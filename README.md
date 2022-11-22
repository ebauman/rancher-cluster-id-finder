# Rancher Cluster ID Finder

This tool locates and outputs the cluster id of the cluster in which it is executed (or via the kubeconfig it is run with). 

It can optionally also output this value into a desired configmap.

## Usage

```
rancher-cluster-id-finder [-kubeconfig path] [-configmap-name name] [-configmap-namespace] [-configmap-key] [-debug]
```

## Building

```
go build -o rcidf
```