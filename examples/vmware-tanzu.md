# Using kubecfg with VMware Tanzu

VMware Tanzu clusters often require a distribution-specific login flow before `kubectl` can access a cluster using the `kubectl-vsphere` plugin. Kubecfg supports running distribution specific login flows before the kubeconfig is generated and ready to be used. Here is an example on how to set it up.

## Configuration

Following `kubecfg` kubeconfig declaration allows you to run `kubecfg login [KUBECONFIG] [CONTEXT]`. That will run the `login` command, authenticate the user with Tanzu, export the authentication token and inject it into a rendered kubeconfig.

```yaml
kubeconfigs:
  tanzu:
    clusters:
      supervisor-prod:
        name: supervisor-prod
        server: https://supervisor-prod.amimof.com:6443
        insecure_skip_tls_verify: true
    auth_infos:
      amimof:
        name: amimof
        provider: tanzu-vsphere
        login:
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
          copy_from_context_name: supervisor-api.amimof.com
    contexts:
      supervisor-prod:
        name: supervisor-prod
        cluster: supervisor-prod
        authinfo: amimof

```

With this configuration you may run the following:

```bash
kubecfg login tanzu supervisor-prod
```

If the auth info contains encrypted fields such as `encryptedToken`, provide the age identity used to decrypt them during compile:

```bash
kubecfg login tanzu supervisor-prod --identity-file ~/.config/kubecfg/age.txt
```

> NOTE! The login flow is by default executed with `kubecfg use` if the `auth_info` contains a `login` flow.
