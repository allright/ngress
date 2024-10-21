package main

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"ngress/internal/nginx"
)

func main() {
	f := newFlags()
	project := NewProject("ngress")
	defer project.Close()

	cfg, err := clientcmd.BuildConfigFromFlags(*f.masterURL, *f.kubeconfig)
	if err != nil {
		klog.Fatalf("error building kubeconfig: %v", err)
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("error building kubernetes clientset: %v", err)
	}

	configController := nginx.NewConfigController(f.nginx, kubeClient)
	project.WaitStopSignal()
	configController.Stop()
}
