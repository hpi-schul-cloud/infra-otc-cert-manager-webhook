package main

import (
	"os"

	"github.com/hpi-schul-cloud/infra-otc-cert-manager-webhook/otcdns"
	"github.com/jetstack/cert-manager/pkg/acme/webhook/cmd"
)

var GroupName = getGroupName()

func main() {
	// infra-otc-cert-manager-webhook.hpi-schul-cloud.github.com
	if GroupName == "" {
		panic("GROUP_NAME must be specified")
	}

	// This will register our custom DNS provider with the webhook serving
	// library, making it available as an API under the provided GroupName.
	// You can register multiple DNS provider implementations with a single
	// webhook, where the Name() method will be used to disambiguate between
	// the different implementations.
	cmd.RunWebhookServer(GroupName, otcdns.NewSolver())
}

func getGroupName() string {
	var groupName string = "infra-otc-cert-manager-webhook.hpi-schul-cloud.github.com"
	if os.Getenv("GROUP_NAME") == "" {
		return groupName
	} else {
		return os.Getenv("GROUP_NAME")
	}
}
