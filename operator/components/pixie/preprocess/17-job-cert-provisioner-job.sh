#!/usr/bin/env bash
set -eup pipefail

PROCESS_DIR=$(dirname $0)

"$PROCESS_DIR/update-images.sh" <&0