REPO_DIR=$(shell git rev-parse --show-toplevel)
SEMVER_CLI_BIN:=$(if $(which semver-cli),$(which semver-cli),$(GOPATH)/bin/semver-cli)

.PHONY: semver-cli
semver-cli: $(SEMVER_CLI_BIN)

$(SEMVER_CLI_BIN):
	@(CGO_ENABLED=0 go install github.com/davidrjonas/semver-cli@latest)

#----- GKE -----#
GCP_PROJECT?=wavefront-gcp-dev
GCP_REGION=us-central1
GCP_ZONE?=c
NUMBER_OF_NODES?=3

connect-to-gke:
	gcloud config set project $(GCP_PROJECT)
	gcloud auth configure-docker --quiet

gke-connect-to-cluster: gke-cluster-name-check
	gcloud container clusters get-credentials $(GKE_CLUSTER_NAME) --zone $(GCP_REGION)-$(GCP_ZONE) --project $(GCP_PROJECT)

gke-cluster-name-check:
	@if [ -z ${GKE_CLUSTER_NAME} ]; then echo "Need to set GKE_CLUSTER_NAME" && exit 1; fi

delete-gke-cluster: gke-cluster-name-check connect-to-gke
	echo "Deleting GKE K8s Cluster: $(GKE_CLUSTER_NAME)"
	gcloud container clusters delete $(GKE_CLUSTER_NAME) --zone $(GCP_REGION)-$(GCP_ZONE) --quiet

create-gke-cluster: gke-cluster-name-check connect-to-gke
	echo "Creating GKE K8s Cluster: $(GKE_CLUSTER_NAME)"
	gcloud container clusters create $(GKE_CLUSTER_NAME) --machine-type=e2-standard-2 --zone=$(GCP_REGION)-$(GCP_ZONE) --enable-ip-alias --create-subnetwork range=/21 --num-nodes=$(NUMBER_OF_NODES)
	gcloud container clusters get-credentials $(GKE_CLUSTER_NAME) --zone $(GCP_REGION)-$(GCP_ZONE) --project $(GCP_PROJECT)
	kubectl create clusterrolebinding --clusterrole cluster-admin \
		--user $$(gcloud auth list --filter=status:ACTIVE --format="value(account)") \
		clusterrolebinding


#----- AKS -----#
aks-subscription-id-check:
	@if [ -z ${AKS_SUBSCRIPTION_ID} ]; then echo "Need to set AKS_SUBSCRIPTION_ID" && exit 1; fi

aks-connect-to-cluster: aks-subscription-id-check
	az account set --subscription

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