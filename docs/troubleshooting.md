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
## How to prepare your service for load and performance testing
We prepared this great [documentation](https://hellofresh.atlassian.net/l/cp/Fii0HScP) to help with some common questions:
- Understanding your service
- Test data
- Test requirements
- Monitoring
- Planning and running
