module github.com/hpi-schul-cloud/infra-otc-cert-manager-webhook

//module local

// https://golang.org/doc/devel/release.html
// https://golang.org/doc/go1.13 >> 03.09.2019
// https://golang.org/doc/go1.16 >> 16.02.2021
go 1.13

require (
	// https://github.com/jetstack/cert-manager
	// The Jetstack Cert-Manager.
	// v1.2.0 >> 11.02.2021, https://github.com/jetstack/cert-manager/releases/tag/v1.2.0
	// v1.3.0 >> 07.04.2021, https://github.com/jetstack/cert-manager/releases/tag/v1.3.0
	// v1.3.1 >> 14.04.2021, https://github.com/jetstack/cert-manager/releases/tag/v1.3.1
	// See also: terraform/modules/sc-ionos-certificate-issuer/main.tf. Version must match.
	// The cert-manager uses Kubernetes API 0.19 since 26 Aug 2020. https://github.com/jetstack/cert-manager/commit/14ea7c3f653e07a7a326bef2c3689b0596d706bc#diff-33ef32bf6c23acb95f5902d7097b7a1d5128ca061167ec0716715b0b9eeaa5f6
	github.com/jetstack/cert-manager v1.3.1

	// https://github.com/opentelekomcloud/gophertelekomcloud
	// The Open Telekom Cloud API
	github.com/opentelekomcloud/gophertelekomcloud v0.3.2

	// Miek Gieben DNS. A DNS library.
	// github.com/miekg/dns v1.1.31

	// A test library.
	github.com/stretchr/testify v1.6.1

	// https://github.com/kubernetes/apiextensions-apiserver
	// This API server provides the implementation for CustomResourceDefinitions which is included as delegate server inside of kube-apiserver.
	// apiextensions-apiserver v0.18.0 >>> Kubernetes 1.18
	k8s.io/apiextensions-apiserver v0.19.0

	k8s.io/apimachinery v0.19.0

	// https://github.com/kubernetes/client-go
	// Client library to talk to Kubernetes. client-go v0.18.0 >>> Kubernetes 1.18
	k8s.io/client-go v0.19.0

	// https://github.com/kubernetes/klog/tree/v2.9.0
	k8s.io/klog v1.0.0
)

// replace github.com/hpi-schul-cloud/infra-otc-cert-manager-webhook/otcdns => ../otcdns
