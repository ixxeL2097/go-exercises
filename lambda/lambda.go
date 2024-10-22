package main

import (
	"lambda/k8s"
	"lambda/logger"
	"lambda/server"
	"net/http"
	"os"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

func main() {

	kubeConfigPath := k8s.GetKubeConfigPath()
	kubeClient, err := k8s.CreateKubeClient(kubeConfigPath, "static")
	if err != nil {
		logger.Logger.Error("Failed to create static kube client", err)
	}
	kubeDynamicClient, err := k8s.CreateKubeClient(kubeConfigPath, "dynamic")
	if err != nil {
		logger.Logger.Error("Failed to create dynamic kube client", err)
	}

	apiHandler := &server.APIHandler{
		KubeClient:        kubeClient.(kubernetes.Interface),
		KubeDynamicClient: kubeDynamicClient.(dynamic.Interface),
	}
	mux := http.NewServeMux()
	mux.Handle("/v1/deployments/restart", apiHandler)

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8081"
	}

	logger.Logger.Info("Starting server on port", "port", port)
	err = http.ListenAndServe(":"+port, mux)
	if err != nil {
		logger.Logger.Fatalf("Failed to serve on port %v : %v", port, err)
	}
}
