kill $(jobs -p) &>/dev/null || true
start_forward_test_proxy "$NS" "$PROXY_NAME" /dev/null
trap 'stop_forward_test_proxy /dev/null' EXIT

RES=$(mktemp)

if [ -f "${SCRIPT_DIR}/overlays/test-$K8S_ENV/metrics/${METRICS_FILE_NAME}.jsonl" ]; then
  cat "${METRICS_FILE_DIR}/${METRICS_FILE_NAME}.jsonl" "${SCRIPT_DIR}/overlays/test-$K8S_ENV/metrics/${METRICS_FILE_NAME}.jsonl" >${METRICS_FILE_DIR}/combined-metrics.jsonl
else
  cat "${METRICS_FILE_DIR}/${METRICS_FILE_NAME}.jsonl" >${METRICS_FILE_DIR}/combined-metrics.jsonl
fi

FLUSH_INTERVAL=15
RETRIES=14 # TODO make configurable
printf "Diffing metrics .."
for (( i=1; i<="$RETRIES"; i++ )) do
  printf "."
  sleep "$FLUSH_INTERVAL"

  while true; do # wait until we get a good connection
    RES_CODE=$(curl --silent --output "$RES" --write-out "%{http_code}" --data-binary "@${METRICS_FILE_DIR}/combined-metrics.jsonl" "http://localhost:8888/metrics/diff" || echo "000")
    [[ $RES_CODE -lt 200 ]] || break
  done

  cat "${RES}" > "${SCRIPT_DIR}/res.txt"

  if [[ $RES_CODE -gt 399 ]]; then
    red "INVALID METRICS"
    jq -r '.[]' "${RES}"
    exit 1
  fi

  DIFF_COUNT=$(jq "(.Missing | length) + (.Unwanted | length)" "$RES")

  if [[ $DIFF_COUNT -eq 0 ]]; then
    printf " in %d tries" "$i"
    break
  fi
done
echo " done."

jq -c '.Missing[]' "$RES" | sort >"${SCRIPT_DIR}/missing.jsonl"
jq -c '.Extra[]' "$RES" | sort >"${SCRIPT_DIR}/extra.jsonl"
jq -c '.Unwanted[]' "$RES" | sort >"${SCRIPT_DIR}/unwanted.jsonl"

echo "Metrics diff: $RES"
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
  if which pbcopy &>/dev/null; then
    echo "$RES" | pbcopy
  fi
  exit 1
fi

green "SUCCEEDED"

stop_forward_test_proxy /dev/null
