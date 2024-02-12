package customresource

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"kubectl/charm"
	"kubectl/logger"
	"time"
)

type CustomResourceDefinition interface {
	GetCRList(kubeClient dynamic.Interface, namespace string) ([][]string, []map[string]string)
	DisplayCRIssue(CRListIssue []map[string]string)
	NewCR(group string, kind string, successCondition string, version string, prettyName string) CustomResourceDefinition
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

type GitRepository struct {
	CustomResource
}

type HelmRelease struct {
	CustomResource
}

type HelmRepository struct {
	CustomResource
}

func (e *ExternalSecret) NewCR(group string, kind string, successCondition string, version string, prettyName string) CustomResourceDefinition {
	if group == "" {
		group = "external-secrets.io"
	}
	if kind == "" {
		kind = "externalsecrets"
	}
	if successCondition == "" {
		successCondition = "SecretSynced"
	}
	if version == "" {
		version = "v1beta1"
	}
	if prettyName == "" {
		prettyName = "ExternalSecret"
	}
	return &ExternalSecret{
		CustomResource: CustomResource{
			Group:            group,
			Version:          version,
			Kind:             kind,
			SuccessCondition: successCondition,
			PrettyName:       prettyName,
		},
	}
}

func (e *Kustomization) NewCR(group string, kind string, successCondition string, version string, prettyName string) CustomResourceDefinition {
	if group == "" {
		group = "kustomize.toolkit.fluxcd.io"
	}
	if kind == "" {
		kind = "kustomizations"
	}
	if successCondition == "" {
		successCondition = "ReconciliationSucceeded"
	}
	if version == "" {
		version = "v1"
	}
	if prettyName == "" {
		prettyName = "Kustomization"
	}
	return &Kustomization{
		CustomResource: CustomResource{
			Group:            group,
			Version:          version,
			Kind:             kind,
			SuccessCondition: successCondition,
			PrettyName:       prettyName,
		},
	}
}

func (e *GitRepository) NewCR(group string, kind string, successCondition string, version string, prettyName string) CustomResourceDefinition {
	if group == "" {
		group = "source.toolkit.fluxcd.io"
	}
	if kind == "" {
		kind = "gitrepositories"
	}
	if successCondition == "" {
		successCondition = "Succeeded"
	}
	if version == "" {
		version = "v1"
	}
	if prettyName == "" {
		prettyName = "GitRepository"
	}
	return &GitRepository{
		CustomResource: CustomResource{
			Group:            group,
			Version:          version,
			Kind:             kind,
			SuccessCondition: successCondition,
			PrettyName:       prettyName,
		},
	}
}

func (e *HelmRelease) NewCR(group string, kind string, successCondition string, version string, prettyName string) CustomResourceDefinition {
	if group == "" {
		group = "helm.toolkit.fluxcd.io"
	}
	if kind == "" {
		kind = "helmreleases"
	}
	if successCondition == "" {
		successCondition = "ReconciliationSucceeded"
	}
	if version == "" {
		version = "v2beta1"
	}
	if prettyName == "" {
		prettyName = "HelmRelease"
	}
	return &HelmRelease{
		CustomResource: CustomResource{
			Group:            group,
			Version:          version,
			Kind:             kind,
			SuccessCondition: successCondition,
			PrettyName:       prettyName,
		},
	}
}

func (e *HelmRepository) NewCR(group string, kind string, successCondition string, version string, prettyName string) CustomResourceDefinition {
	if group == "" {
		group = "source.toolkit.fluxcd.io"
	}
	if kind == "" {
		kind = "helmrepositories"
	}
	if successCondition == "" {
		successCondition = "Succeeded"
	}
	if version == "" {
		version = "v2beta1"
	}
	if prettyName == "" {
		prettyName = "HelmRepository"
	}
	return &HelmRepository{
		CustomResource: CustomResource{
			Group:            group,
			Version:          version,
			Kind:             kind,
			SuccessCondition: successCondition,
			PrettyName:       prettyName,
		},
	}
}

func (cr *CustomResource) GetCRList(kubeClient dynamic.Interface, namespace string) ([][]string, []map[string]string) {
	logger.Logger.Debug("Looking for customResource", "kind", cr.PrettyName, "namespace", namespace)
	var customResource = schema.GroupVersionResource{Group: cr.Group, Version: cr.Version, Resource: cr.Kind}
	customResources, err := kubeClient.Resource(customResource).Namespace(namespace).List(context.Background(), metav1.ListOptions{})
	if customResources == nil || len(customResources.Items) == 0 {
		logger.Logger.Info("No CustomResource found", "kind", cr.PrettyName, "namespace", namespace)
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
			fmt.Println("")
		}
	} else {
		logger.Logger.Info("All CustomResources are healthy", "kind", cr.PrettyName)
		fmt.Println("")
	}
}

func (cr *CustomResource) AnalyzeCRStatus(kubeClient dynamic.Interface, namespace string) {
	CRList, CRListIssue := cr.GetCRList(kubeClient, namespace)
	if CRList != nil {
		logger.Logger.Debug("Listing customResource", "kind", cr.PrettyName, "namespace", namespace)
		charm.CreateObjectArray(CRList, make([]string, 0))
		if CRListIssue != nil {
			cr.DisplayCRIssue(CRListIssue)
		}
	}
}
