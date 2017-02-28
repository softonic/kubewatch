package main

//-----------------------------------------------------------------------------
// Package factored import statement:
//-----------------------------------------------------------------------------

import (

	// Stdlib:
	"encoding/json"
	"fmt"
	"os"
	"time"

	// Kubernetes:
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"

	// Community:
	"gopkg.in/alecthomas/kingpin.v2"
)

//-----------------------------------------------------------------------------
// Setup command and flags:
//-----------------------------------------------------------------------------

var (

	// Root level command:
	app = kingpin.New("kubewatch", "Watches Kubernetes resources via its API")

	// Flags:
	kubeconfig = app.Flag("kubeconfig",
		"Absolute path to the kubeconfig file.").
		Default(os.Getenv("HOME") + "/.kube/config").ExistingFile()

	resource = app.Flag("resource",
		"Set the resource type to be watched.").
		Default("services").String()

	namespace = app.Flag("namespace",
		"Set the namespace to be watched.").
		Default(v1.NamespaceAll).String()
)

//-----------------------------------------------------------------------------
// func init() is called after all the variable declarations in the package
// have evaluated their initializers, and those are evaluated only after all
// the imported packages have been initialized:
//-----------------------------------------------------------------------------

func init() {

	// Customize kingpin:
	app.Version("v0.1.0").Author("Marc Villacorta Morera")
	app.UsageTemplate(usageTemplate)
	app.HelpFlag.Short('h')
}

//-----------------------------------------------------------------------------
// Entry point:
//-----------------------------------------------------------------------------

func main() {

	// Parse command flags:
	kingpin.MustParse(app.Parse(os.Args[1:]))

	// Map resource to runtime object:
	m := map[string]runtime.Object{

		// v1:
		"configMaps":             &v1.ConfigMap{},
		"endpoints":              &v1.Endpoints{},
		"events":                 &v1.Event{},
		"limitranges":            &v1.LimitRange{},
		"namespaces":             &v1.Namespace{},
		"persistentvolumeclaims": &v1.PersistentVolumeClaim{},
		"persistentvolumes":      &v1.PersistentVolume{},
		"pods":                   &v1.Pod{},
		"podtemplates":           &v1.PodTemplate{},
		"replicationcontrollers": &v1.ReplicationController{},
		"resourcequotas":         &v1.ResourceQuota{},
		"secrets":                &v1.Secret{},
		"serviceaccounts":        &v1.ServiceAccount{},
		"services":               &v1.Service{},

		// v1beta1:
		"deployments":              &v1beta1.Deployment{},
		"horizontalpodautoscalers": &v1beta1.HorizontalPodAutoscaler{},
		"ingresses":                &v1beta1.Ingress{},
		"jobs":                     &v1beta1.Job{},
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

	// Controller providing event notifications:
	_, controller := cache.NewInformer(
		watchlist,
		m[*resource],
		time.Second*0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				jsn, _ := json.Marshal(obj)
				fmt.Printf("%s added: %s\n", *resource, jsn)
			},
			DeleteFunc: func(obj interface{}) {
				jsn, _ := json.Marshal(obj)
				fmt.Printf("%s deleted: %s\n", *resource, jsn)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				fmt.Printf("%s changed\n", *resource)
			},
		},
	)

	stop := make(chan struct{})
	go controller.Run(stop)

	// Loop forever:
	for {
		time.Sleep(time.Second)
	}
}
