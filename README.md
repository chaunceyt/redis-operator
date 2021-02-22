# Redis Operator

A Golang based Redis operator that will make/manage Redis in standalone mode on top of the Kubernetes.

## Supported Features


* Redis standalone mode setup
* Enable Redis Modules:  [RedisGraph](https://oss.redislabs.com/redisgraph/), [RedisJSON](https://oss.redislabs.com/redisjson/) and/or [RedisSearch](https://oss.redislabs.com/redisearch/)
* Password/Password-less setup
* Resources restrictions with k8s requests
* SecurityContext to create immutable pods during runtime.

## TODO
* Storage provisioning (PVC) 
* Prometheus exporter
* Affinity and node selector



