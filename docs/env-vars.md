# Kangal environment variables

## Proxy
| Parameter                     | Description                                                                    | Default                                    |
|-------------------------------|--------------------------------------------------------------------------------|--------------------------------------------|
| `ALLOWED_CUSTOM_IMAGES`       | Allow custom Images to be defined in the request                               | `false`                                    |
| `KUBE_CLIENT_TIMEOUT`         | Timeout for each operation done by kube client                                 | `5s`                                       |
| `MAX_LIST_LIMIT`              | Output of LIST endpoint                                                        | `50`                                       |
| `OPEN_API_SERVER_DESCRIPTION` | Description to the OpenAPI server URL                                          | `Kangal proxy default value`               |
| `OPEN_API_SERVER_URL`         | URL to the OpenAPI specification server                                        | `https://kangal-proxy.example.com/openapi` |
| `OPEN_API_SPEC_PATH`          | Path to the openapi spec file                                                  | `/etc/kangal`                              |
| `OPEN_API_SPEC_FILE`          | Name of the openapi spec file                                                  | `openapi.json`                             |
| `OPEN_API_UI_URL`             | URL to the OpenAPI UI                                                          | `https://kangal-openapi-ui.example.com`    |
| `OPEN_API_CORS_ALLOW_ORIGIN`  | List of origins a cross-domain request can be executed from                    | `*`                                        |
| `OPEN_API_CORS_ALLOW_HEADERS` | List of non simple headers client is allowed to use with cross-domain requests | `Content-Type,api_key,Authorization`       |
| `WEB_HTTP_PORT`               |                                                                                | `8080`                                     |

## Controller
| Parameter                       | Description                                             | Default             |
|---------------------------------|---------------------------------------------------------|---------------------|
| `CLEANUP_THRESHOLD`             | Life time of a load test (disable by setting value to 0)| `1h`                |
| `KANGAL_PROXY_URL`              | Endpoints used to store load test reports               | `""`                |
| `KUBE_CLIENT_TIMEOUT`           | Timeout for each operation done by kube client          | `5s`                |
| `SYNC_HANDLER_TIMEOUT`          | Time limit for each sync operation                      | `60s`               |
| `WEB_HTTP_PORT`                 |                                                         | `8080`              |

## Backend specific configuration
### JMeter
| Parameter                                          | Description                                                              | Default                           |
|----------------------------------------------------|--------------------------------------------------------------------------|-----------------------------------|
| `JMETER_MASTER_IMAGE_NAME`                         | JMeter master image name/repository                                      | `hellofresh/kangal-jmeter-master` |
| `JMETER_MASTER_IMAGE_TAG`                          | Tag of the JMeter master image above                                     | `latest`                          |
| `JMETER_MASTER_CPU_LIMIT`                          | Master CPU limit                                                         |                                   |
| `JMETER_MASTER_CPU_REQUESTS`                       | Master CPU requests                                                      |                                   |
| `JMETER_MASTER_MEMORY_LIMITS`                      | Master memory limits                                                     |                                   |
| `JMETER_MASTER_MEMORY_REQUESTS`                    | Master memory requests                                                   |                                   |
| `JMETER_WORKER_IMAGE_NAME`                         | JMeter worker image name/repository                                      | `hellofresh/kangal-jmeter-worker` |
| `JMETER_WORKER_IMAGE_TAG`                          | Tag of the JMeter worker image above                                     | `latest`                          |
| `JMETER_WORKER_CPU_LIMITS`                         | Worker container CPU limits                                              |                                   |
| `JMETER_WORKER_CPU_REQUESTS`                       | Worker CPU requests                                                      |                                   |
| `JMETER_WORKER_MEMORY_LIMITS`                      | Worker memory limits                                                     |                                   |
| `JMETER_WORKER_MEMORY_REQUESTS`                    | Worker memory requests                                                   |                                   |
| `JMETER_WORKER_REMOTE_CUSTOM_DATA_ENABLED`         | Enable remote custom data                                                | `false`                           |
| `JMETER_WORKER_REMOTE_CUSTOM_DATA_BUCKET`          | The name of the bucket where remote data is                              |                                   |
| `JMETER_WORKER_REMOTE_CUSTOM_DATA_VOLUME_SIZE`     | Volume size used by download remote data                                 | `1Gi`                             |
| `RCLONE_CONFIG_REMOTECUSTOMDATA_TYPE`              | [Rclone](https://rclone.org/) environment variable for type              |                                   |
| `RCLONE_CONFIG_REMOTECUSTOMDATA_ACCESS_KEY_ID`     | [Rclone](https://rclone.org/) environment variable for access key ID     |                                   |
| `RCLONE_CONFIG_REMOTECUSTOMDATA_SECRET_ACCESS_KEY` | [Rclone](https://rclone.org/) environment variable for secret access key |                                   |
| `RCLONE_CONFIG_REMOTECUSTOMDATA_REGION`            | [Rclone](https://rclone.org/) environment variable for region            |                                   |
| `RCLONE_CONFIG_REMOTECUSTOMDATA_ENDPOINT`          | [Rclone](https://rclone.org/) environment variable for endpoint          |                                   |

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

### k6
| Parameter            | Description     | Default         |
|----------------------|-----------------|-----------------|
| `K6_IMAGE_NAME`      | K6 image name   | `loadimpact/k6` |
| `K6_IMAGE_TAG`       | K6 image tag    | `latest`        |
| `K6_CPU_LIMITS`      | CPU limits      |                 |
| `K6_CPU_REQUESTS`    | CPU requests    |                 |
| `K6_MEMORY_LIMITS`   | Memory limits   |                 |
| `K6_MEMORY_REQUESTS` | Memory requests |                 |

## Logger config
| Parameter                  | Description                                                  | Default                               |
|----------------------------|--------------------------------------------------------------|---------------------------------------|
| `LOG_LEVEL`                | Log level                                                    | `info`                                |
| `LOG_TYPE`                 | Log type                                                     | `kangal`                              |

## Report config
| Parameter                  | Description                                                  | Default   |
|----------------------------|--------------------------------------------------------------|-----------|
| `AWS_ACCESS_KEY_ID`        | AWS access key ID. If not defined report will not be stored  |           |
| `AWS_BUCKET_NAME`          | The name of the bucket for saving reports                    |           |
| `AWS_DEFAULT_REGION`       | Storage connection parameter                                 |           |
| `AWS_ENDPOINT_URL`         | Storage connection parameter                                 |           |
| `AWS_PRESIGNED_EXPIRES`    | Expiration time for Presigned URLs                           |           |
| `AWS_SECRET_ACCESS_KEY`    | AWS secret access key                                        |           |
| `AWS_USE_HTTPS`            | Set to "true" to use HTTPS                                   | `false`   |

## Swagger
| Parameter              | Description                              | Default                                    |
|------------------------|------------------------------------------|--------------------------------------------|
| `PORT`                 | The PORT of OpenAPI definition           | `8080`                                     |
| `URL`                  | The URL pointing to OpenAPI definition   | `https://kangal-proxy.example.com/openapi` |
| `VALIDATOR_URL`        | The URL to spec validator                | `null`                                     |
| `KANGAL_PROXY_URL`     | Kangal Proxy URL used to persist reports | `https://kangal-proxy.example.com`         |
