include ../Makefile

PREFIX?=projects.registry.vmware.com/tanzu_observability_keights_saas
DOCKER_IMAGE?=kubernetes-collector
WAVEFRONT_CLUSTER?=nimba

REPO_ROOT=$(shell git rev-parse --show-toplevel)
COLLECTOR_REPO_ROOT=$(shell git rev-parse --show-toplevel)/collector
TEST_DIR=$(COLLECTOR_REPO_ROOT)/hack/test
OUT_DIR?=$(COLLECTOR_REPO_ROOT)/_output
INTEGRATION_TEST_ARGS?=-r default -r real-proxy

BINARY_NAME=wavefront-collector

RC_NUMBER?=1
GIT_BRANCH=$(shell git rev-parse --abbrev-ref HEAD)
GIT_HUB_REPO=wavefrontHQ/wavefront-collector-for-kubernetes

K8S_ENV=$(shell $(REPO_ROOT)/scripts/get-k8s-cluster-env.sh)
KIND_LOCATION=$(shell $(REPO_ROOT)/scripts/kind-location.sh)

ifndef TEMP_DIR
TEMP_DIR:=$(shell mktemp -d /tmp/wavefront.XXXXXX)
endif

ifndef VERSION_POSTFIX
VERSION_POSTFIX:=-dev-$(shell whoami)-$(shell date +"%y%m%d%H%M%S")
endif
NEXT_RELEASE_VERSION?=$(shell cat ./release/NEXT_RELEASE_VERSION)
VERSION?=$(NEXT_RELEASE_VERSION)$(VERSION_POSTFIX)
GIT_COMMIT:=$(shell git rev-parse --short HEAD)

# for testing, the built image will also be tagged with this name provided via an environment variable
OVERRIDE_IMAGE_NAME?=${COLLECTOR_TEST_IMAGE}

LDFLAGS=-w -X github.com/wavefronthq/observability-for-kubernetes/collector/internal/version.Version=$(NEXT_RELEASE_VERSION) -X github.com/wavefronthq/observability-for-kubernetes/collector/internal/version.Commit=$(GIT_COMMIT)

.PHONY: release

.PHONY: all
all: container

.PHONY: fmt
fmt:
	find . -type f -name "*.go" | grep -v "./vendor*" | xargs goimports -w

.PHONY: deploy-targets clean-targets clean-deployment

deploy-targets:
	@($(REPO_ROOT)/scripts/deploy/deploy-targets.sh)

clean-targets:
	@($(REPO_ROOT)/scripts/deploy/uninstall-targets.sh)

clean-deployment:
	@($(REPO_ROOT)/scripts/deploy/uninstall-wavefront-helm-release.sh)
	@(cd $(TEST_DIR) && ./clean-deploy.sh)

k8s-env:
	@echo "\033[92mK8s Environment: $(shell kubectl config current-context)\033[0m"

k8s-nodes-arch:
	kubectl get nodes --label-columns='kubernetes.io/arch'

delete-images:
ifeq ($(K8S_ENV)-$(KIND_RUNNING_LOCALLY), Kind-local)
	make delete-images-kind
endif

push-to-kind: container
	echo $(PREFIX)/$(DOCKER_IMAGE):$(VERSION)

	@kind load docker-image $(PREFIX)/$(DOCKER_IMAGE):$(VERSION) --name kind

cover-push-to-kind: cover-container
	echo $(PREFIX)/$(DOCKER_IMAGE):$(VERSION)

	@kind load docker-image $(PREFIX)/$(DOCKER_IMAGE):$(VERSION) --name kind

delete-images-kind:
	@docker exec -it kind-control-plane crictl rmi $(PREFIX)/$(DOCKER_IMAGE):$(VERSION) || true

push-images:
ifeq ($(K8S_ENV)-$(KIND_RUNNING_LOCALLY), Kind-local)
	make push-to-kind
else
	make docker-xplatform-build
endif

cover-push-images:
ifeq ($(K8S_ENV)-$(KIND_RUNNING_LOCALLY), Kind-local)
	make cover-push-to-kind
else
	make docker-xplatform-build
endif

# TODO: exclude certain keys from sorting
# because we want them to be at the top and visible when we open the file!
.PHONY: sort-integrations-keys
sort-integrations-keys:
	# TODO: uncomment to run this on all of our dashboards when we're comfortable
	@#$(COLLECTOR_REPO_ROOT)/hack/sort-json-keys-inplace.sh $(HOME)/workspace/integrations/kubernetes/dashboards/*.json
	@$(COLLECTOR_REPO_ROOT)/hack/sort-json-keys-inplace.sh $(HOME)/workspace/integrations/kubernetes/dashboards/integration-kubernetes-control-plane.json

.PHONY: checkfmt
checkfmt:
	@if [ $$(goimports -d $$(find . -type f -name '*.go' -not -path "./vendor/*") | wc -l) -gt 0 ]; then \
		echo $$'\e[31mgoimports FAILED!!!\e[0m'; \
		goimports -d $$(find . -type f -name '*.go' -not -path "./vendor/*"); \
		exit 1; \
	fi

.PHONY: tests
tests:
	go clean -testcache
	go test -timeout 30s -race ./...

.PHONY: test
test: tests

.PHONY: build
build: fmt vet
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -a -o $(OUT_DIR)/$(GOOS)/$(GOARCH)/$(BINARY_NAME) cmd/wavefront-collector/main.go
	cp open_source_licenses.txt $(OUT_DIR)

.PHONY: vet
vet:
	go vet -composites=false ./...

.PHONY: containers
containers: container test-proxy-container

.PHONY: container
container:
	CGO_ENABLED=0 GOOS=linux GOARCH=$(GOARCH) make build -o fmt -o vet
	docker build -f Dockerfile \
	--pull -t $(PREFIX)/$(DOCKER_IMAGE):$(VERSION) $(OUT_DIR)
ifneq ($(OVERRIDE_IMAGE_NAME),)
	docker tag $(PREFIX)/$(DOCKER_IMAGE):$(VERSION) $(OVERRIDE_IMAGE_NAME)
endif

cover-container:
	# Run build in a container in order to have reproducible builds
	docker build \
	-f $(COLLECTOR_REPO_ROOT)/Dockerfile.cover-non-cross-platform \
	--build-arg BINARY_NAME=$(BINARY_NAME) \
	--build-arg RELEASE_VERSION=$(NEXT_RELEASE_VERSION) --build-arg GIT_COMMIT="$(GIT_COMMIT)" \
	--pull -t $(PREFIX)/$(DOCKER_IMAGE):$(VERSION) .
ifneq ($(OVERRIDE_IMAGE_NAME),)
	docker tag $(PREFIX)/$(DOCKER_IMAGE):$(VERSION) $(OVERRIDE_IMAGE_NAME)
endif

BUILDER_SUFFIX=$(shell echo $(PREFIX) | cut -d '/' -f1)

.PHONY: docker-xplatform-build
docker-xplatform-build:
	GOOS=linux GOARCH=amd64 make build -o fmt -o vet
	GOOS=linux GOARCH=arm64 make build -o fmt -o vet
	docker buildx create --use --node wavefront_collector_builder_$(BUILDER_SUFFIX)
	docker buildx build --platform linux/amd64,linux/arm64 --push \
	--pull -t $(PREFIX)/$(DOCKER_IMAGE):$(VERSION) -f Dockerfile $(OUT_DIR)
	@echo "Built collector version: $(VERSION)"

.PHONY: peg
peg:
	@which peg > /dev/null || \
		(cd $(COLLECTOR_REPO_ROOT)/..; GOARCH=$(GOARCH) CGO_ENABLED=0 go install github.com/pointlander/peg@latest)

%.peg.go: %.peg
	peg -switch -inline $<

.PHONY: container_rhel
container_rhel:
	docker build \
		-f $(COLLECTOR_REPO_ROOT)/hack/deploy/docker/Dockerfile-rhel \
		--build-arg COLLECTOR_VERSION=$(NEXT_RELEASE_VERSION) --build-arg LDFLAGS="$(LDFLAGS)" \
		-t $(PREFIX)/$(DOCKER_IMAGE):$(VERSION) .
ifneq ($(OVERRIDE_IMAGE_NAME),)
	sudo docker tag $(PREFIX)/$(DOCKER_IMAGE):$(VERSION) $(OVERRIDE_IMAGE_NAME)
endif

.PHONY: push_rhel_redhat_connect
push_rhel_redhat_connect: container_rhel
	docker tag  $(PREFIX)/$(DOCKER_IMAGE):$(VERSION) $(PREFIX)/$(DOCKER_IMAGE):$(NEXT_RELEASE_VERSION)-rc
	docker push $(PREFIX)/$(DOCKER_IMAGE):$(NEXT_RELEASE_VERSION)-rc

.PHONY: clean
clean:
	@rm -rf $(OUT_DIR)

.PHONY: token-check
token-check:
	@if [ -z ${WAVEFRONT_TOKEN} ]; then echo "Need to set WAVEFRONT_TOKEN" && exit 1; fi

METRICS_RETRY_COUNT?=18

.PHONY: proxy-test
proxy-test: token-check
ifeq ($(INTEGRATION_TEST_ARGS),all)
	$(eval INTEGRATION_TEST_ARGS := -r cluster-metrics-only -r node-metrics-only -r combined -r single-deployment -r default -r real-proxy-metrics)
endif
	(cd $(TEST_DIR) && ./test-integration.sh -c $(WAVEFRONT_CLUSTER) -t $(WAVEFRONT_TOKEN) -v $(VERSION) $(INTEGRATION_TEST_ARGS))

.PHONE: build-image
build-image:
ifneq ($(INTEGRATION_TEST_BUILD), ci)
	make delete-images push-images
endif

.PHONY: integration-test
integration-test: token-check k8s-env clean-deployment deploy-targets build-image proxy-test

# Get code coverage of integration test
coverage-test: token-check k8s-env clean-deployment deploy-targets delete-images cover-push-images proxy-test
	kubectl exec -n wavefront-collector -it ds/wavefront-collector -- curl localhost:19999
	kubectl exec -n wavefront-collector -it ds/wavefront-collector -- cat cover.out > coverage/integration-report.txt
	go tool cover -html=coverage/integration-report.txt -o coverage/integration-browser.html
	go tool cover -func=coverage/integration-report.txt -o coverage/integration-by-func.txt

	go clean -testcache
	go test -timeout 30s ./... -cover -covermode=count -coverpkg=./... -coverprofile=coverage/unit-report.txt
	go tool cover -html=coverage/unit-report.txt -o coverage/unit-browser.html
	go tool cover -func=coverage/unit-report.txt -o coverage/unit-by-func.txt

	echo "mode: set" > coverage/merged.out && cat *-report.txt | grep -v mode: | sort -r | awk '{if($1 != last) {print $0;last=$1}}' >> coverage/merged.out
	go tool cover -html=coverage/merged.out -o coverage/merged-browser.html
	go tool cover -func=coverage/merged.out -o coverage/merged-by-func.txt

# creating this as separate and distinct for now,
# but would like to recombine as a flag on integration-test
.PHONY: integration-test-rhel
integration-test-rhel: token-check k8s-env clean-deployment deploy-targets
	VERSION=$(VERSION)-rhel make container_rhel test-proxy-container delete-images push-images proxy-test
