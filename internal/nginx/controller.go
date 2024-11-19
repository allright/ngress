package nginx

import (
	"bytes"
	"fmt"
	core "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"ngress/internal/utils"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Controller struct {
	mu      sync.Mutex
	chStop  chan struct{}
	factory informers.SharedInformerFactory

	ingresses map[string]*Ingress
	hosts     map[string]*Host
	services  map[string]struct{}
	certs     map[string][]byte

	secrets       *Secrets
	opts          *Opts
	debounced     func(f func())
	configData    []byte
	certsCheckSum uint32
	hostname      string
}

func NewConfigController(opts *Opts, kubeClient kubernetes.Interface) *Controller {
	hostname, err := os.Hostname()
	if err != nil {
		klog.Fatalf("error getting hostname: %v", err)
	}
	c := &Controller{
		factory:   informers.NewSharedInformerFactory(kubeClient, 30*time.Second),
		chStop:    make(chan struct{}),
		ingresses: make(map[string]*Ingress),
		services:  make(map[string]struct{}),
		hosts:     make(map[string]*Host),
		certs:     make(map[string][]byte),
		secrets:   newSecrets(),
		debounced: utils.NewDebounce(*opts.reloadDebounceInterval),
		opts:      opts,
		hostname:  hostname,
	}

	sharedInformers := []cache.SharedInformer{
		c.factory.Networking().V1().Ingresses().Informer(),
		c.factory.Core().V1().Secrets().Informer(),
		c.factory.Core().V1().Services().Informer(),
	}
	for _, informer := range sharedInformers {
		_, err = informer.AddEventHandler(c)
		if err != nil {
			klog.Fatalf("error: %v, registering informer: %v", err, informer)
		}
	}

	c.factory.Start(c.chStop)
	klog.Infof("started on host: %s success", hostname)

	return c
}

func (c *Controller) Stop() {
	close(c.chStop)
	c.factory.Shutdown()
}

func (c *Controller) add(obj interface{}) {
	switch v := obj.(type) {
	case *networking.Ingress:
		c.OnAddIngress(v)
	case *core.Secret:
		c.secrets.add(v)
	case *core.Service:
		service := serviceString(v)
		klog.Infof("added service: %v", service)
		c.services[service] = struct{}{}
	}
}

func (c *Controller) remove(obj interface{}) {
	switch v := obj.(type) {
	case *networking.Ingress:
		c.OnRemoveIngress(v)
	case *core.Secret:
		c.secrets.remove(v)
	case *core.Service:
		service := serviceString(v)
		klog.Infof("removed service: %v", service)
		delete(c.services, service)
	}
}

func (c *Controller) OnAdd(obj interface{}, _ bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.add(obj)
	c.debounced(c.debounce)
}

func (c *Controller) OnUpdate(old interface{}, obj interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if reflect.DeepEqual(old, obj) {
		return
	}

	c.remove(old)
	c.add(obj)
	c.debounced(c.debounce)
}

func (c *Controller) OnDelete(obj interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.remove(obj)
	c.debounced(c.debounce)
}

func ingressName(ingress *networking.Ingress) string {
	return fmt.Sprintf("%s.%s", ingress.Namespace, ingress.Name)
}

func (c *Controller) OnAddIngress(ingress *networking.Ingress) {
	name := ingressName(ingress)
	ing, ok := c.ingresses[name]
	if !ok {
		ing = newIngress(ingress, c.hosts, c.secrets, c.services, c.hostname)
		c.ingresses[name] = ing
	} else {
		klog.Warningf("INGRESS:%v.%v already exists", ingress.Namespace, ingress.Name)
	}
}

func (c *Controller) OnRemoveIngress(ingress *networking.Ingress) {
	name := ingressName(ingress)
	ing, ok := c.ingresses[name]
	if ok {
		ing.Remove(ingress)
	}
	delete(c.ingresses, name)
}

func (c *Controller) debounce() {
	certs := make(map[string][]byte)
	var sb strings.Builder
	c.secrets.resetWriteMarks() // during buildServers marks for needed secrets will be set
	for _, host := range utils.SortedArrayFromMap(c.hosts) {
		host.buildServers(*c.opts.certsDir, &sb)
	}
	c.secrets.fillCerts(*c.opts.certsDir, certs)
	c.applyNginxConfiguration(sb.String(), certs)
}

func (c *Controller) nginxReload() {
	pid, err := utils.GetPidFromFile(*c.opts.pidFileName)
	if err != nil {
		klog.Errorf("error getting nginx pid: %v", err)
		return
	}

	klog.Infof("sending 'nginx -s reload' to nginx pid: %d", pid)
	err = syscall.Kill(pid, syscall.SIGHUP)
	if err != nil {
		klog.Errorf("error send SIGHUP to nginx: %v pid: %v", err, pid)
		return
	}
}

func (c *Controller) writeCerts() {
	_ = utils.RemoveGlob(fmt.Sprintf("%v/*", *c.opts.certsDir))

	for path, cert := range c.certs {
		dir := filepath.Dir(path)
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			klog.Errorf("error: %v, creating directory: %v", err, dir)
		}
		err = os.WriteFile(path, cert, 0644)
		if err != nil {
			klog.Errorf("error: %v, writing certificate to file: %v", err, path)
		} else {
			klog.Infof("success writing certificate to file: %v", path)
		}
	}
}

func (c *Controller) applyNginxConfiguration(config string, certs map[string][]byte) {

	certsChanged := !reflect.DeepEqual(c.certs, certs)
	c.certs = certs
	if certsChanged {
		klog.Infof("certificates changed")
		c.writeCerts()
	}

	configData := []byte(config)
	configChanged := bytes.Compare(c.configData, configData) != 0
	if configChanged {
		c.configData = configData
		klog.Infof("nginx config changed\n%v", config)

		path := filepath.Join(*c.opts.confDir, "ngress.conf")
		err := os.WriteFile(path, configData, 0644)
		if err != nil {
			klog.Errorf("error writing ngress.conf: %v", err)
			return
		}
	} else {
		klog.Infof("nginx config not changed")
	}

	if configChanged || certsChanged {
		c.nginxReload()
	}
}

func servicePort(v *core.Service) int32 {
	var port int32
	if len(v.Spec.Ports) > 0 {
		port = v.Spec.Ports[0].Port
	}
	return port
}

func serviceString(v *core.Service) string {
	return fmt.Sprintf("%v.%v:%v", v.Name, v.Namespace, servicePort(v))
}
