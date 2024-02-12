package main

import (
	"context"
	"fmt"
	"github.com/charmbracelet/huh"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"kubectl/charm"
	"kubectl/customresource"
	"kubectl/logger"
	"os"
	"os/exec"
	"path/filepath"
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

func ListPods(namespace string, kubeClient kubernetes.Interface) (*v1.PodList, []string, error) {
	pods, err := kubeClient.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		err = fmt.Errorf("error getting pods: %v\n", err)
		return nil, nil, err
	}
	podsName := make([]string, 0)
	for _, n := range pods.Items {
		podsName = append(podsName, n.Name)
	}
	return pods, podsName, nil
}

func GetPodList(kubeClient kubernetes.Interface, namespace string) ([][]string, []map[string]string, error) {
	pods, err := kubeClient.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	logger.ErrHandle(err)

	podList := make([][]string, 0)
	podListIssue := make([]map[string]string, 0)

	for _, pod := range pods.Items {
		name := pod.Name
		phase := string(pod.Status.Phase)
		var ready string

		for _, condition := range pod.Status.Conditions {
			if condition.Type == v1.PodReady {
				ready = string(condition.Status)
			}
		}

		row := []string{
			name,
			phase,
			ready,
		}
		podList = append(podList, row)

		if podNotHealthy(&pod) {
			podListIssue = append(podListIssue, map[string]string{
				"Name":  name,
				"Phase": phase,
				"Ready": ready,
			})
		}
	}
	return podList, podListIssue, nil
}

func podNotHealthy(pod *v1.Pod) bool {
	if pod.Status.Phase != v1.PodRunning && pod.Status.Phase != v1.PodSucceeded {
		return true
	}
	if pod.Status.Phase == v1.PodSucceeded {
		return false
	}
	for _, condition := range pod.Status.Conditions {
		if condition.Type == v1.PodReady && condition.Status == v1.ConditionTrue {
			return false
		}
	}
	return true
}

func ListNameSpaces(kubeClient kubernetes.Interface) (*v1.NamespaceList, []string, error) {
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

	Es := (&customresource.ExternalSecret{}).NewCR("", "", "", "", "").(*customresource.ExternalSecret)
	Ks := (&customresource.Kustomization{}).NewCR("", "", "", "", "").(*customresource.Kustomization)
	GitRepo := (&customresource.GitRepository{}).NewCR("", "", "", "", "").(*customresource.GitRepository)
	Hc := (&customresource.HelmRepository{}).NewCR("", "", "", "", "").(*customresource.HelmRepository)
	Hr := (&customresource.HelmRelease{}).NewCR("", "", "", "", "").(*customresource.HelmRelease)

	Es.AnalyzeCRStatus(kubeDynamicClient, nsChoice)
	Ks.AnalyzeCRStatus(kubeDynamicClient, nsChoice)
	GitRepo.AnalyzeCRStatus(kubeDynamicClient, nsChoice)
	Hc.AnalyzeCRStatus(kubeDynamicClient, nsChoice)
	Hr.AnalyzeCRStatus(kubeDynamicClient, nsChoice)

	podList, podListIssue, err := GetPodList(kubeClient, nsChoice)

	if podList != nil {
		charm.CreateObjectArray(podList, []string{"NAME", "STATUS", "READY"})
	}

	if len(podListIssue) > 0 {
		logger.Logger.Error("ISSUE DETECTED")
		for _, pb := range podListIssue {
			logger.Logger.Error("Unsynced/NotReady", "name", pb["Name"], "status", pb["Status"], "ready", pb["Ready"], "hint", pb["Message"])
			fmt.Println("")
		}
	} else {
		logger.Logger.Info("All Pods are healthy")
		fmt.Println("")
	}

	//fmt.Print("pod list")
	//fmt.Println(podList)
	//fmt.Print("pod list issue")
	//fmt.Println(podListIssue)

}
