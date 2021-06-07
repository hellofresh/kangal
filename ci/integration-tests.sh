#!/usr/bin/env bash
set -e

export AWS_ACCESS_KEY_ID=""
export AWS_SECRET_ACCESS_KEY=""

# Create a simple service to be called from the created loadTest
# This service will log all the requests
kubectl create ns test-busybox

kubectl run --generator=run-pod/v1 busybox --namespace=test-busybox --port=8280 --image=busybox -- sh -c "echo 'Hello' > /var/www/index.html && httpd -f -p 8280 -h /var/www/ -vv"

kubectl expose pod busybox --type=NodePort --namespace=test-busybox

kubectl wait --for=condition=ready pod busybox -n test-busybox

#extract the node IP
NODE_IP=$(kubectl describe pod busybox -n test-busybox | grep "Node:" | cut -d '/' -f 2)

#extract the node port
NODE_PORT=$(kubectl get service busybox -n test-busybox | grep busybox | cut -d ':' -f 2 | cut -d '/' -f 1)

#update JMX test to use node:port
sed -i "s/TEST_IP/$NODE_IP/g" ./pkg/controller/testdata/valid/integration_test.jmx
sed -i "s/TEST_PORT/${NODE_PORT}/g" ./pkg/controller/testdata/valid/integration_test.jmx

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
echo "Starting integration tests"

# Run the integration tests
KUBECONFIG="$HOME/.kube/config" make test-integration

# check the logs of busybox server and count the number of requests sent by JMeter
# integration_test.jmx is designed to send 30 requests during 30 seconds.
# jmeter_integration_test.go creates a loadTest with 2 distributed pods, which leads us to 60 desired requests to the server
DESIRED_REQUESTS_COUNT=60
REQUEST_COUNT=$(kubectl logs busybox -n test-busybox | grep -c "response:200")
if [ "${REQUEST_COUNT}" -ne "${DESIRED_REQUESTS_COUNT}" ]; then
  echo "JMeter Integration Test sent $REQUEST_COUNT requests, but $DESIRED_REQUESTS_COUNT requests were expected. Test failed."
  exit 1
fi
