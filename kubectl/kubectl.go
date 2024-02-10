package main

import (
	"context"
	"fmt"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/charmbracelet/log"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const (
	purple    = lipgloss.Color("99")
	gray      = lipgloss.Color("245")
	lightGray = lipgloss.Color("241")
)

type CustomResource struct {
	group            string
	version          string
	kind             string
	successCondition string
	prettyName       string
}

var (
	re = lipgloss.NewRenderer(os.Stdout)
	// HeaderStyle is the lipgloss style used for the table headers.
	HeaderStyle = re.NewStyle().Foreground(purple).Bold(true).Align(lipgloss.Center)
	// CellStyle is the base lipgloss style used for the table rows.
	CellStyle = re.NewStyle().Padding(0, 1).Width(14)
	// OddRowStyle is the lipgloss style used for odd-numbered table rows.
	OddRowStyle = CellStyle.Copy().Foreground(gray)
	// EvenRowStyle is the lipgloss style used for even-numbered table rows.
	EvenRowStyle = CellStyle.Copy().Foreground(lightGray)
	// BorderStyle is the lipgloss style used for the table border.
	BorderStyle = lipgloss.NewStyle().Foreground(purple)
	// Logger for output
	Logger *log.Logger
	Es     = CustomResource{
		group:            "external-secrets.io",
		version:          "v1beta1",
		kind:             "externalsecrets",
		successCondition: "SecretSynced",
		prettyName:       "ExternalSecret",
	}
	Ks = CustomResource{
		group:            "kustomize.toolkit.fluxcd.io",
		version:          "v1",
		kind:             "kustomizations",
		successCondition: "SecretSynced",
		prettyName:       "Kustomization",
	}
	GitRepo = CustomResource{
		group:            "source.toolkit.fluxcd.io",
		version:          "v1",
		kind:             "gitrepository",
		successCondition: "SecretSynced",
		prettyName:       "GitRepository",
	}
)

func init() {
	styles := log.DefaultStyles()
	styles.Levels[log.FatalLevel] = lipgloss.NewStyle().SetString("FATAL!!").Padding(0, 1, 0, 1).Background(lipgloss.AdaptiveColor{Light: "#9966CC", Dark: "#9966CC"}).Foreground(lipgloss.Color("0")).Bold(true)
	styles.Levels[log.ErrorLevel] = lipgloss.NewStyle().SetString("ERROR!!").Padding(0, 1, 0, 1).Background(lipgloss.AdaptiveColor{Light: "203", Dark: "203"}).Foreground(lipgloss.Color("0")).Bold(true)
	styles.Levels[log.InfoLevel] = lipgloss.NewStyle().SetString("INFO >>").Padding(0, 1, 0, 1).Background(lipgloss.AdaptiveColor{Light: "45", Dark: "45"}).Foreground(lipgloss.Color("0")).Bold(true)
	styles.Levels[log.DebugLevel] = lipgloss.NewStyle().SetString("DEBUG ::").Padding(0, 1, 0, 1).Background(lipgloss.AdaptiveColor{Light: "75", Dark: "75"}).Foreground(lipgloss.Color("0")).Bold(true)
	styles.Keys["critical"] = lipgloss.NewStyle().Foreground(lipgloss.Color("#9966CC"))
	styles.Values["critical"] = lipgloss.NewStyle().Bold(true)
	styles.Keys["err"] = lipgloss.NewStyle().Foreground(lipgloss.Color("204"))
	styles.Values["err"] = lipgloss.NewStyle().Bold(true)
	styles.Keys["hint"] = lipgloss.NewStyle().Foreground(lipgloss.Color("204"))
	styles.Values["hint"] = lipgloss.NewStyle().Bold(true)
	styles.Keys["status"] = lipgloss.NewStyle().Foreground(lipgloss.Color("204"))
	styles.Values["status"] = lipgloss.NewStyle().Bold(true)
	styles.Keys["object"] = lipgloss.NewStyle().Foreground(lipgloss.Color("204"))
	styles.Values["object"] = lipgloss.NewStyle().Bold(true)
	styles.Keys["ready"] = lipgloss.NewStyle().Foreground(lipgloss.Color("204"))
	styles.Values["ready"] = lipgloss.NewStyle().Bold(true)
	Logger = log.New(os.Stderr)
	Logger.SetStyles(styles)
	Logger.SetLevel(log.DebugLevel)
}

func errHandle(err error) {
	if err != nil {
		Logger.Fatal("Exit", "critical", err)
	}
}

func runCmd(command string) (string, error) {
	cmd := exec.Command("bash", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		Logger.Fatal("Exit", "critical", err)
	}
	return string(output), nil
}

func createOptionsFromStrings(strings []string) []huh.Option[string] {
	var options []huh.Option[string]
	for _, str := range strings {
		options = append(options, huh.NewOption(string(str), string(str)))
	}
	return options
}

func getForm[T comparable](selects ...*huh.Select[T]) *huh.Form {
	var fields []huh.Field
	for _, sel := range selects {
		fields = append(fields, sel)
	}
	group := huh.NewGroup(fields...)
	return huh.NewForm(group)
}

func getKubeConfigPath() string {
	userHomeDir, err := os.UserHomeDir()
	errHandle(err)

	kubeConfigPath := filepath.Join(userHomeDir, ".kube", "config")
	return kubeConfigPath
}

func getKubeConfigFromFile(kubeConfigPath string) *api.Config {
	config, err := clientcmd.LoadFromFile(kubeConfigPath)
	errHandle(err)
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
	errHandle(err)
}

func switchKubeContextNamespace(context string, kubeConfigPath string, namespace string, config *api.Config) {
	config.Contexts[context].Namespace = namespace
	err := clientcmd.WriteToFile(*config, kubeConfigPath)
	errHandle(err)
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

func ListPods(namespace string, coreClient kubernetes.Interface) (*v1.PodList, []string, error) {
	pods, err := coreClient.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
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

func ListNameSpaces(coreClient kubernetes.Interface) (*v1.NamespaceList, []string, error) {
	namespaces, err := coreClient.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
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

func getCRList(cr CustomResource, kubeClient dynamic.Interface, namespace string) ([][]string, []map[string]string) {
	var customResource = schema.GroupVersionResource{Group: cr.group, Version: cr.version, Resource: cr.kind}
	customResources, err := kubeClient.Resource(customResource).Namespace(namespace).List(context.Background(), metav1.ListOptions{})
	if customResources == nil || len(customResources.Items) == 0 {
		Logger.Info("No CustomResource found", "kind", cr.prettyName)
		return nil, nil
	}
	errHandle(err)

	CRList := make([][]string, 0)
	CRListIssue := make([]map[string]string, 0)

	for _, custom := range customResources.Items {

		statusMap, foundStatus, err := unstructured.NestedMap(custom.Object, "status")
		errHandle(err)

		if foundStatus {
			conditions, foundConditions, err := unstructured.NestedSlice(statusMap, "conditions")
			errHandle(err)

			if foundConditions {
				var latestCondition map[string]interface{}
				latestTime := time.Time{}
				latestIndex := -1

				for i, condition := range conditions {
					conditionMap, ok := condition.(map[string]interface{})
					if !ok {
						Logger.Fatal("condition is not a Map object", "object", custom.Object)
					}
					transitionTimeStr, _ := conditionMap["lastTransitionTime"].(string)
					transitionTime, err := time.Parse(time.RFC3339, transitionTimeStr)
					errHandle(err)

					if transitionTime.After(latestTime) || (transitionTime.Equal(latestTime) && i > latestIndex) {
						latestTime = transitionTime
						latestCondition = conditionMap
						latestIndex = i
					}
				}
				if latestCondition != nil {
					reason := latestCondition["reason"].(string)
					status := latestCondition["status"].(string)
					message := latestCondition["message"].(string)

					row := []string{
						custom.Object["metadata"].(map[string]interface{})["name"].(string),
						reason,
						status,
						message,
					}
					CRList = append(CRList, row)

					if reason != cr.successCondition {
						CRListIssue = append(CRListIssue, map[string]string{
							"Name":    custom.Object["metadata"].(map[string]interface{})["name"].(string),
							"Status":  reason,
							"Ready":   status,
							"Message": message,
						})
					}
				}
			} else {
				Logger.Fatal("No sub-entry 'conditions' found in entry 'status'", "object", custom.Object)
			}
		} else {
			Logger.Fatal("No entry 'status' found", "object", custom.Object)
		}
	}
	return CRList, CRListIssue
}

func displayCRIssue(CRListIssue []map[string]string, cr CustomResource) {
	if len(CRListIssue) > 0 {
		Logger.Error("ISSUE DETECTED", "object", cr.prettyName)
		for _, pb := range CRListIssue {
			Logger.Error("Unsynced/NotReady", "kind", cr.prettyName, "name", pb["Name"], "status", pb["Status"], "ready", pb["Ready"], "hint", pb["Message"])
		}
	} else {
		Logger.Info("All CustomResources are healthy", "kind", cr.prettyName)
	}
}

func createObjectArray(ObjectList [][]string) {
	t := table.New().Border(lipgloss.ThickBorder()).BorderStyle(BorderStyle).StyleFunc(func(row, col int) lipgloss.Style {
		var style lipgloss.Style

		switch {
		case row == 0:
			return HeaderStyle
		default:
			style = OddRowStyle
		}

		if col == 0 {
			style = style.Copy().Width(30)
		}
		if col == 1 {
			style = style.Copy().Width(25)
		}
		if col == 3 {
			style = style.Copy().Width(90)
		}

		return style
	}).Headers("NAME", "STATUS", "READY", "MESSAGE").Rows(ObjectList...)
	fmt.Println(t)
}

func main() {

	kubeConfigPath := getKubeConfigPath()
	config := getKubeConfigFromFile(kubeConfigPath)
	kubeContextsList := getKubeContexts(config)

	var ctxChoice string
	contextChoiceForm := getForm(
		huh.NewSelect[string]().Title("Kubernetes Context").Description("Please choose a context to operate in").Options(createOptionsFromStrings(kubeContextsList)...).Value(&ctxChoice),
	)
	err := contextChoiceForm.Run()
	errHandle(err)

	switchKubeContext(ctxChoice, kubeConfigPath, config)

	kubeClient, err := createClient(kubeConfigPath)
	_, nsName, err := ListNameSpaces(kubeClient)

	var nsChoice string
	nsChoiceForm := getForm(
		huh.NewSelect[string]().Title("Kubernetes Namespace").Description("Please choose a namespace to operate in").Options(createOptionsFromStrings(nsName)...).Value(&nsChoice),
	)
	err = nsChoiceForm.Run()
	errHandle(err)
	nsChoiceForm.View()

	switchKubeContextNamespace(ctxChoice, kubeConfigPath, nsChoice, config)

	kubeDynamicClient, err := createDynamicClient(kubeConfigPath)

	ESList, ESListIssue := getCRList(Es, kubeDynamicClient, nsChoice)
	Logger.Debug("Listing customResource", "kind", Es.prettyName, "namespace", nsChoice)
	createObjectArray(ESList)
	displayCRIssue(ESListIssue, Es)
	KSList, KSListIssue := getCRList(Ks, kubeDynamicClient, nsChoice)
	Logger.Debug("Listing customResource", "kind", Ks.prettyName, "namespace", nsChoice)
	createObjectArray(KSList)
	displayCRIssue(KSListIssue, Ks)

}
