# Kangal

## Table of content
- [Kangal user flow](Kangal-user-flow.md) 
- [Load generators](#load-generator-types-aka-backends)
    - [Requirements for new backends](#adding-a-new-load-generator)
    - [JMeter in Kangal](jmeter-in-kangal/JMeter-load-generator-in-kangal.md)
- [Reporting](#reporting-in-kangal)
- [Troubleshooting](Troubleshooting.md)

Welcome to the Kangal - Kubernetes and Go Automatic Loader!
To start using Kangal you will need to install it in your cluster. Installation guide using Helm can be found [here](https://github.com/hellofresh/kangal/blob/master/charts/kangal/README.md).
In this section you can find information about load generators and how to write tests.
    
## Load generator types aka backends
Currently, there are two load generator types implemented for Kangal:
- Fake - mock up provider used for testing controller logic. Not generating any load. Useful for debugging.

- JMeter - the first real load generator implemented for Kangal. Kangal creates JMeter load test environments based on [Kangal-JMeter](https://github.com/hellofresh/kangal-jmeter) docker image. 
JMeter is a powerful tool which can be used for different performance testing tasks. 
Please read [JMeter Load generator in Kangal](jmeter-in-kangal/JMeter-load-generator-in-kangal.md) for further details.

### Adding a new load generator
Kangal offers an opportunity to add different load generators as backends. 
Requirements to new load generators:
1. Create a docker image that must contain an executable of a new load generator and all required scripts to run it. Docker image should exit once load test is finished and it should provide logs to stdout which will be used by Kangal Proxy API.
2. Create a new backend resource definition in Kangal source code: 
 - [/pkg/backends](https://github.com/hellofresh/kangal/tree/master/pkg/backends). 
 - [backend.go](https://github.com/hellofresh/kangal/blob/master/pkg/backends/backend.go#L33)
 - [CRD definition](https://github.com/hellofresh/kangal/blob/master/charts/kangal/crd.yaml#L43)
 - [openapi.json](https://github.com/hellofresh/kangal/blob/master/openapi.json#L280)
The basic resource is a job that manages all the other resources and sets pods to the `finished` state when the test is over.

## Reporting in Kangal
Reporting is an important part of load testing process. It basically contains in two parts:
1. Live metrics during the running load test - Kangal proxy scrapes logs from main job stdout Docker container.
2. Solid report generated after the end of the test. 
Currently, Kangal relies on report creation implemented in backend. You can read more about JMeter backend implementation in [Reporting in JMeter](jmeter-in-kangal/Reporting-in-JMeter.md).

To persist the reports the backend receives a Pre-Signed URL where which can use to upload the report. If the report contains multiple files it will be necessary to archieve/compress into a single file.

Kangal also provides an API endpoint that allows to retrieve reports from storage bucket.

To allow Kangal to serve the report static files is necessary to explicit set the file as a `tar` archive with no compression and **no enclosing directory**, otherwise the endpoint will just force the report download.

The script below shows how to properly persist to the storage.

```sh
if [[ -n "${REPORT_PRESIGNED_URL}" ]]; then
  echo "=== Saving report to Object storage ==="
  tar -C /path/to/reports/ -cf /tmp/report-archive.tar .
  curl -X PUT -H "Content-Type: application/x-tar" -T /tmp/report-archive.tar -L "${REPORT_PRESIGNED_URL}"
fi
```
