package nginx

import (
	"fmt"
	networking "k8s.io/api/networking/v1"
	"k8s.io/klog/v2"
)

type Ingress struct {
	tag      string
	hosts    map[string]*Host
	secrets  *Secrets
	hostname string
}

func newIngress(
	ingress *networking.Ingress,
	hosts map[string]*Host,
	secrets *Secrets,
	services map[string]struct{},
	hostname string) *Ingress {

	c := &Ingress{
		tag:      fmt.Sprintf("INGRESS:%s.%s", ingress.Namespace, ingress.Name),
		hosts:    hosts,
		secrets:  secrets,
		hostname: hostname,
	}

	annotations := newAnnotations(ingress.Annotations)
	if len(annotations.hostAffinity) > 0 {
		if annotations.hostAffinity != c.hostname {
			klog.Infof("%v > skip %+v, this host: %v != %v ",
				c.tag, annotations.string(), c.hostname, annotations.hostAffinity)
			return c
		}
	}

	for _, r := range ingress.Spec.Rules {
		if r.Host == "" {
			klog.Warningf("%v skip empty host: %v -> %v", c.tag, r.Host, r.HTTP)
			continue
		}
		host, ok := c.hosts[r.Host]
		if !ok {
			host = newHost(&r, secrets, services)
			c.hosts[r.Host] = host
		}
		host.applyAnnotations(annotations)
		host.addRef()
		host.update(ingress.Namespace, &r)
	}

	tlsStr := ""
	for _, tls := range ingress.Spec.TLS {
		sn := makeSecretName(ingress.Namespace, tls.SecretName)
		tlsStr = fmt.Sprintf(", <SECRET:%v> -> %v", sn, tls.Hosts)
		for _, tlsHost := range tls.Hosts {
			host, ok := c.hosts[tlsHost]
			if ok {
				host.attachTLSSecret(sn)
			}
		}
	}

	klog.Infof("%v create %+v%v", c.tag, annotations.string(), tlsStr)
	return c
}

func (c *Ingress) Remove(ingress *networking.Ingress) {
	klog.Infof("%v remove", c.tag)

	for _, tls := range ingress.Spec.TLS {
		sn := makeSecretName(ingress.Namespace, tls.SecretName)
		for _, tlsHost := range tls.Hosts {
			host, ok := c.hosts[tlsHost]
			if ok {
				host.detachTLSSecret(sn)
			}
		}
	}

	for _, r := range ingress.Spec.Rules {
		host, ok := c.hosts[r.Host]
		if ok {
			host.remove(&r)
			if host.removeRef() {
				delete(c.hosts, r.Host)
			}
		}
	}
}
