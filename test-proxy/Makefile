include ../Makefile

VERSION?=
PREFIX?=projects.registry.vmware.com/tanzu_observability_keights_saas
DOCKER_IMAGE:=test-proxy
BINARY_NAME:=test-proxy
GIT_COMMIT:=$(shell git rev-parse --short HEAD)
LDFLAGS=-w -X main.version=$(VERSION) -X main.commit=$(GIT_COMMIT)
BUILD_DIR:=$(MONOREPO_DIR)/test-proxy/build

.PHONY: clean
clean:
	@rm -rf $(BUILD_DIR)

.PHONY: build
build: fmt vet peg test-proxy

.PHONY: fmt
fmt:
	find . -type f -name "*.go" | grep -v "./vendor*" | xargs goimports -w

.PHONY: vet
vet:
	go vet ./...

.PHONY: test
test:
	go test ./...

.PHONY: peg
peg:
	@which peg > /dev/null || \
		(cd $(MONOREPO_DIR); GOARCH=$(GOARCH) CGO_ENABLED=0 go install github.com/pointlander/peg@latest)

%.peg.go: %.peg
	peg -switch -inline $<

.PHONY: test-proxy
test-proxy: peg internal/testproxy/metricline/metric_grammar.peg.go internal/testproxy/eventline/event_grammar.peg.go
	@if [ -z "$(VERSION)" ]; then echo "Need to set test-proxy VERSION" && exit 1; fi
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -a -o $(BUILD_DIR)/$(GOOS)/$(GOARCH)/$(BINARY_NAME) cmd/test-proxy/main.go

.PHONY: test-proxy-container
test-proxy-container: clean
	GOOS=linux GOARCH=$(GOARCH) $(MAKE) build
	docker build -f Dockerfile \
		--pull -t $(PREFIX)/$(DOCKER_IMAGE):$(VERSION) $(BUILD_DIR)

.PHONY: publish-test-proxy
publish-test-proxy: test-proxy-container
	docker push $(PREFIX)/$(DOCKER_IMAGE):$(VERSION)

.PHONY: push-to-kind
push-to-kind: test-proxy-container
	@echo $(PREFIX)/$(DOCKER_IMAGE):$(VERSION)
	@kind load docker-image $(PREFIX)/$(DOCKER_IMAGE):$(VERSION) --name kind

BUILDER_SUFFIX=$(shell echo $(PREFIX) | cut -d '/' -f1)

.PHONY: docker-xplatform-build
docker-xplatform-build: clean
	GOOS=linux GOARCH=amd64 $(MAKE) test-proxy
	GOOS=linux GOARCH=arm64 $(MAKE) test-proxy
	docker buildx create --use --node testproxy_builder_$(BUILDER_SUFFIX)
	docker buildx build -f Dockerfile \
		--platform linux/amd64,linux/arm64 --push --pull \
		-t $(PREFIX)/$(DOCKER_IMAGE):$(VERSION) $(BUILD_DIR)
	@echo "Built test-proxy version: $(VERSION)"
