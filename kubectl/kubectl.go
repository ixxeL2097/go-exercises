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
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"kubectl/customresource"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	purple    = lipgloss.Color("99")
	gray      = lipgloss.Color("245")
	lightGray = lipgloss.Color("241")
)

//type CustomResourceDefinition interface {
//	getCRList(kubeClient dynamic.Interface, namespace string) ([][]string, []map[string]string)
//	NewCR() CustomResourceDefinition
//}
//
//type CustomResource struct {
//	group            string
//	version          string
//	kind             string
//	successCondition string
//	prettyName       string
//}
//
//type ExternalSecret struct {
//	CustomResource
//}
//
//type Kustomization struct {
//	CustomResource
//}

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

	//Es = (&customresource.ExternalSecret{}).NewCR().(*customresource.ExternalSecret)
	//Ks = (&customresource.Kustomization{}).NewCR().(*customresource.Kustomization)
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

	Es := (&customresource.ExternalSecret{}).NewCR().(*customresource.ExternalSecret)
	Ks := (&customresource.Kustomization{}).NewCR().(*customresource.Kustomization)

	ESList, ESListIssue := Es.GetCRList(kubeDynamicClient, nsChoice)
	Logger.Debug("Listing customResource", "kind", Es.PrettyName, "namespace", nsChoice)
	createObjectArray(ESList)
	Es.DisplayCRIssue(ESListIssue)
	KSList, KSListIssue := Ks.GetCRList(kubeDynamicClient, nsChoice)
	Logger.Debug("Listing customResource", "kind", Ks.PrettyName, "namespace", nsChoice)
	createObjectArray(KSList)
	Ks.DisplayCRIssue(KSListIssue)

}
