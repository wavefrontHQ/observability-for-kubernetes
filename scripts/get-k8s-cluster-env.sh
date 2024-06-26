#!/bin/bash -e

CURRENT_CONTEXT=$(kubectl config current-context 2>/dev/null)

if grep -q "kind" <<< "$CURRENT_CONTEXT"; then
  echo "Kind"
elif grep -q "gke" <<< "$CURRENT_CONTEXT"; then
  echo "GKE"
elif grep -q "aks" <<< "$CURRENT_CONTEXT"; then
  echo "AKS"
elif grep -q "k8po-ci" <<< "$CURRENT_CONTEXT"; then
  echo "AKS"
elif grep -q "eks" <<< "$CURRENT_CONTEXT"; then
  echo "EKS"
elif grep -q "openshift" <<< "$CURRENT_CONTEXT"; then
  echo "Openshift"
#  name for sheepctl pool enviroment
elif grep -q "tkg-mgmt" <<< "$CURRENT_CONTEXT"; then
  echo "TKGm"
else
  echo "No matching env for ${CURRENT_CONTEXT}"
fi