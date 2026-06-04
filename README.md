# kubecfg

[![Go](https://github.com/amimof/kubecfg/actions/workflows/go.yaml/badge.svg)](https://github.com/amimof/kubecfg/actions/workflows/go.yaml)

`kubecfg` is a small CLI for managing kubeconfigs as named workspaces. It keeps cluster definitions, output paths, and login/import flows in a single YAML file, then renders the kubeconfig files you actually use.

<!--toc:start-->
- [What It Does](#what-it-does)
- [Quickstart](#quickstart)
- [Highlights](#highlights)
- [How It Works](#how-it-works)
- [Usage](#usage)
  - [Rendering kubeconfigs](#rendering-kubeconfigs)
  - [Login Sources And Imports](#login-sources-and-imports)
  - [Encrypted Fields](#encrypted-fields)
  - [Describing Workspaces](#describing-workspaces)
  - [Selecting kubeconfigs](#selecting-kubeconfigs)
- [API Reference](#api-reference)
- [CLI Reference](#cli-reference)
- [License](#license)
<!--toc:end-->


# What It Does
If you manage more than one cluster, `~/.kube` tends to turn into a junk drawer pretty quickly. `kubecfg` gives that sprawl some structure without adding much ceremony. It groups kubeconfigs into workspaces, lets you select them interactively or directly, and can refresh credentials by calling external login commands and importing the resulting context data.

`kubecfg` is useful when you want one source of truth for cluster access instead of hand-editing kubeconfigs or keeping a pile of half-documented files around. It fits well when different environments need different output files, different default contexts, or different login/import flows, but you still want the result to be plain kubeconfig files on disk.


# Highlights
- Declarative kubeconfig management from a single YAML file
- Named workspaces for grouping related kubeconfigs
- Interactive fuzzy selection backed by `fzf`
- Workspace-level default context resolution
- Kubeconfig-level login sources for refreshing credentials on demand
- Context-level imports from login-generated kubeconfigs
- Relative kubeconfig paths via `@/` and `--base-dir`
- Plain files, plain CLI, no daemon or background state
- Built-in per-field encryption using `age`

# How It Works

`kubecfg` treats `~/.config/kubecfg.yaml` as the source of truth and rendered kubeconfig files as disposable build artifacts. Instead of editing kubeconfig files directly, you describe your intended kubeconfig setup in `kubecfg.yaml`: workspaces, clusters, auth info, contexts, login sources, imports, defaults, and optional encrypted fields. When you render a kubeconfig, `kubecfg` turns that declarative intent into a regular kubeconfig file that tools like `kubectl`, `helm`, and `kubectx` can use.

When you run `kubecfg render`, kubecfg reads the configuration file, resolves the selected workspace, assembles the referenced clusters, users, and contexts, runs any configured login sources, imports referenced clusters and auth infos from the temporary kubeconfigs they produce, applies any workspace defaults, and writes the rendered kubeconfig to the configured output directory.

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
        path: "@/mainframe.yaml"

        login_sources:
          oidc:
            command: kubectl
            args:
              - oidc-login
              - get-token

        contexts:
          mainframe:
            namespace: default
            import_ref:
              login_source: oidc
              context: imported
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

## Login Sources And Imports

A kubeconfig definition can include one or more `login_sources`. A login source runs a command that writes a temporary kubeconfig to the path provided in `$KUBECONFIG`. Contexts can then use `import_ref` to select which context, cluster, and auth info to copy from that temporary kubeconfig into the rendered kubeconfig.

Example:
```yaml
kubeconfigs:
  company:
    path: "@/company.yaml"
    login_sources:
      oidc:
        command: kubectl
        args:
          - oidc-login
          - get-token
    contexts:
      production:
        namespace: default
        import_ref:
          login_source: oidc
          context: imported
```

When you run `kubecfg render company`, kubecfg runs the `oidc` login source first. The command writes a temporary kubeconfig to the path provided in `$KUBECONFIG`. Kubecfg then reads that temporary kubeconfig, resolves `contexts.production.import_ref.context`, and copies the effective cluster and auth info into the rendered kubeconfig.

If `import_ref.cluster` or `import_ref.auth_info` is omitted, kubecfg defaults those names from the imported context inside the temporary kubeconfig.

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
        user: admin
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

## Selecting Kubeconfigs

Use `kubecfg use` to set a kubeconfig as the active one. The `use` command updates the symlink to `~/.kube/config` pointing it to a rendered kubeconfig. See [Rendering Kubeconfigs](#rendering-kubeconfigs) for more information.

`kubecfg use` searches for kubeconfig files and displays them in a fuzzy finder allowing you to interactively select a kubeconfig to activate.

`kubecfg use` searches for kubeconfigs using this pattern by default `~/.kube/*.yaml`. This can be overridden with `--glob`. 

For example:

```bash
kubecfg use --glob ~/Projects/kube/*.yaml --glob ~/.kube/conf.d/*.yml
```

# API Reference

This example is meant to be copied into `kubecfg.yaml` and edited in place. It uses the canonical field spellings accepted by the current decoder. Keep one primary auth mechanism uncommented per `auth_infos.<name>` entry.

```yaml
# Config version. The empty in-memory default uses v1.
version: v1

# Used when --workspace is omitted.
default_workspace: examples

# Base directory used for paths starting with "@/".
# If omitted, kubecfg defaults to ~/.kube.
# base_dir: ~/.kube

workspaces:
  examples:
    # Free-form description shown by `kubecfg workspaces`.
    description: "Reference workspace with all supported kubeconfig scenarios"

    # Each entry must match a key under `kubeconfigs`.
    kubeconfigs:
      - static-token
      - login-import
      - exec-plugin
      - token-file
      - mtls
      - basic-auth
      - legacy-auth-provider

kubeconfigs:
  static-token:
    # Absolute path, or "@/..." relative to base_dir.
    path: "@/generated/static-token.yaml"

    # Local context name to render as current-context for this kubeconfig.
    # This is resolved only within static-token.contexts.
    current_context: admin

    # Local context name that should become both the default and current
    # context in the rendered kubeconfig.
    # default_context: admin

    # Default namespace applied to contexts that omit namespace.
    # default_namespace: default

    # Present in the schema, but not currently enforced by CLI commands.
    # protected: true

    # Aliases must be unique across all kubeconfigs.
    aliases:
      - token
      - mainframe

    clusters:
      mainframe:
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

        # Use `exec` when the rendered kubeconfig itself should carry a
        # client-go exec plugin configuration.
        # exec:
        #   command: kubelogin
        #   args:
        #     - get-token
        #   env:
        #     - name: AZURE_CONFIG_DIR
        #       value: /Users/you/.azure
        #   env_file: ~/.config/kubecfg/exec.env
        #   apiVersion: client.authentication.k8s.io/v1beta1
        #   installHint: Install kubelogin and make sure it is on PATH.
        #   provideClusterInfo: true
        #   interactiveMode: IfAvailable
        #   stdinUnavailable: false
        #   stdinUnavailableMessage: stdin required for interactive login

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

        # Must reference an existing auth info key from this kubeconfig.
        user: admin

        # Optional namespace stored on the rendered context.
        namespace: default

        # Optional internal metadata passed through to client-go.
        # locationOfOrigin: kubecfg

        # Present in the schema, but not practical to configure from plain YAML
        # with the current decoder.
        # extensions: {}

  login-import:
    path: "@/generated/login-import.yaml"

    # Login sources are reusable within a single kubeconfig definition.
    login_sources:
      oidc:
        command: /usr/local/bin/cluster-login
        args:
          - --cluster
          - login-cluster

        # Extra environment passed to the login subprocess.
        env:
          - AWS_PROFILE=dev
          - AWS_REGION=eu-west-1

        # Environment can also be loaded from a dotenv-style file.
        # Values from env_file override duplicate keys from env.
        # env_file: ~/.config/kubecfg/login.env

        # Present in the schema, but not currently used by kubecfg.
        # output_mode: json

    contexts:
      imported:
        # For imported contexts, local cluster/user are optional.
        namespace: default

        import_ref:
          # Must match a key under login_sources.
          login_source: oidc

          # Must match a context name inside the temporary kubeconfig produced
          # by the login command.
          context: imported

          # Optional explicit overrides. If omitted, kubecfg uses the cluster
          # and auth info referenced by the imported context.
          # cluster: imported-cluster
          # auth_info: imported-user

      utbildning-dev:
        # Minimal import-only context matching a common Tanzu workflow.
        namespace: default
        import_ref:
          login_source: oidc
          context: utbildning-dev
          auth_info: utbildning-dev

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
    contexts:
      exec:
        cluster: exec-cluster
        user: exec-user

  token-file:
    # Absolute paths are also supported.
    path: /Users/you/.kube/token-file.yaml
    clusters:
      tokenfile-cluster:
        server: https://tokenfile.example.com
    auth_infos:
      tokenfile-user:
        tokenFile: /var/run/secrets/kubernetes.io/serviceaccount/token
    contexts:
      tokenfile:
        cluster: tokenfile-cluster
        user: tokenfile-user

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
    contexts:
      mtls:
        cluster: mtls-cluster
        user: mtls-user

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
    contexts:
      basic:
        cluster: basic-cluster
        user: basic-user

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
    contexts:
      legacy:
        cluster: legacy-cluster
        user: legacy-user
```

> **Note**: This project is under active early development and unstable. Features, API, and behavior are subject to change at any time and may not be backwards compatible between versions. Expect breaking changes.

# CLI Reference
Use `kubecfg --help` for CLI usage

# License
kubecfg is licensed under the MIT license. See the [`LICENSE`](./LICENSE) file for details.
