GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)
PREFIX?=projects.registry.vmware.com/tanzu_observability
DOCKER_IMAGE?=pixie-sizer
ifeq ($(origin VERSION_POSTFIX), undefined)
VERSION_POSTFIX:=-alpha-$(shell whoami)-$(shell date +"%y%m%d%H%M%S")
endif
NEXT_RELEASE_VERSION?=0.1.0
VERSION?=$(NEXT_RELEASE_VERSION)$(VERSION_POSTFIX)
IMG?=$(PREFIX)/$(DOCKER_IMAGE):$(VERSION)


.PHONY: build
build: build/$(GOOS)/$(GOARCH)/pixie-sizer build/$(GOOS)/$(GOARCH)/open_source_licenses.txt

.PHONY: build-image
build-image:
	GOOS=linux GOARCH=amd64 make build
	GOOS=linux GOARCH=arm64 make build
	docker buildx create --use --node wavefront_pixie_sizer_builder
	docker buildx build --platform linux/amd64,linux/arm64 --push --pull -t $(IMG) -f Dockerfile build

.PHONY: build-deployment
build-deployment: build-image
	sed 's%image: projects.registry.vmware.com/tanzu_observability/pixie-sizer:VERSION%image: $(IMG)%g' pixie-sizer.yaml > build/pixie-sizer.yaml

.PHONY: deploy
deploy: build-deployment
	kubectl delete -f build/pixie-sizer.yaml || true
	kubectl apply -f build/pixie-sizer.yaml


.PHONY: build/$(GOOS)/$(GOARCH)/pixie-sizer
build/$(GOOS)/$(GOARCH)/pixie-sizer:
	CGO_ENABLED=0 go build -o build/$(GOOS)/$(GOARCH)/pixie-sizer ./...

build/$(GOOS)/$(GOARCH)/open_source_licenses.txt: open_source_licenses.txt
	mkdir -p build/$(GOOS)/$(GOARCH)
	cp open_source_licenses.txt build/$(GOOS)/$(GOARCH)