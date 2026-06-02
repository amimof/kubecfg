# kubecfg

[![Go](https://github.com/amimof/kubecfg/actions/workflows/go.yaml/badge.svg)](https://github.com/amimof/kubecfg/actions/workflows/go.yaml)

`kubecfg` is a small CLI for managing kubeconfigs as named workspaces. It keeps cluster definitions, output paths, and credential refresh hooks in a single YAML file, then renders the kubeconfig files you actually use.

<!--toc:start-->
- [What It Does](#what-it-does)
- [Quickstart](#quickstart)
- [Highlights](#highlights)
- [How It Works](#how-it-works)
- [Usage](#usage)
  - [Rendering kubeconfigs](#rendering-kubeconfigs)
  - [Login pre-hooks](#login-pre-hooks)
  - [Encrypted Fields](#encrypted-fields)
  - [Describing Workspaces](#describing-workspaces)
- [API Reference](#api-reference)
- [CLI Reference](#cli-reference)
- [License](#license)
<!--toc:end-->


# What It Does
If you manage more than one cluster, `~/.kube` tends to turn into a junk drawer pretty quickly. `kubecfg` gives that sprawl some structure without adding much ceremony. It groups kubeconfigs into workspaces, lets you select them interactively or directly, and can refresh credentials by calling an external login command and importing the resulting auth info.

`kubecfg` is useful when you want one source of truth for cluster access instead of hand-editing kubeconfigs or keeping a pile of half-documented files around. It fits well when different environments need different output files, different default contexts, or different login flows, but you still want the result to be plain kubeconfig files on disk.


# Highlights
- Declarative kubeconfig management from a single YAML file
- Named workspaces for grouping related kubeconfigs
- Interactive fuzzy selection backed by `fzf`
- Workspace-level default context resolution
- External login flow for refreshing credentials on demand
- Relative kubeconfig paths via `@/` and `--base-dir`
- Plain files, plain CLI, no daemon or background state
- Built-in per-field encryption using `age`

# How It Works

`kubecfg` treats `~/.config/kubecfg.yaml` as the source of truth and rendered kubeconfig files as disposable build artifacts. Instead of editing kubeconfig files directly, you describe your intended kubeconfig setup in `kubecfg.yaml`: workspaces, clusters, auth info, contexts, hooks, defaults, and optional encrypted fields. When you render a kubeconfig, `kubecfg` turns that declarative intent into a regular kubeconfig file that tools like `kubectl`, `helm`, and `kubectx` can use.

When you run `kubecfg render`, kubecfg reads the configuration file, resolves the selected workspace, assembles the referenced clusters, users, and contexts, applies any workspace defaults, and writes the rendered kubeconfig to the configured output directory.

Because rendered kubeconfigs are generated outputs, they are intentionally disposable. If a rendered kubeconfig is deleted, overwritten, or becomes stale, you can render it again from the source configuration. 

This lets you manage `kubecfg.yaml` like a dotfile. Version it, sync it across machines, or share it with others to reproduce the same kubeconfig setup consistently.

# Quickstart

1. Install `kubecfg` from GitHub [Releases](https://github.com/amimof/kubecfg/releases)
   
   ```bash
   curl -LO https://github.com/amimof/kubecfg/releases/latest/download/kubecfg-linux-amd64
   sudo install -m 755 kubecfg-linux-amd64 /usr/local/bin/kubecfg
   ```

   > **Make sure you download the precompiled binary for your OS and Platform**

2. Create `~/.config/kubecfg.yaml`. Here's an example configuration

   ```yaml
   workspaces:
     homelab:
       description: "Homelab"
       kubeconfigs:
       - mainframe

   kubeconfigs:
     mainframe:
       path: "@mainframe.yaml"

       clusters:
         mainframe-dev:
           server: https://mainframe-dev.amimof.com

       auth_infos:
         admin:
           exec:
             command: oidc-login
             args: ["get-token", "--issuer-url", "https://issuer.example.com"]

       contexts:
         mainframe:
           cluster: mainframe-dev
           authinfo: admin
   ```

3. Render a kubeconfig

   ```bash
   kubecfg render
   ```

   Press Enter to render the selected kubeconfig. kubecfg writes the rendered kubeconfig to `base_dir`, which defaults to `~/.kube/`, and updates `~/.kube/config` to point to it.

> See [Examples](/examples/) for more information on how to configure kubecfg in various ways

# Usage

## Rendering kubeconfigs

`kubecfg render` renders one or more kubeconfig definitions from your `~/.config/kubecfg.yaml` configuration file. A kubeconfig definition describes the intent for a kubeconfig. Rendering turns that intent into an actual kubeconfig file on disk.

`kubecfg render mainframe` renders a specific kubeconfig in the default workspace set by `default_workspace`

`kubecfg render homelab/mainframe` renders a specific kubeconfig in a specific workspace.

`kubecfg render mainframe --workspace homelab` same as previous command

`kubecfg render mainframe --identity-file ~/.config/kubecfg/age.txt` decrypts `encryptedToken` and other encrypted auth fields during compile.

## Login pre-hooks

A kubeconfig definition can include a `login` pre-hook. When present, the hook is executed before the kubeconfig is rendered. This is useful for workflows where you need to authenticate before generating or activating a kubeconfig, for example with OIDC, cloud provider CLIs, or custom login commands.

Example:
```yaml
kubeconfigs:
  company:
    path: "@/company.yaml"
    auth_infos:
      amimof:
        login:
          command: kubectl
          args:
            - oidc-login
            - get-token
          copy_from_context_name: imported
    contexts:
      production:
        cluster: company
        authInfo: amimof
```

When you run `kubecfg render company`, the login hook runs first. The command writes credentials to the kubeconfig path provided in `$KUBECONFIG`, and `kubecfg` imports the auth info referenced by `login.copy_from_context_name` before rendering and activating the final kubeconfig.

## Encrypted Fields

Use `kubecfg encrypt` to generate an armored age string and paste it into a encrypted auth field.

```bash
kubecfg encrypt --public-key age1...
```

```yaml
kubeconfigs:
  mainframe:
    path: "@/generated/mainframe.yaml"
    auth_infos:
      admin:
        encryptedToken: |
          -----BEGIN AGE ENCRYPTED FILE-----
          ...
          -----END AGE ENCRYPTED FILE-----
    contexts:
      admin:
        cluster: mainframe
        authInfo: admin
```

Render that kubeconfig with either an age identity file or a passphrase-backed age secret:

```bash
kubecfg render mainframe --identity-file ~/.config/kubecfg/age.txt
kubecfg login mainframe admin --identity-file ~/.config/kubecfg/age.txt
```

If both `token` and `encryptedToken` are set, `encryptedToken` wins.

## Describing Workspaces

Use `kubecfg describe workspace` to inspect a workspace and the kubeconfigs defined in it. This is useful when you want to understand what `kubecfg` will render before selecting or activating a kubeconfig.

```bash
kubecfg describe workspace
```

To inspect a single workspace, pass the workspace name:

```bash
kubecfg describe workspace homelab
```

If a workspace contains kubeconfigs with encrypted fields, provide an identity file when describing it:

```bash
kubecfg describe workspace homelab --identity-file ~/.config/kubecfg/age.txt
```

# API Reference

This example is meant to be copied into `kubecfg.yaml` and edited in place. It uses the canonical field spellings accepted by the current decoder. Keep one primary auth mechanism uncommented per `auth_infos.<name>` entry.

```yaml
# Config version. The empty in-memory default uses v1.
version: v1

# Used when --workspace is omitted.
default_workspace: examples

# Top-level fallback namespace. Stored in runtime config, but not currently
# written into generated kubeconfigs by kubecfg.
# default_namespace: default

workspaces:
  examples:
    # Free-form description shown by `kubecfg workspaces`.
    description: "Reference workspace with all supported kubeconfig scenarios"

    # Each entry must match a key under `kubeconfigs`.
    kubeconfigs:
      - static-token
      - login-refresh
      - exec-plugin
      - token-file
      - mtls
      - basic-auth
      - legacy-auth-provider

kubeconfigs:
  static-token:
    # Absolute path, or "@/..." relative to --base-dir.
    path: "@/generated/static-token.yaml"

    # Local context name to render as current-context for this kubeconfig.
    # This is resolved only within static-token.contexts.
    current_context: admin

    # Present in the schema, but not currently enforced by CLI commands.
    # protected: true

    # Aliases must be unique across all kubeconfigs.
    aliases:
      - token
      - mainframe

    clusters:
      mainframe:
        # The cluster map key is the name used in the rendered kubeconfig.
        # `name` exists in the schema, but the compiler currently uses the
        # map key above.
        # name: mainframe

        # Optional internal metadata passed through to client-go.
        # location_of_origin: kubecfg

        # Kubernetes API server URL.
        server: https://mainframe.example.com

        # Optional transport settings.
        # tls_server_name: api.internal.example.com

        # Choose one CA source: file path or inline PEM data.
        certificate_authority: /etc/kubernetes/ca.pem
        # certificate_authority_data: |
        #   -----BEGIN CERTIFICATE-----
        #   ...
        #   -----END CERTIFICATE-----

        # Optional transport tweaks.
        # insecure_skip_tls_verify: false
        # proxy_url: http://proxy.internal:8080
        # disable_compression: false

        # Present in the schema, but not practical to configure from plain YAML
        # with the current decoder.
        # extensions: {}

    auth_infos:
      admin:
        # Pick one primary auth mechanism for this auth info.
        token: "<redacted>"

        # Encrypted variants are decrypted during Compile() when you run
        # `kubecfg render` or `kubecfg login` with `--identity-file` or a
        # passphrase on stdin.
        # encryptedToken: |
        #   -----BEGIN AGE ENCRYPTED FILE-----
        #   ...
        #   -----END AGE ENCRYPTED FILE-----
        # encryptedPassword: |
        #   -----BEGIN AGE ENCRYPTED FILE-----
        #   ...
        #   -----END AGE ENCRYPTED FILE-----

        # Other primary auth mechanisms:
        # tokenFile: /var/run/secrets/kubernetes.io/serviceaccount/token
        # username: demo
        # password: change-me
        # clientCertificate: /etc/kubernetes/client.crt
        # clientKey: /etc/kubernetes/client.key
        # clientCertificateData: |
        #   -----BEGIN CERTIFICATE-----
        #   ...
        #   -----END CERTIFICATE-----
        # clientKeyData: |
        #   -----BEGIN PRIVATE KEY-----
        #   ...
        #   -----END PRIVATE KEY-----
        # authProvider:
        #   name: oidc
        #   config:
        #     client-id: kubectl
        #     idp-issuer-url: https://issuer.example.com
        # exec:
        #   command: kubelogin
        #   args:
        #     - get-token
        # login:
        #   command: /usr/local/bin/cluster-login
        #   args:
        #     - --cluster
        #     - mainframe
        #   env:
        #     - AWS_PROFILE=dev
        #   outputMode: json
        #   copy_from_context_name: imported

        # Optional fields that can be layered on top of the chosen auth
        # mechanism.
        # locationOfOrigin: kubecfg
        # impersonate: alice@example.com
        # impersonateUID: "1001"
        # impersonateGroups:
        #   - developers
        #   - oncall
        # impersonateUserExtra:
        #   example.com/team:
        #     - platform

        # Present in the schema, but not practical to configure from plain YAML
        # with the current decoder.
        # extensions: {}

    contexts:
      admin:
        # Must reference an existing cluster key from this kubeconfig.
        cluster: mainframe

        # Current decoder expects authInfo here.
        # Must reference an existing auth info key from this kubeconfig.
        authInfo: admin

        # Optional namespace stored on the rendered context.
        namespace: default

        # Optional internal metadata passed through to client-go.
        # locationOfOrigin: kubecfg

        # Present in the schema, but not practical to configure from plain YAML
        # with the current decoder.
        # extensions: {}

  login-refresh:
    path: "@/generated/login-refresh.yaml"
    clusters:
      login-cluster:
        server: https://login.example.com
    auth_infos:
      oidc:
        # Use `login` when kubecfg should run a command, read the temporary
        # kubeconfig it produced, and import credentials from a context inside it.
        login:
          command: /usr/local/bin/cluster-login
          args:
            - --cluster
            - login-cluster

          # Present in the schema, but not currently used by kubecfg.
          # outputMode: json

          # Extra environment passed to the login subprocess.
          env:
            - AWS_PROFILE=dev
            - AWS_REGION=eu-west-1

          # This must match the context name inside the temporary kubeconfig
          # produced by the login command.
          copy_from_context_name: imported

        # Other primary auth mechanisms:
        # token: "<redacted>"
        # exec:
        #   command: kubelogin
        #   args:
        #     - get-token
    contexts:
      imported:
        cluster: login-cluster
        authInfo: oidc
        namespace: default

  exec-plugin:
    path: "@/generated/exec-plugin.yaml"
    clusters:
      exec-cluster:
        server: https://exec.example.com
    auth_infos:
      exec-user:
        # Use `exec` when the rendered kubeconfig itself should carry a
        # client-go exec plugin configuration.
        exec:
          command: kubelogin
          args:
            - get-token
            - --environment
            - AzurePublicCloud
          env:
            - name: AZURE_CONFIG_DIR
              value: /Users/you/.azure
          apiVersion: client.authentication.k8s.io/v1beta1
          installHint: Install kubelogin and make sure it is on PATH.
          provideClusterInfo: true

          # Optional exec settings.
          # interactiveMode: IfAvailable
          # stdinUnavailable: false
          # stdinUnavailableMessage: stdin required for interactive login

          # Present in the schema, but not practical to configure from plain YAML
          # with the current decoder.
          # config: {}

        # Other primary auth mechanisms:
        # token: "<redacted>"
        # login:
        #   command: /usr/local/bin/cluster-login
        #   args:
        #     - --cluster
        #     - exec-cluster
        #   copy_from_context_name: imported
    contexts:
      exec:
        cluster: exec-cluster
        authInfo: exec-user

  token-file:
    # Absolute paths are also supported.
    path: /Users/you/.kube/token-file.yaml
    clusters:
      tokenfile-cluster:
        server: https://tokenfile.example.com
    auth_infos:
      tokenfile-user:
        tokenFile: /var/run/secrets/kubernetes.io/serviceaccount/token

        # Other primary auth mechanisms:
        # token: "<redacted>"
        # login:
        #   command: /usr/local/bin/cluster-login
        #   args:
        #     - --cluster
        #     - tokenfile-cluster
        #   copy_from_context_name: imported
    contexts:
      tokenfile:
        cluster: tokenfile-cluster
        authInfo: tokenfile-user

  mtls:
    path: "@/generated/mtls.yaml"
    clusters:
      mtls-cluster:
        server: https://mtls.example.com

        # Choose one CA source: file path or inline PEM data.
        certificate_authority: /etc/kubernetes/ca.pem
        # certificate_authority_data: |
        #   -----BEGIN CERTIFICATE-----
        #   ...
        #   -----END CERTIFICATE-----
    auth_infos:
      mtls-user:
        # Choose one certificate source: file paths or inline PEM data.
        clientCertificate: /etc/kubernetes/client.crt
        clientKey: /etc/kubernetes/client.key
        # clientCertificateData: |
        #   -----BEGIN CERTIFICATE-----
        #   ...
        #   -----END CERTIFICATE-----
        # clientKeyData: |
        #   -----BEGIN PRIVATE KEY-----
        #   ...
        #   -----END PRIVATE KEY-----

        # Other primary auth mechanisms:
        # token: "<redacted>"
        # exec:
        #   command: kubelogin
        #   args:
        #     - get-token
    contexts:
      mtls:
        cluster: mtls-cluster
        authInfo: mtls-user

  basic-auth:
    path: "@/generated/basic-auth.yaml"
    clusters:
      basic-cluster:
        server: https://basic.example.com
    auth_infos:
      basic-user:
        username: demo
        password: change-me

        # Optional impersonation fields can be layered on top of the selected
        # auth mechanism.
        impersonate: alice@example.com
        # impersonateUID: "1001"
        # impersonateGroups:
        #   - developers
        #   - oncall
        # impersonateUserExtra:
        #   example.com/team:
        #     - platform

        # Other primary auth mechanisms:
        # token: "<redacted>"
        # login:
        #   command: /usr/local/bin/cluster-login
        #   args:
        #     - --cluster
        #     - basic-cluster
        #   copy_from_context_name: imported
    contexts:
      basic:
        cluster: basic-cluster
        authInfo: basic-user

  legacy-auth-provider:
    path: "@/generated/legacy-auth-provider.yaml"
    clusters:
      legacy-cluster:
        server: https://legacy.example.com
    auth_infos:
      legacy-user:
        # Legacy client-go auth provider config.
        authProvider:
          name: oidc
          config:
            client-id: kubectl
            client-secret: "<redacted>"
            id-token: "<redacted>"
            refresh-token: "<redacted>"
            idp-issuer-url: https://issuer.example.com

        # Other primary auth mechanisms:
        # token: "<redacted>"
        # exec:
        #   command: kubelogin
        #   args:
        #     - get-token
    contexts:
      legacy:
        cluster: legacy-cluster
        authInfo: legacy-user
```

> **Note**: This project is under active early development and unstable. Features, API, and behavior are subject to change at any time and may not be backwards compatible between versions. Expect breaking changes.

# CLI Reference
Use `kubecfg --help` for CLI usage

# License
kubecfg is licensed under the MIT license. See the [`LICENSE`](./LICENSE) file for details.
