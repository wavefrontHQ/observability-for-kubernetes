#!/bin/bash -e

function main() {
	cd "$(dirname "$0")"

	./uninstall-deploy-local.sh || true
	kubectl delete namespace load-test || true
}

main $@
