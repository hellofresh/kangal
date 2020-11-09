# Troubleshooting

## Problems with Kangal installation
To troubleshoot Kangal you will need access to your Kubernetes cluster.

- Log in to the cluster where Kangal is installed
- Check if Kangal Proxy and Controller pods are up and running
- Check logs of Kangal Proxy and Kangal Controller

## Problems with a specific load test
You can make basic troubleshooting using Kangal API endpoints or either exploring load test Pods if is the case of your backend.

- Get status of the load test 
```bash
curl --location --request GET 'http://${KANGAL_PROXY_ADDRESS}/load-test/loadtest-random-name/' 
```
- Get logs from the master pod
```bash
curl --location --request GET 'http://${KANGAL_PROXY_ADDRESS}/load-test/loadtest-random-name/logs' 
```
- Get logs from the worker pod
```bash
curl --location --request GET 'http://${KANGAL_PROXY_ADDRESS}/load-test/loadtest-random-name/logs/loadtest-worker-000' 
```
