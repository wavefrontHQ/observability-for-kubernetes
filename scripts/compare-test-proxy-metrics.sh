kill $(jobs -p) &>/dev/null || true
sleep 3
kubectl --namespace "$NS" port-forward "deploy/${PROXY_NAME}" 8888 &
trap 'kill $(jobs -p) &>/dev/null || true' EXIT
sleep 3

echo "waiting for logs..."
sleep ${SLEEP_TIME}

RES=$(mktemp)

if [ -f "${SCRIPT_DIR}/overlays/test-$K8S_ENV/metrics/${METRICS_FILE_NAME}.jsonl" ]; then
  cat "${METRICS_FILE_DIR}/${METRICS_FILE_NAME}.jsonl" "${SCRIPT_DIR}/overlays/test-$K8S_ENV/metrics/${METRICS_FILE_NAME}.jsonl" >${METRICS_FILE_DIR}/combined-metrics.jsonl
else
  cat "${METRICS_FILE_DIR}/${METRICS_FILE_NAME}.jsonl" >${METRICS_FILE_DIR}/combined-metrics.jsonl
fi

while true; do # wait until we get a good connection
  RES_CODE=$(curl --silent --output "$RES" --write-out "%{http_code}" --data-binary "@${METRICS_FILE_DIR}/combined-metrics.jsonl" "http://localhost:8888/metrics/diff")
  [[ $RES_CODE -lt 200 ]] || break
done

cat "${RES}" > "${SCRIPT_DIR}/res.txt"

if [[ $RES_CODE -gt 399 ]]; then
  red "INVALID METRICS"
  jq -r '.[]' "${RES}"
  exit 1
fi

DIFF_COUNT=$(jq "(.Missing | length) + (.Unwanted | length)" "$RES")
EXIT_CODE=0

jq -c '.Missing[]' "$RES" | sort >"${SCRIPT_DIR}/missing.jsonl"
jq -c '.Extra[]' "$RES" | sort >"${SCRIPT_DIR}/extra.jsonl"
jq -c '.Unwanted[]' "$RES" | sort >"${SCRIPT_DIR}/unwanted.jsonl"

echo "$RES"
if [[ $DIFF_COUNT -gt 0 ]]; then
  red "Missing: $(jq "(.Missing | length)" "$RES")"
  if [[ $(jq "(.Missing | length)" "$RES") -le 10 ]]; then
    cat "${SCRIPT_DIR}/missing.jsonl"
  fi
  red "Unwanted: $(jq "(.Unwanted | length)" "$RES")"
  if [[ $(jq "(.Unwanted | length)" "$RES") -le 10 ]]; then
    cat "${SCRIPT_DIR}/unwanted.jsonl"
  fi
  red "Extra: $(jq "(.Extra | length)" "$RES")"
  red "FAILED: METRICS OUTPUT DID NOT MATCH"
  if which pbcopy >/dev/null; then
    echo "$RES" | pbcopy
  fi
  exit 1
else
  green "SUCCEEDED"
fi

kill $(jobs -p) &>/dev/null || true
