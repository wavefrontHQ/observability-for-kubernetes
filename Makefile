export # Used to let all sub-make use the initialized value of variables whose names consist solely of alphanumerics and underscores

SEMVER_CLI_BIN:=$(if $(which semver-cli),$(which semver-cli),$(GOPATH)/bin/semver-cli)

GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)

.PHONY: semver-cli
semver-cli: $(SEMVER_CLI_BIN)

$(SEMVER_CLI_BIN):
	@(CGO_ENABLED=0 go install github.com/davidrjonas/semver-cli@latest)

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

.PHONY: rebase
rebase:
	git fetch origin
	git rebase origin/main
	git log --oneline -n 10

.PHONY: nuke-kind
nuke-kind:
	kind delete cluster
	kind create cluster --image kindest/node:v1.24.7 #setting to 1.24.* to avoid floating to 1.25 which we currently don't support

.PHONY: nuke-kind-ha
nuke-kind-ha:
	kind delete cluster
	kind create cluster --config "make/k8s-envs/kind-ha.yml"

.PHONY: kind-connect-to-cluster
kind-connect-to-cluster:
	kubectl config use kind-kind

.PHONY: target-kind
target-kind:
	kubectl config use kind-kind
