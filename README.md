# 3scale-operator

A Kubernetes Operator based on the Operator SDK for installing and syncing resources in 3Scale.

[![Build Status](https://travis-ci.org/integr8ly/3scale-operator.svg?branch=master)](https://travis-ci.org/integr8ly/3scale-operator)


|                 | Project Info  |
| --------------- | ------------- |
| License:        | Apache License, Version 2.0                      |
| IRC             | [#integreatly](https://webchat.freenode.net/?channels=integreatly) channel in the [freenode](http://freenode.net/) network. |


## Deploying

```sh
#create required resources
make install
#deploy the latest version of the operator
make deploy
#create example threescale custom resource
make create-examples
```

##Development 

```sh
make install run
```

You should see something like:

```go
operator-sdk up local --namespace=3scale --operator-flags="--resync=10 --log-level=debug"
INFO[0000] Go Version: go1.10.1                         
INFO[0000] Go OS/Arch: linux/amd64                      
INFO[0000] operator-sdk Version: 0.0.6                  
INFO[0000] 3Scale Version: 2.2.0.GA                     
INFO[0001] Watching 3scale.net/v1alpha1, ThreeScale, 3scale, 10000000000 
DEBU[0001] starting threescales controller
```

## Building

```sh
#builds image: quay.io/integreatly/3scale-operator:<tag>
make build
```

## Running tests

```
#Runs both gofmt checks and unit tests
make test
```