package nginx

import (
	"k8s.io/klog/v2"
	"strconv"
)

type Annotations struct {
	proto        ProtoOpts
	hostAffinity string
	unixSocket   string
	staticSite   string
}

func parsePort(annotations map[string]string, key string, portType string, defaultValue uint16) uint16 {
	portStr := annotations[key]
	if len(portStr) > 0 {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			klog.Warningf("failed to parse nginx %v port %s: %v, use default", portType, portStr, err)
			return defaultValue
		}
		return uint16(port)
	}
	return defaultValue
}

func newAnnotations(annotations map[string]string) *Annotations {
	proto := ProtoOpts{
		http2:        annotations["ngress.proto/http2"] == "true",
		http3:        annotations["ngress.proto/http3"] == "true",
		websocket:    annotations["ngress.proto/websocket"] == "true",
		unsecurePort: parsePort(annotations, "ngress.port/unsecure", "secure", defaultUnsecurePort),
		securePort:   parsePort(annotations, "ngress.port/secure", "secure", defaultSecurePort),
	}
	return &Annotations{
		proto:        proto,
		hostAffinity: annotations["ngress.affinity/host"],
		unixSocket:   annotations["ngress.unix/socket"],
		staticSite:   annotations["ngress.static/site"],
	}
}

func (c *Annotations) string() string {
	s := "["
	s += c.proto.string()
	if c.hostAffinity != "" {
		s += " hostAffinity: " + c.hostAffinity
	}
	if c.unixSocket != "" {
		s += " unixSocket: " + c.unixSocket
	}
	return s + " ]"
}

func (c *Annotations) merge(a *Annotations) {
	if len(a.hostAffinity) > 0 {
		c.hostAffinity = a.hostAffinity
	}
	if len(a.unixSocket) > 0 {
		c.unixSocket = a.unixSocket
	}
	c.proto.merge(&a.proto)
}
