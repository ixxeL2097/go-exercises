package server

import (
	"lambda/controllers"
	"lambda/health"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

func NewRouter(kubeClient kubernetes.Interface, kubeDynamicClient dynamic.Interface) *gin.Engine {
	gin.ForceConsoleColor()
	router := gin.Default()
	router.SetTrustedProxies(nil)
	router.Use(gzip.Gzip(gzip.DefaultCompression))
	healthRouter := new(health.HealthController)

	deploymentController := &controllers.DeploymentController{
		KubeClient:        kubeClient,
		KubeDynamicClient: kubeDynamicClient,
	}

	router.GET("/readyz", healthRouter.Ready)
	router.GET("/healthz", healthRouter.Status)

	// router.GET("/readyz", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ready"}) })
	// router.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"status": "healthy"}) })

	v1 := router.Group("/v1")
	{
		deployments := v1.Group("/deployments")
		{
			deployments.POST("/restart", deploymentController.HandleRestartDeployment)
			deployments.POST("/list", deploymentController.HandleListDeployments)
		}
	}
	return router
}
