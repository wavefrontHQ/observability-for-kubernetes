export # Used to let all sub-make use the initialized value of variables whose names consist solely of alphanumerics and underscores

SEMVER_CLI_BIN:=$(if $(which semver-cli),$(which semver-cli),$(GOPATH)/bin/semver-cli)

REPO_DIR=$(shell git rev-parse --show-toplevel)

GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)

.PHONY: semver-cli
semver-cli: $(SEMVER_CLI_BIN)

$(SEMVER_CLI_BIN):
	@(CGO_ENABLED=0 go install github.com/davidrjonas/semver-cli@latest)

promote-internal:
	cp -a $(REPO_DIR)/operator/dev-internal/* $(REPO_DIR)/

	mkdir -p $(REPO_DIR)/deploy/crd/
	cp $(REPO_DIR)/operator/config/crd/bases/wavefront.com_wavefronts.yaml $(REPO_DIR)/deploy/crd/

.PHONY: promote-release-images
promote-release-images:
	$(eval ALPHA_TAG := $(shell $(REPO_DIR)/scripts/get-latest-alpha-tag.sh))
	$(eval OPERATOR_VERSION := $(shell cat $(REPO_DIR)/operator/release/OPERATOR_VERSION))
	$(eval COLLECTOR_VERSION := $(shell cat $(REPO_DIR)/collector/release/VERSION))
	echo $(OPERATOR_VERSION), $(COLLECTOR_VERSION), $(ALPHA_TAG)
	crane copy "projects.registry.vmware.com/tanzu_observability_keights_saas/kubernetes-operator:$(ALPHA_TAG)" "projects.registry.vmware.com/tanzu_observability_keights_saas/kubernetes-operator-snapshot:$(OPERATOR_VERSION)"
	crane copy "projects.registry.vmware.com/tanzu_observability_keights_saas/kubernetes-collector:$(ALPHA_TAG)" "projects.registry.vmware.com/tanzu_observability_keights_saas/kubernetes-collector-snapshot:$(COLLECTOR_VERSION)"
#	echo "kubernetes-operator:$(ALPHA_TAG)" | ./operator/hack/docker/copy-image-refs.sh -s $(SOURCE_PREFIX) -d $(DEST_PREFIX)
#	echo "kubernetes-collector:$(ALPHA_TAG)" | ./operator/hack/docker/copy-image-refs.sh -s $(SOURCE_PREFIX) -d $(DEST_PREFIX)