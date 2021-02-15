#!/usr/bin/env bash
set -e

export AWS_ACCESS_KEY_ID=""
export AWS_SECRET_ACCESS_KEY=""

echo "Starting kangal proxy"
./bin/kangal proxy --kubeconfig="$HOME/.kube/config" --max-load-tests 1 >/tmp/kangal_proxy.log 2>&1 &
PID_PROXY=$!
sleep 1
echo "Check if proxy is running"
if ! kill -s 0 "${PID_PROXY}"; then
  echo "Failed to run kangal proxy"
  cat /tmp/kangal_proxy.log
  exit 1
fi

echo "Proxy is running"
echo "Starting kangal controller"
WEB_HTTP_PORT=8888 ./bin/kangal controller --kubeconfig="$HOME/.kube/config" >/tmp/kangal_controller.log 2>&1 &
PID_CONTROLLER=$!
sleep 1
echo "Check if controller is running"
if ! kill -s 0 "${PID_CONTROLLER}"; then
  echo "Failed to run kangal controller"
  cat /tmp/kangal_controller.log
  exit 1
fi

echo "Controller is running"
echo "Starting integration tests"

# Run the integration tests
KUBECONFIG="$HOME/.kube/config" make test-integration
