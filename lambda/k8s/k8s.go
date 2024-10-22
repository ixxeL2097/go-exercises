package k8s

import (
	"context"
	"fmt"
	"lambda/logger"
	"lambda/requests"
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

func PathExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, err
		} else {
			return false, err
		}
	}
	return !info.IsDir(), nil
}

func GetKubeConfigPath() string {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		logger.Logger.Warn("fail", err)
	}
	kubeConfigPath := filepath.Join(userHomeDir, ".kube", "config")
	return kubeConfigPath
}

func GetKubeConfigFromFile(kubeConfigPath string) *api.Config {
	config, err := clientcmd.LoadFromFile(kubeConfigPath)
	if err != nil {
		logger.Logger.Error("fail", err)
	}
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
	if err != nil {
		logger.Logger.Fatal("critical", err)
	}
}

func SwitchKubeContextNamespace(context string, kubeConfigPath string, namespace string, config *api.Config) {
	config.Contexts[context].Namespace = namespace
	err := clientcmd.WriteToFile(*config, kubeConfigPath)
	if err != nil {
		logger.Logger.Fatal("critical", err)
	}
}

func CreateKubeClient(kubeConfigPath string, clientType string) (interface{}, error) {
	var kubeConfig *rest.Config
	if _, err := PathExists(kubeConfigPath); err != nil {
		logger.Logger.Info("Loading in-cluster kube config")
		kubeConfig, err = rest.InClusterConfig()
		if err != nil {
			logger.Logger.Error("Failed to load in-cluster kube config", err)
			return nil, err
		}
	} else {
		logger.Logger.Info("Loading kube config", "kubeconfig", kubeConfigPath)
		kubeConfig, err = clientcmd.BuildConfigFromFlags("", kubeConfigPath)
		if err != nil {
			logger.Logger.Error("Failed to load kube config from path", err)
			return nil, err
		}
	}
	switch clientType {
	case "static":
		logger.Logger.Info("Creating static kube client")
		client, err := kubernetes.NewForConfig(kubeConfig)
		if err != nil {
			return nil, err
		}
		return client, nil
	case "dynamic":
		logger.Logger.Info("Creating dynamic kube client")
		dynamicClient, err := dynamic.NewForConfig(kubeConfig)
		if err != nil {
			return nil, err
		}
		return dynamicClient, nil
	default:
		return nil, fmt.Errorf("unknown client type: %s", clientType)
	}
}

// func CreateClient(kubeConfigPath string) (kubernetes.Interface, error) {
// 	var kubeConfig *rest.Config
// 	if _, err := PathExists(kubeConfigPath); err != nil {
// 		logger.Logger.Info("Loading in cluster kube config")
// 		config, err := rest.InClusterConfig()
// 		if err != nil {
// 			logger.Logger.Error("fail", err)
// 		}
// 		kubeConfig = config
// 	} else {
// 		logger.Logger.Info("Loading kube config", "kubeconfig", kubeConfigPath)
// 		config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
// 		if err != nil {
// 			logger.Logger.Error("fail", err)
// 		}
// 		kubeConfig = config
// 	}
// 	client, err := kubernetes.NewForConfig(kubeConfig)
// 	return client, err
// }

// func CreateDynamicClient(kubeConfigPath string) (dynamic.Interface, error) {
// 	var kubeConfig *rest.Config
// 	if _, err := PathExists(kubeConfigPath); err != nil {
// 		logger.Logger.Info("Loading in cluster kube config")
// 		config, err := rest.InClusterConfig()
// 		if err != nil {
// 			logger.Logger.Error("fail", err)
// 		}
// 		kubeConfig = config
// 	} else {
// 		logger.Logger.Info("Loading kube config", "kubeconfig", kubeConfigPath)
// 		config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
// 		if err != nil {
// 			logger.Logger.Error("fail", err)
// 		}
// 		kubeConfig = config
// 	}
// 	dynamicClient, err := dynamic.NewForConfig(kubeConfig)
// 	return dynamicClient, err
// }

func GetGVKFromObject(obj runtime.Object) (schema.GroupVersionKind, error) {
	gvks, _, err := scheme.Scheme.ObjectKinds(obj)
	if err != nil {
		logger.Logger.Error("Failed loading GVK from object", "obj", obj, "error", err)
		return schema.GroupVersionKind{}, err
	}
	if len(gvks) == 0 {
		logger.Logger.Error("No GVK found for object", "obj", obj, "error", err)
		return schema.GroupVersionKind{}, err
	}
	return gvks[0], nil
}

func GetDeployment(deploymentName string, namespace string, kubeClient kubernetes.Interface) (*appsv1.Deployment, error) {
	deployClient := kubeClient.AppsV1().Deployments(namespace)

	deploy, err := deployClient.Get(context.Background(), deploymentName, metav1.GetOptions{})
	if err != nil {
		logger.Logger.Error("Failed getting deployment", "deploy", deploymentName, "namespace", namespace, "error", err)
		return nil, err
	}

	gvk, err := GetGVKFromObject(deploy)
	if err != nil {
		logger.Logger.Error("Failed to inject GVK", "deploy", deploymentName, "namespace", namespace, "error", err)
		return nil, err
	}
	deploy.SetGroupVersionKind(gvk)

	return deploy, nil
}

func GetDeploymentsList(namespace string, kubeClient kubernetes.Interface) (*appsv1.DeploymentList, error) {
	deploys, err := kubeClient.AppsV1().Deployments(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		logger.Logger.Error("Failed getting deployment list", "namespace", namespace, "error", err)
		return nil, err
	}
	return deploys, nil
}

func GetPodsList(namespace string, kubeClient kubernetes.Interface) (*corev1.PodList, error) {
	pods, err := kubeClient.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		logger.Logger.Error("Failed getting pods list", "namespace", namespace, "error", err)
		return nil, err
	}
	return pods, nil
}

func GetNamespacesList(kubeClient kubernetes.Interface) (*corev1.NamespaceList, error) {
	namespaces, err := kubeClient.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		logger.Logger.Error("Failed listing namespaces", "error", err)
		return nil, err
	}
	return namespaces, nil
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
	pods, err := GetPodsList(namespace, kubeClient)
	if err != nil {
		logger.Logger.Error(err)
	}

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

func UpdateResource(ctx context.Context, dynamicClient dynamic.Interface, obj runtime.Object, modifyRequest requests.ModifyRequest) error {
	metaObj, ok := obj.(metav1.Object)
	if !ok {
		return fmt.Errorf("object does not implement metav1.Object")
	}

	gvk := obj.GetObjectKind().GroupVersionKind()
	gvr := schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: strings.ToLower(gvk.Kind) + "s",
	}
	logger.Logger.Debug("GVK acquired", "group", gvk.Group, "version", gvk.Version, "kind", gvk.Kind)

	liveObject, err := dynamicClient.Resource(gvr).Namespace(metaObj.GetNamespace()).Get(ctx, metaObj.GetName(), metav1.GetOptions{})
	if err != nil {
		logger.Logger.Error("Failed to get live object", "object", metaObj.GetName(), "namespace", metaObj.GetNamespace(), "error", err)
		return err
	}
	logger.Logger.Debug("Live object acquired", "name", metaObj.GetName(), "namespace", metaObj.GetNamespace(), "object", liveObject.Object)

	switch modifyRequest.Operation {
	case "update":
		logger.Logger.Debug("Update required", "value", modifyRequest.Value, "path", modifyRequest.Path)
		if err := unstructured.SetNestedField(liveObject.Object, modifyRequest.Value, modifyRequest.Path...); err != nil {
			logger.Logger.Error("Failed to update field", "value", modifyRequest.Value, "path", modifyRequest.Path, "error", err)
			return err
		}
	case "merge":
		logger.Logger.Debug("Merge required", "value", modifyRequest.Value, "path", modifyRequest.Path)
		currentMap, found, err := unstructured.NestedMap(liveObject.Object, modifyRequest.Path...)
		if err != nil {
			logger.Logger.Error("Failed to retrieve current nested map", "path", modifyRequest.Path, "error", err)
			return err
		}
		updateMap, ok := modifyRequest.Value.(map[string]string)
		if !ok {
			logger.Logger.Error("modifyRequest.Value is not a map[string]string", "value", modifyRequest.Value)
			return err
		}
		if found {
			for k, v := range updateMap {
				currentMap[k] = v
			}
			if err := unstructured.SetNestedField(liveObject.Object, currentMap, modifyRequest.Path...); err != nil {
				logger.Logger.Error("Failed to set nested field", "path", modifyRequest.Path, "error", err)
				return err
			}
		} else {
			if err := unstructured.SetNestedField(liveObject.Object, modifyRequest.Value, modifyRequest.Path...); err != nil {
				logger.Logger.Error("Failed to set nested field", "path", modifyRequest.Path, "error", err)
				return err
			}
		}

	// case "delete":
	// 	err = unstructured.RemoveNestedField(cr.Object, modifyRequest.Path...)

	default:
		return fmt.Errorf("unknown operation: %s", modifyRequest.Operation)
	}

	if _, err := dynamicClient.Resource(gvr).Namespace(metaObj.GetNamespace()).Update(ctx, liveObject, metav1.UpdateOptions{}); err != nil {
		logger.Logger.Error("Failed to update live object", "object", liveObject, "error", err)
		return err
	}
	return nil
}
