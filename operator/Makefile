include ../Makefile

export # Used to let all sub-make use the initialized value of variables whose names consist solely of alphanumerics and underscores

# Image URL to use all building/pushing image targets
PREFIX?=projects.registry.vmware.com/tanzu_observability
DOCKER_IMAGE?=kubernetes-operator-snapshot

GO_IMPORTS_BIN:=$(if $(shell which goimports),$(shell which goimports),$(GOPATH)/bin/goimports)

OS := $(shell uname -s | tr A-Z a-z)

GOLANGCI_LINT_VERSION:="1.52.2"
ifeq ($(OS), darwin)
DARWIN_GOLANGCI_LINT_BIN:=$(or $(shell which golangci-lint),"/usr/local/bin/golangci-lint")
else
LINUX_GOLANGCI_LINT_BIN:=$(or $(shell which golangci-lint),$(GOPATH)/bin/golangci-lint)
endif

ifeq ($(origin VERSION_POSTFIX), undefined)
VERSION_POSTFIX:=-alpha-$(shell whoami)-$(shell date +"%y%m%d%H%M%S")
endif

#COLLECTOR_VERSION?=
NEXT_RELEASE_VERSION?=$(shell ../scripts/get-next-operator-version.sh)
VERSION?=$(NEXT_RELEASE_VERSION)$(VERSION_POSTFIX)
IMG?=$(PREFIX)/$(DOCKER_IMAGE):$(VERSION)
NS?=observability-system
LDFLAGS=-X main.version=$(NEXT_RELEASE_VERSION)

OBSERVABILITY_FOR_KUBERNETES_HEADER=
ifneq ($(TOKEN),)
OBSERVABILITY_FOR_KUBERNETES_HEADER=-H "Authorization: token $(TOKEN)"
endif

REPO_DIR=$(shell git rev-parse --show-toplevel)/operator

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
$(GO_IMPORTS_BIN):
	@(cd $(REPO_DIR)/..; CGO_ENABLED=0 go install golang.org/x/tools/cmd/goimports@latest)

.PHONY: manifests
manifests: controller-gen config/crd/bases/wavefront.com_wavefronts.yaml

config/crd/bases/wavefront.com_wavefronts.yaml: api/v1alpha1/wavefront_types.go controllers/wavefront_controller.go
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen api/v1alpha1/zz_generated.deepcopy.go

api/v1alpha1/zz_generated.deepcopy.go: hack/boilerplate.go.txt api/v1alpha1/wavefront_types.go
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: $(GO_IMPORTS_BIN)
	find . -type f -name "*.go" | grep -v "./vendor*" | xargs goimports -w

.PHONY: checkfmt
checkfmt: $(GO_IMPORTS_BIN)
	@if [ $$(goimports -d $$(find . -type f -name '*.go' -not -path "./vendor/*") | wc -l) -gt 0 ]; then \
		echo $$'\e[31mgoimports FAILED!!!\e[0m'; \
		goimports -d $$(find . -type f -name '*.go' -not -path "./vendor/*"); \
		exit 1; \
	fi

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: delete-old-linter
delete-old-linter: ## Add as a prerequisite for linux-golangci-lint if you bump GOLANGCI_LINT_VERSION
	$(eval LINT_VERSION := $(if $(shell command -v golangci-lint),$(shell golangci-lint version | awk '{print $$4}'),))
	if [ ${GOLANGCI_LINT_VERSION} != "$(LINT_VERSION)" ]; then \
		if which golangci-lint; then \
			echo "Deleting golangci-lint version: $(LINT_VERSION)"; \
			sudo rm -f "$(shell which golangci-lint)" || true; \
		fi; \
	fi

.PHONY: golangci-lint
golangci-lint: $(OS)-golangci-lint
	golangci-lint run

.PHONY: darwin-golangci-lint
darwin-golangci-lint: $(DARWIN_GOLANGCI_LINT_BIN)

$(DARWIN_GOLANGCI_LINT_BIN):
	brew install golangci-lint

.PHONY: linux-golangci-lint
linux-golangci-lint: $(LINUX_GOLANGCI_LINT_BIN)

$(LINUX_GOLANGCI_LINT_BIN):
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sudo sh -s -- -b $(go env GOPATH)/bin v${GOLANGCI_LINT_VERSION}

.PHONY: test
test: manifests generate fmt vet ## Run tests.
	go test ./... -coverprofile cover.out

##@ Build

.PHONY: build
build: generate fmt vet ## Build manager binary.
	go build -ldflags "$(LDFLAGS)" -o build/$(GOOS)/$(GOARCH)/manager main.go
	rm -rf build/$(GOOS)/$(GOARCH)/deploy
	cp -r deploy build/$(GOOS)/$(GOARCH)
	cp open_source_licenses.txt build/

.PHONY: clean
clean: clean-build
	rm -rf bin

.PHONY: clean-build
clean-build:
	rm -rf build

.PHONY: docker-build
docker-build: $(SEMVER_CLI_BIN) ## Build docker image with the manager.
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

CONTROLLER_GEN = $(REPO_DIR)/bin/controller-gen
.PHONY: controller-gen
controller-gen:
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.11.3)

KUSTOMIZE = $(REPO_DIR)/bin/kustomize
.PHONY: kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v4@v4.5.7)

IMGPKG = $(REPO_DIR)/bin/imgpkg
$(IMGPKG):
	curl --create-dirs --output $(IMGPKG) -L https://github.com/vmware-tanzu/carvel-imgpkg/releases/download/v0.33.0/imgpkg-$(shell go env GOOS)-$(shell go env GOARCH)
	chmod +x $(IMGPKG)

.PHONY: imgpkg
imgpkg: $(IMGPKG)

KUBE_LINTER = $(REPO_DIR)/bin/kube-linter
.PHONY: install-kube-linter
install-kube-linter: ## Download kube-linter locally if necessary.
	$(call go-get-tool,$(KUBE_LINTER),golang.stackrox.io/kube-linter/cmd/kube-linter@v0.4.0)

KUBE_SCORE = $(REPO_DIR)/bin/kube-score
.PHONY: install-kube-score
install-kube-score: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUBE_SCORE),github.com/zegl/kube-score/cmd/kube-score@v1.14.0)

# go-get-tool will 'go get' any package $2 and install it to $1.
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
echo "Downloading $(2)" ;\
GOOS= GOARCH= GOBIN=$(REPO_DIR)/bin go install $(2) ;\
}
endef

OPERATOR_BUILD_DIR:=$(REPO_DIR)/build/operator

OPERATOR_YAML_TYPE?=kind
KUBERNETES_YAML:=$(OPERATOR_BUILD_DIR)/wavefront-operator.yaml

KUSTOMIZATION_TYPE?=base
KUSTOMIZATION_YAML:=$(OPERATOR_BUILD_DIR)/kustomization.yaml

$(OPERATOR_BUILD_DIR):
	mkdir -p $(OPERATOR_BUILD_DIR)

.PHONY: base-kustomization-yaml
base-kustomization-yaml: $(OPERATOR_BUILD_DIR)
	cp $(REPO_DIR)/hack/build/kustomization.yaml $(OPERATOR_BUILD_DIR)

.PHONY: custom-kustomization-yaml
custom-kustomization-yaml: $(OPERATOR_BUILD_DIR)
	sed "s%YOUR_IMAGE_REGISTRY%$(PREFIX)%g" $(REPO_DIR)/dev-internal/deploy/kustomization.yaml | \
		sed "s%YOUR_NAMESPACE%$(NS)%g" > $(KUSTOMIZATION_YAML)

.PHONY: kubernetes-yaml
kubernetes-yaml: manifests kustomize $(OPERATOR_BUILD_DIR)
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/default > $(KUBERNETES_YAML)

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
	cp $(KUBERNETES_YAML) $(REPO_DIR)/dev-internal/deploy/wavefront-operator.yaml

.PHONY: kind-kubernetes-yaml
kind-kubernetes-yaml: docker-build copy-kind-patches kubernetes-yaml
	kind load docker-image $(IMG)

.PHONY: copy-kind-patches
copy-kind-patches:
	cp config/manager/patches-kind.yaml config/manager/patches.yaml

.PHONY: operator-yaml
operator-yaml: $(OPERATOR_YAML_TYPE)-kubernetes-yaml $(KUSTOMIZATION_TYPE)-kustomization-yaml

.PHONY: deploy
deploy: operator-yaml
	kubectl apply -k $(OPERATOR_BUILD_DIR)
	kubectl create -n $(NS) secret generic wavefront-secret --from-literal token=$(WAVEFRONT_TOKEN) || true

.PHONY: undeploy
undeploy: operator-yaml
	kubectl delete --ignore-not-found=$(ignore-not-found) -n $(NS) secret wavefront-secret || true
	kubectl delete --ignore-not-found=$(ignore-not-found) -k $(OPERATOR_BUILD_DIR) || true

.PHONY: integration-test
integration-test: install-kube-score install-kube-linter undeploy deploy
	(cd $(REPO_DIR)/hack/test && ./run-e2e-tests.sh -t $(WAVEFRONT_TOKEN) -d $(NS) -v $(NEXT_RELEASE_VERSION) $(INTEGRATION_TEST_ARGS))

.PHONY: clean-cluster
clean-cluster:
	(cd $(REPO_DIR)/hack/test && ./clean-cluster.sh)

# create a new branch from main
# usage: make branch JIRA=XXXX OR make branch NAME=YYYY
branch:
	$(eval NAME := $(if $(JIRA),K8SSAAS-$(JIRA),$(NAME)))
	@if [ -z "$(NAME)" ]; then \
		echo "usage: make branch JIRA=XXXX OR make branch NAME=YYYY"; \
		exit 1; \
  	fi
	git stash
	git checkout main
	git pull
	git checkout -b $(NAME)

git-rebase:
	git fetch origin
	git rebase origin/main
	git log --oneline -n 10
