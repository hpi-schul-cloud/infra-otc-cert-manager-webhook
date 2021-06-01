package otcdns

import (
	"encoding/json"
	"fmt"

	otc "github.com/opentelekomcloud/gophertelekomcloud"
	otcos "github.com/opentelekomcloud/gophertelekomcloud/openstack"
	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"

	// cmmeta1 "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	cmmeta1 "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
)

// ===========================================================================
// Kubernetes configuration
// ===========================================================================

// OtcDnsConfig is a structure that is used to decode into when
// solving a DNS01 challenge.
//
// This information is provided by cert-manager, and may be a reference to
// additional configuration that's needed to solve the challenge for this
// particular certificate or issuer.
//
// This typically includes references to Secret resources containing DNS
// provider credentials, in cases where a 'multi-tenant' DNS solver is being
// created.
//
// If you do *not* require per-issuer or per-certificate configuration to be
// provided to your webhook, you can skip decoding altogether in favour of
// using CLI flags or similar to provide configuration.
//
// You should not include sensitive information here. If credentials need to
// be used by your provider here, you should reference a Kubernetes Secret
// resource and fetch these credentials using a Kubernetes clientset.
type OtcDnsConfig struct {
	// Only for testing. Fallback to store secret as Kuberentes configuration, when it is not configured as Kubernetes secret.
	// Especially useful until it becomes clear how to inject secrets in kubebuilder.
	AccessKey string `json:"accessKey"`
	// Only for testing. Fallback to store secret as Kuberentes configuration, when it is not configured as Kubernetes secret.
	// Especially useful until it becomes clear how to inject secrets in kubebuilder.
	SecretKey string `json:"secretKey"`
	// Location of the access key secret. The access key will be loaded from this secret reference.
	AccessKeySecretRef cmmeta1.SecretKeySelector `json:"accessKeySecretRef"`
	// Location of the secret key secret.  The secret key will be loaded from this secret reference.
	SecretKeySecretRef cmmeta1.SecretKeySelector `json:"secretKeySecretRef"`
	//
	Region string `json:"region"`
	//
	AuthURL string `json:"authURL"`
}

//
// The "config" part of the solver configuration is given to us with the ChallengeRequest
// in a plain json format.
// We unmarshal that json here and integrate it into our OtcDnsConfig object.
//
// Note that the returned configuration should not contain secrets only references to
// the secrets we need to access the otcdns.
//
func configJsonToOtcDnsConfig(cfgJSON *extapi.JSON) (OtcDnsConfig, error) {
	cfg := OtcDnsConfig{}
	// handle the 'base case' where no configuration has been provided
	if cfgJSON == nil {
		return cfg, nil
	}
	if err := json.Unmarshal(cfgJSON.Raw, &cfg); err != nil {
		return cfg, fmt.Errorf("error decoding solver config: %v", err)
	}

	return cfg, nil
}

// ===========================================================================
// Local configuration (Environment, cloud.yaml)
// ===========================================================================

const (
	envPrefix string = "OS_"
	// otcuser, otcaksk
	OtcProfileNameUser string = "otcuser"
	OtcProfileNameAkSk string = "otcaksk"
)

//
// Creates a ProviderClient and authenticates it, with a configuration we load from Kubernetes.
//
// https://github.com/opentelekomcloud/gophertelekomcloud/blob/v0.3.2/auth_options.go
//
func getProviderClientWithAccessKeyAuth(authOpts otc.AuthOptionsProvider) (*otc.ProviderClient, error) {
	provider, err := otcos.AuthenticatedClient(authOpts)
	if err != nil {
		return nil, fmt.Errorf("provider creation has failed: %s", err)
	}
	return provider, nil
}

var EnvOS = otcos.NewEnv(envPrefix)

//
// Creates a ProviderClient and authenticates it, with the local cloud configuration.
// A local ~/.config/openstack/clouds.yaml with an <otcProfileName> profile must be loadable.
// See also: gophertelekomcloud/acceptance/clients/clients.go
//
func getProviderClient() (*otc.ProviderClient, error) {
	return getProviderClientProfile(OtcProfileNameUser)
}

//
// Creates a ProviderClient and authenticates it, with the local cloud configuration.
// A local ~/.config/openstack/clouds.yaml with an <otcProfileName> profile must be loadable.
// See also: gophertelekomcloud/acceptance/clients/clients.go
//
func getProviderClientProfile(otcProfileName string) (*otc.ProviderClient, error) {

	client, err := EnvOS.AuthenticatedClient(otcProfileName)
	if err != nil {
		return nil, fmt.Errorf("cloud and provider creation has failed: %s", err)
	}

	return client, nil
}

//
// Loads the configuration(s) into a 'Cloud' object.
// The configuration is loaded from the environment and/or the clouds.yaml.
//
func getCloud() (*otcos.Cloud, error) {
	return getCloudProfile(OtcProfileNameUser)
}

//
// Loads the configuration(s) into a 'Cloud' object.
// The configuration is loaded from the environment and/or the clouds.yaml.
//
func getCloudProfile(otcProfileName string) (*otcos.Cloud, error) {

	cloud, err := EnvOS.Cloud(otcProfileName)
	if err != nil {
		return nil, fmt.Errorf("error constructing cloud configuration: %s", err)
	}

	cloud, err = copyCloud(cloud)
	if err != nil {
		return nil, fmt.Errorf("error copying cloud: %s", err)
	}

	return cloud, nil
}

//
// Creates a deep copy of the given cloud data.
// Returns the copy.
// See also gophertelekomcloud/acceptance/clients/clients.go
//
func copyCloud(src *otcos.Cloud) (*otcos.Cloud, error) {
	srcJson, err := json.Marshal(src)
	if err != nil {
		return nil, fmt.Errorf("error marshalling cloud: %s", err)
	}

	res := new(otcos.Cloud)
	if err := json.Unmarshal(srcJson, res); err != nil {
		return nil, fmt.Errorf("error unmarshalling cloud: %s", err)
	}

	return res, nil
}
