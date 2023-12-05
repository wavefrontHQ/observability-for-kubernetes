#!/usr/bin/env bash
set -uo pipefail

STREAM_NAME=experiment6
RECEIVER_NAME="$STREAM_NAME-receiver"
LEMANS_RESOURCE_SERVER="localhost:8001"
RECEIVER_URI=http://localhost:8000/report
CSP_SECRET="$(echo -n "$(cat tmp/csp_username):$(cat tmp/csp_password)" | base64)"

token_file=$(mktemp)

printf "\nauthenticating\n"
curl --fail-with-body --location --request POST 'https://console-stg.cloud.vmware.com/csp/gateway/am/api/auth/authorize' \
  --header 'Content-Type: application/x-www-form-urlencoded' \
  --header "Authorization: Basic $CSP_SECRET" \
  --data-urlencode 'grant_type=client_credentials' \
  -o "$token_file"

CSP_AUTH_TOKEN="$(gojq -r .access_token  "$token_file")"

printf "\ncreating consumer receiver\n"

receiver_json_file="$(mktemp)"
printf "\n$receiver_json_file\n"
cat <<-JSON >"$receiver_json_file"
{
  "name": "$RECEIVER_NAME",
  "address": "$RECEIVER_URI",
  "useHttp2": false
}
JSON

curl --location --fail-with-body --request POST "http://$LEMANS_RESOURCE_SERVER/le-mans/v2/resources/receivers" \
  --header "x-xenon-auth-token: $CSP_AUTH_TOKEN" \
  --header 'Content-Type: application/json' \
  --data @"$receiver_json_file"

printf "\ncreating consumer receiver starter\n"

consumer_receiver_starter_json_file="$(mktemp)"
cat <<-JSON >"$consumer_receiver_starter_json_file"
{
   "name": "$STREAM_NAME-consumer",
   "factoryLink": "/le-mans/consumers/kafka",
   "startJsonState": "{'topic': '$STREAM_NAME', 'retryTopic': '$STREAM_NAME', 'statusCodesToRetryIndefinitely': [503], 'contentType': 'application/json', 'receiverLink': '/le-mans/v2/resources/receivers/$STREAM_NAME-receiver', 'maxRetryLimit': 10000, 'kafkaProperties': {'group.id': 'le-mans', 'fetch.min.bytes': 1, 'key.deserializer': 'org.apache.kafka.common.serialization.StringDeserializer', 'max.poll.records': 150, 'max.partition.fetch.bytes': 2097152, 'auto.offset.reset': 'latest', 'bootstrap.servers': 'localhost:9092', 'value.deserializer': 'org.apache.kafka.common.serialization.StringDeserializer'}, 'kafkaProducerPath': '/le-mans/receivers/kafka-producer/$STREAM_NAME-producer'}"
}
JSON

curl --location --fail-with-body --request PATCH "http://$LEMANS_RESOURCE_SERVER/le-mans/v2/resources/receiver-starters/$STREAM_NAME-consumer" \
  --header "x-xenon-auth-token: $CSP_AUTH_TOKEN" \
  --header 'Content-Type: application/json' \
  --data @"$consumer_receiver_starter_json_file"

printf "\ncreating producer receiver starter\n"

producer_receiver_starter_json_file="$(mktemp)"
cat <<-JSON >"$producer_receiver_starter_json_file"
{
    "name": "$STREAM_NAME-producer",
    "factoryLink": "/le-mans/receivers/kafka-producer",
    "startJsonState": "{'topicName':'$STREAM_NAME','numberOfPartitions':'1','replicationFactor':'1','retentionPeriod':'7200000','kafkaProperties':{'bootstrap.servers':'localhost:9092','key.serializer':'org.apache.kafka.common.serialization.StringSerializer','value.serializer':'org.apache.kafka.common.serialization.StringSerializer','retries':'0','linger.ms':'5','lemans.KafkaProducerService.KAFKA_KEY':'org_id','partitioner.class':'com.vmware.lemans.receivers.kafka.RoundRobinPartitioner'}}"

}
JSON

curl --location --fail-with-body --request POST "http://$LEMANS_RESOURCE_SERVER/le-mans/v2/resources/receiver-starters" \
  --header "x-xenon-auth-token: $CSP_AUTH_TOKEN" \
  --header 'Content-Type: application/json' \
  --data @"$producer_receiver_starter_json_file"

printf "\ncreating producer receiver\n"

producer_receiver_json_file="$(mktemp)"
printf "\n$producer_receiver_json_file"
cat <<-JSON >"$producer_receiver_json_file"
{
    "name": "$STREAM_NAME-kafka-producer-receiver",
    "address": "/le-mans/receivers/kafka-producer/$STREAM_NAME-producer",
    "useHttp2": true
}
JSON

curl --location --fail-with-body --request POST "http://$LEMANS_RESOURCE_SERVER/le-mans/v2/resources/receivers" \
  --header "x-xenon-auth-token: $CSP_AUTH_TOKEN" \
  --header 'Content-Type: application/json' \
  --data @"$producer_receiver_json_file"

printf "\ncreating stream\n"

stream_json_file="$(mktemp)"
printf "\n$stream_json_file\n"

cat <<-JSON >"$stream_json_file"
{
    "name": "$STREAM_NAME",
    "deliveryPolicy": "WAIT_ALL",
    "receiverLinks": ["/le-mans/v2/resources/receivers/$STREAM_NAME-kafka-producer-receiver"]
}
JSON

curl --location --fail-with-body --request POST "http://$LEMANS_RESOURCE_SERVER/le-mans/v2/resources/streams" \
  --header "x-xenon-auth-token: $CSP_AUTH_TOKEN" \
  --header 'Content-Type: application/json' \
  --data @"$stream_json_file"
