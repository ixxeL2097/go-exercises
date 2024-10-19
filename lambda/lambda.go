package main

import (
	"fmt"
	"lambda/k8s"
	"lambda/logger"
	"lambda/server"
	"net/http"
	"os"
)

func main() {

	kubeConfigPath := k8s.GetKubeConfigPath()
	kubeClient := k8s.CreateClient(kubeConfigPath)
	kubeDynamicClient := k8s.CreateDynamicClient(kubeConfigPath)

	apiHandler := &server.APIHandler{
		KubeClient:        kubeClient,
		KubeDynamicClient: kubeDynamicClient,
	}
	mux := http.NewServeMux()
	mux.Handle("/v1/deployments/restart", apiHandler)

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Starting server on port :%s\n", port)
	err := http.ListenAndServe(":"+port, mux)
	logger.ErrHandle(err)
}
