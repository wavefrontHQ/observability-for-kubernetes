include ../Makefile

# test trigger build in jenkins

export # Used to let all sub-make use the initialized value of variables whose names consist solely of alphanumerics and underscores

# Image URL to use all building/pushing image targets
PREFIX?=projects.registry.vmware.com/tanzu_observability
DOCKER_IMAGE?=kubernetes-operator-snapshot

OS := $(shell uname -s | tr A-Z a-z)

ifeq ($(origin VERSION_POSTFIX), undefined)
VERSION_POSTFIX:=-alpha-$(shell whoami)-$(shell date +"%y%m%d%H%M%S")
endif

NEXT_RELEASE_VERSION?=$(shell ../scripts/get-next-operator-version.sh)
VERSION?=$(NEXT_RELEASE_VERSION)$(VERSION_POSTFIX)
IMG?=$(PREFIX)/$(DOCKER_IMAGE):$(VERSION)
NS?=observability-system
LDFLAGS=-X main.version=$(NEXT_RELEASE_VERSION)

OBSERVABILITY_FOR_KUBERNETES_HEADER=
ifneq ($(TOKEN),)
OBSERVABILITY_FOR_KUBERNETES_HEADER=-H "Authorization: token $(TOKEN)"
endif

MONOREPO_DIR=$(shell git rev-parse --show-toplevel)
OPERATOR_DIR=$(shell git rev-parse --show-toplevel)/operator

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

ALLOW_AUTO_INSTRUMENTATION?=true

.PHONY: all
all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development
.PHONY: manifests
manifests: config/crd/bases/wavefront.com_wavefronts.yaml

config/crd/bases/wavefront.com_wavefronts.yaml: api/v1alpha1/wavefront_types.go controllers/wavefront_controller.go
	controller-gen rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: api/v1alpha1/zz_generated.deepcopy.go

api/v1alpha1/zz_generated.deepcopy.go: hack/boilerplate.go.txt api/v1alpha1/wavefront_types.go
	controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./..."

FIND_GOIMPORTS_FILES:=find . -type f -name '*.go' -not -path "./vendor/*" -not -path "./pixie-sizer/vendor/*"

.PHONY: fmt
fmt:
	$(FIND_GOIMPORTS_FILES) | xargs goimports -w

.PHONY: checkfmt
checkfmt:
	@if [ $$(goimports -d $$($(FIND_GOIMPORTS_FILES)) | wc -l) -gt 0 ]; then \
		echo $$'\e[31mgoimports FAILED!!!\e[0m'; \
		goimports -d $$($(FIND_GOIMPORTS_FILES)); \
		exit 1; \
	fi

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: golangci-lint
golangci-lint:
	golangci-lint run

.PHONY: test
test: manifests generate fmt vet ## Run tests.
	go test ./... -coverprofile cover.out

##@ Build

.PHONY: build
build: generate fmt vet ## Build manager binary.
	go build -ldflags "$(LDFLAGS)" -o build/$(GOOS)/$(GOARCH)/manager main.go
	rm -rf build/$(GOOS)/$(GOARCH)/components
	find components -type f -name "*.yaml" | xargs dirname | uniq | xargs -I{} mkdir -p build/$(GOOS)/$(GOARCH)/{}
	find components -type f -name "*.yaml" | xargs -I{} /bin/bash -c 'cp {} build/$(GOOS)/$(GOARCH)/$$(dirname {})'
	cp open_source_licenses.txt build/

.PHONY: clean
clean: clean-build
	rm -rf bin

.PHONY: clean-build
clean-build:
	rm -rf build

.PHONY: docker-build
docker-build: ## Build docker image with the manager.
	CGO_ENABLED=0 GOOS=linux GOARCH=$(GOARCH) make build -o fmt -o vet
	docker build -t ${IMG} -f Dockerfile build

BUILDER_SUFFIX=$(shell echo $(PREFIX) | cut -d '/' -f1)

.PHONY: docker-xplatform-build
docker-xplatform-build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make build -o fmt -o vet
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 make build -o fmt -o vet
	docker buildx create --use --node wavefront_operator_builder_$(BUILDER_SUFFIX)
	docker buildx build --platform linux/amd64,linux/arm64 --push --pull -t ${IMG} -f Dockerfile build

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	docker push ${IMG}

docker-copy-images:
	@test $${SOURCE_PREFIX?Please set variable SOURCE_PREFIX}
	./hack/component-image-refs.sh | ./hack/docker/copy-image-refs.sh -d $(PREFIX) -s $(SOURCE_PREFIX)
	echo "$(DOCKER_IMAGE):$(VERSION)" | ./hack/docker/copy-image-refs.sh -d $(PREFIX) -s $(SOURCE_PREFIX)

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = true
endif

.PHONY: copy-base-patches
copy-base-patches:
	cp config/manager/patches-base.yaml config/manager/patches.yaml

# go-get-tool will 'go get' any package $2 and install it to $1.
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
echo "Downloading $(2)" ;\
GOOS= GOARCH= GOBIN=$(OPERATOR_DIR)/bin go install $(2) ;\
}
endef

OPERATOR_BUILD_DIR:=$(OPERATOR_DIR)/build/operator

ifeq (Kind,$(shell $(MONOREPO_DIR)/scripts/get-k8s-cluster-env.sh))
OPERATOR_YAML_TYPE?=kind
else
OPERATOR_YAML_TYPE?=xplatform
endif

KUBERNETES_YAML:=$(OPERATOR_BUILD_DIR)/wavefront-operator.yaml

KUSTOMIZATION_TYPE?=base
KUSTOMIZATION_YAML:=$(OPERATOR_BUILD_DIR)/kustomization.yaml

$(OPERATOR_BUILD_DIR):
	mkdir -p $(OPERATOR_BUILD_DIR)

.PHONY: base-kustomization-yaml
base-kustomization-yaml: $(OPERATOR_BUILD_DIR)
	cp $(OPERATOR_DIR)/hack/build/kustomization.yaml $(OPERATOR_BUILD_DIR)

.PHONY: custom-kustomization-yaml
custom-kustomization-yaml: $(OPERATOR_BUILD_DIR)
	sed "s%YOUR_IMAGE_REGISTRY%$(PREFIX)%g" $(OPERATOR_DIR)/dev-internal/deploy/kustomization.yaml | \
		sed "s%YOUR_NAMESPACE%$(NS)%g" > $(KUSTOMIZATION_YAML)

.PHONY: kubernetes-yaml
kubernetes-yaml: manifests $(OPERATOR_BUILD_DIR)
	cd config/manager && kustomize edit set image controller=$(IMG)
	kustomize build config/default > $(KUBERNETES_YAML)

.PHONY: rc-kubernetes-yaml
rc-kubernetes-yaml: $(OPERATOR_BUILD_DIR)
	curl -v https://raw.githubusercontent.com/wavefrontHQ/observability-for-kubernetes/$(OPERATOR_YAML_RC_SHA)/operator/wavefront-operator-$(GIT_BRANCH).yaml \
		-H "Accept: application/vnd.github.raw" \
		$(OBSERVABILITY_FOR_KUBERNETES_HEADER) \
		-o $(KUBERNETES_YAML)

.PHONY: xplatform-kubernetes-yaml
xplatform-kubernetes-yaml: docker-xplatform-build copy-base-patches kubernetes-yaml

.PHONY: released-kubernetes-yaml
released-kubernetes-yaml: copy-base-patches kubernetes-yaml
	cp $(KUBERNETES_YAML) $(OPERATOR_DIR)/dev-internal/deploy/wavefront-operator.yaml

.PHONY: kind-kubernetes-yaml
kind-kubernetes-yaml: docker-build copy-kind-patches kubernetes-yaml
	kind load docker-image $(IMG)

.PHONY: copy-kind-patches
copy-kind-patches:
	cp config/manager/patches-kind.yaml config/manager/patches.yaml

.PHONY: copy-rbac-kustomization-yaml
copy-rbac-kustomization-yaml:
	@if [ ${ALLOW_AUTO_INSTRUMENTATION} != true ]; then \
    	cp config/rbac/base_kustomization.yaml config/rbac/kustomization.yaml; \
    else \
    	cp config/rbac/auto_instrumentation_kustomization.yaml config/rbac/kustomization.yaml; \
	fi

.PHONY: operator-yaml
operator-yaml: copy-rbac-kustomization-yaml $(OPERATOR_YAML_TYPE)-kubernetes-yaml $(KUSTOMIZATION_TYPE)-kustomization-yaml

.PHONY: deploy
deploy: operator-yaml
	kubectl apply -k $(OPERATOR_BUILD_DIR)
	kubectl create -n $(NS) secret generic wavefront-secret --from-literal token=$(WAVEFRONT_TOKEN)  || true
	kubectl create -n $(NS) secret generic insights-secret --from-literal ingestion-token='a-token' || true

.PHONY: undeploy
undeploy: operator-yaml
	kubectl delete --ignore-not-found=$(ignore-not-found) -n $(NS) secret wavefront-secret || true
	kubectl delete --ignore-not-found=$(ignore-not-found) -k $(OPERATOR_BUILD_DIR) || true

METRICS_RETRY_COUNT?=18

.PHONY: integration-test
integration-test: undeploy deploy
	(cd $(OPERATOR_DIR)/hack/test && ./run-e2e-tests.sh -t $(WAVEFRONT_TOKEN) -d $(NS) -v $(NEXT_RELEASE_VERSION) $(INTEGRATION_TEST_ARGS))
