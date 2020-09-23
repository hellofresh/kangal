# Kangal User Flow

We expect users to communicate with Kangal by only using API, which is provided by Kangal Proxy.
Here you can find recommendations for users to communicate with Kangal using JMeter as load generator.

1 - Prepare JMeter scripts, see details in [How to write JMeter tests](jmeter-in-kangal/How-to-write-tests.md).
 Examples can be found in [examples folder](../examples). To generate the load against the real server these tests should be tuned.

2 - Create a new load test by making a POST request to Kangal Proxy.
This example CURL command uses example JMeter test some fake test data and environment variable data just as an example.
<p align="center">  
<img src="jmeter-in-kangal/images/sending_request_postman.png" height="500">
</p>

```
curl -X POST http://${KANGAL_PROXY_ADDRESS}/load-test \
  -H 'Content-Type: multipart/form-data' \
  -F distributedPods=1 \
  -F testFile=@./examples/constant_load.jmx \
  -F testData=@./artifacts/loadtests/testData.csv \
  -F envVars=@./artifacts/loadtests/envVars.csv \
  -F type=JMeter
```
3 - Check the status of the load test

```
curl -X GET \
  http://${KANGAL_PROXY_ADDRESS}/load-test/loadtest-name
```

4 - Get logs and monitor your tests. Example tests offer to use the Backend Listener element to collect metrics in InfluxDB. 
You can also monitor the behavior of your service with your custom tools e.g. Graphite.

```
curl -X GET \
  http://${KANGAL_PROXY_ADDRESS}/load-test/loadtest-name/logs
```

5 - View the report. When the test is finished successfully Kangal will generate the JMeter report and save it in S3 bucket. 
The report for a particular test can be found by the link `https://${KANGAL_PROXY_ADDRESS}/load-test/loadtest-name/report/main.html`

6 - Delete your finished load test

```
curl -X DELETE \
  http://${KANGAL_PROXY_ADDRESS}/load-test/loadtest-name
```