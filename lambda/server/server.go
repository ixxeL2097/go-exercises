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
	case "/v1/deployments/restart":
		h.handleRestartDeployment(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *APIHandler) handleRestartDeployment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Deploy    string `json:"deploy"`
		Namespace string `json:"namespace"`
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	logger.ErrHandle(err)

	if req.Deploy == "" {
		http.Error(w, "You should specify a deployment name", http.StatusBadRequest)
		return
	}

	deployment := k8s.GetDeployment(req.Deploy, req.Namespace, h.KubeClient)

	k8s.UpdateResource(context.Background(), h.KubeDynamicClient, deployment, requests.RESTART_DEPLOY())

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Deployment %s restarted successfully", req.Deploy)))
}
