# Kangal Chart
[Kangal](https://github.com/hellofresh/kangal) is a tool to spin up an isolated environment in a Kubernetes cluster to run performance tests using different load test providers.

## Introduction
This chart bootstraps a [kangal](https://github.com/hellofresh/kangal) deployment using the [Helm](https://helm.sh) package manager.

## Prerequisites
- Helm 2+
- Kubernetes 1.12+

## Installing the Chart
To add the the repository to Helm:
```shell
$ helm repo add kangal https://hellofresh.github.io/kangal
```

To install the Custom Resource Definition:
```shell
$ kubectl apply -f https://raw.githubusercontent.com/hellofresh/kangal/master/charts/kangal/crd.yaml
```

To install the chart with the release name `kangal`:
```shell
$ helm install \
  --set environment=dev \
  kangal kangal/kangal
```

> for Helm v2:
> ```shell
> $ helm install \
>   --set environment=dev \
>   --name kangal kangal/kangal
> ```

To install the chart with the release name `kangal` and use an specific version:
```shell
$ helm install \
  --set environment=dev \
  --set proxy.image.tag=1.0.3 \
  --set controller.image.tag=1.0.3 \
  kangal kangal/kangal
```

> for Helm v2:
> ```shell
> $ helm install \
>   --set environment=dev \
>   --set proxy.image.tag=1.0.3 \
>   --set controller.image.tag=1.0.3 \
>   --name kangal kangal/kangal
> ```

The command deploys Kangal on the Kubernetes cluster in the default configuration.
It also applies the latest version of Custom Resource Definition (CRD) to the cluster.

> **Tip**: List all releases using `helm list`

## Uninstalling the Chart
To uninstall/delete the `kangal` deployment:

```shell
$ helm delete kangal
$ kubectl delete crd loadtests.kangal.hellofresh.com
```

## Configuration
To install Kangal to your infrastructure you need 3 deployments: Kangal-Proxy, Kangal-Controller and Kangal-openapi-UI

The following table lists the common configurable parameters for `Kangal` chart:

| Parameter                         | Description                                                                                         | Default                      |
|-----------------------------------|-----------------------------------------------------------------------------------------------------|------------------------------|
| `fullnameOverride`                | String to fully override kangal.fullname template with a string                                     | `nil`                        |
| `nameOverride`                    | String to partially override kangal.fullname template with a string (will prepend the release name) | `nil`                        |
| `configmap.AWS_ACCESS_KEY_ID`     | AWS access key ID. If not defined report will not be stored                                         | ``                           |
| `configmap.AWS_SECRET_ACCESS_KEY` | AWS secret access key                                                                               | ``                           |
| `configmap.AWS_BUCKET_NAME`       | The name of the bucket for saving reports                                                           | `my-bucket`                  |
| `configmap.AWS_ENDPOINT_URL`      | Storage connection parameter                                                                        | `s3.us-east-1.amazonaws.com` |
| `configmap.AWS_DEFAULT_REGION`    | Storage connection parameter                                                                        | `us-east-1`                  |

Deployment specific configurations:

### Kangal Proxy
| Parameter                               | Description                                           | Default                                |
|-----------------------------------------|-------------------------------------------------------|----------------------------------------|
| `proxy.image.repository`                | Repository of the image                               | `hellofreshtech/kangal`                |
| `proxy.image.tag`                       | Tag of the image                                      | `latest`                               |
| `proxy.image.pullPolicy`                | Pull policy of the image                              | `Always`                               |
| `proxy.args`                            | Argument for `kangal` command                         | `["proxy"]`                            |
| `proxy.replicaCount`                    | Number of pod replicas                                | `2`                                    |
| `proxy.service.enabled`                 | Service enabling flag                                 | `true`                                 |
| `proxy.service.enablePrometheus`        | ServiceMonitor for Prometheus enabled flag            | `false`                                |
| `proxy.service.type`                    | Service type                                          | `ClusterIP`                            |
| `proxy.service.ports.http`              | Service port                                          | `80`                                   |
| `proxy.ingress.enabled`                 | Ingress enabled flag                                  | `true`                                 |
| `proxy.ingress.annotations`             | Ingress annotations                                   | `kubernetes.io/ingress.class: "nginx"` |
| `proxy.ingress.path`                    | Ingress path                                          | `/`                                    |
| `proxy.ingress.hosts`                   | Ingress hosts. *Required* if ingress is enabled       | `kangal-proxy.local`                   |
| `proxy.resources`                       | CPU/Memory resource requests/limits                   | Default values of the cluster          |
| `proxy.nodeSelector`                    | Node labels for pod assignment                        | `{}`                                   |
| `proxy.tolerations`                     | Tolerations for nodes that have taints on them        | `[]`                                   |
| `proxy.affinity`                        | Pod scheduling preferences                            | `{}`                                   |
| `proxy.podAnnotations`                  | Annotation to be added to pod                         | `{}`                                   |
| `proxy.containerPorts.http`             | The ports that the container listens to               | `8080`                                 |
| `proxy.env.OPEN_API_SERVER_URL`         | *Required.* A URL to the OpenAPI specification server | `https://kangal-openapi.local`         |
| `proxy.env.OPEN_API_SERVER_DESCRIPTION` | *Required.* A Description to the OpenAPI server URL   | `Kangal proxy default value`           |
| `proxy.env.OPEN_API_UI_URL`             | A URL to the OpenAPI UI                               | `https://kangal-openapi-ui.local`      |

### OpenAPI UI
| Parameter                             | Description                                     | Default                                |
|---------------------------------------|-------------------------------------------------|----------------------------------------|
| `openapi-ui.enabled`                  | OpenAPI UI enabling flag                        | `true`                                 |
| `openapi-ui.image.repository`         | Repository of the image                         | `swaggerapi/swagger-ui`                |
| `openapi-ui.image.tag`                | Tag of the image                                | `latest`                               |
| `openapi-ui.image.pullPolicy`         | Pull policy of the image                        | `Always`                               |
| `openapi-ui.replicaCount`             | Number of pod replicas                          | `2`                                    |
| `openapi-ui.service.enabled`          | Service enabled flag                            | `true`                                 |
| `openapi-ui.service.enablePrometheus` | ServiceMonitor for Prometheus enabled flag      | `false`                                |
| `openapi-ui.service.type`             | Service type                                    | `ClusterIP`                            |
| `openapi-ui.service.ports.http`       | Service port                                    | `80`                                   |
| `openapi-ui.ingress.enabled`          | Ingress enabled flag                            | `true`                                 |
| `openapi-ui.ingress.annotations`      | Ingress annotations                             | `kubernetes.io/ingress.class: "nginx"` |
| `openapi-ui.ingress.path`             | Ingress path                                    | `/`                                    |
| `openapi-ui.ingress.hosts`            | Ingress hosts. *Required* if ingress is enabled | `kangal-openapi.local`                 |
| `openapi-ui.resources`                | CPU/Memory resource requests/limits             | Default values of the cluster          |
| `openapi-ui.nodeSelector`             | Node labels for pod assignment                  | `{}`                                   |
| `openapi-ui.tolerations`              | Tolerations for nodes that have taints on them  | `[]`                                   |
| `openapi-ui.affinity`                 | Pod scheduling preferences                      | `{}`                                   |
| `openapi-ui.podAnnotations`           | Annotation to be added to pod                   | `{}`                                   |
| `openapi-ui.containerPorts.http`      | The ports that the container listens to         | `8080`                                 |
| `openapi-ui.env.PORT`                 | The PORT of OpenAPI definition                  | `8080`                                 |
| `openapi-ui.env.URL`                  | The URL pointing to OpenAPI definition          | `https://kangal.local/openapi`         |
| `openapi-ui.env.VALIDATOR_URL`        | The URL to spec validator                       | `null`                                 |

### Kangal Controller
| Parameter                             | Description                                | Default                 |
|---------------------------------------|--------------------------------------------|-------------------------|
| `controller.image.repository`         | Repository of the image                    | `hellofreshtech/kangal` |
| `controller.image.tag`                | Tag of the image                           | `latest`                |
| `controller.image.pullPolicy`         | Pull policy of the image                   | `Always`                |
| `controller.args`                     | Argument for `kangal` command              | `["controller"]`        |
| `controller.replicaCount`             | Number of pod replicas                     | `1`                     |
| `controller.service.enabled`          | Service enabled flag                       | `true`                  |
| `controller.service.enablePrometheus` | ServiceMonitor for Prometheus enabled flag | `false`                 |
| `controller.service.type`             | Service type                               | `ClusterIP`             |
| `controller.service.ports.http`       | Service port                               | `80`                    |

### Kangal Controller (Locust specific)
| Parameter                                      | Description                 | Default           |
|------------------------------------------------|-----------------------------|-------------------|
| `controller.env.LOCUST_IMAGE`                  | Locust image                | `locustio/locust` |
| `controller.env.LOCUST_IMAGE_TAG`              | Locust image tag            | `1.3.0`           |
| `controller.env.LOCUST_MASTER_CPU_LIMITS`      | Master container CPU limits | ``                |
| `controller.env.LOCUST_MASTER_CPU_REQUESTS`    | Master CPU requests         | ``                |
| `controller.env.LOCUST_MASTER_MEMORY_LIMITS`   | Master memory limits        | ``                |
| `controller.env.LOCUST_MASTER_MEMORY_REQUESTS` | Master memory requests      | ``                |
| `controller.env.LOCUST_WORKER_CPU_LIMITS`      | Master container CPU limits | ``                |
| `controller.env.LOCUST_WORKER_CPU_REQUESTS`    | Master CPU requests         | ``                |
| `controller.env.LOCUST_WORKER_MEMORY_LIMITS`   | Master memory limits        | ``                |
| `controller.env.LOCUST_WORKER_MEMORY_REQUESTS` | Master memory requests      | ``                |
