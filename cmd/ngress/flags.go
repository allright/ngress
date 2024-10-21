package main

import (
	"flag"
	"ngress/internal/nginx"
)

type flags struct {
	kubeconfig *string
	masterURL  *string
	nginx      *nginx.Opts
}

func newFlags() *flags {
	return &flags{
		kubeconfig: flag.String("kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster."),
		masterURL:  flag.String("master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster."),
		nginx:      nginx.NewOpts(),
	}
}
