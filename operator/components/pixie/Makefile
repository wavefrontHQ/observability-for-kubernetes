PIXIE_VERSION ?= 0.14.2
BUILD_BASE ?= build
BUILD_DIR ?= $(BUILD_BASE)/$(PIXIE_VERSION)
PREPROCESS_DIR ?= preprocess

.PHONY: all

.PHONY: checknodiff
checkyamls: all
	@if [ $$(git status --porcelain=v1 2>/dev/null | grep 'operator/components/pixie/' | grep '.yaml' | wc -l) -gt 0 ]; then \
		echo $$'\e[31mneed to update Pixie YAMLS!\e[0m'; \
		exit 1; \
	fi


define yaml-rule
all: $1-$2-$3.yaml

$1-$2-$3.yaml: $(BUILD_DIR)/splits/$2/$3.yaml $(PREPROCESS_DIR)/common.sh $(if $(wildcard $(PREPROCESS_DIR)/$1-$2-$3.sh),$(PREPROCESS_DIR)/$1-$2-$3.sh)
	$(PREPROCESS_DIR)/common.sh < $$< \
	$$(if $$(word 3, $$^),| $$(word 3, $$^)) \
	> $$@
endef

$(eval $(call yaml-rule,00,secret,pl-cluster-secrets))
$(eval $(call yaml-rule,01,serviceaccount,metadata-service-account))
$(eval $(call yaml-rule,02,serviceaccount,pl-cert-provisioner-service-account))
$(eval $(call yaml-rule,04,serviceaccount,query-broker-service-account))
$(eval $(call yaml-rule,06,configmap,pl-tls-config))
$(eval $(call yaml-rule,07,service,kelvin-service))
$(eval $(call yaml-rule,09,service,vizier-metadata-svc))
$(eval $(call yaml-rule,10,service,vizier-query-broker-svc))
$(eval $(call yaml-rule,11,persistentvolumeclaim,metadata-pv-claim))
$(eval $(call yaml-rule,12,deployment,kelvin))
$(eval $(call yaml-rule,14,deployment,vizier-query-broker))
$(eval $(call yaml-rule,15,statefulset,vizier-metadata))
$(eval $(call yaml-rule,16,daemonset,vizier-pem))
$(eval $(call yaml-rule,17,job,cert-provisioner-job))
$(eval $(call yaml-rule,18,configmap,pl-cloud-config))
$(eval $(call yaml-rule,19,configmap,pl-cluster-config))
$(eval $(call yaml-rule,20,configmap,nats-config))
$(eval $(call yaml-rule,21,service,pl-nats))
$(eval $(call yaml-rule,22,service,pl-nats-mgmt))
$(eval $(call yaml-rule,23,statefulset,pl-nats))

define rbac-rule
all: ../../config/rbac/components/pixie/$1-$2-$3.yaml

../../config/rbac/components/pixie/$1-$2-$3.yaml: $(BUILD_DIR)/splits/$2/$3.yaml $(PREPROCESS_DIR)/common-rbac.sh $(if $(wildcard $(PREPROCESS_DIR)/$1-$2-$3.sh),$(PREPROCESS_DIR)/$1-$2-$3.sh)
	$(PREPROCESS_DIR)/common-rbac.sh < $$< \
	$$(if $$(word 3, $$^),| $$(word 3, $$^)) \
	> $$@
endef

$(eval $(call rbac-rule,00,role,pl-cert-provisioner-role))
$(eval $(call rbac-rule,04,role,pl-vizier-metadata-role))
$(eval $(call rbac-rule,05,role,pl-vizier-query-broker-role))
$(eval $(call rbac-rule,07,clusterrole,pl-node-view))
$(eval $(call rbac-rule,09,clusterrole,pl-vizier-metadata))
$(eval $(call rbac-rule,10,rolebinding,pl-cert-provisioner-binding))
$(eval $(call rbac-rule,15,rolebinding,pl-vizier-metadata-binding))
$(eval $(call rbac-rule,16,rolebinding,pl-vizier-query-broker-binding))
$(eval $(call rbac-rule,21,clusterrolebinding,pl-vizier-metadata-cluster-binding))
$(eval $(call rbac-rule,22,clusterrolebinding,pl-vizier-metadata-node-view-cluster-binding))


.PRECIOUS: $(BUILD_DIR)/splits/%
$(BUILD_DIR)/splits/%: $(BUILD_DIR)/yamls
	mkdir -p $(BUILD_DIR)/splits/{configmap,secret,deployment,statefulset,daemonset,job,serviceaccount,service,role,rolebinding,clusterrole,clusterrolebinding,persistentvolumeclaim}
	yq -s '"$(BUILD_DIR)/splits/" + (.kind | downcase) + "/" + .metadata.name + ".yaml"' $(BUILD_DIR)/yamls/vizier/secrets.yaml
	yq -s '"$(BUILD_DIR)/splits/" + (.kind | downcase) + "/" + .metadata.name + ".yaml"' $(BUILD_DIR)/yamls/vizier/vizier_metadata_persist_prod.yaml
	yq -s '"$(BUILD_DIR)/splits/" + (.kind | downcase) + "/" + .metadata.name + ".yaml"' $(BUILD_DIR)/yamls/vizier_deps/nats_prod.yaml

.PRECIOUS: $(BUILD_DIR)/yamls
$(BUILD_DIR)/yamls: $(BUILD_DIR)/yamls.tar
	mkdir -p $(BUILD_DIR)/yamls
	tar -xf $< -C $(BUILD_DIR)/yamls --strip-components 1
	find $(BUILD_DIR)/yamls -exec touch {} \;

$(BUILD_DIR)/yamls.tar: | $(BUILD_DIR)
	curl --silent -L "https://github.com/pixie-io/pixie/releases/download/release%2Fvizier%2Fv$(PIXIE_VERSION)/vizier_yamls.tar" --output $@

$(BUILD_DIR):
	mkdir -p $@

clean:
	rm -rf $(BUILD_BASE)
	rm -rf *.yaml

