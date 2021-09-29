# Rancherd

Rancherd bootstraps a node with Kubernetes (k3s/rke2) and Rancher such
that all future management of Kubernetes and Rancher can be done from
Kubernetes. Rancherd will only run once per node. Once the system has
been fully bootstrapped it will not run again. It is intended that the
primary use of Rancherd is to be ran from cloud-init or a similar system.

## Quick Start

To create a three node cluster run the following on servers named `server1`,
`server2`, and `server3`.

On `server1`
```bash
mkdir -p /etc/rancher/rancherd
cat > /etc/rancher/rancherd/config.yaml << EOF
role: cluster-init
token: somethingrandom
EOF
curl -fL https://raw.githubusercontent.com/rancher/rancherd/master/install.sh | sh -
```

On `server2`
```bash
mkdir -p /etc/rancher/rancherd
cat > /etc/rancher/rancherd/config.yaml << EOF
role: server
server: https://server1:8443
token: somethingrandom
EOF
curl -fL https://raw.githubusercontent.com/rancher/rancherd/master/install.sh | sh -
```

On `server3`
```bash
mkdir -p /etc/rancher/rancherd
cat > /etc/rancher/rancherd/config.yaml << EOF
role: server
server: https://server1:8443
token: somethingrandom
EOF
curl -fL https://raw.githubusercontent.com/rancher/rancherd/master/install.sh | sh -
```

## Installation

### cloud-init

The primary way of running Rancherd is intended to be done from.
Add to your cloud-init the following for a single node cluster. All
configuration that would be found in the rancherd config.yaml should
be embedded in the `rancherd` key in the cloud-config.

```yaml
#cloud-config
rancherd:
  role: cluster-init
runcmd:
  - curl -fL https://raw.githubusercontent.com/rancher/rancherd/master/install.sh | sh -
```

### Manual

`rancherd` binary can be downloaded from https://github.com/rancher/rancherd/releases/latest
and manually ran.

### Curl script (systemd installation)

The below command will download `rancherd` binary and setup a systemd unit and run it.

```bash
curl -sfL https://https://raw.githubusercontent.com/rancher/rancherd/master/install.sh | sh -
```
 
## Cluster Initialization

Creating a cluster always starts with one node initializing the cluster, by
assigning the `cluster-init` role and then other nodes joining to the cluster.
The new cluster will have a token generated for it or you can manually
assign a unique string.  The token for an existing cluster can be determined
by running `rancherd get-token`.

## Joining Nodes

Nodes can be joined to the cluster as the role `server` to add more control
plane nodes or as the role `agent` to add more worker nodes. To join a node
you must have the Rancher server URL (which is by default running on port
`8443`) and the token.

## Node Roles


Rancherd will bootstrap a node with one of the following roles

1. __cluster-init__: Every cluster must start with one node that has the
    cluster-init role.
2. __server__: Joins the cluster as a new control-plane,etcd,worker node
3. __agent__: Joins the cluster as a worker only node.

## Server discovery

It can be quite cumbersome to automate bringing up a clustered system
that requires one bootstrap node.  Also there are more considerations
around load balancing and replacing nodes in a proper production setup.
Rancherd support server discovery based on https://github.com/hashicorp/go-discover.

When using server discovery the `cluster-init` role is not used, only `server`
and `agent`. The `server` URL is also dropped in place of using the `discovery`
key. The `discovery` configuration will be used to dynamically determine what
is the server URL and if the current node should act as the `cluster-init` node.

Example
```yaml
role: server
discovery:
  params:
    # Corresponds to go-discover provider name
    provider: "mdns"
    # All other key/values are parameters corresponding to what 
    # the go-discover provider is expecting
    service: "rancher-server"
  # If this is a new cluster it will wait until 3 server are 
  # available and they all agree on the same cluster-init node
  expectedServers: 3
  # How long servers are remembered for. It is useful for providers
  # that are not consistent in their responses, like mdns.
  serverCacheDuration: 1m
```
More information on how to use the discovery is in the config examples.

## Configuration

Configuration for rancherd goes in `/etc/rancher/rancherd/config.yaml`.  A full
example configuration with documentation is available in 
[config-example.yaml](./config-example.yaml).

Minimal configuration
```yaml
# /etc/rancher/rancherd/config.yaml

# role: Valid values cluster-init, server, agent
role: cluster-init

# token: A shared secret known by all clusters in the system
token: somethingrandom

# server: The server URL to join a cluster to. By default port 8443.
#         Only valid for roles server and agent, not cluster-init
server: https://example.com:8443
```

### Version Channels

The `kubernetesVersion` and `rancherVersion` accept channel names instead of explict versions.

Valid `kubernetesVersion` channels are as follows:

| Channel Name | Description |
|--------------|-------------|
|  stable | k3s stable (default value of kubernetesVersion) |
| latest | k3s latest |
| testing | k3s test |
|  stable:k3s | Same as stable channel |
| latest:k3s | Same as latest channel |
| testing:k3s | Same as testing channel |
|  stable:rke2 | rke2 stable |
| latest:rke2 | rke2 latest |
| testing:rke2 | rke2 testing |
| v1.21 | Latest k3s v1.21 release. The applies to any Kubernetes minor version |
| v1.21:rke2 | Latest rke2 v1.21 release. The applies to any Kubernetes minor version |

Valid `rancherVersions` channels are as follows:

| Channel Name | Description |
|--------------|-------------|
|  stable | [stable helm repo](https://releases.rancher.com/server-charts/stable/index.yaml) (default value of rancherVersion) |
| latest | [latest helm repo](https://releases.rancher.com/server-charts/latest/index.yaml) |

### Rancher Config

By default Rancher is installed with the following values.yaml.  You can override
any of these settings with the `rancherValues` setting in the rancherd `config.yaml`
```yaml
# Multi-Cluster Management is disabled by default, change to multi-cluster-management=true to enable
features: multi-cluster-management=false

# The Rancher UI will run on the host port 8443 by default. Set to 0 to disable
# and instead use ingress.enabled=true to route traffic through ingress
hostPort: 8443

# Accessing ingress is disabled by default.
ingress:
  enabled: false
  
# Don't create a default admin password
noDefaultAdmin: true

# The negative value means it will up to that many replicas if there are
# at least that many nodes available.  For example, if you have 2 nodes and
# `replicas` is `-3` then 2 replicas will run.  Once you add a third node
# a then 3 replicas will run
replicas: -3

# External TLS is assumed
tls: external
```

A full reference of all parameters in the values.yaml is available in
the [Rancher repo](https://github.com/rancher/rancher/blob/release/v2.6/chart/values.yaml).

## Dashboard/UI

The Rancher UI is running by default on port `:8443`.  There is no default
`admin` user password set.  You must run `rancherd reset-admin` once to
get an `admin` password to login.

## Multi-Cluster Management

By default Multi Cluster Managmement is disables in Rancher.  To enable set the
following in the rancherd config.yaml
```yaml
rancherValues:
  features: multi-cluster-management=true
```

## Upgrading

rancherd itself doesn't need to be upgraded. It is only ran once per node
and then after that provides no value.  What you do need to upgrade after
the fact is Rancher and Kubernetes. 

### Rancher
Rancher is installed as a helm chart following the standard procedure. You can upgrade 
Rancher with the standard procedure documented at
https://rancher.com/docs/rancher/v2.6/en/installation/install-rancher-on-k8s/upgrades/.

### Kubernetes
To upgrade Kubernetes you will use Rancher to orchestrate the upgrade. This is a matter of changing
the Kubernetes version on the `fleet-local/local` `Cluster` in the `provisioning.cattle.io/v1`
apiVersion.  For example

```shell
kubectl edit clusters.provisioning.cattle.io -n fleet-local local
```
```yaml
apiVersion: provisioning.cattle.io/v1
kind: Cluster
metadata:
  name: local
  namespace: fleet-local
spec:
  # Change to new valid k8s version
  kubernetesVersion: v1.21.4+k3s1
```

### Automated

You can also use the `rancherd upgrade` command on a `server` node to automatically do the
above procedure.
