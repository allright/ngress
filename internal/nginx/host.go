package nginx

import (
	"fmt"
	core "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	"k8s.io/klog/v2"
	"ngress/internal/utils"
	"strings"
)

type Host struct {
	tag           string
	refCount      int
	routes        map[string]*Route
	host          string
	tlsSecretName string
	tlsRefCount   int
	secrets       *Secrets
	annotations   *Annotations
	services      map[string]struct{}
}

func newHost(rule *networking.IngressRule, secrets *Secrets, services map[string]struct{}) *Host {

	c := &Host{
		tag:      fmt.Sprintf("HOST:%v", rule.Host),
		host:     rule.Host,
		routes:   make(map[string]*Route),
		secrets:  secrets,
		services: services,
	}
	return c
}

func (c *Host) applyAnnotations(annotations *Annotations) {
	if c.annotations == nil {
		c.annotations = annotations
	}
	c.annotations.merge(annotations)
}

func (c *Host) attachTLSSecret(secretName string) {
	if len(secretName) == 0 {
		klog.Errorf("%v> secret name is empty", c.tag)
		return
	}

	if len(c.tlsSecretName) == 0 {
		c.tlsSecretName = secretName
	} else {
		if secretName != c.tlsSecretName {
			klog.Errorf("%v> try to add incorrect secret, expected %v, got %v", c.tag, c.tlsSecretName, secretName)
			return
		}
	}
	if c.tlsRefCount == 0 {
		klog.Infof("%v> added TLS secret: %v", c.tag, secretName)
	}

	c.tlsRefCount++
	c.tlsSecretName = secretName
}

func (c *Host) detachTLSSecret(secretName string) {
	if secretName != c.tlsSecretName {
		klog.Errorf("%v> try to remove incorrect secret, expected %v, got %v", c.tag, c.tlsSecretName, secretName)
		return
	}
	if c.tlsRefCount > 0 {
		c.tlsRefCount--
		if c.tlsRefCount == 0 {
			c.tlsSecretName = ""
			klog.Infof("%v> removed TLS secret: %v", c.tag, secretName)
		}
	}
}

func (c *Host) addRef() {
	c.refCount++
}

func (c *Host) removeRef() bool {
	if c.refCount > 0 {
		c.refCount--
	}
	return c.refCount == 0
}

func (c *Host) update(namespace string, rule *networking.IngressRule) {
	for _, path := range rule.HTTP.Paths {
		r, ok := c.routes[path.Path]
		if !ok {
			route := &Route{path: &path, namespace: namespace}
			c.routes[path.Path] = route
			if len(path.Backend.Service.Port.Name) > 0 {
				if len(c.annotations.unixSocket) > 0 {
					route.unixSocket = utils.GetStringValue(path.Backend.Service.Port.Name, c.annotations.unixSocket)
				}
				if len(c.annotations.staticSite) > 0 {
					route.staticSite = utils.GetStringValue(path.Backend.Service.Port.Name, c.annotations.staticSite)
				}
			}
		} else {
			klog.Errorf("%v> route %v already exist, ignore current", c.tag, r.string())
		}
	}
}

func (c *Host) remove(rule *networking.IngressRule) {
	for _, path := range rule.HTTP.Paths {
		delete(c.routes, path.Path)
	}
}

func (c *Host) buildServers(certsDir string, sb *strings.Builder) {
	server := newServer(c.host, &c.annotations.proto)

	secret := c.secrets.get(c.tlsSecretName)
	if secret == nil {
		klog.Errorf("%v> SECRET:%v NOT found", c.tag, c.tlsSecretName)
	} else {
		klog.Infof("%v> found <SECRET:%v>(%v)", c.tag, secret.name(), secret.string())
		server.sslCertPath = secret.path(certsDir, core.TLSCertKey)
		server.sslCertKeyPath = secret.path(certsDir, core.TLSPrivateKeyKey)
		secret.markForWrite(true)
	}
	server.addAltSvc = secret != nil // to prevent add altSvc to 80 port (unsecure)

	haveRootPath := false

	// sort alphabetically for correct nginx.conf comparing
	for _, key := range utils.SortedKeys(c.routes) {
		route := c.routes[key]

		if !haveRootPath && route.path.Path == "/" {
			haveRootPath = true
		}

		if route.isRouteToService() {
			// check for service is exist or not ?
			_, ok := c.services[route.destination()]
			if !ok {
				klog.Warningf("%v> %v service not found, route skipped", c.tag, route.string())
				continue
			}
		}

		klog.Infof("%v> %v", c.tag, route.string())

		server.addRoute(route)
	}
	if !haveRootPath {
		server.addBlockRootLocation()
	}

	server.write(sb)
}
