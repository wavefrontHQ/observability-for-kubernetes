PREFIX?=projects.registry.vmware.com/tanzu_observability
DOCKER_IMAGE?=pixie-sizer
ifeq ($(origin VERSION_POSTFIX), undefined)
VERSION_POSTFIX:=-alpha-$(shell whoami)-$(shell date +"%y%m%d%H%M%S")
endif
RELEASE_VERSION?=0.2.0
VERSION?=$(RELEASE_VERSION)$(VERSION_POSTFIX)
IMG?=$(PREFIX)/$(DOCKER_IMAGE):$(VERSION)

IMG_ARCHES:=amd64 arm64
IMG_FILES:=pixie-sizer open_source_licenses.txt
IMG_DEPS:=$(foreach IMG_ARCH,$(IMG_ARCHES),$(foreach IMG_FILE,$(IMG_FILES),build/linux/$(IMG_ARCH)/$(IMG_FILE)))

.PHONY: build-image
build-image: $(IMG_DEPS)
	docker buildx create --use --node wavefront_pixie_sizer_builder
	docker buildx build --platform linux/amd64,linux/arm64 --push --pull -t $(IMG) -f Dockerfile build

.PHONY: build-deployment
build-deployment: build-image
	sed 's%image: projects.registry.vmware.com/tanzu_observability/pixie-sizer:.*%image: $(IMG)%g' pixie-sizer.yaml > build/pixie-sizer.yaml

.PHONY: deploy
deploy: build-deployment
	kubectl delete -f build/pixie-sizer.yaml || true
	PIXIE_SIZER_YAML=file://$(shell realpath build/pixie-sizer.yaml) ./install.sh

.PHONY: release
release: build-deployment
	test -n "$(NEXT_RELEASE_VERSION)" # $$NEXT_RELEASE_VERSION must be set
	cp build/pixie-sizer.yaml pixie-sizer-$(VERSION).yaml
	git add pixie-sizer-$(VERSION).yaml
	sed -i '0,/RELEASE_VERSION?=.*/{s//RELEASE_VERSION?=$(NEXT_RELEASE_VERSION)/}' Makefile
ifeq ($(VERSION_POSTFIX),) # this is a full release
	sed -i 's%pixie-sizer-.*\.yaml%pixie-sizer-$(VERSION).yaml%g' install.sh
endif

.PRECIOUS: build/%/pixie-sizer
build/%/pixie-sizer: always_out_of_date
	$(eval GOOS := $(wordlist 1,1,$(subst /, ,$(basename $*))))
	$(eval GOARCH := $(wordlist 2,2,$(subst /, ,$(basename $*))))
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 go build -o build/$(GOOS)/$(GOARCH)/pixie-sizer ./...

.PRECIOUS: build/%/open_source_licenses.txt
build/%/open_source_licenses.txt: ../open_source_licenses.txt
	$(eval OS := $(wordlist 1,1,$(subst /, ,$(basename $*))))
	$(eval ARCH := $(wordlist 2,2,$(subst /, ,$(basename $*))))
	mkdir -p build/$(OS)/$(ARCH)
	cp $< build/$(OS)/$(ARCH)

.PHONY: always_out_of_date
always_out_of_date: