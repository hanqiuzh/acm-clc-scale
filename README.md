# Cluster lifecycle scaling simulation tool

Goal: Create a simulation tool to help assess the performance of managedcluster-import-controller and klusterlet-addon-controllerr.

## How to build
To build the program, use the following command:
```
go build -o mock mockcluster/main.go 
```
which will generate a binary file locally in the project folder.

## How to use
To get help, after build, run `./mock`

All clusters created will have a label `mock-cluster: "true"`. 

### add clusters

Add 500 clusters name start with `mock-cluster-`, each time add 30 clusters, and sleep 10 seconds before next 30 clusters.
```
./mock add -p mock-cluster- --total 500 -c 30 -s 10
```

### delete clusters
Fire deletion of 500 clusters, name start with `mock-cluster-`, each time delete 30 clusters, sleep 10 seconds before next 30 clusters.
```
./mock delete -p mock-cluster- --total 500 -c 30 -s 10
```

### keep alive
keep all clusters start with `mock-cluster-`
```
./mock keepAlive -p mock-cluster-
```