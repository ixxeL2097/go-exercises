package customresource

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"kubectl/charm"
	"kubectl/k8s"
	"kubectl/logger"
	"time"
)

type CustomResourceDefinition interface {
	setGroup(group string)
	getGroup() string
	setVersion(version string)
	getVersion() string
	setKind(kind string)
	getKind() string
	setSuccessCondition(successCondition string)
	getSuccessCondition() string
	setPrettyName(prettyName string)
	getPrettyName() string
	GetCRList(kubeClient dynamic.Interface, kubeStaticClient kubernetes.Interface, namespace string) ([][]string, []map[string]string)
	DisplayCRIssue(CRListIssue []map[string]string)
	AnalyzeCRStatus(kubeClient dynamic.Interface, kubeStaticClient kubernetes.Interface, namespace string)
}

type CustomResource struct {
	group            string
	version          string
	kind             string
	successCondition string
	prettyName       string
}

func (cr *CustomResource) setGroup(group string) {
	cr.group = group
}

func (cr *CustomResource) getGroup() string {
	return cr.group
}

func (cr *CustomResource) setVersion(version string) {
	cr.version = version
}

func (cr *CustomResource) getVersion() string {
	return cr.version
}
func (cr *CustomResource) setKind(kind string) {
	cr.kind = kind
}

func (cr *CustomResource) getKind() string {
	return cr.kind
}
func (cr *CustomResource) setSuccessCondition(successCondition string) {
	cr.successCondition = successCondition
}

func (cr *CustomResource) getSuccessCondition() string {
	return cr.successCondition
}
func (cr *CustomResource) setPrettyName(prettyName string) {
	cr.prettyName = prettyName
}

func (cr *CustomResource) getPrettyName() string {
	return cr.prettyName
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

func GetCRD(crdType string, group string, kind string, version string) CustomResourceDefinition {
	if crdType == "externalsecret" {
		return NewExternalSecret(group, kind, version)
	}
	if crdType == "kustomization" {
		return NewKustomization(group, kind, version)
	}
	if crdType == "gitrepository" {
		return NewGitRepository(group, kind, version)
	}
	if crdType == "helmrelease" {
		return NewHelmRelease(group, kind, version)
	}
	if crdType == "helmrepository" {
		return NewHelmRepository(group, kind, version)
	}
	logger.ErrHandle(fmt.Errorf("Wrong CRD type passed : %v", crdType))
	return nil
}

func NewExternalSecret(group string, kind string, version string) CustomResourceDefinition {
	if group == "" {
		group = "external-secrets.io"
	}
	if kind == "" {
		kind = "externalsecrets"
	}
	if version == "" {
		version = "v1beta1"
	}
	return &ExternalSecret{
		CustomResource: CustomResource{
			group:            group,
			kind:             kind,
			version:          version,
			successCondition: "SecretSynced",
			prettyName:       "ExternalSecret",
		},
	}
}

func NewKustomization(group string, kind string, version string) CustomResourceDefinition {
	if group == "" {
		group = "kustomize.toolkit.fluxcd.io"
	}
	if kind == "" {
		kind = "kustomizations"
	}
	if version == "" {
		version = "v1"
	}
	return &Kustomization{
		CustomResource: CustomResource{
			group:            group,
			kind:             kind,
			version:          version,
			successCondition: "ReconciliationSucceeded",
			prettyName:       "Kustomization",
		},
	}
}

func NewGitRepository(group string, kind string, version string) CustomResourceDefinition {
	if group == "" {
		group = "source.toolkit.fluxcd.io"
	}
	if kind == "" {
		kind = "gitrepositories"
	}
	if version == "" {
		version = "v1"
	}
	return &GitRepository{
		CustomResource: CustomResource{
			group:            group,
			kind:             kind,
			version:          version,
			successCondition: "Succeeded",
			prettyName:       "GitRepository",
		},
	}
}

func NewHelmRelease(group string, kind string, version string) CustomResourceDefinition {
	if group == "" {
		group = "helm.toolkit.fluxcd.io"
	}
	if kind == "" {
		kind = "helmreleases"
	}
	if version == "" {
		version = "v2beta1"
	}
	return &HelmRelease{
		CustomResource: CustomResource{
			group:            group,
			kind:             kind,
			version:          version,
			successCondition: "ReconciliationSucceeded",
			prettyName:       "HelmRelease",
		},
	}
}

func NewHelmRepository(group string, kind string, version string) CustomResourceDefinition {
	if group == "" {
		group = "source.toolkit.fluxcd.io"
	}
	if kind == "" {
		kind = "helmrepositories"
	}
	if version == "" {
		version = "v2beta1"
	}
	return &HelmRepository{
		CustomResource: CustomResource{
			group:            group,
			kind:             kind,
			version:          version,
			successCondition: "Succeeded",
			prettyName:       "HelmRepository",
		},
	}
}

func (cr *CustomResource) GetCRList(kubeDynamicClient dynamic.Interface, kubeStaticClient kubernetes.Interface, namespace string) ([][]string, []map[string]string) {
	logger.Logger.Debug("Looking for customResource", "kind", cr.getPrettyName(), "namespace", namespace)
	var customResource = schema.GroupVersionResource{Group: cr.getGroup(), Version: cr.getVersion(), Resource: cr.getKind()}
	customResources, err := kubeDynamicClient.Resource(customResource).Namespace(namespace).List(context.Background(), metav1.ListOptions{})
	if customResources == nil || len(customResources.Items) == 0 {
		logger.Logger.Info("No CustomResource found", "kind", cr.getPrettyName(), "namespace", namespace)
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

					if reason != cr.getSuccessCondition() {
						CREvents := k8s.GetEventsFromResource(kubeStaticClient, cr.getPrettyName(), namespace, custom.Object["metadata"].(map[string]interface{})["name"].(string))
						issue := map[string]string{
							"Name":    custom.Object["metadata"].(map[string]interface{})["name"].(string),
							"Status":  reason,
							"Ready":   status,
							"Message": message,
						}
						for _, event := range CREvents {
							issue["EventMessage"] += event.Message
						}
						CRListIssue = append(CRListIssue, issue)
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
		logger.Logger.Error("ISSUE DETECTED", "kind", cr.getPrettyName())
		for _, pb := range CRListIssue {
			logger.Logger.Error("Unsynced/NotReady", "kind", cr.getPrettyName(), "name", pb["Name"], "status", pb["Status"], "ready", pb["Ready"], "errors", pb["EventMessage"]+"\n")
		}
	} else {
		logger.Logger.Info("All CustomResources are healthy", "kind", cr.getPrettyName())
	}
}

func (cr *CustomResource) AnalyzeCRStatus(kubeDynamicClient dynamic.Interface, kubeStaticClient kubernetes.Interface, namespace string) {
	CRList, CRListIssue := cr.GetCRList(kubeDynamicClient, kubeStaticClient, namespace)
	if CRList != nil {
		logger.Logger.Debug("Listing customResource", "kind", cr.getPrettyName(), "namespace", namespace)
		charm.CreateObjectArray(CRList, make([]string, 0))
		if CRListIssue != nil {
			cr.DisplayCRIssue(CRListIssue)
		}
	}
}
