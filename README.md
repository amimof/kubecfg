# kubecfg

[![Go](https://github.com/amimof/kubecfg/actions/workflows/go.yaml/badge.svg)](https://github.com/amimof/kubecfg/actions/workflows/go.yaml)

`kubecfg` is a command line tool that uses `fzf` to find and switch [kubectl](kubectl.docs.kubernetes.io/) configuration files (known as kubeconfigs).

## How it works

kubecfg will recursively find kubeconfigs in desired folder and list them in an interactive terminal. kubecfg uses symlinks to the standard kubectl kubeconfig location. Switching kubeconfigs will simply update the link to `~/.kube/config`.

## Usage

```
Usage:
  kubecfg [PATH] <flags>

kubecfg Kubernetes kubconfig manager

List, search and switch between multiple kubeconfig files within a directory

  -d, --dir string     The symlink kubeconfig (default "~/.kube/config")
  -g, --glob strings   List files matching a pattern to include. This flag can be used multiple times. (default [~/.kube/*.yaml])
      --version        Print version
```

