#!/bin/bash
set -e

function curl_query_to_random_generator() {
  local query="$1"

  curl --silent \
    'https://www.random.org/lists/?mode=advanced' \
    -H 'authority: www.random.org' \
    -H 'content-type: application/x-www-form-urlencoded' \
    -H 'accept: text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9' \
    -H 'sec-fetch-site: same-origin' \
    -H 'sec-fetch-mode: navigate' \
    -H 'sec-fetch-user: ?1' \
    -H 'sec-fetch-dest: document' \
    -H 'referer: https://www.random.org/lists/?mode=advanced' \
    --data "list=${query}&format=plain&rnd=new" \
    --compressed
}

function prepare_query_list() {
  local space='%0D%0A'
  local team_csv_list="$1"

  # replace commas with 'space'
  local query_list="${team_csv_list//,/${space}}"

  echo "${query_list}"
}

function print_random_generator_results() {
  echo ${TEAM_NAME} :
  echo "${TEAM_RESULT}" # Adding quotes around "${}" outputs a newline after each name
}

function print_usage_and_exit() {
  echo "Failure: $1"
  echo "Usage: $0 [flags] [options]"
  echo -e "\t-n team name (required)"
  echo -e "\t-l list of devs (required, ex: Anil,Mark,Priya,John,Yuqi)"
  exit 1
}

function main() {
  TEAM_NAME=''
  local team_dev_list=''

  while getopts "n:l:" opt; do
    case $opt in
      n) TEAM_NAME="$OPTARG" ;;
      l) team_dev_list="$OPTARG" ;;
      \?) print_usage_and_exit "Invalid option: -$OPTARG" ;;
    esac
  done

  if [[ -z "${TEAM_NAME}" ]]; then
    print_usage_and_exit "-n <TEAM_NAME> is required"
  fi

  if [[ -z "${team_dev_list}" ]]; then
    print_usage_and_exit "-l <TEAM_DEV_LIST> is required"
  fi

  # Get the random order results for the team
  local team_query_list="$(prepare_query_list "${team_dev_list}")"
  TEAM_RESULT="$(curl_query_to_random_generator "${team_query_list}")"

  print_random_generator_results
}

main "$@"
