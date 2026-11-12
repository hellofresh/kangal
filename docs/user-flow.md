# User Flow
We expect users to communicate with Kangal by only using API, which is provided by Kangal Proxy.

> You can import [openapi.json](/openapi.json) file to your Postman and have a collection of requests to Kangal.

Here is an example of requests users can send to Kangal API to manage their load test.

## Create
Create a new load test by making a POST request to Kangal Proxy.

> **Note**: The sample CURL commands below use example test files, those files can be found in Kangal repository [/artifacts](https://github.com/hellofresh/kangal/tree/master/artifacts/loadtests) and [/examples](https://github.com/hellofresh/kangal/tree/master/examples).

**The test data will be equally divided between the number of pods set on distributedPods parameter**

The test starts as soon as Kangal creates it. If the response gets lost, [list](#list) your tests by tag before you submit again, or the same test may run twice.

When the proxy already has its maximum number of tests creating or running (`--max-load-tests`, 10 by default), it answers `429 Too Many Requests`. Submit the test again later.

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

### Overwrite an existing test
Kangal recognises a load test by its test file. While a test with the same test file exists, in any phase:

- `overwrite=false` refuses the new test with `400 Bad Request`;
- `overwrite=true` first deletes every test with that test file, including running ones started by someone else.

### Use tags
You can also tag the load test so that you can find them later, the format is `tag1:value1,tag2:value2`. Each value can be at most 63 characters long, the Kubernetes limit for label values.

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

### Use custom image
Specify the container images to use for Master and Worker roles:

```bash
curl -X POST http://${KANGAL_PROXY_ADDRESS}/load-test \
  -H 'Content-Type: multipart/form-data' \
  -F distributedPods=1 \
  -F testFile=@examples/constant_load.jmx \
  -F testData=@artifacts/loadtests/testData.csv \
  -F envVars=@artifacts/loadtests/envVars.csv \
  -F type=JMeter \
  -F tags=tag1:value1,tag2:value2 \
  -F overwrite=true \
  -F masterImage=hellofresh/kangal-jmeter-master:5.5 \
  -F workerImage=hellofresh/kangal-jmeter-worker:5.5
```

Custom images are accepted only when the proxy runs with `ALLOWED_CUSTOM_IMAGES=true`.

To pin an image by digest, give the digest without a tag and with at most two `/`, for example `grafana/k6@sha256:<digest>` or `docker.io/grafana/k6@sha256:<digest>`. Kangal doesn't recognise a tag together with a digest (`grafana/k6:1.3.0@sha256:<digest>`) or a longer path. It then runs the default image, without an error.

## Check
Check the status of the load test.

```bash
curl -X GET \
  http://${KANGAL_PROXY_ADDRESS}/load-test/loadtest-name
```

## Live monitoring
Get logs and monitor your tests.
For the logs of the main load generator process use the following command:
```bash
curl -X GET http://${KANGAL_PROXY_ADDRESS}/load-test/loadtest-name/logs
```

k6 has no main process: each of its pods is a worker, so read k6 logs per worker as shown below (see [k6 logs](k6/README.md#logs)).

### Advanced logs monitoring
For the logs of the worker pod use the index number of the worker.
Index numbers are `0`, `1`, etc., according to the number of workers you created.
```bash
curl -X GET http://${KANGAL_PROXY_ADDRESS}/load-test/loadtest-name/logs/0
```

You can also monitor the behavior of your service with your custom tools e.g. Graphite.

Example of monitoring for JMeter is described at [docs/jmeter/reporting.md](jmeter/reporting.md).

## Get static report
When the test is finished successfully the backend will save the report.

The report for a particular test can be found by the link `https://${KANGAL_PROXY_ADDRESS}/load-test/loadtest-name/report/`.

> Report persistence depends on the backend implementation.

## Delete
Delete your finished load test.

```bash
curl -X DELETE http://${KANGAL_PROXY_ADDRESS}/load-test/loadtest-name
```

## Lifetime
Kangal deletes a finished or errored load test once `CLEANUP_THRESHOLD` (1 hour by default, see [environment variables](env-vars.md#controller)) has passed since it ended. An errored test that never ended is deleted that long after it was created. Its status and logs go with it, and the status endpoint then answers `404 Not Found`.

The [report](#get-static-report) stays available after the test is deleted. Record any status or logs you need before then.

## Limits
- The test file and the test data are stored in ConfigMaps, which Kubernetes limits to 1 MiB each. A larger file fails in the controller and the test never starts. Generate large test data inside the test instead.
- `envVars` values are not hidden. They are stored as plain text in the LoadTest resource, and anyone who can read it, or the pods in the test's namespace, can see them. Use credentials made for testing only.

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
