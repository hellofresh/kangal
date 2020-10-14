# Locust

## Table of content
- [How it works](#how-it-works)
- [Configuring Locust resource requirements](#configuring-locust-resource-requirements)
- [Writing tests](#writing-tests)
- [Reporting](#reporting)

Locust is one of the load generators implemented in Kangal. It uses the official docker image [locustio/locust](https://hub.docker.com/r/locustio/locust).

Kangal requires a py testfile describing the test.

For more information, check [Locust official website](https://locust.io/).

## How it works
You can create Locust load tests using Kangal Proxy.

Let's create a simple test file named `locustfile.py` with this content:
```python
from locust import HttpUser, task, between

class ExampleLoadTest(HttpUser):
    wait_time = between(5, 15)

    @task
    def example_page(self):
        self.client.get('/example-page')
```

Now, send this to Kangal using this command:
```shell
$ curl -X POST http://${KANGAL_PROXY_ADDRESS}/load-test \
  -H 'Content-Type: multipart/form-data' \
  -F distributedPods=1 \
  -F testFile=@locustfile.py \
  -F type=Locust \
  -F duration=10m \
  -F targetURL=http://my-app.example.com
```

Let's break it down the parameters:
- `distributedPods` is the number of Locust workers desired
- `testFile` is the locustfile containing your test
- `type` is the backend you want to use, `Locust` in this case
- `duration` configures how long the load test will run for
- `targetURL` is the host to be prefixed on all relative URLs in the locustfile

> Note: If you don't specify `duration`, your tests will run infinitely.

> Note: If you don't specify `targetURL`, be sure to use absolute URLs on your locustfile or your tests will fail.

Here's another example:
```python
from locust import HttpUser, task, between

class ExampleLoadTest(HttpUser):
    wait_time = between(5, 15)

    @task
    def example_page(self):
        self.client.get('http://my-app.example.com/example-page')
```

And again, upload it to Kangal:
```shell
$ curl -X POST http://${KANGAL_PROXY_ADDRESS}/load-test \
  -H 'Content-Type: multipart/form-data' \
  -F distributedPods=1 \
  -F testFile=@locustfile.py \
  -F type=Locust
```

In this last example, the test will run infinitely and no `targetURL` were specified.

## Configuring Locust resource requirements
By default, Kangal does not specify resource requirements for loadtests run with Locust as a backend.

You can specify resource limits and requests for Locust master and worker containers to ensure that when the load generator runs, it has sufficient resources and will not fail before the service does.

The following environment variables can be specified to configure this parameter:

```
LOCUST_MASTER_CPU_LIMITS
LOCUST_MASTER_CPU_REQUESTS
LOCUST_MASTER_MEMORY_LIMITS
LOCUST_MASTER_MEMORY_REQUESTS
LOCUST_WORKER_CPU_LIMITS
LOCUST_WORKER_CPU_REQUESTS
LOCUST_WORKER_MEMORY_LIMITS
LOCUST_WORKER_MEMORY_REQUESTS
```

You have to specify these variables on Kangal Controller, read more at [charts/kangal/README.md](/charts/kangal/README.md#kangal-controller-locust-specific).

More information regarding resource limits and requests can be found in the following page(s):
- https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
- https://cloud.google.com/blog/products/gcp/kubernetes-best-practices-resource-requests-and-limits

## Writing tests
It's recommended to read [official Locust documentation](https://docs.locust.io/en/stable/writing-a-locustfile.html).

## Reporting
Locust can write test statistics in CSV format, to persist those files, put the code below into your locustfile.

> Note: Kangal Locust implementation automatically exports reports to `/tmp/` folder.

```python
import glob, tarfile, requests, os

from locust import events, runners
from locust.runners import MasterRunner

@events.quitting.add_listener
def hook_quit(environment):
    presigned_url = os.environ.get('REPORT_PRESIGNED_URL')
    if None == presigned_url:
        return
    if False == isinstance(environment.runner, MasterRunner):
        return
    report = '/home/locust/report.tar.gz'
    tar = tarfile.open(report, 'w:gz')
    for item in glob.glob('/tmp/*.csv'):
        print('Adding %s...' % item)
        tar.add(item, arcname=os.path.basename(item))
    tar.close()
    request_headers = {'content-type': 'application/gzip'}
    r = requests.put(presigned_url, data=open(report, 'rb'), headers=request_headers)
```
