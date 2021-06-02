OS ?= $(shell go env GOOS)
ARCH ?= $(shell go env GOARCH)

# Docker and Helm
IMAGE_NAME := "webhook"
IMAGE_TAG := "latest"
# Helm
RELEASE_NAME := "testrelease"
NAMESPACE := "testnamespace"

OUT := $(shell pwd)/_out

KUBEBUILDER_VERSION=2.3.2

$(shell mkdir -p "$(OUT)")

#
# Defaulttarget. Runs the test.
#
test: _test/kubebuilder
	go test -v .

#
# Kubebuilder is a framework for building Kubernetes APIs using custom resource definitions (CRDs).
#
_test/kubebuilder:
	curl -fsSL https://github.com/kubernetes-sigs/kubebuilder/releases/download/v$(KUBEBUILDER_VERSION)/kubebuilder_$(KUBEBUILDER_VERSION)_$(OS)_$(ARCH).tar.gz -o kubebuilder-tools.tar.gz
	mkdir -p _test/kubebuilder
	tar -xvf kubebuilder-tools.tar.gz
	mv kubebuilder_$(KUBEBUILDER_VERSION)_$(OS)_$(ARCH)/bin _test/kubebuilder/
	rm kubebuilder-tools.tar.gz
	rm -R kubebuilder_$(KUBEBUILDER_VERSION)_$(OS)_$(ARCH)

clean: clean-kubebuilder

clean-kubebuilder:
	rm -Rf _test/kubebuilder

#
# Creates a docker image webhook.latest with the compiled webhook.
#
build:
	docker build -t "$(IMAGE_NAME):$(IMAGE_TAG)" .

#
# Use helm to render the manifests for the webhook.
#
.PHONY: rendered-manifest.yaml
rendered-manifest.yaml:
	helm template \
		$(RELEASE_NAME) \
		deploy/infra-otc-cert-manager-webhook \
        --set image.repository=$(IMAGE_NAME) \
        --set image.tag=$(IMAGE_TAG) \
		--namespace=$(NAMESPACE) \
        > "$(OUT)/rendered-manifest.yaml"
