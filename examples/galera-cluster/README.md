

## Galera Cluster for MySQL with StatefulSet 

This example describes how to run a Galera cluster with StatefulSet on Kubernetes. 

### Information
Galera cluster is a synchronous multi-master database cluster based on synchronous replication. So We can read and write at any node in the cluster. It means that we can use a Kubernetes Service to cover all nodes.

Galera cluster need to identify the most advanced node in the cluster. We start the cluster with the most advanced node and add other nodes to the cluster after the first node starting successfully. 

StatefulSet of Kubernetes provides stable hostnames which are available in DNS. We use these hostnames to detect whether the cluster is alive. 

### Mechanism
When a node started, it has a hostname like `container_name-0` , `container_name-1` and so on. Client can connect the cluster with domain name `container_name-{index}.service_name` (In this example, `container_name` and `service_name` are `mysql`).

First the node gets the index from hostname.

Then it detects all nodes in the cluster (Addresses of nodes are defined in `GALERA_CLUSTER_ADDRESS`). 

If there was a alive node, current node join in it.

If current node can't find any alive node, it sleeps `GALERA_START_DELAY` seconds and does another try. It tries `{index} + 1` times in total.

If there was no alive node at last, current node starts a new cluster and let itself as the most advanced node. Then waiting for other nodes joining in.

### Prerequisites
This example assume that you have a running Kubernetes (Version 1.5 or later). Earlier versions of Kubernetes don't support StatefulSet.

If you were familiar with MySQL and Galera, you can configure `my.cnf` and `galera.cnf`. Configs of MySQL and Galera won't be explained in this example.

If you want a `init.sql` for the cluster, you can add a line in Dockerfile:
```dockerfile
COPY path_to_init.sql /init.sql
```

You should build a image from Dockerfile in current directory. 
```
$ docker build -t galera:5.6 .
```
After building, you can upload the image to any plcae which your Kubernetes can pull it (e.g. Docker Hub). 

### Configuration
All configs in `set.yaml`. There are some explanations:
- Number of replicas: We use 3 nodes as default. You can use more nodes by setting the number of replicas.
- Name of Image: We build and tag the galera image with name `galera:5.6`. **You should replace it with real path of image**.
- MYSQL_ROOT_PASSWORD: Password of root user of MySQL.
- MYSQL_DATABASE: Name of Database. It will be created when the cluster starts up at the first time. 
- MYSQL_USER: User of `MYSQL_DATABASE` .
- MYSQL_PASSWORD: Password of `MYSQL_USER` .
- GALERA_USER: User of Galera cluster. The cluster use this user to sync database.
- GALERA_PASSWORD: Password of `GALERA_USER` .
- GALERA_CLUSTER_NAME: Name of cluster.
- GALERA_CLUSTER_ADDRESS: Addresses of all nodes and use comma as delimiter. Address can be IP address or domain name.
- GALERA_START_DELAY: Delay `GALERA_START_DELAY` seconds when it's failed to detect the cluster. 
- Capacity of database: Default is 10Gi. Set the capacity you need. As the same time, you need to set the `storage` in `pv.yaml` .

We store all data of mysql in `hostPath`. If you want to migrate to other place, you need change these configs in `pv.yaml`:
```yaml
spec:
  capacity:
    # same with storage in set.yaml
    storage: 10Gi
  accessModes:
    - ReadWriteOnce
  persistentVolumeReclaimPolicy: Retain
  # change to other way 
  hostPath:
    path: /data/db1
```

NOTE: The number of replicas, number of addresses in `GALERA_CLUSTER_ADDRESS` and number of PV should be same.


### Create a cluster
Create a cluster with `kubectl`:
```
$ kubectl apply -f pv.yaml
$ kubectl apply -f set.yaml
```

Then you can connect these nodes by domain name: `mysql-0.mysql` , `mysql-1.mysql` , `mysql-2.mysql` , `mysql`.

The last domain name `mysql` uses round-robin DNS. It can provide a basic load balancing.
