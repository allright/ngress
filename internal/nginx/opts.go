package nginx

import (
	"flag"
	"time"
)

type Opts struct {
	confDir                *string
	certsDir               *string
	pidFileName            *string
	reloadDebounceInterval *time.Duration
}

func NewOpts() *Opts {
	return &Opts{
		confDir:     flag.String("nginx-confd-dir", "", "path to nginx conf.d directory"),
		certsDir:    flag.String("nginx-certs-dir", "", "path to nginx certs directory"),
		pidFileName: flag.String("nginx-pid-file", "", "nginx pid file location"),
		reloadDebounceInterval: flag.Duration("nginx-reload-debounce-interval",
			10*time.Second,
			"nginx reload debounce interval for prevent very often nginx config reloads on configurations change"),
	}
}
