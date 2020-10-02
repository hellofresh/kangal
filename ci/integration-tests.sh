#!/usr/bin/env bash
set -e

export AWS_ACCESS_KEY_ID=""
export AWS_SECRET_ACCESS_KEY=""

# Adjust resource requests and limits
#export JMETER_MASTER_CPU_LIMITS="2000m"
export JMETER_MASTER_CPU_REQUESTS="150m"
#export JMETER_MASTER_MEMORY_LIMITS="4Gi"
export JMETER_MASTER_MEMORY_REQUESTS="500Mi"
#export JMETER_WORKER_CPU_LIMITS="2000m"
export JMETER_WORKER_CPU_REQUESTS="150m"
#export JMETER_WORKER_MEMORY_LIMITS="4Gi"
export JMETER_WORKER_MEMORY_REQUESTS="500Mi"

echo "Starting kangal proxy"
./bin/kangal proxy --kubeconfig="$HOME/.kube/config" --max-load-tests 1 > /tmp/kangal_proxy.log 2>&1 &
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
WEB_HTTP_PORT=8888 ./bin/kangal controller --kubeconfig="$HOME/.kube/config" > /tmp/kangal_controller.log 2>&1 &
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
