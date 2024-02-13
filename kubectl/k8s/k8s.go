package k8s

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"kubectl/logger"
	"os"
	"path/filepath"
)

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
