## Quick Start guide
### Step 1. Deploy Kubernetes operator using all in one config file
```
kubectl apply -f https://raw.githubusercontent.com/martezr/morpheus-operator/main/deploy/all-in-one.yaml
```
### Step 2. Create vSphere Instance

**1.** Create a Morpheus credentials Secret

```
kubectl create secret generic morpheus-credentials \
         --from-literal="url=<the_morpheus_url>" \
         --from-literal="username=<the_morpheus_username>" \
         --from-literal="password=<the_morpheus_password>" \
         -n morpheus-system
```

**2.** Create an `AtlasProject` Custom Resource

The `AtlasProject` CustomResource represents Atlas Projects in our Kubernetes cluster. You need to specify 
`projectIpAccessList` with the IP addresses or CIDR blocks of any hosts that will connect to the Atlas Cluster.
```
cat <<EOF | kubectl apply -f -
apiVersion: atlas.mongodb.com/v1
kind: AtlasProject
metadata:
  name: my-project
spec:
  name: Test Atlas Operator Project
  projectIpAccessList:
    - ipAddress: "192.0.2.15"
      comment: "IP address for Application Server A"
    - ipAddress: "203.0.113.0/24"
      comment: "CIDR block for Application Server B - D"
EOF
```
**3.** Create an `AtlasCluster` Custom Resource.
The example below is a minimal configuration to create an M10 Atlas cluster in the AWS US East region. For a full list of properties, check
`atlasclusters.atlas.mongodb.com` [CRD specification](config/crd/bases/atlas.mongodb.com_atlasclusters.yaml)):
```
cat <<EOF | kubectl apply -f -
apiVersion: atlas.mongodb.com/v1
kind: AtlasCluster
metadata:
  name: my-atlas-cluster
spec:
  name: "Test-cluster"
  projectRef:
    name: my-project
  providerSettings:
    instanceSizeName: M10
    providerName: AWS
    regionName: US_EAST_1
EOF
```

**4.** Create a database user password Kubernetes Secret
```
kubectl create secret generic the-user-password --from-literal="password=P@@sword%"
```

**5.** Create an `AtlasDatabaseUser` Custom Resource

In order to connect to an Atlas Cluster the database user needs to be created. `AtlasDatabaseUser` resource should reference
the password Kubernetes Secret created in the previous step.
```
cat <<EOF | kubectl apply -f -
apiVersion: atlas.mongodb.com/v1
kind: AtlasDatabaseUser
metadata:
  name: my-database-user
spec:
  roles:
    - roleName: "readWriteAnyDatabase"
      databaseName: "admin"
  projectRef:
    name: my-project
  username: theuser
  passwordSecretRef:
    name: the-user-password
EOF
```
**6.** Wait for the `AtlasDatabaseUser` Custom Resource to be ready

Wait until the AtlasDatabaseUser resource gets to "ready" status (it will wait until the cluster is created that may take around 10 minutes):
```
kubectl get vsphereinstances kubedemo -o=jsonpath='{.status.conditions[?(@.type=="running")].status}'
True
```