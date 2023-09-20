#!/usr/bin/env bash
set -e pipefail

cat <<EOF
dataSource.user=postgres
dataSource.portNumber=5432
dataSource.serverName=postgres
dataSource.databaseName=postgres
dataSource.password=password
EOF