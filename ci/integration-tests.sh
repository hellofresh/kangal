#!/usr/bin/env bash
set -e

export AWS_ACCESS_KEY_ID=""
export AWS_SECRET_ACCESS_KEY=""

# Create a simple service to be called from the created loadTest
# This service will log all the requests
function prepare_jmeter_integration_test {
  kubectl create ns test-busybox
  kubectl run busybox --namespace=test-busybox --port=8280 --image=busybox -- sh -c "echo 'Hello' > /var/www/index.html && httpd -f -p 8280 -h /var/www/ -vv"
  kubectl expose pod busybox --type=NodePort --namespace=test-busybox
  kubectl wait --for=condition=ready pod busybox -n test-busybox

  NODE_IP=$(kubectl describe pod busybox -n test-busybox | grep "Node:" | cut -d '/' -f 2)
  NODE_PORT=$(kubectl get service busybox -n test-busybox | grep busybox | cut -d ':' -f 2 | cut -d '/' -f 1)

  # Update JMX test to use node_ip:port to send requests
  sed -i "s/TEST_IP/$NODE_IP/g" ./pkg/controller/testdata/valid/integration_test.jmx
  sed -i "s/TEST_PORT/${NODE_PORT}/g" ./pkg/controller/testdata/valid/integration_test.jmx
}

# Run dummy grpc server to test ghz
function prepare_ghz_integration_test {
  kubectl create ns test-ghz
  kubectl run greeter-server --namespace=test-ghz --port=50051 --image=greeter_server:local
  kubectl expose pod greeter-server --type=NodePort --namespace=test-ghz
  kubectl wait --for=condition=ready pod greeter-server -n test-ghz

  NODE_IP=$(kubectl describe pod greeter-server -n test-ghz | grep "Node:" | cut -d '/' -f 2)
  NODE_PORT=$(kubectl get service greeter-server -n test-ghz | grep greeter-server | cut -d ':' -f 2 | cut -d '/' -f 1)

  sed -i "s/0.0.0.0/$NODE_IP/g" ./pkg/controller/testdata/ghz/config.json
  sed -i "s/50051/${NODE_PORT}/g" ./pkg/controller/testdata/ghz/config.json
}

# Start Kangal
function prepare_kangal {
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
  WEB_HTTP_PORT=8888 CLEANUP_THRESHOLD=10s SYNC_HANDLER_TIMEOUT=10s ./bin/kangal controller --kubeconfig="$HOME/.kube/config" >/tmp/kangal_controller.log 2>&1 &
  PID_CONTROLLER=$!
  sleep 1
  echo "Check if controller is running"
  if ! kill -s 0 "${PID_CONTROLLER}"; then
    echo "Failed to run kangal controller"
    cat /tmp/kangal_controller.log
    exit 1
  fi

  echo "Controller is running"
}

# Check the logs of busybox server and count the number of requests sent by JMeter.
# integration_test.jmx is designed to send 30 requests during 30 seconds.
# jmeter_integration_test.go creates a loadTest with 2 distributed pods, which leads us to 60 desired requests to the server
function verify_jmeter_integration_test {
  DESIRED_REQUESTS_COUNT=60
  REQUEST_COUNT=$(kubectl logs busybox -n test-busybox | grep -c "response:200")
  if [ "${REQUEST_COUNT}" -ne "${DESIRED_REQUESTS_COUNT}" ]; then
    echo "JMeter Integration Test sent $REQUEST_COUNT requests, but $DESIRED_REQUESTS_COUNT requests were expected. Test failed."
    exit 1
  fi
}

# Verify that greeter-server received something
function verify_ghz_integration_test {
  REQUEST_COUNT=$(kubectl logs greeter-server -n test-ghz --tail=100 | grep -c "Received")
  if [ "$REQUEST_COUNT" -eq "0" ]; then
    echo "Expected dummy grpc server to receive >0 request, but found 0. Test failed."
    exit 1
  fi
}

# Main
# --------
# Prepare environment for integration test
if [[ "$SKIP_JMETER_INTEGRATION_TEST" != "1" ]]; then
  prepare_jmeter_integration_test
fi

if [[ "$SKIP_GHZ_INTEGRATION_TEST" != "1" ]]; then
  prepare_ghz_integration_test
fi

prepare_kangal


# Run the integration tests
echo "Starting integration tests"
KUBECONFIG="$HOME/.kube/config" make test-integration

# Verify results
if [[ "$SKIP_JMETER_INTEGRATION_TEST" != "1" ]]; then
  verify_jmeter_integration_test
fi

if [[ "$SKIP_GHZ_INTEGRATION_TEST" != "1" ]]; then
  verify_ghz_integration_test
fi
