#!/usr/bin/env bash
set -euo pipefail

curl --location --fail-with-body --request POST "http://localhost:2878/" \
  --header 'Content-Type: text/plain' \
  --data "test.luke-to-lemans-via-troll 100 source=luke-dev-mac"
