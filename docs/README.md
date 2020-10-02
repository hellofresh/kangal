# Kangal

## Table of content
- [Load generators types (aka backends)](#load-generator-types-aka-backends)
- [Adding a new load generator](#adding-a-new-load-generator)
- [Reporting](#reporting)
- [Troubleshooting](troubleshooting.md)
- [User flow](user-flow.md) 

Welcome to the Kangal - **K**ubernetes **an**d **G**o **A**utomatic **L**oader!

For installation instructions, read the [Quickstart guide](/README.md#quickstart-guide) or the [Helm Chart](/charts/kangal/README.md).

In this section you can find information about load generators and how to write tests.
    
## Load generator types (aka backends)
Currently, there are two load generator types implemented for Kangal:

- **Fake** - Mock up provider used for testing purpouses, not generating any load.
- **JMeter** - Kangal creates JMeter load test environments based on [hellofreshtech/kangal-jmeter](https://github.com/hellofresh/kangal-jmeter) docker image.

### JMeter
JMeter is a powerful tool which can be used for different performance testing tasks.

Please read [docs/jmeter/README.md](jmeter/README.md) for further details.

## Adding a new load generator
Kangal can be easily extended by adding different load generators as backends. 

### Requirements for adding a new load generators

1. Create a docker image that must contain an executable of a new load generator and all required scripts to run it. Docker image should exit once load test is finished and it should provide logs to stdout which will be used by Kangal Proxy.

2. Create a new backend resource definition in Kangal source code: 
 - [pkg/backends/](/pkg/backends)
 - [backend.go](/pkg/backends/backend.go#L33)
 - [crd.yaml](/charts/kangal/crd.yaml#L43)
 - [openapi.json](/openapi.json#L280)

## Reporting
Reporting is an important part of load testing process. It basically contains in two parts:

1. Live metrics during the running load test, Kangal Proxy scrapes logs from main job stdout container.
2. Solid report generated after the end of the test. 

Kangal Proxy provides an API endpoint that allows to retrieve persisted reports (`/load-test/:name/report/`).

> Kangal relies on report creation to be implemented in the backend.

### Persisting reports
To persist reports backends receives a Pre-Signed URL where which can use to upload it.

> If the report contains multiple files it will be necessary to archieve/compress into a single file.

To allow Kangal to serve the report static files is necessary to explicit set the file as a `tar` archive with no compression and **no enclosing directory**, otherwise the endpoint will just force the report download.

The script below is an example on how to properly persist to the storage.

```sh
if [[ -n "${REPORT_PRESIGNED_URL}" ]]; then
  echo "=== Saving report to Object storage ==="
  tar -C /path/to/reports/ -cf /tmp/report-archive.tar .
  curl -X PUT -H "Content-Type: application/x-tar" -T /tmp/report-archive.tar -L "${REPORT_PRESIGNED_URL}"
fi
```
