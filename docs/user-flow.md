# User Flow
We expect users to communicate with Kangal by only using API, which is provided by Kangal Proxy.

> You can import [openapi.json](/openapi.json) file to your Postman and have a collection of requests to Kangal.

Here is an example of requests users can send to Kangal API to manage their load test.

## Create 
Create a new load test by making a POST request to Kangal Proxy.

> **Note**: The sample CURL commands below use example test files, those files can be found in [Kangal repository](https://github.com/hellofresh/kangal/).

### Using JMeter
```shell
curl -X POST http://${KANGAL_PROXY_ADDRESS}/load-test \
  -H 'Content-Type: multipart/form-data' \
  -F distributedPods=1 \
  -F testFile=@examples/constant_load.jmx \
  -F testData=@artifacts/loadtests/testData.csv \
  -F envVars=@artifacts/loadtests/envVars.csv \
  -F type=JMeter \
  -F overwrite=true
```

### Using Locust
```shell
curl -X POST http://${KANGAL_PROXY_ADDRESS}/load-test \
  -H 'Content-Type: multipart/form-data' \
  -F distributedPods=1 \
  -F testFile=@examples/locustfile.py \
  -F envVars=@artifacts/loadtests/envVars.csv \
  -F type=Locust \
  -F duration=10m \
  -F targetURL=http://my-app.example.com \
  -F overwrite=true
```

You can also tag the load test so that you can find them later, the format is `tag1:value1,tag2:value2`

```bash
curl -X POST http://${KANGAL_PROXY_ADDRESS}/load-test \
  -H 'Content-Type: multipart/form-data' \
  -F distributedPods=1 \
  -F testFile=@examples/constant_load.jmx \
  -F testData=@artifacts/loadtests/testData.csv \
  -F envVars=@artifacts/loadtests/envVars.csv \
  -F type=JMeter \
  -F tags=tag1:value1,tag2:value2 \
  -F overwrite=true
```

## Check 
Check the status of the load test.

```
curl -X GET \
  http://${KANGAL_PROXY_ADDRESS}/load-test/loadtest-name
```

## Live monitoring
Get logs and monitor your tests. 
For the logs of the main load generator process use the following command:
```
curl -X GET http://${KANGAL_PROXY_ADDRESS}/load-test/loadtest-name/logs
```
### Advanced logs monitoring
For the logs of the worker pod use the index number of the worker. 
Index numbers are `0`, `1`, etc, according to the number of workers you created.
```bash
curl -X GET http://${KANGAL_PROXY_ADDRESS}/load-test/loadtest-name/logs/0
```

You can also monitor the behavior of your service with your custom tools e.g. Graphite.

Example of monitoring for JMeter is described at [docs/jmeter/reporting.md](jmeter/reporting.md).

## Get static report. 
When the test is finished successfully the backend will save the report.

The report for a particular test can be found by the link `https://${KANGAL_PROXY_ADDRESS}/load-test/loadtest-name/report/`.

> Report persistence depends on the backend implementation.

## Delete 
Delete your finished load test.

```
curl -X DELETE http://${KANGAL_PROXY_ADDRESS}/load-test/loadtest-name
```

## List

You can find out all the load tests

```bash
curl http://${KANGAL_PROXY_ADDRESS}/load-test
```

Output for this endpoint is paginated and default limit and possible max value per page is set to `50`.
Use `MAX_LIST_LIMIT` env var when running proxy to change default value.

You can filter by `tags`

```bash
curl 'http://${KANGAL_PROXY_ADDRESS}/load-test?tags=tag1:value1'
```

You can filter by `phase`, possible phases are: `creating, starting, running, finished, errored`

```bash
curl 'http://${KANGAL_PROXY_ADDRESS}/load-test?phase=running'
```

All together
```bash
curl 'http://${KANGAL_PROXY_ADDRESS}/load-test?phase=running&tags=tag1:value1'
```

Use custom limit value for your search

```bash
curl 'http://${KANGAL_PROXY_ADDRESS}/load-test?tags=tag1:value1&limit=10'
```
