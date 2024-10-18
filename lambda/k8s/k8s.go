package k8s

import (
	"context"
	"fmt"
	"lambda/logger"
	"os"
	"path/filepath"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/kubectl/pkg/scheme"
)

type ModifyRequest struct {
	Path      []string    // chemin dans l'objet (ex: []string{"spec", "template", "spec"})
	Value     interface{} // valeur à mettre (peut être map[string]string, string, int, etc)
	Operation string      // "update", "merge", "delete" par exemple
}

func GetKubeConfigPath() string {
	userHomeDir, err := os.UserHomeDir()
	logger.ErrHandle(err)

	kubeConfigPath := filepath.Join(userHomeDir, ".kube", "config")
	return kubeConfigPath
}

func GetKubeConfigFromFile(kubeConfigPath string) *api.Config {
	config, err := clientcmd.LoadFromFile(kubeConfigPath)
	logger.ErrHandle(err)
	return config
}

func GetKubeContexts(config *api.Config) []string {
	kubeContexts := make([]string, 0)
	for kubeContext := range config.Contexts {
		kubeContexts = append(kubeContexts, kubeContext)
	}
	return kubeContexts
}

func SwitchKubeContext(context string, kubeConfigPath string, config *api.Config) {
	config.CurrentContext = context
	err := clientcmd.WriteToFile(*config, kubeConfigPath)
	logger.ErrHandle(err)
}

func SwitchKubeContextNamespace(context string, kubeConfigPath string, namespace string, config *api.Config) {
	config.Contexts[context].Namespace = namespace
	err := clientcmd.WriteToFile(*config, kubeConfigPath)
	logger.ErrHandle(err)
}

func CreateClient(kubeConfigPath string) kubernetes.Interface {
	var kubeConfig *rest.Config
	if kubeConfigPath != "" {
		config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
		logger.ErrHandle(err)
		kubeConfig = config
	} else {
		config, err := rest.InClusterConfig()
		logger.ErrHandle(err)
		kubeConfig = config
	}
	client, err := kubernetes.NewForConfig(kubeConfig)
	logger.ErrHandle(err)
	return client
}

func CreateDynamicClient(kubeConfigPath string) dynamic.Interface {
	var kubeConfig *rest.Config
	if kubeConfigPath != "" {
		config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
		logger.ErrHandle(err)
		kubeConfig = config
	} else {
		config, err := rest.InClusterConfig()
		logger.ErrHandle(err)
		kubeConfig = config
	}
	dynamicClient, err := dynamic.NewForConfig(kubeConfig)
	logger.ErrHandle(err)
	return dynamicClient
}

// func GetGVKFromObject(obj runtime.Object, restMapper meta.RESTMapper) (schema.GroupVersionKind, error) {
// 	gvks, _, err := scheme.Scheme.ObjectKinds(obj)
// 	logger.ErrHandle(err)
// 	return gvks[0], nil
// }

func GetGVKFromObject(obj runtime.Object) schema.GroupVersionKind {
	gvks, _, err := scheme.Scheme.ObjectKinds(obj)
	logger.ErrHandle(err)
	return gvks[0]
}

// func GetGVKFromObject(obj runtime.Object, restMapper meta.RESTMapper) (schema.GroupVersionKind, error) {
// 	// Récupérer les metadonnées de l'objet
// 	accessor, err := meta.Accessor(obj)
// 	logger.ErrHandle(err)

// 	// Créer un GVR (GroupVersionResource) à partir du nom
// 	gvk, err := restMapper.KindFor(schema.GroupVersionResource{
// 		Resource: strings.ToLower(accessor.GetName()),
// 	})
// 	logger.ErrHandle(err)
// 	fmt.Println(gvk)
// 	return gvk, nil
// }

func GetDeployment(deploymentName string, namespace string, kubeClient kubernetes.Interface) *appsv1.Deployment {
	// restMapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(kubeClient.Discovery()))

	deployClient := kubeClient.AppsV1().Deployments(namespace)
	deploy, err := deployClient.Get(context.Background(), deploymentName, metav1.GetOptions{})
	logger.ErrHandle(err)

	// gvk, err := GetGVKFromObject(deploy, restMapper)
	gvk := GetGVKFromObject(deploy)
	logger.ErrHandle(err)
	deploy.SetGroupVersionKind(gvk)

	return deploy
}

func GetDeploymentsList(namespace string, kubeClient kubernetes.Interface) *appsv1.DeploymentList {
	deploys, err := kubeClient.AppsV1().Deployments(namespace).List(context.Background(), metav1.ListOptions{})
	logger.ErrHandle(err)
	return deploys
}

func GetPodsList(namespace string, kubeClient kubernetes.Interface) *corev1.PodList {
	pods, err := kubeClient.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	logger.ErrHandle(err)
	return pods
}

func GetNamespacesList(kubeClient kubernetes.Interface) *corev1.NamespaceList {
	namespaces, err := kubeClient.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	logger.ErrHandle(err)
	return namespaces
}

func GetPodStatuses(kubeClient kubernetes.Interface, namespace string, pod *corev1.Pod) (map[string]string, []corev1.Event) {
	podStatuses := map[string]string{
		"podReady":             "",
		"initialized":          "",
		"scheduled":            "",
		"containersReady":      "",
		"reason":               "",
		"waitingReason":        "",
		"waitingReasonMessage": "",
	}
	var podEvents []corev1.Event
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			podStatuses["podReady"] = string(condition.Status)
			if podStatuses["podReady"] != "True" {
				podStatuses["reason"] = condition.Reason
				podEvents = GetWarningEventsFromResource(kubeClient, "Pod", namespace, pod.Name)
				for _, c := range pod.Status.ContainerStatuses {
					if c.State.Waiting != nil && c.State.Waiting.Reason != "" {
						podStatuses["waitingReason"] = c.State.Waiting.Reason
						podStatuses["waitingReasonMessage"] = c.State.Waiting.Message
					}
					if c.State.Terminated != nil && c.State.Terminated.Reason != "" {
						podStatuses["waitingReason"] = c.State.Terminated.Reason
						podStatuses["waitingReasonMessage"] = c.State.Terminated.Message
					}
					if c.State.Running != nil {
						podStatuses["waitingReason"] = "Running"
					}
				}
			}
		}
		if condition.Type == corev1.PodInitialized {
			podStatuses["initialized"] = string(condition.Status)
		}
		if condition.Type == corev1.PodScheduled {
			podStatuses["scheduled"] = string(condition.Status)
		}
		if condition.Type == corev1.ContainersReady {
			podStatuses["containersReady"] = string(condition.Status)
		}
	}
	return podStatuses, podEvents
}

func GetPodListErrors(kubeClient kubernetes.Interface, namespace string) ([][]string, []map[string]string) {
	pods := GetPodsList(namespace, kubeClient)

	podList := make([][]string, 0)
	podListIssue := make([]map[string]string, 0)

	for _, pod := range pods.Items {
		podStatuses, podEvents := GetPodStatuses(kubeClient, namespace, &pod)
		podRow := []string{
			pod.ObjectMeta.Name,
			string(pod.Status.Phase),
			podStatuses["podReady"],
			podStatuses["initialized"],
			podStatuses["scheduled"],
			podStatuses["containersReady"],
			podStatuses["reason"],
			podStatuses["waitingReason"],
		}
		podList = append(podList, podRow)

		if podRow[2] != "True" {
			issue := map[string]string{
				"Name":     pod.ObjectMeta.Name,
				"Phase":    string(pod.Status.Phase),
				"PodReady": podStatuses["podReady"],
				"Reason":   podStatuses["reason"],
				"Status":   podStatuses["waitingReason"],
				"Message":  podStatuses["waitingReasonMessage"],
			}
			for _, event := range podEvents {
				issue["EventMessage"] += event.Message + "\n"
			}
			podListIssue = append(podListIssue, issue)
		}
	}
	return podList, podListIssue
}

func GetWarningEventsFromResource(kubeClient kubernetes.Interface, resourceKind string, namespace string, resourceName string) []corev1.Event {
	var warningEvents []corev1.Event
	events, err := kubeClient.CoreV1().Events(namespace).List(context.Background(), metav1.ListOptions{})
	logger.ErrHandle(err)
	for _, event := range events.Items {
		if event.InvolvedObject.Kind == resourceKind && event.InvolvedObject.Name == resourceName {
			if event.Type != "Normal" {
				warningEvents = append(warningEvents, event)
			}
		}
	}
	return warningEvents
}

func UpdateResource(ctx context.Context, dynamicClient dynamic.Interface, obj runtime.Object, modifyRequest ModifyRequest) {
	metaObj, ok := obj.(metav1.Object)
	if !ok {
		fmt.Errorf("object does not implement metav1.Object")
	}

	gvk := obj.GetObjectKind().GroupVersionKind()
	gvr := schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: strings.ToLower(gvk.Kind) + "s",
	}

	liveObject, err := dynamicClient.Resource(gvr).Namespace(metaObj.GetNamespace()).Get(ctx, metaObj.GetName(), metav1.GetOptions{})
	logger.ErrHandle(err)

	switch modifyRequest.Operation {
	case "update":
		err := unstructured.SetNestedField(liveObject.Object, modifyRequest.Value, modifyRequest.Path...)
		logger.ErrHandle(err)

	case "merge":
		currentMap, found, _ := unstructured.NestedMap(liveObject.Object, modifyRequest.Path...)
		if found {
			updateMap, ok := modifyRequest.Value.(map[string]string)
			if ok {
				for k, v := range updateMap {
					currentMap[k] = v
				}
				err := unstructured.SetNestedMap(liveObject.Object, currentMap, modifyRequest.Path...)
				logger.ErrHandle(err)
			}
		} else {
			err := unstructured.SetNestedField(liveObject.Object, modifyRequest.Value, modifyRequest.Path...)
			logger.ErrHandle(err)
		}

	// case "delete":
	// 	err = unstructured.RemoveNestedField(cr.Object, modifyRequest.Path...)

	default:
		fmt.Errorf("unknown operation: %s", modifyRequest.Operation)
	}

	_, err = dynamicClient.Resource(gvr).Namespace(metaObj.GetNamespace()).Update(ctx, liveObject, metav1.UpdateOptions{})
	logger.ErrHandle(err)
}
