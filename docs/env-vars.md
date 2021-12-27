# Kangal environment variables

## Proxy
| Parameter                       | Description                                          | Default                                    |
|---------------------------------|------------------------------------------------------|--------------------------------------------|
| `ALLOWED_CUSTOM_IMAGES`         | Allow custom Images to be defined in the request     | `false`                                    |
| `DEBUG`                         |        | |
| `KUBE_CLIENT_TIMEOUT`           | Timeout for each operation done by kube client       | `5s`                                       |
| `MAX_LIST_LIMIT`                | Output of LIST endpoint                              | 50                                         |
| `OPEN_API_SERVER_DESCRIPTION`   | Description to the OpenAPI server URL                | `Kangal proxy default value`               |
| `OPEN_API_SERVER_URL`           | URL to the OpenAPI specification server              | `https://kangal-proxy.example.com/openapi` |
| `OPEN_API_SPEC_PATH`            | Path to the openapi spec file                        | `/etc/kangal`                              |
| `OPEN_API_SPEC_FILE`            | Name of the openapi spec file                        | `openapi.json`                             |
| `OPEN_API_UI_URL`               | URL to the OpenAPI UI                                | `https://kangal-openapi-ui.example.com`    |
| `OPEN_API_CORS_ALLOW_ORIGIN`    | List of origins a cross-domain request can be executed from                    | `*`              |
| `OPEN_API_CORS_ALLOW_HEADERS`   | List of non simple headers client is allowed to use with cross-domain requests | `Content-Type,api_key,Authorization`    |
| `WEB_HTTP_PORT`                 |                                                      | `8080`                                     |

## Controller
| Parameter                       | Description                                       | Default             |
|---------------------------------|---------------------------------------------------|---------------------|
| `CLEANUP_THRESHOLD`             | Life time of a load test                          | `1h`                |
| `DEBUG`                         |        | |
| `KANGAL_PROXY_URL`              | Endpoints used to store load test reports         | `""`                |
| `KUBE_CLIENT_TIMEOUT`           | Timeout for each operation done by kube client    | `5s`                |
| `SYNC_HANDLER_TIMEOUT`          | Time limit for each sync operation                | `60s`               |
| `WEB_HTTP_PORT`                 |                                                   | `8080`              |

## Backend specific configuration
### JMeter
| Parameter                       | Description                          | Default                           |
|---------------------------------|--------------------------------------|-----------------------------------|
| `JMETER_MASTER_IMAGE_NAME`      | JMeter master image name/repository  | `hellofresh/kangal-jmeter-master` |
| `JMETER_MASTER_IMAGE_TAG`       | Tag of the JMeter master image above | `latest`                          |
| `JMETER_MASTER_CPU_LIMIT`       | Master CPU limit                     |                                   |
| `JMETER_MASTER_CPU_REQUESTS`    | Master CPU requests                  |                                   |
| `JMETER_MASTER_MEMORY_LIMITS`   | Master memory limits                 |                                   |
| `JMETER_MASTER_MEMORY_REQUESTS` | Master memory requests               |                                   |
| `JMETER_WORKER_IMAGE_NAME`      | JMeter worker image name/repository  | `hellofresh/kangal-jmeter-worker` |
| `JMETER_WORKER_IMAGE_TAG`       | Tag of the JMeter worker image above | `latest`                          |
| `JMETER_WORKER_CPU_LIMITS`      | Worker container CPU limits          |                                   |
| `JMETER_WORKER_CPU_REQUESTS`    | Worker CPU requests                  |                                   |
| `JMETER_WORKER_MEMORY_LIMITS`   | Worker memory limits                 |                                   |
| `JMETER_WORKER_MEMORY_REQUESTS` | Worker memory requests               |                                   |

### Locust
| Parameter                       | Description                 | Default           |
|---------------------------------|-----------------------------|-------------------|
| `LOCUST_IMAGE`                  | Locust image                |                   |
| `LOCUST_IMAGE_NAME`             | Locust image name           |                   |
| `LOCUST_IMAGE_TAG`              | Locust image tag            |                   |
| `LOCUST_MASTER_CPU_LIMITS`      | Master container CPU limits |                   |
| `LOCUST_MASTER_CPU_REQUESTS`    | Master CPU requests         |                   |
| `LOCUST_MASTER_MEMORY_LIMITS`   | Master memory limits        |                   |
| `LOCUST_MASTER_MEMORY_REQUESTS` | Master memory requests      |                   |
| `LOCUST_WORKER_CPU_LIMITS`      | Master container CPU limits |                   |
| `LOCUST_WORKER_CPU_REQUESTS`    | Master CPU requests         |                   |
| `LOCUST_WORKER_MEMORY_LIMITS`   | Master memory limits        |                   |
| `LOCUST_WORKER_MEMORY_REQUESTS` | Master memory requests      |                   |

### `ghz`
| Parameter                    | Description                         | Default                 |
|------------------------------|-------------------------------------|-------------------------|
| `GHZ_IMAGE_NAME`             | Default ghz image name/repository   | `hellofresh/kangal-ghz` |
| `GHZ_IMAGE_TAG`              | Tag of the ghz image above          | `latest`                |
| `GHZ_MASTER_CPU_LIMITS`      | CPU limits                          |                         |
| `GHZ_MASTER_CPU_REQUESTS`    | CPU requests                        |                         |
| `GHZ_MASTER_MEMORY_LIMITS`   | Memory limits                       |                         |
| `GHZ_MASTER_MEMORY_REQUESTS` | Memory requests                     |                         |

## Global config
| Parameter                  | Description                                                  | Default                               |
|----------------------------|--------------------------------------------------------------|---------------------------------------|
| `DEBUG`                    |   |                    |
| `LOG_LEVEL`                | Log level                                                    | `info`                                |
| `LOG_TYPE`                 | Log type                                                     | `kangal`                              |
| `AWS_ACCESS_KEY_ID`        | AWS access key ID. If not defined report will not be stored  | `my-access-key-id`                    |
| `AWS_BUCKET_NAME`          | The name of the bucket for saving reports                    | `my-bucket`                           |
| `AWS_DEFAULT_REGION`       | Storage connection parameter                                 | `us-east-1`                           |
| `AWS_ENDPOINT_URL`         | Storage connection parameter                                 | `s3.us-east-1.amazonaws.com`          |
| `AWS_SECRET_ACCESS_KEY`    | AWS secret access key                                        | `my-secret-access-key`                |

## Report config
| Parameter                  | Description                        | Default                               |
|----------------------------|------------------------------------|---------------------------------------|
| `AWS_PRESIGNED_EXPIRES`    | Expiration time for Presigned URLs | `30m`                                 |
| `AWS_USE_HTTPS`            | Set to "true" to use HTTPS         | `false`                               |

## Swagger
| Parameter              | Description                              | Default                                    |
|------------------------|------------------------------------------|--------------------------------------------|
| `PORT`                 | The PORT of OpenAPI definition           | `8080`                                     |
| `URL`                  | The URL pointing to OpenAPI definition   | `https://kangal-proxy.example.com/openapi` |
| `VALIDATOR_URL`        | The URL to spec validator                | `null`                                     |
| `KANGAL_PROXY_URL`     | Kangal Proxy URL used to persist reports | `https://kangal-proxy.example.com`         |
