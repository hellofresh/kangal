# Kangal - Automatic loader
[![Artifact HUB](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/kangal)](https://artifacthub.io/packages/search?repo=kangal)
[![codecov](https://codecov.io/gh/hellofresh/kangal/branch/master/graph/badge.svg)](https://codecov.io/gh/hellofresh/kangal)
<p align="center">  
<img src="./kangal_logo.svg" width="320">
</p>

Run performance tests in Kubernetes cluster with Kangal.
___

## Table of content
- [Why Kangal?](#why-kangal)
- [Key features](#key-features)
- [How it works](#how-it-works)
- [Architectural diagram](#architectural-diagram)
- [Components](#components)
    - [LoadTest Custom Resource](#loadtest-custom-resource)
    - [Kangal Proxy](#kangal-proxy)
    - [Kangal Controller](#kangal-controller)
- [Quickstart Guide](#quickstart-guide)
    - [Installing using helm](#installing-using-helm)
    - [Installing using helm template](#installing-using-helm-template)
    - [Creating first LoadTest](#creating-first-loadtest)
- [Developing Kangal](#developing-kangal)
- [Contributing](#contributing)
- [Support](#support)

## Why Kangal?
In Kangal project, the name stands for "**K**ubernetes **an**d **G**o **A**utomatic **L**oader".
But originally Kangal is the breed of a shepherd dog. Let the smart and protective dog herd your load testing projects.

With Kangal, you can spin up an isolated environment in a Kubernetes cluster to run performance tests using different load generators.

## Key features
- **create** an isolated Kubernetes environment with an opinionated load generator installation
- **run** load tests against any desired environment
- **monitor** load tests metrics in Grafana
- **save the report** for the successful load test
- **clean up** after the test finishes

## How it works
Kangal application uses Kubernetes [Custom Resources](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/).

LoadTest custom resource (CR) is a main working entity.
LoadTest custom resource definition (CRD) can be found in [charts/kangal/crd.yaml](charts/kangal/crd.yaml).

Kangal application contains two main parts:
 - **Proxy** to create, delete and check load tests and reports via REST API requests
 - **Controller** to operate with LoadTest custom resource and other Kubernetes entities.

Kangal also uses S3 compatible storage to save test reports. 

## Architectural diagram
The diagram below illustrates the workflow for Kangal in Kubernetes infrastructure.

<p align="left">  
 <a href="architectural_diagram.png">
   <img alt="Architectural diagram" src="architectural_diagram.png" >
 </a>
</p>

## Components
### LoadTest Custom Resource
A new custom resource in the Kubernetes cluster which contains requirements for performance testing environments.

More info about the Custom Resources in [Official Kubernetes documentation](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)

### Kangal Proxy
Provides the following HTTP methods for `/load-test` endpoint:
 - POST - allowing the user to create a new LoadTest
 - GET - allowing the user to see current LoadTest status / logs / report / metrics
 - DELETE - allowing the user to stop and delete existing LoadTest

 The Kangal Proxy is documented using the [OpenAPI Spec](https://swagger.io/specification/).

 If you prefer to use Postman you can also import [openapi.json](openapi.json) file into Postman to create a new collection.

### Kangal Controller
The general name for several Kubernetes controllers created to manage all the aspects of the performance testing process.
 - LoadTest controller  
 - Backend jobs controller
 
## Quickstart Guide

### Installing using helm

**TODO** Short tutorial explaining how to install using helm

### Installing using helm template

**TODO** Short tutorial explaining how to install using helm template command and applying manually

### Creating first LoadTest

**TODO** Short tutorial explaining how to create first load test with examples

## Developing Kangal
To start developing Kangal you need a local Kubernetes environment, e.g. [minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/)
or [docker desktop](https://rominirani.com/tutorial-getting-started-with-kubernetes-with-docker-on-mac-7f58467203fd). 
> Note: Depending on load generator type, load test environments created by Kangal may require a lot of resources. Make sure you increased your limits for local Kubernetes cluster. 
> Read more about implemented load generators [here](docs/README.md#load-generator-types-aka-backends). 

1. Clone the repo locally
    ```bash
    git clone https://github.com/hellofresh/kangal.git
    ```

2. Create required Kubernetes resource LoadTest CRD in your cluster
    ```bash
    kubectl apply -f charts/kangal/crd.yaml
    ```
    or just use:
    ```bash
    make appply-crd
    ```
    
3. Download the dependencies
    ```bash
    go mod vendor
    ```

4. Build Kangal binary
   ```bash
   make build
    ```
    
5. Set the environment variables
    ``` bash
    export WEB_HTTP_PORT=8080                    # API port for Kangal Proxy
    export AWS_BUCKET_NAME=YOUR_BUCKET_NAME      # name of the bucket for saving reports
    export AWS_ENDPOINT_URL=YOUR_BUCKET_ENDPOINT # storage connection parameter
    export AWS_DEFAULT_REGION=YOUR_AWS_REGION    # storage connection parameter
    ```

6. Run both Kangal proxy and controller
    ```bash
    ./kangal controller --kubeconfig=$KUBECONFIG 
    ./kangal proxy --kubeconfig=$KUBECONFIG
    ```

## Contributing
To start contributing, please check [CONTRIBUTING](CONTRIBUTING.md).

## Support
If you need support, start with the [Troubleshooting guide](docs/Troubleshooting.md), and work your way through the process that we've outlined.
