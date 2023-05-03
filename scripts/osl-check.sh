#!/usr/bin/env bash

function generateReport() {
    local component=$1
    local added_packages_file=$2
    local removed_packages_file=$3
    local upgraded_packages_file=$4

    if [ "${component}" == 'collector' ]; then
        echo "🚨 Warning: merging this PR will require a new OSL request for the Collector, which will also require one for the Operator! See diff below. 🚨"
    fi

    if [ "${component}" == 'operator' ]; then
        echo "🚨 Warning: merging this PR will require a new OSL request for the Operator! See diff below. 🚨"
    fi

    echo

    echo "## The following packages have been added:"
    echo '```'
    cat "${added_packages_file}"
    echo '```'
    echo

    echo "## The following packages have been removed:"
    echo '```'
    cat "${removed_packages_file}"
    echo '```'
    echo

    echo "## The following packages have been upgraded:"
    echo '```'
    cat "${upgraded_packages_file}"
    echo '```'
}

function main() {
    local component=$1
    local branch_start=$2

    if [ -z "${branch_start}" ]; then
        branch_start='main'
    fi

    local additions_file=$(mktemp)
    local removals_file=$(mktemp)

    git diff "${branch_start}..HEAD" -- "${component}/go.mod" | grep -e '^+\s' | awk '{print $2}' | sort > "${additions_file}"
    git diff "${branch_start}..HEAD" -- "${component}/go.mod" | grep -e '^-\s' | awk '{print $2}' | sort > "${removals_file}"

    local added_packages_file=$(mktemp)
    local removed_packages_file=$(mktemp)
    local upgraded_packages_file=$(mktemp)

    comm -23 "${additions_file}" "${removals_file}" > "${added_packages_file}"
    comm -23 "${removals_file}" "${additions_file}" > "${removed_packages_file}"
    comm -12 "${additions_file}" "${removals_file}" > "${upgraded_packages_file}"

    generateReport "${component}" "${added_packages_file}" "${removed_packages_file}" "${upgraded_packages_file}" > "${component}_report.txt"

    # exit 1 if any packages are added or removed
    [ $(wc -l < "${added_packages_file}") -eq 0 ]
    [ $(wc -l < "${removed_packages_file}") -eq 0 ]
}

main $@