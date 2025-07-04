package html

import (
	"context"
	"log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

type Ekswatch struct {
	Name   string `json:"name"`
	Synced string `json:"synced,omitempty"`
	// Add other fields as necessary
}

func GetEkswatches() []Ekswatch {

	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error creating dynamic client: %s", err.Error())
	}

	// Define the GroupVersionResource for your CRD. Replace with your actual values.
	gvr := schema.GroupVersionResource{
		Group:    "ekstools.devops.automation",
		Version:  "v1alpha1",
		Resource: "ekswatches", // Plural name of your CRD
	}

	// Get a list of all Foo resources.
	unstructuredList, err := dynamicClient.Resource(gvr).Namespace("ekswatch-operator-system-tilt").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Error getting Foo resources: %s", err.Error())
	}

	returnedEkswatches := make([]Ekswatch, 0, len(unstructuredList.Items))
	for _, item := range unstructuredList.Items {
		// Extract the name of the resource from the unstructured object.
		name := item.GetName()
		returnedEkswatches = append(returnedEkswatches, Ekswatch{
			Name:   name,
			Synced: "N/A",
		})
	}

	return returnedEkswatches

	// // creates the in-cluster config
	// config, err := rest.InClusterConfig()
	// if err != nil {
	// 	panic(err.Error())
	// }
	// // creates the clientset
	// clientset, err := kubernetes.NewForConfig(config)
	// if err != nil {
	// 	panic(err.Error())
	// }

	// ekswatches, err := clientset.ekstools.devoops.automation.v1alpha1().Ekswatches("").List(context.TODO(), metav1.ListOptions{})
	// if err != nil {
	// 	panic(err.Error())
	// }

	// returnedEkswatches := make([]string, 0, len(ekswatches.Items))
	// for _, ekswatch := range ekswatches.Items {
	// 	returnedEkswatches = append(returnedEkswatches, ekswatch.Name)
	// }

	// return returnedEkswatches

	// pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	// if err != nil {
	// 	panic(err.Error())
	// }

	// returnedPods := make([]string, 0, len(pods.Items))
	// for _, pod := range pods.Items {
	// 	returnedPods = append(returnedPods, pod.Name)
	// }

	// return returnedPods

	// for {
	// 	// get pods in all the namespaces by omitting namespace
	// 	// Or specify namespace to get pods in particular namespace
	// 	pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	// 	if err != nil {
	// 		panic(err.Error())
	// 	}
	// 	fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))

	// 	// Examples for error handling:
	// 	// - Use helper functions e.g. errors.IsNotFound()
	// 	// - And/or cast to StatusError and use its properties like e.g. ErrStatus.Message
	// 	_, err = clientset.CoreV1().Pods("default").Get(context.TODO(), "example-xxxxx", metav1.GetOptions{})
	// 	if errors.IsNotFound(err) {
	// 		fmt.Printf("Pod example-xxxxx not found in default namespace\n")
	// 	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
	// 		fmt.Printf("Error getting pod %v\n", statusError.ErrStatus.Message)
	// 	} else if err != nil {
	// 		panic(err.Error())
	// 	} else {
	// 		fmt.Printf("Found example-xxxxx pod in default namespace\n")
	// 	}

	// 	time.Sleep(10 * time.Second)
	// }
}
