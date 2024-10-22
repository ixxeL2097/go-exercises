package main

import (
	"lambda/k8s"
	"lambda/server"
	"os"

	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

func main() {
	kubeConfigPath := k8s.GetKubeConfigPath()

	kubeClient, err := k8s.CreateKubeClient(kubeConfigPath, "static")
	if err != nil {
		log.Fatal("Error creating static kube client ", err)
	}
	kubeDynamicClient, err := k8s.CreateKubeClient(kubeConfigPath, "dynamic")
	if err != nil {
		log.Fatal("Error creating dynamic kube client ", err)
	}

	router := server.NewRouter(kubeClient.(kubernetes.Interface), kubeDynamicClient.(dynamic.Interface))

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8081"
	}

	log.Infof("Starting server on port %v", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to serve on port %v : %v", port, err)
	}
}
