package server

import (
	"context"
	"encoding/json"
	"fmt"
	"lambda/k8s"
	"lambda/logger"
	"lambda/requests"
	"net/http"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

type APIHandler struct {
	KubeClient        kubernetes.Interface
	KubeDynamicClient dynamic.Interface
}

func (h *APIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/":
		h.HomePage(w, r)
	case "/v1/deployments/restart":
		h.HandleRestartDeployment(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *APIHandler) HomePage(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintln(w, "<h1>Welcome to lambda Application!</h1>")
}

func (h *APIHandler) HandleRestartDeployment(w http.ResponseWriter, r *http.Request) {
	logger.Logger.Info("Processing deployment restart request")
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		logger.Logger.Error("Wrong API HTTP method", "method", r.Method)
		return
	}

	var req struct {
		Deploy    string `json:"deploy"`
		Namespace string `json:"namespace"`
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		logger.Logger.Error("Failed to decode JSON")
		return
	}

	if req.Deploy == "" {
		http.Error(w, "You should specify a deployment name", http.StatusBadRequest)
		logger.Logger.Error("Missing deployment name in request")
		return
	}
	logger.Logger.Info("Restarting deployment", "deployment", req.Deploy, "namespace", req.Namespace)

	deployment, err := k8s.GetDeployment(req.Deploy, req.Namespace, h.KubeClient)
	if err != nil {
		logger.Logger.Error("Error getting deployment", "deploy", req.Deploy, "error", err)
		http.Error(w, fmt.Sprintf("Error getting deployment: %v", err), http.StatusInternalServerError)
		return
	}
	logger.Logger.Info("Updating resource", "deployment", req.Deploy, "namespace", req.Namespace)

	if err := k8s.UpdateResource(context.Background(), h.KubeDynamicClient, deployment, requests.RESTART_DEPLOY()); err != nil {
		logger.Logger.Error("Error updating resource", "resource", deployment, "error", err)
		http.Error(w, fmt.Sprintf("Error updating resource: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Deployment %s restarted successfully", req.Deploy)))
}
