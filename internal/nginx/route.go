package nginx

import (
	"fmt"
	networking "k8s.io/api/networking/v1"
	"strings"
)

type Route struct {
	namespace  string
	path       *networking.HTTPIngressPath
	unixSocket string // if not empty -> use unix socket
	staticSite string // if not empty -> use unix socket
}

func (c *Route) destination() string {
	if len(c.staticSite) > 0 {
		return fmt.Sprintf("static-site:%v", c.staticSite)
	}
	if len(c.unixSocket) > 0 {
		return fmt.Sprintf("unix:%v", c.unixSocket)
	}
	return fmt.Sprintf("%v.%v:%v",
		c.path.Backend.Service.Name,
		c.namespace,
		c.path.Backend.Service.Port.Number)
}

func (c *Route) isRouteToService() bool {
	return len(c.staticSite) == 0 && len(c.unixSocket) == 0
}

func (c *Route) upstreamName() string {
	if len(c.staticSite) > 0 {
		return ""
	}
	if len(c.unixSocket) > 0 {
		upstreamName := strings.ReplaceAll(c.unixSocket, "/", "_")
		return fmt.Sprintf("unix%v", upstreamName)
	}
	return fmt.Sprintf("%v_%v_%v",
		c.path.Backend.Service.Name,
		c.namespace,
		c.path.Backend.Service.Port.Number)
}

func (c *Route) location() string {
	p := c.path
	var s strings.Builder
	if *p.PathType == networking.PathTypeExact {
		s.WriteString("= ")
	}
	s.WriteString(p.Path)
	return s.String()
}

func (c *Route) string() string {
	return fmt.Sprintf("'%v' => '%v'", c.location(), c.destination())
}

func (c *Route) write(addAltSvc bool, opts *ProtoOpts, sb *strings.Builder) {
	locationPrefix := ""
	if *c.path.PathType == networking.PathTypeExact {
		locationPrefix = "="
	}
	sb.WriteString(fmt.Sprintf(
		`
 location %s %s {`,
		locationPrefix, c.path.Path))

	if len(c.staticSite) > 0 {
		sb.WriteString(fmt.Sprintf(
			`
  root %v;
  try_files $uri $uri/ /index.html;`,
			c.staticSite))
	} else {
		sb.WriteString(fmt.Sprintf(
			`
  proxy_http_version 1.1;
  proxy_set_header Host $http_host;
  proxy_pass http://%v;`,
			c.destination()))
	}

	if opts.websocket {
		sb.WriteString(`
  proxy_set_header Upgrade $http_upgrade;
  proxy_set_header Connection "upgrade";`)
	}

	if addAltSvc {
		c.addAltSvcHeaderIfNeeded(opts, sb)
	}
	sb.WriteString("\n }\n")
}

func (c *Route) addAltSvcHeaderIfNeeded(opts *ProtoOpts, sb *strings.Builder) {
	http := func(n int, port uint16) string {
		return fmt.Sprintf("h%d=\":%v\"", n, port)
	}

	if opts.http2 || opts.http3 {
		sb.WriteString("\n  add_header Alt-Svc ")
	}

	if opts.http2 {
		sb.WriteString(http(2, opts.securePort))
	}

	if opts.http3 {
		if opts.http2 {
			sb.WriteString(",")
		}
		sb.WriteString(http(3, opts.securePort))
	}

	if opts.http2 || opts.http3 {
		sb.WriteString(";")
	}
}

func locationRoot444(sb *strings.Builder) {
	sb.WriteString(`
 location / {
   return 444;
 }
`)
}
