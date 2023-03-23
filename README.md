# kubecfg

[![Go](https://github.com/amimof/kubecfg/actions/workflows/go.yaml/badge.svg)](https://github.com/amimof/kubecfg/actions/workflows/go.yaml)

`kubecfg` is an interactive command line tool which lets you switch between [kubectl](kubectl.docs.kubernetes.io/) configuration files (known as kubeconfigs).

<img src="img/index.gif" alt="drawing" width="800"/>

## How it works
kubecfg will recursively find kubeconfigs in desired folder and list them in an interactive terminal. kubecfg uses symlinks to the standard kubectl kubeconfig location. Switching kubeconfigs will simply update the link to `~/.kube/config`.

## Usage

```
Usage:
  kubecfg [PATH] <flags>

kubecfg Kubernetes kubconfig manager

List, search and switch between multiple kubeconfig files within a directory

      --version   Print version
```