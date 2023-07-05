#!/usr/bin/env bash

TKGM_CONTEXT_NAME=tkg-mgmt-vc-admin@tkg-mgmt-vc

function main() {
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
  sheepctl pool lock tkg-2.1-vcenter-7.0.0 --from-namespace shepherd-official --lifetime 15h --output lock.json 2>&1 sheepctl_output
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
