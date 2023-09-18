#!/usr/bin/env bash
set -e pipefail

cat <<EOF
dataSource.user=$(whoami)
dataSource.portNumber=5432
dataSource.serverName=localhost
dataSource.databaseName=postgres
dataSource.password=
EOF