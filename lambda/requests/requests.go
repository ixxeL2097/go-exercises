package requests

import (
	"time"
)

type ModifyRequest struct {
	Path      []string
	Value     interface{}
	Operation string
}

func RESTART_DEPLOY() ModifyRequest {
	time := time.Now().Format(time.RFC3339)
	return ModifyRequest{
		Path: []string{"spec", "template", "metadata", "annotations"},
		Value: map[string]string{
			"kubectl.kubernetes.io/restartedAt": time,
		},
		Operation: "merge",
	}
}
