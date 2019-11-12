package main

//-----------------------------------------------------------------------------
// Package factored import statement:
//-----------------------------------------------------------------------------

import (

	// Stdlib:
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"reflect"
	"time"

	// Kubernetes:
	apps "k8s.io/api/apps/v1"
	autoscaling "k8s.io/api/autoscaling/v1"
	batch "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"

	// Community:
	log "github.com/sirupsen/logrus"
)

//-----------------------------------------------------------------------------
// Command, flags and arguments:
//-----------------------------------------------------------------------------

var (

	// Resources:
	resources = []string{
		"configMaps", "endpoints", "events", "limitranges", "replicasets",
		"persistentvolumeclaims", "persistentvolumes", "pods", "podtemplates",
		"replicationcontrollers", "resourcequotas", "secrets", "serviceaccounts",
		"services", "deployments", "nodes", "horizontalpodautoscalers", "damonsets", "statefulsets", "ingresses", "jobs"}

	flgKubeconfig = flag.String("config", kubeconfigPath(), "a string")

	flgNamespace = flag.String("namespace", "default", "a string")

	flgAllNamespaces = flag.Bool("allnamespaces", false, "a bool")

	flgFlatten = flag.Bool("flatten", true, "a bool")

	argResources = flag.Args()

	namespacestring = "default"
)

//-----------------------------------------------------------------------------
// Types and structs:
//-----------------------------------------------------------------------------

type verObj struct {
	apiVersion    string
	runtimeObject runtime.Object
}

type strIfce map[string]interface{}

//-----------------------------------------------------------------------------
// Map resources to runtime objects:
//-----------------------------------------------------------------------------

var resourceObject = map[string]verObj{

	// v1:
	"configMaps":             {"v1", &v1.ConfigMap{}},
	"endpoints":              {"v1", &v1.Endpoints{}},
	"events":                 {"v1", &v1.Event{}},
	"limitranges":            {"v1", &v1.LimitRange{}},
	"persistentvolumeclaims": {"v1", &v1.PersistentVolumeClaim{}},
	"persistentvolumes":      {"v1", &v1.PersistentVolume{}},
	"pods":                   {"v1", &v1.Pod{}},
	"podtemplates":           {"v1", &v1.PodTemplate{}},
	"replicationcontrollers": {"v1", &v1.ReplicationController{}},
	"resourcequotas":         {"v1", &v1.ResourceQuota{}},
	"secrets":                {"v1", &v1.Secret{}},
	"serviceaccounts":        {"v1", &v1.ServiceAccount{}},
	"services":               {"v1", &v1.Service{}},
	"nodes":                  {"v1", &v1.Node{}},

	// apps :
	"deployments":  {"apps", &apps.Deployment{}},
	"statefulsets": {"apps", &apps.StatefulSet{}},
	"daemonsets":   {"apps", &apps.DaemonSet{}},

	// autoscaling :
	"horizontalpodautoscalers": {"autoscaling", &autoscaling.HorizontalPodAutoscaler{}},

	// v1beta1 :
	"ingresses":   {"v1beta1", &extensions.Ingress{}},
	"replicasets": {"v1beta1", &extensions.Ingress{}},
	"jobs":        {"batch", &batch.Job{}},
}

//-----------------------------------------------------------------------------
// func init() is called after all the variable declarations in the package
// have evaluated their initializers, and those are evaluated only after all
// the imported packages have been initialized:
//-----------------------------------------------------------------------------

func init() {

	// Customize the default logger:
	log.SetFormatter(&log.TextFormatter{ForceColors: true})
	log.SetOutput(os.Stderr)
	log.SetLevel(log.InfoLevel)
}

//-----------------------------------------------------------------------------
// Entry point:
//-----------------------------------------------------------------------------

func main() {

	flag.Parse()

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
	for _, resource := range flag.Args() {
		watchResource(clientset, resource, *flgNamespace, *flgAllNamespaces)
	}

	// Block forever:
	select {}
}

//-----------------------------------------------------------------------------
// watchResource:
//-----------------------------------------------------------------------------

func watchResource(clientset *kubernetes.Clientset, resource string, namespace string, all bool) {

	var client rest.Interface

	var definition string

	// Set the API endpoint:
	switch resourceObject[resource].apiVersion {
	case "v1":
		client = clientset.CoreV1().RESTClient()
	case "v1beta1":
		client = clientset.ExtensionsV1beta1().RESTClient()
	case "apps":
		client = clientset.AppsV1().RESTClient()
	case "autoscaling":
		client = clientset.AutoscalingV1().RESTClient()
	case "batch":
		client = clientset.BatchV1().RESTClient()
	}

	switch all {
	case true:
		definition = ""
	case false:
		definition = namespace
	}

	listWatch := cache.NewListWatchFromClient(
		client, resource, definition,
		fields.Everything())

	// Ugly hack to suppress sync events:
	listWatch.ListFunc = func(options metav1.ListOptions) (runtime.Object, error) {
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

	// Log this watch:
	log.WithField("type", resource).Info("Watching for new resources")

	// Start the controller:
	go controller.Run(wait.NeverStop)
}

//-----------------------------------------------------------------------------
// printEvent:
//-----------------------------------------------------------------------------

func printEvent(obj interface{}) {

	// Variables:
	var jsn []byte
	var err error

	// Marshal obj into JSON:
	if jsn, err = json.Marshal(obj); err != nil {
		log.Error("Ops! Cannot marshal JSON")
		return
	}

	if *flgFlatten == true {

		// Unmarshal JSON into dat:
		dat := strIfce{}
		if err = json.Unmarshal(jsn, &dat); err != nil {
			log.Error("Ops! Cannot unmarshal JSON")
			return
		}

		// Flatten dat into r:
		r := strIfce{}
		flatten(r, "kubewatch", reflect.ValueOf(dat))

		// Marshal r into JSON:
		if jsn, err = json.Marshal(r); err != nil {
			log.Error("Ops! Cannot marshal JSON")
			return
		}
	}

	// Print to stdout:
	fmt.Printf("%s\n", jsn)
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

		// Log and return:
		log.WithField("file", kubeconfig).Info("Running out-of-cluster using kubeconfig")
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}

	// ...otherwise assume in-cluster:
	log.Info("Running in-cluster using environment variables")
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
	l, err := clientset.CoreV1().Namespaces().List(metav1.ListOptions{})
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

func flatten(r strIfce, p string, v reflect.Value) {

	// Append '_' to prefix:
	if p != "" {
		p = p + "_"
	}

	// Set the value:
	if v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		v = v.Elem()
	}

	// Return if !valid:
	if !v.IsValid() {
		return
	}

	// Set the type:
	t := v.Type()

	// Flatten each type kind:
	switch t.Kind() {
	case reflect.Bool:
		flattenBool(v, p, r)
	case reflect.Float64:
		flattenFloat64(v, p, r)
	case reflect.Map:
		flattenMap(v, p, r)
	case reflect.Slice:
		flattenSlice(v, p, r)
	case reflect.String:
		flattenString(v, p, r)
	default:
		log.Error("Unknown: " + p)
	}
}

//-----------------------------------------------------------------------------
// flattenBool:
//-----------------------------------------------------------------------------

func flattenBool(v reflect.Value, p string, r strIfce) {
	if v.Bool() {
		r[p[:len(p)-1]] = "true"
	} else {
		r[p[:len(p)-1]] = "false"
	}
}

//-----------------------------------------------------------------------------
// flattenFloat64:
//-----------------------------------------------------------------------------

func flattenFloat64(v reflect.Value, p string, r strIfce) {
	r[p[:len(p)-1]] = fmt.Sprintf("%f", v.Float())
}

//-----------------------------------------------------------------------------
// flattenMap:
//-----------------------------------------------------------------------------

func flattenMap(v reflect.Value, p string, r strIfce) {
	for _, k := range v.MapKeys() {
		if k.Kind() == reflect.Interface {
			k = k.Elem()
		}
		if k.Kind() != reflect.String {
			log.Errorf("%s: map key is not string: %s", p, k)
		}
		flatten(r, p+k.String(), v.MapIndex(k))
	}
}

//-----------------------------------------------------------------------------
// flattenSlice:
//-----------------------------------------------------------------------------

func flattenSlice(v reflect.Value, p string, r strIfce) {
	r[p+"#"] = fmt.Sprintf("%d", v.Len())
	for i := 0; i < v.Len(); i++ {
		flatten(r, fmt.Sprintf("%s%d", p, i), v.Index(i))
	}
}

//-----------------------------------------------------------------------------
// flattenString:
//-----------------------------------------------------------------------------

func flattenString(v reflect.Value, p string, r strIfce) {
	r[p[:len(p)-1]] = v.String()
}
