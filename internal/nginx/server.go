package nginx

import (
	"fmt"
	"strings"
)

type Server struct {
	*ProtoOpts
	addAltSvc         bool
	name              string
	sslCertPath       string
	sslCertKeyPath    string
	routes            []*Route
	blockRootLocation bool
}

func newServer(name string, opts *ProtoOpts) *Server {
	return &Server{
		name:      name,
		ProtoOpts: opts,
		routes:    make([]*Route, 0),
	}
}

func (c *Server) writeServerForwardBlock(sb *strings.Builder) {
	sb.WriteString(fmt.Sprintf(`
server { 
 listen %v;
 server_name %v;
 return 301 https://$host$request_uri; 
}
`, c.ProtoOpts.unsecurePort, c.name))
}

func (c *Server) addRoute(route *Route) {
	c.routes = append(c.routes, route)
}

func (c *Server) addBlockRootLocation() {
	c.blockRootLocation = true
}

func (c *Server) write(sb *strings.Builder) {
	// makes two server blocks -> redirect from unsecure to https
	https := len(c.sslCertPath) > 0 && len(c.sslCertKeyPath) > 0
	if len(c.routes) == 0 {
		return
	}

	if !https {
		sb.WriteString("\nserver { ")
		sb.WriteString(fmt.Sprintf(`
 listen %v;
 server_name %v;`,
			c.unsecurePort, c.name))
	} else {
		c.writeServerForwardBlock(sb)
		sb.WriteString(`
server {
 ssl_protocols TLSv1.2 TLSv1.3;
 ssl_session_timeout 10m;
 ssl_session_cache shared:SSL:10m;`)

		sb.WriteString(fmt.Sprintf(`
 listen %v ssl;
 server_name %v;`, c.securePort, c.name))

		if c.http2 {
			sb.WriteString("\n http2 on;")
		}

		if c.http3 {
			sb.WriteString(fmt.Sprintf(`
 http3 on;
 listen %v quic reuseport;
 ssl_early_data on;`,
				c.securePort))
		}

		sb.WriteString(fmt.Sprintf(`
 ssl_certificate %v;
 ssl_certificate_key %v;
`,
			c.sslCertPath,
			c.sslCertKeyPath))
	}

	if c.blockRootLocation {
		locationRoot444(sb)
	}

	// alt-svc header actual only for https!
	for _, location := range c.routes {
		location.write(c.addAltSvc, c.ProtoOpts, sb)
	}

	sb.WriteString("\n}\n")
}
