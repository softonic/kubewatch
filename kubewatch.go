package main

//-----------------------------------------------------------------------------
// Package factored import statement:
//-----------------------------------------------------------------------------

import (

	// Stdlib:
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"time"

	// Kubernetes:
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/pkg/runtime"
	"k8s.io/client-go/pkg/util/wait"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"

	// Community:
	"gopkg.in/alecthomas/kingpin.v2"
)

//-----------------------------------------------------------------------------
// Command, flags and arguments:
//-----------------------------------------------------------------------------

var (

	// Root level command:
	app = kingpin.New("kubewatch", "Watches Kubernetes resources via its API.")

	// Resources:
	resources = []string{
		"configMaps", "endpoints", "events", "limitranges",
		"persistentvolumeclaims", "persistentvolumes", "pods", "podtemplates",
		"replicationcontrollers", "resourcequotas", "secrets", "serviceaccounts",
		"services", "deployments", "horizontalpodautoscalers", "ingresses", "jobs"}

	// Flags:
	flgKubeconfig = app.Flag("kubeconfig",
		"Absolute path to the kubeconfig file.").
		Default(kubeconfigPath()).ExistingFileOrDir()

	flgNamespace = app.Flag("namespace",
		"Set the namespace to be watched.").
		Default(v1.NamespaceAll).HintAction(listNamespaces).String()

	flgFlatten = app.Flag("flatten",
		"Whether to produce flatten JSON output or not.").Bool()

	// Arguments:
	argResources = app.Arg("resources",
		"Space delimited list of resources to be watched.").
		Required().HintOptions(resources...).Enums(resources...)
)

//-----------------------------------------------------------------------------
// Type structs:
//-----------------------------------------------------------------------------

type verObj struct {
	apiVersion    string
	runtimeObject runtime.Object
}

//-----------------------------------------------------------------------------
// Map resources to runtime objects:
//-----------------------------------------------------------------------------

var resourceObject = map[string]verObj{

	// v1:
	"configMaps":             verObj{"v1", &v1.ConfigMap{}},
	"endpoints":              verObj{"v1", &v1.Endpoints{}},
	"events":                 verObj{"v1", &v1.Event{}},
	"limitranges":            verObj{"v1", &v1.LimitRange{}},
	"persistentvolumeclaims": verObj{"v1", &v1.PersistentVolumeClaim{}},
	"persistentvolumes":      verObj{"v1", &v1.PersistentVolume{}},
	"pods":                   verObj{"v1", &v1.Pod{}},
	"podtemplates":           verObj{"v1", &v1.PodTemplate{}},
	"replicationcontrollers": verObj{"v1", &v1.ReplicationController{}},
	"resourcequotas":         verObj{"v1", &v1.ResourceQuota{}},
	"secrets":                verObj{"v1", &v1.Secret{}},
	"serviceaccounts":        verObj{"v1", &v1.ServiceAccount{}},
	"services":               verObj{"v1", &v1.Service{}},

	// v1beta1:
	"deployments":              verObj{"v1beta1", &v1beta1.Deployment{}},
	"horizontalpodautoscalers": verObj{"v1beta1", &v1beta1.HorizontalPodAutoscaler{}},
	"ingresses":                verObj{"v1beta1", &v1beta1.Ingress{}},
	"jobs":                     verObj{"v1beta1", &v1beta1.Job{}},
}

//-----------------------------------------------------------------------------
// func init() is called after all the variable declarations in the package
// have evaluated their initializers, and those are evaluated only after all
// the imported packages have been initialized:
//-----------------------------------------------------------------------------

func init() {

	// Customize kingpin:
	app.Version("v0.3.1").Author("Marc Villacorta Morera")
	app.UsageTemplate(usageTemplate)
	app.HelpFlag.Short('h')
}

//-----------------------------------------------------------------------------
// Entry point:
//-----------------------------------------------------------------------------

func main() {

	// Parse command flags:
	kingpin.MustParse(app.Parse(os.Args[1:]))

	// Build the config:
	config, err := buildConfig(*flgKubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// Create the clientset:
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	// Watch for the given resource:
	for _, resource := range *argResources {
		watchResource(clientset, resource, *flgNamespace)
	}

	// Loop forever:
	for {
		time.Sleep(time.Second)
	}
}

//-----------------------------------------------------------------------------
// watchResource:
//-----------------------------------------------------------------------------

func watchResource(clientset *kubernetes.Clientset, resource, namespace string) {

	var client rest.Interface

	// Set the API endpoint:
	switch resourceObject[resource].apiVersion {
	case "v1":
		client = clientset.Core().RESTClient()
	case "v1beta1":
		client = clientset.Extensions().RESTClient()
	}

	// Watch for resource in namespace:
	listWatch := cache.NewListWatchFromClient(
		client, resource, namespace,
		fields.Everything())

	// Ugly hack to suppress sync events:
	listWatch.ListFunc = func(options api.ListOptions) (runtime.Object, error) {
		return client.Get().Namespace("none").Resource(resource).Do().Get()
	}

	// Controller providing event notifications:
	_, controller := cache.NewInformer(
		listWatch, resourceObject[resource].runtimeObject,
		time.Second*0, cache.ResourceEventHandlerFuncs{
			AddFunc:    printEvent,
			DeleteFunc: printEvent,
		},
	)

	// Start the controller:
	go controller.Run(wait.NeverStop)
}

//-----------------------------------------------------------------------------
// printEvent:
//-----------------------------------------------------------------------------

func printEvent(obj interface{}) {
	if *flgFlatten {
		flatten(reflect.ValueOf(obj), "kubewatch")
	} else if jsn, err := json.Marshal(obj); err == nil {
		fmt.Printf("%s\n", jsn)
	}
}

//-----------------------------------------------------------------------------
// kubeconfigPath:
//-----------------------------------------------------------------------------

func kubeconfigPath() (path string) {

	// Return ~/.kube/config if exists...
	if _, err := os.Stat(os.Getenv("HOME") + "/.kube/config"); err == nil {
		return os.Getenv("HOME") + "/.kube/config"
	}

	// ...otherwise return '.':
	return "."
}

//-----------------------------------------------------------------------------
// buildConfig:
//-----------------------------------------------------------------------------

func buildConfig(kubeconfig string) (*rest.Config, error) {

	// Use kubeconfig if given...
	if kubeconfig != "" && kubeconfig != "." {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}

	// ...otherwise assume in-cluster:
	return rest.InClusterConfig()
}

//-----------------------------------------------------------------------------
// listNamespaces:
//-----------------------------------------------------------------------------

func listNamespaces() (list []string) {

	// Build the config:
	config, err := buildConfig(*flgKubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// Create the clientset:
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	// Get the list of namespace objects:
	l, err := clientset.Namespaces().List(v1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	// Extract the name of each namespace:
	for _, v := range l.Items {
		list = append(list, v.Name)
	}

	return
}

//-----------------------------------------------------------------------------
// flatten:
//-----------------------------------------------------------------------------

func flatten(v reflect.Value, prefix string) {

	// Append '.' to prefix:
	if prefix != "" {
		prefix = prefix + "."
	}

	// Set the value:
	if v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		v = v.Elem()
	}

	// Set the type:
	t := v.Type()

	// Flatten each type kind:
	switch t.Kind() {
	case reflect.Bool:
		flattenBool(prefix)
	case reflect.Int:
		flattenInt(prefix)
	case reflect.Map:
		flattenMap(prefix)
	case reflect.Slice:
		flattenSlice(prefix)
	case reflect.String:
		flattenString(prefix)
	case reflect.Struct:
		flattenStruct(t, v, prefix)
	}
}

//-----------------------------------------------------------------------------
// flattenBool:
//-----------------------------------------------------------------------------

func flattenBool(prefix string) {
	fmt.Println("Bool: " + prefix)
}

//-----------------------------------------------------------------------------
// flattenInt:
//-----------------------------------------------------------------------------

func flattenInt(prefix string) {
	fmt.Println("Int: " + prefix)
}

//-----------------------------------------------------------------------------
// flattenMap:
//-----------------------------------------------------------------------------

func flattenMap(prefix string) {
	fmt.Println("Map: " + prefix)
}

//-----------------------------------------------------------------------------
// flattenSlice:
//-----------------------------------------------------------------------------

func flattenSlice(prefix string) {
	fmt.Println("Slice: " + prefix)
}

//-----------------------------------------------------------------------------
// flattenString:
//-----------------------------------------------------------------------------

func flattenString(prefix string) {
	fmt.Println("String: " + prefix)
}

//-----------------------------------------------------------------------------
// flattenStruct:
//-----------------------------------------------------------------------------

func flattenStruct(t reflect.Type, v reflect.Value, prefix string) {
	fmt.Println("Struct: " + prefix)
	for i := 0; i < v.NumField(); i++ {
		childValue := v.Field(i)
		childKey := t.Field(i).Name
		flatten(childValue.Addr(), prefix+childKey)
	}
}
