export # Used to let all sub-make use the initialized value of variables whose names consist solely of alphanumerics and underscores

SEMVER_CLI_BIN:=$(if $(shell which semver-cli),$(shell which semver-cli),$(GOPATH)/bin/semver-cli)

MONOREPO_DIR=$(shell git rev-parse --show-toplevel)

GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)

.PHONY: semver-cli
semver-cli: $(SEMVER_CLI_BIN)

$(SEMVER_CLI_BIN):
	@(CGO_ENABLED=0 go install github.com/davidrjonas/semver-cli@latest)

promote-internal:
	cp -a $(MONOREPO_DIR)/operator/dev-internal/* $(MONOREPO_DIR)/

	mkdir -p $(MONOREPO_DIR)/deploy/crd/
	cp $(MONOREPO_DIR)/operator/config/crd/bases/wavefront.com_wavefronts.yaml $(MONOREPO_DIR)/deploy/crd/

#----- KIND ----#
.PHONY: nuke-kind
nuke-kind:
	kind delete cluster
	kind create cluster --image kindest/node:v1.24.7 #setting to 1.24.7 to avoid floating to 1.25 which we currently don't support

nuke-kind-ha:
	kind delete cluster
	kind create cluster --config "$(MONOREPO_DIR)/make/kind-ha.yml"

kind-connect-to-cluster:
	kubectl config use kind-kind

target-kind:
	kubectl config use kind-kind

#----- GKE -----#
GCP_PROJECT?=wavefront-gcp-dev
GCP_REGION=us-central1
GCP_ZONE?=c
NUMBER_OF_NODES?=3

target-gke: connect-to-gke gke-connect-to-cluster

connect-to-gke:
	gcloud config set project $(GCP_PROJECT)
	gcloud auth configure-docker --quiet

gke-connect-to-cluster: gke-cluster-name-check
	gcloud container clusters get-credentials $(GKE_CLUSTER_NAME) --zone $(GCP_REGION)-$(GCP_ZONE) --project $(GCP_PROJECT)

gke-cluster-name-check:
	@if [ -z ${GKE_CLUSTER_NAME} ]; then echo "Need to set GKE_CLUSTER_NAME" && exit 1; fi

delete-gke-cluster: gke-cluster-name-check gke-connect-to-cluster
	echo "Deleting GKE K8s Cluster: $(GKE_CLUSTER_NAME)"
	gcloud container clusters delete $(GKE_CLUSTER_NAME) --zone $(GCP_REGION)-$(GCP_ZONE) --quiet

create-gke-cluster: gke-cluster-name-check
	echo "Creating GKE K8s Cluster: $(GKE_CLUSTER_NAME)"
	gcloud container clusters create $(GKE_CLUSTER_NAME) --machine-type=e2-standard-2 \
		--zone=$(GCP_REGION)-$(GCP_ZONE) --enable-ip-alias --create-subnetwork range=/21 --num-nodes=$(NUMBER_OF_NODES)
	gcloud container clusters get-credentials $(GKE_CLUSTER_NAME) --zone $(GCP_REGION)-$(GCP_ZONE) --project $(GCP_PROJECT)
	kubectl create clusterrolebinding --clusterrole cluster-admin \
		--user $$(gcloud auth list --filter=status:ACTIVE --format="value(account)") \
		clusterrolebinding
