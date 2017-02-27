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
	"k8s.io/client-go/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {

	// Handle flags:
	kubeconfig := flag.String("kubeconfig", "./config", "Absolute path to the kubeconfig file.")
	resource := flag.String("resource", "services", "Set the resource type to be watched.")
	namespace := flag.String("namespace", "default", "Set the namespace to be watched.")
	flag.Parse()

	// Map resource to runtime object:
	m := map[string]runtime.Object{
		"pods":     &v1.Pod{},
		"services": &v1.Service{},
	}

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

	// Watch for resource in namespace:
	watchlist := cache.NewListWatchFromClient(
		clientset.Core().RESTClient(),
		*resource, *namespace,
		fields.Everything())

	_, controller := cache.NewInformer(
		watchlist,
		m[*resource],
		time.Second*0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				fmt.Printf("%s added: %s \n", *resource, obj)
			},
			DeleteFunc: func(obj interface{}) {
				fmt.Printf("%s deleted: %s \n", *resource, obj)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				fmt.Printf("%s changed \n", *resource)
			},
		},
	)

	stop := make(chan struct{})
	go controller.Run(stop)

	for {
		time.Sleep(time.Second)
	}
}
