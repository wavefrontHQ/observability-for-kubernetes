COLLECTOR_REPO_ROOT=$(git rev-parse --show-toplevel)/collector

for f in $COLLECTOR_REPO_ROOT/hack/code-health-and-support/command/*.sh; do
    echo "sourcing '${f}'"
    source "${f}"
done
