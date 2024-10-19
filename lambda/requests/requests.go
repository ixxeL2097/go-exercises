package requests

import (
	"time"
)

type ModifyRequest struct {
	Path      []string    // chemin dans l'objet (ex: []string{"spec", "template", "spec"})
	Value     interface{} // valeur à mettre (peut être map[string]string, string, int, etc)
	Operation string      // "update", "merge", "delete" par exemple
}

func RESTART_DEPLOY() ModifyRequest {
	return ModifyRequest{
		Path: []string{"spec", "template", "metadata", "annotations"},
		Value: map[string]string{
			"kubectl.kubernetes.io/restartedAt": time.Now().Format(time.RFC3339),
		},
		Operation: "merge",
	}
}
