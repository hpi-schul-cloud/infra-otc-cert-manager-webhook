# Let's Encrypt ACME Webhook for the Open Telekom Cloud DNS (OTCDNS)

This project provides a cert-manager webhook for the [Open Telekom Cloud (OTC)](https://open-telekom-cloud.com/de) DNS. 

This webhook is available on GitHub [hpi-schul-cloud /
infra-otc-cert-manager-webhook](https://github.com/hpi-schul-cloud/infra-otc-cert-manager-webhook). It is written in Go and uses the Go API of the OTC [gophertelekomcloud](https://github.com/opentelekomcloud/gophertelekomcloud). The gophertelekomcloud is part of the Open Telekom Cloud (T-Systems, Deutsche Telekom) project available on GitHub https://github.com/opentelekomcloud.

## Requirements

- [kubernetes](https://kubernetes.io/) >= v1.18.0
- [cert-manager](https://cert-manager.io/) >= 1.3.1
- [helm](https://helm.sh/) >= v3.0.0

## Configuration

The Helm chart for this project is located in the [deploy/infra-otc-cert-manager-webhook](deploy/infra-otc-cert-manager-webhook) directory.

The following table lists the configurable parameters of the infra-otc-cert-manager-webhook chart and their default values.

| Parameter | Description | Default |
| --------- | ----------- | ------- |
| `groupName` | The groupName  is used to identify your company or business unit that created this webhook. For example, this may be "acme.mycompany.com". This name will need to be referenced in each Issuer's `webhook` stanza to inform cert-manager of where to send ChallengePayload resources in order to solve the DNS01 challenge. This group name should be **unique**, hence using your own company's domain here is recommended. | `infra-otc-cert-manager-webhook.hpi-schul-cloud.github.com` |
| `credentialsSecretRef` | The name of secret where the credentials to access the OTCDNS are stored. | `otcdns-credentials` |
| `certManager.namespace` | Namespace where cert-manager is deployed to. | `cert-manager` |
| `certManager.serviceAccountName` | Service account of cert-manager installation. | `cert-manager` |
| `image.repository` | Image repository | `schulcloud/infra-otc-cert-manager-webhook` |
| `image.tag` | Image tag | `sha-6e4a13b` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `nameOverride` | Override for the chartname | `` |
| `fullnameOverride` | Override for the fullname of the chart | `` |
| `loglevel` | Number for the log level verbosity of webhook. | 2 |
| `service.type` | API service type | `ClusterIP` |
| `service.port` | API service port | `443` |
| `resources` | CPU/memory resource requests/limits | `{}` |
| `nodeSelector` | Node labels for pod assignment | `{}` |
| `affinity` | Node affinity for pod assignment | `{}` |
| `tolerations` | Node tolerations for pod assignment | `[]` |

## Installation

### cert-manager

Follow the [instructions](https://cert-manager.io/docs/installation/) using the cert-manager documentation to install it within your cluster.

### OTC Credentials

To access the OTC IAM and OTC DNS an access key and and a secret key (AK/SK) are needed. See [Automating the Open Telekom Cloud with APIs](https://open-telekom-cloud.com/en/support/tutorials/automating-opentelekomcloud-apis), chapter *API authentication*. The webhook will read this information to get access to the OTC. The user that provides the key must have grants to create and read DNS records.

An example file is provided in [_examples/secret-otcdns-credentials.yaml](_examples/secret-otcdns-credentials.yaml):
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: otcdns-credentials
  namespace: cert-manager
type: Opaque
data:
  accessKey: "[OTCDNS ACCESSKEY BASE64]"
  secretKey: "[OTCDNS SECRETKEY BASE64]"
```
- Copy the example to another directory. Preferably ignored by Git (e.g. "testdata").
- Replace the placeholders with the base64 encoded values of your OTC access user.
- Apply the secret-otcdns-credentials.yaml to your Kubernetes installation.

```kubectl apply -f secret-otcdns-credentials.yaml```

### Webhook

Install the webhook

```bash
helm repo add otcdnswebhook https://hpi-schul-cloud.github.io/infra-otc-cert-manager-webhook/
helm repo update
helm install --namespace cert-manager otcdns-release otcdnswebhook/infra-otc-cert-manager-webhook
```

To uninstall run

```bash
helm uninstall --namespace cert-manager otcdns-release
```

## Issuer

When the cert-manager finds an Ingress annotation or Certificate resource it can handle, it will start the issuing process. Multiple issuers can coexist and each issuer can have multiple solvers that help to solve the challenges. This OTCDNS webhook can be configured as solver in a `ClusterIssuer` or `Issuer` resource. For more information, see [Issuing an ACME certificate using DNS validation](https://cert-manager.io/docs/tutorials/acme/dns-validation/#issuing-an-acme-certificate-using-dns-validation)

Example files are provided in [_examples/clusterissuer-solver-dns01-webhook.yaml](_examples/clusterissuer-solver-dns01-webhook.yaml) and [_examples/clusterissuer-staging-solver-dns01-webhook.yaml](_examples/clusterissuer-staging-solver-dns01-webhook.yaml).

This is an example for Let's Encrypt staging:

```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: sc-cert-manager-clusterissuer-letsencrypt-staging-otcdns
spec:
  acme:
    # The ACME server URL
    server: https://acme-staging-v02.api.letsencrypt.org/directory

    # Email address used for ACME registration
    email: mail@example.com # REPLACE THIS WITH YOUR EMAIL!!!

    # Name of a secret used to store the ACME account private key
    privateKeySecretRef:
      name: letsencrypt-staging-otcdms

    solvers:
      - dns01:
          webhook:
            groupName: infra-otc-cert-manager-webhook.hpi-schul-cloud.github.com
            solverName: otcdns
            config:
              authURL: "https://iam.eu-de.otc.t-systems.com:443/v3"
              region: "eu-de"
              
              # Only for local testing, if no secrets are available.
              # accessKey: ACCESSKEY
              # secretKey: SECRETKEY

              accessKeySecretRef:
                name: otcdns-credentials
                key: accessKey
              secretKeySecretRef:
                name: otcdns-credentials
                key: secretKey
```
The groupName must match the groupName in the Helm chart configuration. The default value is set here and should usually be fine.

The commented out accessKey and secretKey entries are for local testing only. They shall be removed if used on Kubernetes.

accessKeySecretRef.name and secretKeySecretRef.name point to the secret created above. This will give the webhook access to the OTC API.

- Copy the example to another directory. Preferably ignored by Git (e.g. "testdata"). Use the staging or the prod yaml as template.
- Usually it is necessary to edit the email field only. The other values should be fine as they are in the template.
- Apply the edited [_examples/clusterissuer-solver-dns01-webhook.yaml](_examples/clusterissuer-solver-dns01-webhook.yaml) or [_examples/clusterissuer-staging-solver-dns01-webhook.yaml](_examples/clusterissuer-staging-solver-dns01-webhook.yaml) to your Kubernetes installation.

The cert-manager can now identify the installed OTCDNS webhook and forward the selected solver configuration to it.

## Create a certificate

To trigger the certificate creation you can a) create a Certificate resource or b) define an Ingress annotation for the cert-manager. We use method a) here.

Examples Certificate resources can be found here: [_examples/wildcard-certificate-examplesubdomain.yaml](_examples/wildcard-certificate-examplesubdomain.yaml) and [_examples/wildcard-certificate-staging-examplesubdomain.yaml](_examples/wildcard-certificate-staging-examplesubdomain.yaml)

```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: wildcard-certificate-staging-examplesubdomain
  namespace: examplesubdomain
spec:
  # commonName: *.examplesubdomain.example.com
  dnsNames:
  - '*.examplesubdomain.example.com'
  - '*.dev.examplesubdomain.example.com'
  issuerRef:
    kind: ClusterIssuer
    name: sc-cert-manager-clusterissuer-letsencrypt-staging-otcdns
  secretName: wildcard-certificate-staging-examplesubdomain-tls
```
The dnsNames will appear as common name (the first one) and als subject alternative names in the issued certificate. You must be the legitimized owner of the domain.

The issuerRef.name must match the Issuer you want to use (see above).

The secretName is the name of the secret where the certificate given by the issuer is finally stored. This is the secret that must be configured in the Ingress of your application as tls.secretName, if you want to use the certificate.

- Create the certificate yaml and upload it to Kubernetes

The cert-manager will detect it and start the issuing process. See [Troubleshooting a failed certificate request](https://cert-manager.io/docs/faq/troubleshooting/) to see how to track its state in detail.

## Development

### Requirements

- [go](https://golang.org/) >= 1.13.0

### Configure the tests

#### clouds.yaml

There is an example clouds.yaml configuration in [_examples/clouds.yaml](_examples/clouds.yaml). The clouds.yaml is part of the Openstack Telekom configuration.

- Copy it to ~/.config/openstack/
- Add the OTC credentials you want to use for testing.

There are tests that have no credential input parameters. These use the local clouds.yaml config (EnvOS). E.g. all tests that call NewDNSV2Client, will behave that way.

#### config.json

There is an example config.json in [_examples/config.json](_examples/config.json)

- Copy it to [testdata/otcdns/manifests/](testdata/otcdns/manifests/)
- Configure the OTC credentials in the accessKey and secretKey variables.

Note that the ...secretRef cannot be used in a local context. For local tests use "accessKey" and "secretKey". In Kubernetes use the "...SecretRef" entries.

The config.json is used in tests that have credentials as input parameters. E.g. all tests that call NewDNSV2Client**WithAuth** and especially the conformance test in main_test.go.

### Run the tests

#### Makefile

    Run "make" to download kubebuilder into _test/kubebuilder/bin and to run the main testsuite.

When the credentials are configured as described above, the tests shall immediatly succeed.

    Optional: Run "make rendered-manifest.yaml" to render the Helmchart into the "_out" directory. This give you an impression about the Kubernetes configuration.
    Optional: Run "make build" to locally build the Docker container.

The tests you just ran using the makefile are described in the next two chapters.

#### OTC DNS Client Tests

The test functionality concerning the OTC API is in [otcdns/client_test.go](otcdns/client_test.go).

As of today a valid OTC setup is needed. This means you need a local ~/.config/openstack/clouds.yaml. The clouds.yaml must contain a profile "otcaksk" and "otcuser" (see config.go > otcProfileName). More details can be found here [Telekom - Open Telekom Cloud extensions Python configuration](https://python-otcextensions.readthedocs.io/en/latest/install/configuration.html). There is an example clouds.yaml in the [_examples/clouds.yaml](_examples/clouds.yaml) directory.

To run all OTC DNS Client tests from the command line:

    cd otcdns
    go test -v .

#### Cert-Manager Solver Tests

The solver tests are located in main_test.go.

The solver tests rely on the kubebuilder binaries. They are installed by the first target in the Makefile.

- cd into the main project directory where the Makefile is´
- Run:

```make```

This will install the kubebuilder testenvironment and run the cert-manager solver testsuite tests within it. A docker image of the Webhook application is not needed for this.

