package nginx

import "fmt"

const defaultSecurePort = 443
const defaultUnsecurePort = 80

type ProtoOpts struct {
	http2        bool
	http3        bool
	websocket    bool
	unsecurePort uint16
	securePort   uint16
}

func (c *ProtoOpts) string() string {
	s := "["
	if c.http2 {
		s += " HTTP2"
	}
	if c.http3 {
		s += " HTTP3"
	}
	if c.websocket {
		s += " WebSocket"
	}
	if c.securePort != defaultSecurePort {
		s += fmt.Sprintf(" securePort: %d", c.securePort)
	}
	if c.unsecurePort != defaultUnsecurePort {
		s += fmt.Sprintf(" unsecurePort: %d", c.unsecurePort)
	}
	return s + " ]"
}

func (c *ProtoOpts) merge(a *ProtoOpts) {
	if a.securePort != defaultSecurePort {
		c.securePort = a.securePort
	}
	if a.unsecurePort != defaultUnsecurePort {
		c.unsecurePort = a.unsecurePort
	}
	if a.http2 {
		c.http2 = a.http2
	}
	if a.http3 {
		c.http3 = a.http3
	}
	if a.websocket {
		c.websocket = a.websocket
	}
}
