package main

import (
	"os"
	"testing"

	"github.com/cert-manager/cert-manager/pkg/acme/webhook"
	dns "github.com/cert-manager/cert-manager/test/acme"
	"github.com/hpi-schul-cloud/infra-otc-cert-manager-webhook/otcdns"
	"k8s.io/klog"
)

// Allows to overwrite the default value with a custom value.
func getTestZone() string {
	var testZone string = "hpi-schul-cloud.dev."
	if os.Getenv("TEST_ZONE_NAME") == "" {
		return testZone
	} else {
		return os.Getenv("TEST_ZONE_NAME")
	}
}

func TestRunsSuite(t *testing.T) {
	// The manifest path should contain a file named config.json that is a
	// snippet of valid configuration that should be included on the
	// ChallengeRequest passed as part of the test cases.

	klog.Info("Running test")
	klog.V(2).Infof("Verbosity 2 message")

	var otcdnsSolver webhook.Solver = otcdns.NewSolver()

	// The test will automatically discover these authorative servers.
	// ns1.open-telekom-cloud.com. = 80.158.48.19
	// ns2.open-telekom-cloud.com. = 93.188.242.252
	fixture := dns.NewFixture(otcdnsSolver,
		dns.SetResolvedZone(getTestZone()),
		//dns.SetResolvedFQDN(fqdn),
		dns.SetAllowAmbientCredentials(false),
		dns.SetManifestPath("testdata/otcdns/manifests"),
		//dns.SetBinariesPath("_test/kubebuilder/bin"),
		dns.SetDNSServer("80.158.48.19:53"),
		//dns.SetDNSName(testZone),
		// Enable extended tests with multiple TXT entries in one recordset.
		dns.SetStrict(true),
	)
	fixture.RunConformance(t)
}
