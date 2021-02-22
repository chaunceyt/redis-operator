# Redis Operator

A Golang based Redis operator that will make/manage Redis in standalone mode on top of the Kubernetes.

## Supported Features


* Redis standalone mode setup
* Enable Redis Modules:  [RedisGraph](https://oss.redislabs.com/redisgraph/), [RedisJSON](https://oss.redislabs.com/redisjson/), [RedisBloom](https://oss.redislabs.com/redisbloom/), [RedisTimeSeries](https://github.com/RedisTimeSeries/RedisTimeSeries/blob/master/docs/index.md) and/or [RedisSearch](https://oss.redislabs.com/redisearch/)
* Password/Password-less setup
* Resources restrictions with k8s requests
* SecurityContext to create immutable pods during runtime.

## TODO
* Storage provisioning (PVC) 
* Prometheus exporter
* Affinity and node selector


## Usage


```
# Create a single node KinD cluster.
kind create cluster

# Install CRD
make install

# Deploy operator to cluster.
make deploy

# Create some redis custom resources.

# Deploy redis environment
kubectl apply -f config/samples/redis-as-a-graph-database.yaml
kubectl apply -f config/samples/redis-as-a-search-engine.yaml
kubectl apply -f config/samples/redis-as-a-cache-engine.yaml
```



