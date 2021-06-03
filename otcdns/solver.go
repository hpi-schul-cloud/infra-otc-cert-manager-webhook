package otcdns

import (
	"context"
	"fmt"
	"strings"

	"github.com/jetstack/cert-manager/pkg/acme/webhook"
	"github.com/jetstack/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	otc "github.com/opentelekomcloud/gophertelekomcloud"

	// apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	// "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
)

func NewSolver() webhook.Solver {
	return &OtcDnsSolver{}
}

// Solver implements the provider-specific logic needed to
// 'present' an ACME challenge TXT record for your own DNS provider.
// To do so, it must implement the `github.com/jetstack/cert-manager/pkg/acme/webhook.Solver`
// interface.
type OtcDnsSolver struct {
	client *kubernetes.Clientset
}

type otcdnsSecrets struct {
	AccessKey string
	SecretKey string
}

// Name is used as the name for this DNS solver when referencing it on the ACME
// Issuer resource.
// This should be unique **within the group name**, i.e. you can have two
// solvers configured with the same Name() **so long as they do not co-exist
// within a single webhook deployment**.
// For example, `cloudflare` may be used as the name of a solver.
func (s *OtcDnsSolver) Name() string {
	return "otcdns"
}

// Initialize will be called when the webhook first starts.
// This method can be used to instantiate the webhook, i.e. initialising
// connections or warming up caches.
// Typically, the kubeClientConfig parameter is used to build a Kubernetes
// client that can be used to fetch resources from the Kubernetes API, e.g.
// Secret resources containing credentials used to authenticate with DNS
// provider accounts.
// The stopCh can be used to handle early termination of the webhook, in cases
// where a SIGTERM or similar signal is sent to the webhook process.
func (s *OtcDnsSolver) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {

	clientSet, err := kubernetes.NewForConfig(kubeClientConfig)
	if err != nil {
		return err
	}

	s.client = clientSet
	return nil
}

//
// Present is responsible for actually presenting the DNS record with the DNS provider.
// This method should tolerate being called multiple times with the same value.
// cert-manager itself will later perform a self check to ensure that the solver has correctly configured the DNS provider.
//
// challengeRequest: The challenge request to resolve. The challenge request contains the configuration, that is defined in
//   the webhook sections of the ClusterIssuer solver configuration.
//
func (s *OtcDnsSolver) Present(challengeRequest *v1alpha1.ChallengeRequest) error {
	klog.V(6).Infof("call function Present: namespace=%s, zone=%s, fqdn=%s", challengeRequest.ResourceNamespace, challengeRequest.ResolvedZone, challengeRequest.ResolvedFQDN)

	otcdnsClient, err := s.getOtcDnsClientFromChallengeRequest(challengeRequest)
	if err != nil {
		return fmt.Errorf("cannot present. Failed to get dns client. %s", err)
	}

	// Check, if the TXT record already exists.
	zone, err := otcdnsClient.GetHostedZone(challengeRequest.ResolvedZone)
	if err != nil {
		return fmt.Errorf("cannot present. Failed to get hosted zone %s. %s", challengeRequest.ResolvedZone, err)
	}

	safeChallengeRequestKey := s.getSafeTxtValue(challengeRequest.Key)
	challengeExists, existingRecordset, err := otcdnsClient.HasTxtRecordValue(zone, safeChallengeRequestKey)
	if err != nil {
		return fmt.Errorf("failed to check existence of DNS TXT entry. %s", err)
	}

	if challengeExists {
		// The TXT challenge request entry is already present.
		klog.V(6).Infof("challenge request entry is already present. Skipping create.")
	} else if existingRecordset == nil {
		// The whole recordset of the challenge request does not exist. Create it.
		createdRecordset, err := otcdnsClient.NewTxtRecordSet(zone, safeChallengeRequestKey)
		if err != nil {
			return fmt.Errorf("failed to create new challenge request DNS TXT entry. %s", err)
		}
		klog.V(6).Infof("created new challenge request DNS TXT entry %s with values %s", createdRecordset.Name, createdRecordset.Records)
	} else {
		// The recordset exists, but the challenge request value is missing.
		// Add record with challenge key.
		changedRecords := append(existingRecordset.Records, safeChallengeRequestKey)
		changedRecordset, err := otcdnsClient.UpdateTxtRecordValues(zone, existingRecordset, changedRecords)
		if err != nil {
			return fmt.Errorf("failed to update challenge DNS TXT entry. %s", err)
		}
		klog.V(6).Infof("changed challenge request DNS TXT entry %s with values %s", changedRecordset.Name, changedRecordset.Records)
	}

	klog.V(6).Infof("call function Present succeeded: namespace=%s, zone=%s, fqdn=%s", challengeRequest.ResourceNamespace, challengeRequest.ResolvedZone, challengeRequest.ResolvedFQDN)
	return nil
}

// CleanUp should delete the relevant TXT record from the DNS provider console.
// If multiple TXT records exist with the same record name (e.g. _acme-challenge.example.com) then **only** the record with
// the same `key` value provided on the ChallengeRequest should be cleaned up.
// This is in order to facilitate multiple DNS validations for the same domain concurrently.
func (s *OtcDnsSolver) CleanUp(challengeRequest *v1alpha1.ChallengeRequest) error {
	klog.V(6).Infof("CleanUp: namespace=%s, zone=%s, fqdn=%s", challengeRequest.ResourceNamespace, challengeRequest.ResolvedZone, challengeRequest.ResolvedFQDN)

	otcdnsClient, err := s.getOtcDnsClientFromChallengeRequest(challengeRequest)
	if err != nil {
		return fmt.Errorf("cannot present. Failed to get dns client. %s", err)
	}

	// Check, if the TXT record exists.
	zone, err := otcdnsClient.GetHostedZone(challengeRequest.ResolvedZone)
	if err != nil {
		return fmt.Errorf("cannot CleanUp. Failed to get hosted zone. %s", err)
	}

	safeChallengeRequestKey := s.getSafeTxtValue(challengeRequest.Key)
	challengeValueExists, existingRecordset, err := otcdnsClient.HasTxtRecordValue(zone, safeChallengeRequestKey)
	if err != nil {
		return fmt.Errorf("failed to check existence of DNS TXT entry. %s", err)
	}

	if challengeValueExists {
		// The TXT challenge record exists. Delete the value or the whole recordset, if it is the last TXT value.
		changedRecordSet, err := otcdnsClient.DeleteTxtRecordValue(zone, safeChallengeRequestKey, true)
		if err != nil {
			return fmt.Errorf("failed to delete DNS TXT entry %s. %s", safeChallengeRequestKey, err)
		}
		if changedRecordSet != nil {
			klog.V(6).Infof("CleanUp removed one challenge key %s from the recordset %s", safeChallengeRequestKey, changedRecordSet.Name)
		} else {
			klog.V(6).Infof("CleanUp detected that this was the last TXT value in the recordset. Recordset %s deleted", existingRecordset.Name)
		}
	} else if existingRecordset == nil {
		// The TXT challenge recordset does not exist. Nothing to do.
		klog.V(6).Infof("CleanUp not needed. The challenge request DNS TXT recordset does not exit. Skipping delete for challenge value %s", safeChallengeRequestKey)
	} else {
		// The TXT challenge value does not exist. Nothing to do.
		klog.V(6).Infof("CleanUp no needed. The challenge request DNS TXT challenge value %s does not exit in recordset %s. Skipping delete", safeChallengeRequestKey, existingRecordset.Name)
	}

	klog.V(6).Infof("CleanUp succeeded: namespace=%s, zone=%s, fqdn=%s", challengeRequest.ResourceNamespace, challengeRequest.ResolvedZone, challengeRequest.ResolvedFQDN)
	return nil
}

//
// Create a otcDnsClient using the given information in the challenge.
//
func (s *OtcDnsSolver) getOtcDnsClientFromChallengeRequest(challengeRequest *v1alpha1.ChallengeRequest) (*OtcDnsClient, error) {
	// Get the configuration from the challenge request.
	// For the test this is injected via the config.json located in the ManifestPath (see SetManifestPath).
	// For a real Kubernetes environment an example for the manifest yaml file can be found in _examples/secret_otcdns_credential.yaml
	solverWebhookConfig, err := configJsonToOtcDnsConfig(challengeRequest.Config)
	if err != nil {
		return nil, fmt.Errorf("cannot create otcDnsClient. Json not converted. %s", err)
	}
	// fmt.Printf("Decoded configuration %v", solverWebhookConfig)
	// klog.V(6).Infof("decoded configuration %v", solverWebhookConfig)

	// Get the secrets from Kubernetes
	secrets, err := s.getOtcDnsSecrets(&solverWebhookConfig, challengeRequest.ResourceNamespace)
	if err != nil {
		return nil, fmt.Errorf("cannot create otcDnsClient. Secrets not read. %s", err)
	}

	// Create the input parameters for the OtcDnsClient
	// We use the accesskey/secretkey authentication here.
	authOpts := otc.AKSKAuthOptions{
		IdentityEndpoint: solverWebhookConfig.AuthURL,
		AccessKey:        secrets.AccessKey,
		SecretKey:        secrets.SecretKey,
	}

	endpointOpts := otc.EndpointOpts{
		Region: solverWebhookConfig.Region,
	}

	klog.Infof("========================================================================================")
	klog.Infof("authOpts.IdentityEndpoint=%s, authOpts.AccessKey=%s, endpointOpts.Region=%s, endpointOpts=%s", authOpts.IdentityEndpoint, authOpts.AccessKey, endpointOpts.Region, endpointOpts)

	// Create the client
	otcDnsClient, err := NewDNSV2ClientWithAuth(authOpts, endpointOpts)
	// This is an alternative way to create a client
	// otcdnsClient, err := NewDNSV2Client()
	if err != nil {
		return nil, fmt.Errorf("cannot create otcDnsClient. Failed to instantiate. %s", err)
	}

	subdomain, _ := s.extractDomainAndSubdomainFromChallengeRequest(challengeRequest)
	otcDnsClient.Subdomain = subdomain

	return otcDnsClient, err
}

//
// Turns the given challenge key into a safe value we can store in DNS.
//
func (s *OtcDnsSolver) getSafeTxtValue(key string) string {
	safeKey := "\"" + key + "\""
	return safeKey
}

//
// Extract domain and subdomain in a form we can process from the challenge request.
//
func (s *OtcDnsSolver) extractDomainAndSubdomainFromChallengeRequest(challengeRequest *v1alpha1.ChallengeRequest) (string, string) {
	// Extract subdomain by trimming challengeRequest.ResolvedZone (e.g. example.com.) from challengeRequest.ResolvedFQDN (e.g. _acme-challenge.example.com.)
	subDomain := strings.TrimSuffix(challengeRequest.ResolvedFQDN, challengeRequest.ResolvedZone)
	// Trim trailing '.'
	subDomain = strings.TrimSuffix(subDomain, ".")
	// Trim trailing '.'
	domain := strings.TrimSuffix(challengeRequest.ResolvedZone, ".")
	return subDomain, domain
}

//
// The given webhook configuration contains the definitions of references to the secrets we want to load.
//
func (s *OtcDnsSolver) getOtcDnsSecrets(config *OtcDnsConfig, namespace string) (*otcdnsSecrets, error) {

	secs := otcdnsSecrets{}

	if config.AccessKey != "" {
		// Secret configured directly in configuration. This shortcut must never be used in production.
		secs.AccessKey = config.AccessKey
	} else {
		var err error
		secs.AccessKey, err = s.getReferencedSecret(namespace, config.AccessKeySecretRef.Name, config.AccessKeySecretRef.Key)
		if err != nil {
			return nil, fmt.Errorf("cannot get access key: %s", err)
		}
	}

	if config.SecretKey != "" {
		// Secret configured directly in configuration. This shortcut must never be used in production.
		secs.SecretKey = config.SecretKey
	} else {
		var err error
		secs.SecretKey, err = s.getReferencedSecret(namespace, config.SecretKeySecretRef.Name, config.SecretKeySecretRef.Key)
		if err != nil {
			return nil, fmt.Errorf("cannot get secret: %s", err)
		}
	}

	return &secs, nil
}

//
// Takes the given references and tries to load the secrets from the reference locations.
//
func (s *OtcDnsSolver) getReferencedSecret(namespace string, keyRefName string, keyRefKey string) (string, error) {
	secret, err := s.client.CoreV1().Secrets(namespace).Get(context.Background(), keyRefName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to load secret %q. %s", namespace+"/"+keyRefName, err)
	}
	if accessKey, ok := secret.Data[keyRefKey]; ok {
		return string(accessKey), nil
	} else {
		return "", fmt.Errorf("key %q not found in secret %q", keyRefKey, namespace+"/"+keyRefName)
	}
}
