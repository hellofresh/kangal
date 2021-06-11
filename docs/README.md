# Kangal

## Table of content
- [Load generators types (aka backends)](#load-generator-types-aka-backends)
- [User flow](user-flow.md)
- [Adding a new load generator](#adding-a-new-load-generator)
- [Reporting](#reporting)
- [Developer guide](#developer-guide)
- [Troubleshooting](troubleshooting.md)

Welcome to the Kangal - **K**ubernetes **an**d **G**o **A**utomatic **L**oader!

For installation instructions, read the [Quickstart guide](/README.md#quickstart-guide) or the [Helm Chart](/charts/kangal/README.md).

In this section you can find information about load generators and how to write tests.

## Load generator types (aka backends)
Currently, there are the following load generator types implemented for Kangal:

- **Fake** - Mock up provider used for testing purpouses, not generating any load.
- **JMeter** - Kangal creates JMeter load test environments based on [hellofresh/kangal-jmeter](https://github.com/hellofresh/kangal-jmeter) docker image.
- **Locust** - Kangal creates Locust load test environments based on official docker image [locustio/locust](https://hub.docker.com/r/locustio/locust).
- **`ghz`** - Kangal creates `ghz` load test environments using [hellofresh/kangal-ghz](https://github.com/hellofresh/kangal-ghz) docker image.

### JMeter
JMeter is a powerful tool which can be used for different performance testing tasks.

Please read [docs/jmeter/README.md](jmeter/README.md) for further details.

### Locust
Locust is an easy to use, scriptable and scalable performance testing tool. You define the behaviour of your users in regular Python code, instead of using a clunky UI or domain specific language. This makes Locust infinitely expandable and very developer friendly.

Please read [docs/locust/README.md](locust/README.md) for further details.

### `ghz`
`ghz` is a gRPC benchmarking and load testing tool.

Please read [docs/ghz/README.md](ghz/README.md) for further details.

## User flow
Read more at [docs/user-flow.md](user-flow.md).

## Adding a new load generator
Kangal can be easily extended by adding different load generators as backends.

### Requirements for adding a new load generators
1. Create a docker image that must contain an executable of a new load generator and all required scripts to run it. Docker image should exit once load test is finished and it should provide logs to stdout which will be used by Kangal Proxy.

2. Create a new backend resource definition in Kangal source code:
 - [main.go](/main.go)
 - [pkg/backends/](/pkg/backends)
 - [charts/kangal/crds/loadtest.yaml](/charts/kangal/crds/loadtest.yaml#L43)
 - [openapi.json](/openapi.json#L280)

## Reporting
Reporting is an important part of load testing process. It basically contains in two parts:

1. Live metrics during the running load test, Kangal Proxy scrapes logs from main job stdout container.
2. Solid report generated after the end of the test.

Kangal Proxy provides an API endpoint that allows to retrieve persisted reports (`/load-test/:name/report/`).

> Kangal relies on report creation to be implemented in the backend.

### Persisting reports
Kangal generates a Pre-Signed URL and backend can use it to persist a report.

> If the report contains multiple files it will be necessary to archieve/compress into a single file.

To allow Kangal to serve the report static files it is necessary to explicitly set the file as a `tar` archive with no compression and **no enclosing directory**, otherwise, the endpoint will just force the report download.

The script below is an example of how to properly persist to the storage.

```sh
if [[ -n "${REPORT_PRESIGNED_URL}" ]]; then
  echo "=== Saving report to Object storage ==="
  tar -C /path/to/reports/ -cf /tmp/report-archive.tar .
  curl -X PUT -H "Content-Type: application/x-tar" -T /tmp/report-archive.tar -L "${REPORT_PRESIGNED_URL}"
fi
```

## Developer guide
To start developing Kangal you need a local Kubernetes environment, e.g. [minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/) or [docker desktop](https://www.docker.com/products/docker-desktop).
> Note: Depending on load generator type, load test environments created by Kangal may require a lot of resources. Make sure you increased your limits for local Kubernetes cluster.

### 1. Clone the repo locally

```bash
git clone https://github.com/hellofresh/kangal.git
cd kangal
```

### 2. Create required Kubernetes resource LoadTest CRD in your cluster

```bash
kubectl apply -f charts/kangal/crds/loadtest.yaml
```

or just use:

```bash
make apply-crd
```

### 3 Get project dependencies

```bash
go mod vendor
```

### 4. Build Kangal binary

```bash
make build
```

### 5. Set the environment variables

``` bash
export AWS_BUCKET_NAME=YOUR_BUCKET_NAME       # name of the bucket for saving reports
export AWS_ENDPOINT_URL=YOUR_BUCKET_ENDPOINT  # storage connection parameter
export AWS_DEFAULT_REGION=YOUR_AWS_REGION     # storage connection parameter
export KANGAL_PROXY_URL=http://localhost:8080 # used to persist reports
```

### 6. Run both Kangal proxy and controller

```bash
WEB_HTTP_PORT=8888 ./kangal controller --kubeconfig=$KUBECONFIG
WEB_HTTP_PORT=8080 ./kangal proxy --kubeconfig=$KUBECONFIG
```

## Troubleshooting

Read more at [docs/troubleshooting.md](troubleshooting.md).
