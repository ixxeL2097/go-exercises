package customresource

import (
	"context"
	//"github.com/charmbracelet/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"kubectl/logger"
	"time"
)

type CustomResourceDefinition interface {
	GetCRList(kubeClient dynamic.Interface, namespace string) ([][]string, []map[string]string)
	DisplayCRIssue(CRListIssue []map[string]string)
	NewCR() CustomResourceDefinition
}

type CustomResource struct {
	Group            string
	Version          string
	Kind             string
	SuccessCondition string
	PrettyName       string
}

type ExternalSecret struct {
	CustomResource
}

type Kustomization struct {
	CustomResource
}

func (e *ExternalSecret) NewCR() CustomResourceDefinition {
	return &ExternalSecret{
		CustomResource: CustomResource{
			Group:            "external-secrets.io",
			Version:          "v1beta1",
			Kind:             "externalsecrets",
			SuccessCondition: "SecretSynced",
			PrettyName:       "ExternalSecret",
		},
	}
}

func (e *Kustomization) NewCR() CustomResourceDefinition {
	return &Kustomization{
		CustomResource: CustomResource{
			Group:            "kustomize.toolkit.fluxcd.io",
			Version:          "v1",
			Kind:             "kustomizations",
			SuccessCondition: "SecretSynced",
			PrettyName:       "Kustomization",
		},
	}
}

func (cr *CustomResource) GetCRList(kubeClient dynamic.Interface, namespace string) ([][]string, []map[string]string) {
	var customResource = schema.GroupVersionResource{Group: cr.Group, Version: cr.Version, Resource: cr.Kind}
	customResources, err := kubeClient.Resource(customResource).Namespace(namespace).List(context.Background(), metav1.ListOptions{})
	if customResources == nil || len(customResources.Items) == 0 {
		logger.Logger.Info("No CustomResource found", "kind", cr.PrettyName)
		return nil, nil
	}
	logger.ErrHandle(err)

	CRList := make([][]string, 0)
	CRListIssue := make([]map[string]string, 0)

	for _, custom := range customResources.Items {

		statusMap, foundStatus, err := unstructured.NestedMap(custom.Object, "status")
		logger.ErrHandle(err)

		if foundStatus {
			conditions, foundConditions, err := unstructured.NestedSlice(statusMap, "conditions")
			logger.ErrHandle(err)

			if foundConditions {
				var latestCondition map[string]interface{}
				latestTime := time.Time{}
				latestIndex := -1

				for i, condition := range conditions {
					conditionMap, ok := condition.(map[string]interface{})
					if !ok {
						logger.Logger.Fatal("condition is not a Map object", "object", custom.Object)
					}
					transitionTimeStr, _ := conditionMap["lastTransitionTime"].(string)
					transitionTime, err := time.Parse(time.RFC3339, transitionTimeStr)
					logger.ErrHandle(err)

					if transitionTime.After(latestTime) || (transitionTime.Equal(latestTime) && i > latestIndex) {
						latestTime = transitionTime
						latestCondition = conditionMap
						latestIndex = i
					}
				}
				if latestCondition != nil {
					reason := latestCondition["reason"].(string)
					status := latestCondition["status"].(string)
					message := latestCondition["message"].(string)

					row := []string{
						custom.Object["metadata"].(map[string]interface{})["name"].(string),
						reason,
						status,
						message,
					}
					CRList = append(CRList, row)

					if reason != cr.SuccessCondition {
						CRListIssue = append(CRListIssue, map[string]string{
							"Name":    custom.Object["metadata"].(map[string]interface{})["name"].(string),
							"Status":  reason,
							"Ready":   status,
							"Message": message,
						})
					}
				}
			} else {
				logger.Logger.Fatal("No sub-entry 'conditions' found in entry 'status'", "object", custom.Object)
			}
		} else {
			logger.Logger.Fatal("No entry 'status' found", "object", custom.Object)
		}
	}
	return CRList, CRListIssue
}

func (cr *CustomResource) DisplayCRIssue(CRListIssue []map[string]string) {
	if len(CRListIssue) > 0 {
		logger.Logger.Error("ISSUE DETECTED", "object", cr.PrettyName)
		for _, pb := range CRListIssue {
			logger.Logger.Error("Unsynced/NotReady", "kind", cr.PrettyName, "name", pb["Name"], "status", pb["Status"], "ready", pb["Ready"], "hint", pb["Message"])
		}
	} else {
		logger.Logger.Info("All CustomResources are healthy", "kind", cr.PrettyName)
	}
}
