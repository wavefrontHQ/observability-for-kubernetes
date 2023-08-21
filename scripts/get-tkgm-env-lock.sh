#!/usr/bin/env bash

TKGM_CONTEXT_NAME=tkg-mgmt-vc-admin@tkg-mgmt-vc

function main() {
  local lease_time=$1
  if [ -z "${lease_time}" ]; then
    lease_time=15h
  fi

  if [[ ! -x "$(command -v sheepctl)" ]]; then
    if [[ ! "$(uname -a)" =~ "Darwin" ]]; then
      curl -O http://files.pks.eng.vmware.com/ci/artifacts/shepherd/latest/sheepctl-linux-amd64
      chmod +x sheepctl-linux-amd64
      sudo mv sheepctl-linux-amd64 /usr/local/bin/sheepctl
    else
      echo "Please download the sheepctl cli: http://docs.shepherd.run/content/home/installation.html"
      exit 1
    fi
  fi

  sheepctl pool list --public -u shepherd.run
  sheepctl target set -u shepherd.run -n k8po-team

  set +e
  sheepctl pool lock tkg-2.1-vcenter-7.0.0 \
    --from-namespace shepherd-official \
    --lifetime "${lease_time}" --output lock.json \
    > sheepctl_output 2>&1

  local exit_code=$?
  if [[ $exit_code != 0 ]]; then
    cat sheepctl_output
    exit 1
  fi
  set -e

  echo "TKGm lock acquired."
  if [[ "$(uname -a)" =~ "Darwin" ]]; then
    echo "Use scripts/connect-to-tkgm.sh to connect."
  fi
}

main $@
