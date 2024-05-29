package main

import (
	"os"

	"github.com/cert-manager/cert-manager/pkg/acme/webhook/cmd"
	"github.com/hpi-schul-cloud/infra-otc-cert-manager-webhook/otcdns"
	"k8s.io/klog"
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
	klog.V(6).Infof("GroupName is %s. Running webhook server", GroupName)
	cmd.RunWebhookServer(GroupName, otcdns.NewSolver())
	klog.V(6).Infof("Webhook server started")
}

func getGroupName() string {
	var groupName string = "infra-otc-cert-manager-webhook.hpi-schul-cloud.github.com"
	if os.Getenv("GROUP_NAME") == "" {
		return groupName
	} else {
		return os.Getenv("GROUP_NAME")
	}
}
