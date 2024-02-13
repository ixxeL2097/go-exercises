package main

import (
	"context"
	"fmt"
	"github.com/charmbracelet/huh"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"kubectl/charm"
	"kubectl/customresource"
	"kubectl/k8s"
	"kubectl/logger"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

func runCmd(command string) (string, error) {
	cmd := exec.Command("bash", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {

		logger.Logger.Fatal("Exit", "critical", err)
	}
	return string(output), nil
}

func getKubeConfigPath() string {
	userHomeDir, err := os.UserHomeDir()
	logger.ErrHandle(err)

	kubeConfigPath := filepath.Join(userHomeDir, ".kube", "config")
	return kubeConfigPath
}

func getKubeConfigFromFile(kubeConfigPath string) *api.Config {
	config, err := clientcmd.LoadFromFile(kubeConfigPath)
	logger.ErrHandle(err)
	return config
}

func getKubeContexts(config *api.Config) []string {
	kubeContexts := make([]string, 0)
	for kubeContext := range config.Contexts {
		kubeContexts = append(kubeContexts, kubeContext)
	}
	return kubeContexts
}

func switchKubeContext(context string, kubeConfigPath string, config *api.Config) {
	config.CurrentContext = context
	err := clientcmd.WriteToFile(*config, kubeConfigPath)
	logger.ErrHandle(err)
}

func switchKubeContextNamespace(context string, kubeConfigPath string, namespace string, config *api.Config) {
	config.Contexts[context].Namespace = namespace
	err := clientcmd.WriteToFile(*config, kubeConfigPath)
	logger.ErrHandle(err)
}

func createClient(kubeConfigPath string) (kubernetes.Interface, error) {
	var kubeConfig *rest.Config
	if kubeConfigPath != "" {
		config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
		if err != nil {
			return nil, fmt.Errorf("unable to load kubeconfig from %s: %v", kubeConfigPath, err)
		}
		kubeConfig = config
	} else {
		config, err := rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("unable to load in-cluster config: %v", err)
		}
		kubeConfig = config
	}
	client, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create a client: %v", err)
	}
	return client, nil
}

func createDynamicClient(kubeConfigPath string) (dynamic.Interface, error) {
	var kubeConfig *rest.Config
	if kubeConfigPath != "" {
		config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
		if err != nil {
			return nil, fmt.Errorf("unable to load kubeconfig from %s: %v", kubeConfigPath, err)
		}
		kubeConfig = config
	} else {
		config, err := rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("unable to load in-cluster config: %v", err)
		}
		kubeConfig = config
	}
	dynamicClient, err := dynamic.NewForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create a client: %v", err)
	}
	return dynamicClient, nil
}

func getPodListErrors(kubeClient kubernetes.Interface, namespace string) ([][]string, []map[string]string) {
	pods := k8s.GetPodsList(namespace, kubeClient)

	podList := make([][]string, 0)
	podListIssue := make([]map[string]string, 0)

	for _, pod := range pods.Items {
		podStatuses, podEvents := k8s.GetPodStatuses(kubeClient, namespace, &pod)
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

func detectPodsErrors(kubeClient kubernetes.Interface, namespace string) ([][]string, []map[string]string) {
	podList := make([][]string, 0)
	podListIssue := make([]map[string]string, 0)

	timeout := time.After(3 * time.Second)
	watcher, err := kubeClient.CoreV1().Pods(namespace).Watch(context.Background(),
		metav1.ListOptions{
			FieldSelector: "",
		})
	logger.ErrHandle(err)

	for {
		select {
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return podList, podListIssue
			}
			fmt.Println("EVENT")

			pod, ok := event.Object.(*corev1.Pod)
			if !ok {
				continue
			}
			var podReady, initialized, scheduled, containersReady, reason, waitingReason, waitingReasonMessage string
			var podEvents []corev1.Event
			//fmt.Println("Pod name:", pod.ObjectMeta.Name)
			for _, condition := range pod.Status.Conditions {
				//fmt.Println("Condition type:", condition.Type, "Status:", condition.Status, "timestamp:", condition.LastTransitionTime)
				if condition.Type == corev1.PodReady {
					podReady = string(condition.Status)
					if podReady != "True" {
						reason = condition.Reason
						podEvents = k8s.GetEventsFromResource(kubeClient, "Pod", namespace, pod.Name)
						for _, c := range pod.Status.ContainerStatuses {
							if c.State.Waiting != nil && c.State.Waiting.Reason != "" {
								waitingReason = c.State.Waiting.Reason
								waitingReasonMessage = c.State.Waiting.Message
							}
							if c.State.Terminated != nil && c.State.Terminated.Reason != "" {
								waitingReason = c.State.Terminated.Reason
								waitingReasonMessage = c.State.Terminated.Message
							}
							if c.State.Running != nil {
								waitingReason = "Running"
							}
						}
					}
				}
				if condition.Type == corev1.PodInitialized {
					initialized = string(condition.Status)
				}
				if condition.Type == corev1.PodScheduled {
					scheduled = string(condition.Status)
				}
				if condition.Type == corev1.ContainersReady {
					containersReady = string(condition.Status)
				}
			}

			podRow := []string{
				pod.ObjectMeta.Name,
				string(pod.Status.Phase),
				podReady,
				initialized,
				scheduled,
				containersReady,
				reason,
				waitingReason,
			}
			podList = append(podList, podRow)

			if podRow[2] != "True" {
				issue := map[string]string{
					"Name":     pod.ObjectMeta.Name,
					"Phase":    string(pod.Status.Phase),
					"PodReady": podReady,
					"Reason":   reason,
					"Status":   waitingReason,
					"Message":  waitingReasonMessage,
				}

				for _, event := range podEvents {
					issue["EventMessage"] += event.Message + "\n"
				}
				podListIssue = append(podListIssue, issue)
			}

		case <-timeout:
			return podList, podListIssue
		}
	}
}

func GetDeploymentList(kubeClient kubernetes.Interface, namespace string) ([][]string, []map[string]string, error) {
	deployments, err := kubeClient.AppsV1().Deployments(namespace).List(context.Background(), metav1.ListOptions{})
	logger.ErrHandle(err)

	deploymentList := make([][]string, 0)
	deploymentListIssue := make([]map[string]string, 0)

	for _, deployment := range deployments.Items {
		name := deployment.Name
		availableReplicas := deployment.Status.AvailableReplicas
		replicas := deployment.Status.Replicas
		readyReplicas := deployment.Status.ReadyReplicas
		updatedReplicas := deployment.Status.UpdatedReplicas

		row := []string{
			name,
			fmt.Sprintf("%d/%d", availableReplicas, replicas),
			fmt.Sprintf("%d/%d", readyReplicas, replicas),
			fmt.Sprintf("%d/%d", updatedReplicas, replicas), // Ajout de la colonne UP-TO-DATE
		}
		deploymentList = append(deploymentList, row)

		if deploymentNotHealthy(&deployment) {
			deploymentListIssue = append(deploymentListIssue, map[string]string{
				"Name":      name,
				"Available": fmt.Sprintf("%d/%d", availableReplicas, replicas),
				"Ready":     fmt.Sprintf("%d/%d", readyReplicas, replicas),
				"UpToDate":  fmt.Sprintf("%d/%d", updatedReplicas, replicas), // Ajout de la colonne UP-TO-DATE
			})
		}
	}
	return deploymentList, deploymentListIssue, nil
}

func deploymentNotHealthy(deployment *appsv1.Deployment) bool {
	// Critères de non-santé :
	// - Replicas disponibles inférieures aux replicas souhaitées
	// - Répliques non prêtes supérieures à un seuil configurable (exemple : 0)
	// - Présence de conditions d'erreurs

	// Récupérer le seuil de répliques non prêtes acceptable
	maxUnreadyReplicas, err := strconv.Atoi(os.Getenv("DEPLOYMENT_UNREADY_THRESHOLD"))
	if err != nil || maxUnreadyReplicas < 0 {
		// Valeur par défaut en cas d'erreur ou de valeur invalide
		maxUnreadyReplicas = 0
	}

	availableReplicas := deployment.Status.AvailableReplicas
	desiredReplicas := deployment.Spec.Replicas
	unreadyReplicas := deployment.Status.Replicas - deployment.Status.ReadyReplicas
	conditions := deployment.Status.Conditions

	// Vérifier les conditions
	for _, condition := range conditions {
		if condition.Type == appsv1.DeploymentProgressing && condition.Status == corev1.ConditionFalse {
			return true
		}
		if condition.Type == appsv1.DeploymentReplicaFailure && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	// Vérifier le nombre de répliques disponibles et non prêtes
	return availableReplicas < *desiredReplicas || unreadyReplicas > int32(maxUnreadyReplicas)
}

func ListNameSpaces(kubeClient kubernetes.Interface) (*corev1.NamespaceList, []string, error) {
	namespaces, err := kubeClient.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		err = fmt.Errorf("error getting namespoaces: %v\n", err)
		return nil, nil, err
	}
	namespacesName := make([]string, 0)
	for _, n := range namespaces.Items {
		namespacesName = append(namespacesName, n.Name)
	}
	return namespaces, namespacesName, nil
}

func main() {
	kubeConfigPath := getKubeConfigPath()
	config := getKubeConfigFromFile(kubeConfigPath)
	kubeContextsList := getKubeContexts(config)

	var ctxChoice string
	contextChoiceForm := charm.GetForm(
		huh.NewSelect[string]().Title("Kubernetes Context").Description("Please choose a context to operate in").Options(charm.CreateOptionsFromStrings(kubeContextsList)...).Value(&ctxChoice),
	)
	err := contextChoiceForm.Run()
	logger.ErrHandle(err)

	switchKubeContext(ctxChoice, kubeConfigPath, config)

	kubeClient, err := createClient(kubeConfigPath)
	_, nsName, err := ListNameSpaces(kubeClient)

	var nsChoice string
	nsChoiceForm := charm.GetForm(
		huh.NewSelect[string]().Title("Kubernetes Namespace").Description("Please choose a namespace to operate in").Options(charm.CreateOptionsFromStrings(nsName)...).Value(&nsChoice),
	)
	err = nsChoiceForm.Run()
	logger.ErrHandle(err)
	nsChoiceForm.View()

	switchKubeContextNamespace(ctxChoice, kubeConfigPath, nsChoice, config)

	kubeDynamicClient, err := createDynamicClient(kubeConfigPath)

	Es := customresource.GetCRD("externalsecret", "", "", "")
	Ks := customresource.GetCRD("kustomization", "", "", "")
	Gr := customresource.GetCRD("gitrepository", "", "", "")
	Hc := customresource.GetCRD("helmrepository", "", "", "")
	Hr := customresource.GetCRD("helmrelease", "", "", "")

	Es.AnalyzeCRStatus(kubeDynamicClient, kubeClient, nsChoice)
	Ks.AnalyzeCRStatus(kubeDynamicClient, kubeClient, nsChoice)
	Gr.AnalyzeCRStatus(kubeDynamicClient, kubeClient, nsChoice)
	Hc.AnalyzeCRStatus(kubeDynamicClient, kubeClient, nsChoice)
	Hr.AnalyzeCRStatus(kubeDynamicClient, kubeClient, nsChoice)

	podList, podListIssue := getPodListErrors(kubeClient, nsChoice)
	if podList != nil {
		charm.CreateObjectArray(podList, []string{"NAME", "PHASE", "PODREADY", "INIT", "SCHEDULED", "CTRREADY", "REASON", "STATUS"})
	}
	if len(podListIssue) > 0 {
		logger.Logger.Error("ISSUE DETECTED")
		for _, pb := range podListIssue {
			logger.Logger.Error("Unsynced/NotReady", "name", pb["Name"], "ready", pb["PodReady"], "status", pb["Status"], "errors", pb["EventMessage"])
			fmt.Println("")
		}
	} else {
		logger.Logger.Info("All Pods are healthy")
		fmt.Println("")
	}

	//deployList, deployListIssue, err := GetDeploymentList(kubeClient, nsChoice)
	//if deployList != nil {
	//	charm.CreateObjectArray(deployList, []string{"NAME", "AVAILABLE", "READY", "UP-TO-DATE"})
	//}
	//if len(deployListIssue) > 0 {
	//	logger.Logger.Error("ISSUE DETECTED")
	//	for _, pb := range deployListIssue {
	//		logger.Logger.Error("Unsynced/NotReady", "name", pb["Name"], "available", pb["Available"], "ready", pb["Ready"], "uptodate", pb["UpToDate"])
	//		fmt.Println("")
	//	}
	//} else {
	//	logger.Logger.Info("All Pods are healthy")
	//	fmt.Println("")
	//}

	//fmt.Print("pod list")
	//fmt.Println(podList)
	//fmt.Print("pod list issue")
	//fmt.Println(podListIssue)

}
