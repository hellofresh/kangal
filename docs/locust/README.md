# Locust

## Table of content
- [Configuring Locust resource requirements](#configuring-locust-resource-requirements)
- [Writing tests](#writing-tests)
- [Reporting](#reporting)

Locust is one of the load generators used in Kangal. An easy to use, scriptable and scalable performance testing tool. You define the behaviour of your users in regular Python code, instead of using a clunky UI or domain specific language. This makes Locust infinitely expandable and very developer friendly.

Currently Kangal uses the official docker image [locustio/locust](https://hub.docker.com/r/locustio/locust).

Kangal requires a py testfile describing the test.

Check it out [Locust official website](https://locust.io/).

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
