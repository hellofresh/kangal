# k6

## Table of content
- [How it works](#how-it-works)
- [Configuring k6 resource requirements](#configuring-k6-resource-requirements)
- [Writing tests](#writing-tests)
- [Reporting](#reporting)
- [Logs](#logs)
- [Using k6 extensions](#using-k6-extensions)

K6 is one of the load generators implemented in Kangal. It uses the official docker image [grafana/k6](https://hub.docker.com/r/grafana/k6).

Kangal requires a JavaScript testfile describing the test.

For more information, check [k6 official site](https://k6.io/).

## How it works
You can create k6 load tests using Kangal Proxy.

Let's create a simple test file named `test.js` with this content:
```js
import { check, sleep } from 'k6';
import http from 'k6/http';

// you can define settings for your test by exporting the options object
// see https://k6.io/docs/using-k6/options/
export const options = {
  vus: 10
};

export default function () {
  // our HTTP request, note that we are saving the response to res, which can be accessed later
  const res = http.get('http://test.k6.io');

  sleep(1);

  const checkRes = check(res, {
    'status is 200': (r) => r.status === 200
  });
}
```

Now, send this to Kangal using this command:
```shell
$ curl -X POST http://${KANGAL_PROXY_ADDRESS}/load-test \
  -H 'Content-Type: multipart/form-data' \
  -F distributedPods=1 \
  -F testFile=@test.js \
  -F duration=10m \
  -F type=K6
```

Let's break it down the parameters:

- `distributedPods` is the number of k6 workers desired
- `testFile` is the JavaScript file containing your test
- `type` is the backend you want to use, `K6` in this case
- `duration` configures how long the load test will run for

> Note: You can specify test configurations through the [options object](https://k6.io/docs/getting-started/running-k6/#using-options) in the javascript code or by using [environment variables](https://k6.io/docs/using-k6/options/).
<!-- comment -->

> Note: If your test script is configured to run 500 VUs (Virtual Users) and `distributedPods` is set to 5, kangal will create five k6 jobs, each running 100 VUs to achieve the desired VU count.

## Configuring k6 resource requirements
By default, Kangal does not specify resource requirements for loadtests run with k6 as a backend.

You can specify resource limits and requests for k6 containers to ensure that when the load generator runs, it has sufficient resources and will not fail before the service does.

The following environment variables can be specified to configure this parameter:

```bash
K6_CPU_LIMITS
K6_CPU_REQUESTS
K6_MEMORY_LIMITS
K6_MEMORY_REQUESTS
```

You have to specify these variables on Kangal Controller, read more at [charts/kangal/README.md](https://github.com/hellofresh/kangal/blob/master/charts/kangal/README.md#kangal-controller-k6-specific).

More information regarding resource limits and requests can be found in the following page(s):

- <https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/>
- <https://cloud.google.com/blog/products/gcp/kubernetes-best-practices-resource-requests-and-limits>

## Writing tests
It's recommended to read [official k6 documentation](https://k6.io/docs/).

### Tests with Test Data

K6 has support for Data Parameterization via the [k6-data module](https://k6.io/docs/javascript-api/#k6-data).
See some examples on the usage of Data Parameterization in the [k6 examples](https://k6.io/docs/examples/data-parameterization/).

Kangal also supports test data with K6, the supported format file is `.csv`.

1. Prepare your test data in a `.csv` file
2. Configure test script accordingly. See [the k6 data parameterization from a CSV file example](https://k6.io/docs/examples/data-parameterization/#from-a-csv-file).
   * **Important**: _"k6 doesn't parse CSV files natively, but you can use an external library, [Papa Parse](https://www.papaparse.com/)."_
3. Add the file in the `testData` field:

```shell
$ curl -X POST http://${KANGAL_PROXY_ADDRESS}/load-test \
  -H 'Content-Type: multipart/form-data' \
  -F distributedPods=1 \
  -F testFile=@test.js \
  -F testData=@/path/to/test-data.csv \
  -F duration=10m \
  -F type=K6
```

## Reporting
k6 can write test statistics in multiple formats using the `K6_OUT` environment variable. To persist the summary, put the code below into your testfile using the [handleSummary function](https://k6.io/docs/results-visualization/end-of-test-summary/#handlesummary-callback).

```js
import http from 'k6/http';

export default function () {
  // your test code here....
}

export function handleSummary(data) {
    console.log('Preparing the end-of-test summary...');

    if (!__ENV.REPORT_PRESIGNED_URL) {
        return
    } 
    const resp = http.put(__ENV.REPORT_PRESIGNED_URL, JSON.stringify(data),
        {
           headers: { 'Content-Type': 'application/json'}
        });
    
    if (resp.status != 200) {
        console.error('Could not send summary, got status ' + resp.status);
    }
}
```

You can use <https://github.com/benc-uk/k6-reporter> to create an HTML Report.

> Note: There aren't any concept of master/worker in k6. Metrics will not be automatically aggregated. To be able to aggregate your metrics and analyse them together, you’ll need to set K6_OUT env to send statics to another service (influxdb, prometheus, etc.).

## Logs

There aren't any concept of master/worker in k6. All pods running k6 tests are workers.

For the logs of the worker pod use the index number of the worker.
Index numbers are `0`, `1`, etc., according to the number of workers you created.

```bash
curl -X GET http://${KANGAL_PROXY_ADDRESS}/load-test/loadtest-name/logs/0
```

## Using k6 extensions

By default, kangal will use grafana/k6:latest as the container image for the test jobs. If you want to use extensions built with **xk6** you'll need to create your own image. Example:

```Dockerfile
# Build the k6 binary with the extension
FROM golang:1.16.13-buster as builder

RUN go install go.k6.io/xk6/cmd/xk6@latest
RUN xk6 build --output /k6 --with github.com/walterwanderley/xk6-stomp@latest

# Use the official base image and override the k6 binary
FROM grafana/k6:latest
COPY --from=builder /k6 /usr/bin/k6
```
