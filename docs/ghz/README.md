# `ghz`

[`ghz`] is a gRPC benchmarking and load testing tool.  

Kangal supports `ghz` as a loadtest backend using a custom docker image: [`hellofresh/kangal-ghz`][kangal-ghz] published to [dockerhub].

For more information, please check [ghz official website][`ghz`].

## Usage
To create a loadtest, simply send a request to Kangal proxy with the `ghz` configuration in the `testFile` field in JSON:

```shell
$ curl -X POST http://${KANGAL_PROXY_ADDRESS}/load-test \
  -H 'Content-Type: multipart/form-data' \
  -F distributedPods=1 \
  -F testFile=@config.json \
  -F type=Ghz \
  -F targetURL=http://my-app.example.com
```

Example `config.json`:

```json
{
  "call": "helloworld.Greeter.SayHello",
  "total": 2000,
  "concurrency": 50,
  "data": {
    "name": "Joe"
  },
  "metadata": {
    "foo": "bar",
    "trace_id": "{{.RequestNumber}}",
    "timestamp": "{{.TimestampUnix}}"
  },
  "max-duration": "10s",
  "host": "0.0.0.0:50051"
}
```

### Providing protobuf schema

While `ghz` accepts `.proto` files to not depend on server reflection, Kangal currently only supports `.protoset` files.

To do so, use the `testData` form field to provide the `.protoset` file and add the following key to your JSON configuration file:

```json
{
  "protoset": "/data/testdata.protoset",
  ...
}
```

For information about how to [create `.protoset` files][ghz protoset-example] and the complete list of configuration parameter, please check [ghz documentation][ghz params].

Since `ghz` does not use master-worker pattern, `distributedPods` simply creates replicas of the load-generating pod.  
This means `distributedPods` value of `5` would mean that it creates 5 identical pods, generating 5x the load with 5x concurrency, etc.


## Configuring resource limits and requirements
By default, Kangal does not specify resource requirements for loadtests run with `ghz` backend.

You can specify resource limits and requests for the containers to ensure that when the load generator runs, it has sufficient resources and will not fail before the service does.

The following environment variables can be specified to configure this parameter:

```
GHZ_CPU_LIMITS
GHZ_CPU_REQUESTS
GHZ_MEMORY_LIMITS
GHZ_MEMORY_REQUESTS
```

You have to specify these variables on Kangal Controller, read more at [charts/kangal/README.md](/charts/kangal/README.md#kangal-controller-ghz-specific).

More information regarding resource limits and requests can be found in the following page(s):
- https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
- https://cloud.google.com/blog/products/gcp/kubernetes-best-practices-resource-requests-and-limits


## Notes
1. `ghz` supports configuration file in `toml` format, but this is also currently not supported
2. Kangal overrides the following options:
  * The output format is always set to html
  * Output directory is always set to `/results`
  * This is done so Kangal is able to pick up the results and persist the results.  
  * Because they are set as container arguments, this cannot be overridden with `config.json`.


[`ghz`]: https://ghz.sh/
[ghz params]: https://ghz.sh/docs/options
[ghz protoset-example]: https://ghz.sh/docs/options#--protoset
[kangal-ghz]: https://github.com/hellofresh/kangal-ghz
[dockerhub]: hub.docker.com/r/hellofresh/kangal-ghz/
