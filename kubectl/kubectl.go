package main

import (
	"context"
	"fmt"
	"github.com/charmbracelet/huh"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"kubectl/charm"
	"kubectl/customresource"
	"kubectl/k8s"
	"kubectl/logger"
	"os"
	"os/exec"
	"strconv"
)

func runCmd(command string) (string, error) {
	cmd := exec.Command("bash", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {

		logger.Logger.Fatal("Exit", "critical", err)
	}
	return string(output), nil
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

func main() {
	kubeConfigPath := k8s.GetKubeConfigPath()
	config := k8s.GetKubeConfigFromFile(kubeConfigPath)
	kubeContextsList := k8s.GetKubeContexts(config)

	var ctxChoice string
	contextChoiceForm := charm.GetForm(
		huh.NewSelect[string]().Title("Kubernetes Context").Description("Please choose a context to operate in").Options(charm.CreateOptionsFromStrings(kubeContextsList)...).Value(&ctxChoice),
	)
	err := contextChoiceForm.Run()
	logger.ErrHandle(err)

	k8s.SwitchKubeContext(ctxChoice, kubeConfigPath, config)

	kubeClient := k8s.CreateClient(kubeConfigPath)
	kubeDynamicClient := k8s.CreateDynamicClient(kubeConfigPath)

	nsList := k8s.GetNamespacesList(kubeClient)

	var nsChoice string
	nsChoiceForm := charm.GetForm(
		huh.NewSelect[string]().Title("Kubernetes Namespace").Description("Please choose a namespace to operate in").Options(charm.CreateOptionsFromStrings(func(nsList *corev1.NamespaceList) []string {
			nsName := make([]string, 0)
			for _, ns := range nsList.Items {
				nsName = append(nsName, ns.Name)
			}
			return nsName
		}(nsList))...).Value(&nsChoice),
	)
	err = nsChoiceForm.Run()
	logger.ErrHandle(err)
	nsChoiceForm.View()

	k8s.SwitchKubeContextNamespace(ctxChoice, kubeConfigPath, nsChoice, config)

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

	podList, podListIssue := k8s.GetPodListErrors(kubeClient, nsChoice)
	if podList != nil {
		charm.CreateObjectArray(podList, []string{"NAME", "PHASE", "PODREADY", "INIT", "SCHEDULED", "CTRREADY", "REASON", "STATUS"})
	}
	if len(podListIssue) > 0 {
		logger.Logger.Error("ISSUE DETECTED", "kind", "Pod")
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
