#!/bin/bash -e

function wait_for_user() {
  wait_msg=$1
  read -p "${wait_msg} Press [Enter] to continue or [ctrl-c] to skip the rest of the script. "
}

function confirm_user() {
  wait_msg=$1
  local answer
  read -p "${wait_msg} [Y/n]: " answer

  if [ "${answer}" == 'n' ]; then
    echo 'As you wish. I will not.'
    return 1
  fi

  echo 'Yes, I will do your bidding.'
  return 0
}

ALL_REPOS=(
  git@github.com:sunnylabs/integrations.git
  git@github.com:wavefrontHQ/observability-for-kubernetes.git
  git@github.com:wavefrontHQ/wavefront-kubernetes-adapter.git
  git@github.com:wavefrontHQ/wavefront-proxy.git
  git@github.com:wavefrontHQ/prometheus-storage-adapter.git
  git@github.com:wavefrontHQ/helm.git
  git@github.com:wavefrontHQ/wavefront-kubernetes.git
  git@github.com:wavefrontHQ/docs.git
  git@gitlab.eng.vmware.com:tobs-k8s-group/tmc-wavefront-operator.git
)

function clone_repos() {
  local install_type=$1
  for repo in ${ALL_REPOS[@]}; do
    if [ $install_type == "all" ]; then
      git clone $repo
    elif [ $install_type == "confirm_each" ]; then
      confirm_user "Clone '${repo}'?" \
        && git clone $repo
    fi
  done
}

function initialize_workspace() {
  mkdir -p ~/workspace/
  pushd ~/workspace/
    if confirm_user "Would you like me to clone each repo? Any key for yes or 'n' to decide on each repo."; then
      clone_repos 'all'
    else
      clone_repos 'confirm_each'
    fi
  popd
}

function setup_dev_environment() {
  if ! command -v brew; then
    NONINTERACTIVE=1 /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
  fi

  confirm_user 'Should I install Oh My Zsh?' \
    && sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)"


}

function main() {
  confirm_user 'Would you like me to open the Onboarding Confluence doc in your browser?' \
    && open 'https://confluence.eng.vmware.com/display/CNA/K8PO+Onboarding'

  confirm_user 'Should I install xcode command line tools? (Is this your first time running commands on this computer?)' \
    && xcode-select --install

  confirm_user 'Should I initialize your workspace directory and repos?' \
    && initialize_workspace

  confirm_user 'Should I set up your development environment?' \
    && setup_dev_environment

  echo "TODO: this can be extended to mirror or replace the Onboarding Confluence doc but we need to decide if it's worth the time."
}

main "${@}"
