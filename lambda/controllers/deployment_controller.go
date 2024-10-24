package controllers

import (
	"context"
	"fmt"
	"lambda/k8s"
	"lambda/requests"
	"lambda/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

type ModifyRequest struct {
	Path      []string
	Value     interface{}
	Operation string
}

type DeploymentController struct {
	KubeClient        kubernetes.Interface
	KubeDynamicClient dynamic.Interface
}

func GetRestartDeploymentAnnotations() ModifyRequest {
	time := time.Now().Format(time.RFC3339)
	return ModifyRequest{
		Path: []string{"spec", "template", "metadata", "annotations"},
		Value: map[string]string{
			"kubectl.kubernetes.io/restartedAt": time,
		},
		Operation: "merge",
	}
}

func (h *DeploymentController) HandleListDeployments(c *gin.Context) {
	log.Info("Processing deployments list request")

	var req struct {
		Namespace string `json:"namespace"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid JSON payload")
		return
	}
	if req.Namespace == "" {
		utils.Error(c, http.StatusBadRequest, "You should specify a namespace name")
		return
	}
	log.Info("Listing deployments in namespace: ", req.Namespace)
	deployments, err := k8s.GetDeploymentsList(req.Namespace, h.KubeClient)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, fmt.Sprintf("Error listing deployments: %v", err))
		return
	}
	var deploymentList []struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	}
	for _, deploy := range deployments.Items {
		deploymentList = append(deploymentList, struct {
			Name      string "json:\"name\""
			Namespace string "json:\"namespace\""
		}{Name: deploy.Name, Namespace: deploy.Namespace})
	}
	utils.Success(c, "Deployments listed successfully", deploymentList)
}

func (h *DeploymentController) HandleRestartDeployment(c *gin.Context) {
	log.Info("Processing deployment restart request")

	var req struct {
		Deploy    string `json:"deploy"`
		Namespace string `json:"namespace"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if req.Deploy == "" {
		utils.Error(c, http.StatusBadRequest, "You should specify a deployment name")
		return
	}

	log.Info("Restarting deployment: ", "deployment-> ", req.Deploy, " namespace-> ", req.Namespace)

	deployment, err := k8s.GetDeployment(req.Deploy, req.Namespace, h.KubeClient)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, fmt.Sprintf("Error getting deployment: %v", err))
		return
	}

	log.Info("Updating resource: ", "deployment-> ", req.Deploy, " namespace-> ", req.Namespace)
	if err := k8s.UpdateResource(context.Background(), h.KubeDynamicClient, deployment, requests.GetRestartDeploymentAnnotations()); err != nil {
		utils.Error(c, http.StatusInternalServerError, fmt.Sprintf("Error updating resource: %v", err))
		return
	}
	utils.Success(c, fmt.Sprintf("Deployment %s restarted successfully", req.Deploy), nil)
}
