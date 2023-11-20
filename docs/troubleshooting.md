# Troubleshooting

## Problems with Kangal installation
To troubleshoot Kangal you will need access to your Kubernetes cluster.

- Log in to the cluster where Kangal is installed
- Check if Kangal Proxy and Controller pods are up and running
- Check logs of Kangal Proxy and Kangal Controller

## Load test killed or interrupted during execution
Kangal backends usually run their load tests using Kubernetes Jobs.
Jobs will spawn Pods, to perform the workload until:

- The expected number of completions (usually the same as distributed pods spec);
- The backoff limit is reached (subject to backends that can recover from failures);

The Pods created by Jobs are ephemeral resources subject to the lifecycle
of the Node they were scheduled. Which means they won't survive an eviction
in case of lack of resources or Node maintenance.

In case a load test was interrupted, and the backend can't recover from a failure,
you should run a new test.

## Problems with a specific load test
You can make basic troubleshooting using Kangal API endpoints or either exploring load test Pods if is the case of your backend.

- Get status of the load test
```bash
curl -X GET 'http://${KANGAL_PROXY_ADDRESS}/load-test/loadtest-random-name/' 
```
- Get logs from the master pod
```bash
curl -X GET 'http://${KANGAL_PROXY_ADDRESS}/load-test/loadtest-random-name/logs' 
```
- Get logs from the worker pod
```bash
curl -X GET 'http://${KANGAL_PROXY_ADDRESS}/load-test/loadtest-random-name/logs/0' 
```

## I want to use a specific version of docker image for my backend but another version is used automatically
If you want to use a custom docker image for your load tests, as describe here, check the following:

- you have `ALLOWED_CUSTOM_IMAGES=true` environment variable set for your Kangal Proxy deployment. If not - no custom images are allowed

The default images and tags are specified as constants in `/pkg/backends/your_backend_name/backend.go` files. You can find
an example of K6 [here](https://github.com/hellofresh/kangal/blob/master/pkg/backends/k6/backend.go#L26).

If you want to redefine the default images and tags - use deployment [environment variables](https://github.com/hellofresh/kangal/blob/master/docs/env-vars.md#backend-specific-configuration) for Proxy and Controller.

- `JMETER_MASTER_IMAGE_NAME` and `JMETER_MASTER_IMAGE_TAG` for JMeter master pods and `JMETER_WORKER_IMAGE_NAME` and `JMETER_WORKER_IMAGE_TAG` for JMeter worker pods
- `LOCUST_IMAGE_NAME` and `LOCUST_IMAGE_TAG` for Locust
- `GHZ_IMAGE_NAME` and `GHZ_IMAGE_TAG` for Ghz
- `K6_IMAGE_NAME` and `K6_IMAGE_TAG` for K6
