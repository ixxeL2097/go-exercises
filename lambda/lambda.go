package main

import (
	"context"
	"flag"
	"fmt"
	"lambda/k8s"
	"os"
	"time"
)

func GetDeploy() (string, string) {
	deployName := flag.String("deploy", "", "Name of the deployment")
	namespace := flag.String("namespace", "default", "Namespace of the deployment")
	flag.Parse()

	if *deployName == "" {
		fmt.Println("You should specify a deployment name")
		os.Exit(1)
	}
	return *deployName, *namespace
}

func main() {
	deploy, ns := GetDeploy()
	kubeConfigPath := k8s.GetKubeConfigPath()

	kubeClient := k8s.CreateClient(kubeConfigPath)
	kubeDynamicClient := k8s.CreateDynamicClient(kubeConfigPath)

	deploysList := k8s.GetDeploymentsList(ns, kubeClient)
	for _, deploy := range deploysList.Items {
		fmt.Println("Deploy: ", deploy.Name)
	}

	deployment := k8s.GetDeployment(deploy, ns, kubeClient)

	modif1 := k8s.ModifyRequest{
		Path: []string{"spec", "template", "metadata", "annotations"},
		Value: map[string]string{
			"kubectl.kubernetes.io/restartedAt": time.Now().Format(time.RFC3339),
		},
		Operation: "merge",
	}

	k8s.UpdateResource(context.Background(), kubeDynamicClient, deployment, modif1)

}
