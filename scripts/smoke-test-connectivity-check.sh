for _ in {1..12}; do
  printf "Testing proxy connectivity ..."
  if kubectl get pods --all-namespaces >/dev/null ; then
    green "Proxy connectivity smoke test succeeded!"
    exit 0
  fi
  printf "."
  sleep 5
done

red "Proxy connectivity smoke test failed!"
exit 1
