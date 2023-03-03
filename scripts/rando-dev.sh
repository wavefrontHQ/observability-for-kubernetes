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
  echo ${TEAM_1_NAME} : ${TEAM_1_RESULT} # Adding quotes around "${}" outputs a newline after each name, which we don't want
  echo ${TEAM_2_NAME} : ${TEAM_2_RESULT}
}

function main() {
  # Joe's team
  TEAM_1_NAME='Team Helios :sun_with_face:'
  local team_1_dev_list='Anil,Devon,Ginwoo,Glenn,Priya'
  # Amanda's team
  TEAM_2_NAME='Team Raven :raven:'
  local team_2_dev_list='Jeremy,Jerry,Jesse,John,Peter,Yuqi'

  # Get the random order results for team 1
  local team_1_query_list="$(prepare_query_list "${team_1_dev_list}")"
  TEAM_1_RESULT="$(curl_query_to_random_generator "${team_1_query_list}")"
  # Get the random order results for team 3
  local team_2_query_list="$(prepare_query_list "${team_2_dev_list}")"
  TEAM_2_RESULT="$(curl_query_to_random_generator "${team_2_query_list}")"

  print_random_generator_results
}

main "$@"
