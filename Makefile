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

nuke-kind-ha-workers:
	kind delete cluster
	kind create cluster --config "$(MONOREPO_DIR)/make/kind-ha-workers.yml"

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

#----- AKS -----#
AKS_CLUSTER_NAME?=k8po-ci
AKS_RESOURCE_GROUP?=K8sSaaS

aks-subscription-id-check:
	@if [ -z ${AKS_SUBSCRIPTION_ID} ]; then echo "Need to set AKS_SUBSCRIPTION_ID" && exit 1; fi
	@az account set --subscription $(AKS_SUBSCRIPTION_ID)

aks-login-check:
	@az aks list || az login --scope https://management.core.windows.net//.default

aks-connect-to-cluster: aks-subscription-id-check aks-login-check
	az aks get-credentials --resource-group $(AKS_RESOURCE_GROUP) --name $(AKS_CLUSTER_NAME)

#----- EKS -----#
ECR_REPO_PREFIX=tobs/k8s/saas
WAVEFRONT_DEV_AWS_ACC_ID=095415062695
AWS_PROFILE=wavefront-dev
AWS_REGION=us-west-2
ECR_ENDPOINT=$(WAVEFRONT_DEV_AWS_ACC_ID).dkr.ecr.$(AWS_REGION).amazonaws.com
COLLECTOR_ECR_REPO=$(ECR_REPO_PREFIX)/wavefront-kubernetes-collector
TEST_PROXY_ECR_REPO=$(ECR_REPO_PREFIX)/test-proxy

ecr-host:
	echo $(ECR_ENDPOINT)/$(ECR_REPO_PREFIX)/wavefront-kubernetes-collector

docker-login-eks:
	@aws ecr get-login-password --region $(AWS_REGION) --profile $(AWS_PROFILE) |  docker login --username AWS --password-stdin $(ECR_ENDPOINT)

target-eks: docker-login-eks
	@aws eks --region $(AWS_REGION) update-kubeconfig --name k8s-saas-team-dev --profile $(AWS_PROFILE)
