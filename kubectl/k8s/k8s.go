package k8s

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"kubectl/logger"
)

func GetPodsList(namespace string, kubeClient kubernetes.Interface) *corev1.PodList {
	pods, err := kubeClient.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	logger.ErrHandle(err)
	return pods
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
				podEvents = GetEventsFromResource(kubeClient, "Pod", namespace, pod.Name)
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

func GetEventsFromResource(kubeClient kubernetes.Interface, resourceKind string, namespace string, resourceName string) []corev1.Event {
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
