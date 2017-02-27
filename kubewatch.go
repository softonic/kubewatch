package main

import (

	// Stdlib:
	"flag"
	"fmt"
	"time"

	// Kubernetes:
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {

	// Handle flags:
	kubeconfig := flag.String("kubeconfig", "./config", "absolute path to the kubeconfig file")
	flag.Parse()

	// Uses the current context in kubeconfig:
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// Creates the clientset:
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	// Watch for services in the default namespace:
	watchlist := cache.NewListWatchFromClient(
		clientset.Core().RESTClient(),
		"services", v1.NamespaceDefault,
		fields.Everything())

	_, controller := cache.NewInformer(
		watchlist,
		&v1.Service{},
		time.Second*0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				fmt.Printf("service added: %s \n", obj)
			},
			DeleteFunc: func(obj interface{}) {
				fmt.Printf("service deleted: %s \n", obj)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				fmt.Printf("service changed \n")
			},
		},
	)

	stop := make(chan struct{})
	go controller.Run(stop)

	for {
		time.Sleep(time.Second)
	}
}
