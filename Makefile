export # Used to let all sub-make use the initialized value of variables whose names consist solely of alphanumerics and underscores

MONOREPO_DIR=$(shell git rev-parse --show-toplevel)

GOPATH?=$(or $(shell go env GOPATH),$(HOME)/go)
GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)

.PHONY: promote-internal
promote-internal:
	cp -a $(MONOREPO_DIR)/operator/dev-internal/* $(MONOREPO_DIR)/

	mkdir -p $(MONOREPO_DIR)/deploy/crd/
	cp $(MONOREPO_DIR)/operator/config/crd/bases/wavefront.com_wavefronts.yaml $(MONOREPO_DIR)/deploy/crd/

.PHONY: combined-deploy
combined-deploy:
	cd $(MONOREPO_DIR) && ./scripts/combined-deploy.sh $(COMBINED_DEPLOY_ARGS)

.PHONY: combined-integration-tests
combined-integration-tests:
	$(MAKE) -C operator clean-cluster
	cd $(MONOREPO_DIR) && ./scripts/combined-deploy.sh $(COMBINED_DEPLOY_ARGS)
	$(MAKE) -C operator integration-test -o undeploy -o deploy

.PHONY: clean-cluster
clean-cluster:
	@$(MONOREPO_DIR)/scripts/clean-cluster.sh

#----- KIND ----#
KIND_K8S_VERSION?=v1.25.9
.PHONY: nuke-kind
nuke-kind:
	kind delete cluster
	kind create cluster --image kindest/node:$(KIND_K8S_VERSION)

nuke-kind-ha:
	kind delete cluster
	kind create cluster --config "$(MONOREPO_DIR)/make/kind-ha.yml" --image kindest/node:v1.25.9 # setting to v1.25.9 to avoid floating to 1.26 which we currently don't support

nuke-kind-ha-workers:
	kind delete cluster
	kind create cluster --config "$(MONOREPO_DIR)/make/kind-ha-workers.yml" --image kindest/node:v1.25.9 # setting to v1.25.9 to avoid floating to 1.26 which we currently don't support

kind-connect-to-cluster:
	kubectl config use kind-kind

target-kind:
	kubectl config use kind-kind

#----- GKE -----#
GCP_PROJECT?=wavefront-gcp-dev
GCP_REGION=us-central1
GCP_ZONE?=b
GKE_NODE_POOL?=default-pool
GKE_MONITORING?=NONE
GKE_LOGGING?=NONE
GKE_MACHINE_TYPE?=e2-standard-2
NUMBER_OF_NODES?=3
NUMBER_OF_ARM_NODES?=1
GKE_CLUSTER_VERSION?=1.26

.PHONY: target-gke connect-to-gke gke-connect-to-cluster gke-cluster-name-check delete-gke-cluster create-gke-cluster

target-gke: connect-to-gke gke-connect-to-cluster

connect-to-gke:
	gcloud config set project $(GCP_PROJECT)
	gcloud auth configure-docker --quiet

gke-connect-to-cluster: gke-cluster-name-check
	gcloud container clusters get-credentials $(GKE_CLUSTER_NAME) --zone $(GCP_REGION)-$(GCP_ZONE) --project $(GCP_PROJECT)

gke-cluster-name-check:
	@if [ -z ${GKE_CLUSTER_NAME} ]; then echo "Need to set GKE_CLUSTER_NAME" && exit 1; fi

GKE_WAIT_FOR_COMPLETE?=true
delete-gke-cluster: gke-cluster-name-check gke-connect-to-cluster
	@echo "Deleting GKE K8s Cluster: $(GKE_CLUSTER_NAME)"
	$(eval ASYNC_FLAG := $(if $(GKE_WAIT_FOR_COMPLETE),,--async))
	gcloud container clusters delete $(GKE_CLUSTER_NAME) \
		--zone $(GCP_REGION)-$(GCP_ZONE) \
		--quiet $(ASYNC_FLAG)


# create a GKE cluster without weekly cleanup
# usage: make create-gke-cluster GKE_CLUSTER_NAME=XXXX NOCLEANUP=true
create-gke-cluster: gke-cluster-name-check
	$(eval GKE_LABELS := $(if $(NOCLEANUP),,--labels=delete-me=true))
	@echo "Creating GKE K8s Cluster: $(GKE_CLUSTER_NAME)"
	gcloud container clusters create $(GKE_CLUSTER_NAME) --machine-type=$(GKE_MACHINE_TYPE) \
		--zone=$(GCP_REGION)-$(GCP_ZONE) --enable-ip-alias --create-subnetwork range=/21 \
		--num-nodes=$(NUMBER_OF_NODES)  \
		--cluster-version $(GKE_CLUSTER_VERSION) $(GKE_LABELS) \
		--monitoring $(GKE_MONITORING) --logging $(GKE_LOGGING)
	gcloud container clusters get-credentials $(GKE_CLUSTER_NAME) \
		--zone $(GCP_REGION)-$(GCP_ZONE) --project $(GCP_PROJECT)
	kubectl create clusterrolebinding --clusterrole cluster-admin \
		--user $$(gcloud auth list --filter=status:ACTIVE --format="value(account)") \
		clusterrolebinding

create-gke-cluster-with-arm-nodes: create-gke-cluster add-arm-node-pool-gke-cluster

resize-node-pool-gke-cluster: gke-cluster-name-check
	gcloud container clusters resize $(GKE_CLUSTER_NAME) --zone=$(GCP_REGION)-$(GCP_ZONE) \
        --node-pool $(GKE_NODE_POOL) --num-nodes $(NUMBER_OF_NODES) -q

add-arm-node-pool-gke-cluster: gke-cluster-name-check
	gcloud container  node-pools create arm-pool --cluster=$(GKE_CLUSTER_NAME) --zone=$(GCP_REGION)-$(GCP_ZONE) \
        --machine-type=t2a-standard-1 --num-nodes=$(NUMBER_OF_ARM_NODES)

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
# AWS_PROFILE=wavefront-dev
AWS_REGION=us-west-2
ECR_ENDPOINT=$(WAVEFRONT_DEV_AWS_ACC_ID).dkr.ecr.$(AWS_REGION).amazonaws.com
COLLECTOR_ECR_REPO=$(ECR_REPO_PREFIX)/wavefront-kubernetes-collector
TEST_PROXY_ECR_REPO=$(ECR_REPO_PREFIX)/test-proxy

ecr-host:
	echo $(ECR_ENDPOINT)/$(ECR_REPO_PREFIX)/wavefront-kubernetes-collector

docker-login-eks:
	@aws ecr get-login-password --region $(AWS_REGION) |  docker login --username AWS --password-stdin $(ECR_ENDPOINT)

target-eks: docker-login-eks
	@aws eks --region $(AWS_REGION) update-kubeconfig --name k8s-saas-team-ci --alias k8s-saas-team-ci-eks

#----- TKGm -----#
target-tkgm:
	@$(MONOREPO_DIR)/scripts/connect-to-tkgm.sh

get-tkgm-lock:
	@$(MONOREPO_DIR))/scripts/get-tkgm-env-lock.sh

# create a new branch from main
# usage: make branch JIRA=XXXX OR make branch NAME=YYYY
.PHONY: branch
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

.PHONY: git-rebase
git-rebase:
	git fetch origin
	git rebase origin/main
	git log --oneline -n 10

# list the available makefile targets
.PHONY: no_targets__ list
list:
	@sh -c "$(MAKE) -p no_targets__ | awk -F':' '/^[a-zA-Z0-9][^\$$#\/\\t=]*:([^=]|$$)/ {split(\$$1,A,/ /);for(i in A)print A[i]}' | grep -v '__\$$' | sort"
