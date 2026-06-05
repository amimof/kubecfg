# Using kubecfg with VMware Tanzu

VMware Tanzu clusters often require a distribution-specific login flow before `kubectl` can access a workload cluster through `kubectl-vsphere`. `kubecfg` supports this by running a kubeconfig-level login source and importing the referenced context from the temporary kubeconfig that login command produces.

## Configuration

The following `kubecfg` definition lets you run `kubecfg login [KUBECONFIG] [CONTEXT]` or `kubecfg render [KUBECONFIG]`. Both commands will execute the Tanzu login source, read the temporary kubeconfig written by `kubectl-vsphere`, and copy the imported cluster and auth info into the rendered kubeconfig.

```yaml
kubeconfigs:
  tanzu:
    path: "@/tanzu.yaml"

    login_sources:
      tanzu-got-amimof:
        command: kubectl-vsphere
        env:
          - KUBECTL_VSPHERE_PASSWORD=beyonce
        args:
          - login
          - --server
          - https://supervisor-api.amimof.com:6443
          - --tanzu-kubernetes-cluster-name
          - prod-cluster
          - --tanzu-kubernetes-cluster-namespace
          - prod
          - --insecure-skip-tls-verify
          - --vsphere-username
          - amimof

    contexts:
      supervisor-prod:
        namespace: default
        import_ref:
          login_source: tanzu-got-amimof
          context: supervisor-api.amimof.com

      utbildning-dev:
        namespace: default
        import_ref:
          login_source: tanzu-got-amimof
          context: utbildning-dev
          auth_info: utbildning-dev
```

`import_ref.context` must match a context name in the temporary kubeconfig emitted by `kubectl-vsphere`. If `import_ref.cluster` or `import_ref.auth_info` is omitted, `kubecfg` defaults those names from the imported context.

## Commands

Refresh a specific imported context:

```bash
kubecfg login tanzu supervisor-prod
```

Render the kubeconfig and make it the active one:

```bash
kubecfg render tanzu
```

If your config contains encrypted auth fields elsewhere in the kubeconfig definition, provide the age identity used to decrypt them during compile:

```bash
kubecfg login tanzu supervisor-prod
kubecfg render tanzu
```

`kubecfg render` runs configured login sources by default before writing the final kubeconfig. Use `--skip-login` only when you intentionally want to reuse previously imported credentials.
