package main

import (
	"context"
	"fmt"
	"github.com/charmbracelet/huh"
	"github.com/fatih/color"
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
)

var errColor = color.New(color.FgRed)
var reset = "\033[0m"
var bold = "\033[1m"
var underline = "\033[4m"
var strike = "\033[9m"
var italic = "\033[3m"

var cRed = "\033[31m"
var cGreen = "\033[32m"
var cYellow = "\033[33m"
var cBlue = "\033[34m"
var cPurple = "\033[35m"
var cCyan = "\033[36m"
var cWhite = "\033[37m"

func errHandle(err error) {
	if err != nil {
		_, err2 := errColor.Printf("[ EXIT ] > %v\n", err)
		if err2 != nil {
			os.Exit(1)
		}
		os.Exit(1)
	}
}

func runCmd(command string) (string, error) {
	cmd := exec.Command("bash", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		_, err2 := errColor.Printf("[ EXEC ERROR ] >> error executing <%v>\n", command)
		if err2 != nil {
			os.Exit(1)
		}
		fmt.Printf(string(output))
		return string(output), err
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

func troubleShootExternalSecret(cr CustomResource, kubeClient dynamic.Interface, namespace string) {
	var externalSecret = schema.GroupVersionResource{Group: cr.group, Version: cr.version, Resource: cr.kind}
	externalSecrets, err := kubeClient.Resource(externalSecret).Namespace(namespace).List(context.Background(), metav1.ListOptions{})
	errHandle(err)

	problematicExternalSecrets := make([]map[string]string, 0)

	for _, es := range externalSecrets.Items {

		statusMap, foundStatus, err := unstructured.NestedMap(es.Object, "status")
		errHandle(err)

		if foundStatus {
			conditions, foundConditions, err := unstructured.NestedSlice(statusMap, "conditions")
			errHandle(err)

			if foundConditions {
				for _, condition := range conditions {
					conditionMap, ok := condition.(map[string]interface{})
					if !ok {
						fmt.Println("Erreur: le type de la condition n'est pas une carte.")
						continue
					}

					reason, _ := conditionMap["reason"].(string)
					status, _ := conditionMap["status"].(string)
					cType, _ := conditionMap["type"].(string)

					fmt.Printf("externalSecret  %s >> Reason: %s, Status: %s, Type: %s\n", es.Object["metadata"].(map[string]interface{})["name"], reason, status, cType)

					if reason != "SecretSynced" {
						problematicExternalSecrets = append(problematicExternalSecrets, map[string]string{
							"Name":   es.Object["metadata"].(map[string]interface{})["name"].(string),
							"Status": reason,
							"Ready":  status,
						})
					}
				}
			} else {
				fmt.Println("Pas de champ 'conditions' trouvé dans le champ 'status'.")
			}
		} else {
			fmt.Println("Pas de champ 'status' trouvé.")
		}
	}
	if len(problematicExternalSecrets) > 0 {
		fmt.Printf("%v[ ERROR ] >> There are some issues with externalSecrets':\n", cRed)
		for _, es := range problematicExternalSecrets {
			fmt.Printf("%vName: %s, Status: %s, Ready: %s \n", cBlue, es["Name"], es["Status"], es["Ready"])
		}
	} else {
		fmt.Println("No issues found with externalSecrets")
	}

}

type CustomResource struct {
	group   string
	version string
	kind    string
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

	// Spécifier le groupe, la version et le plural de la ressource personnalisée
	var cr CustomResource
	cr.group = "external-secrets.io"
	cr.version = "v1beta1"
	cr.kind = "externalsecrets"

	troubleShootExternalSecret(cr, kubeDynamicClient, nsChoice)

}
