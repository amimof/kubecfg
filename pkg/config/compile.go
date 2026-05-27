package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"k8s.io/client-go/tools/clientcmd/api"
)

type Compiler struct {
	BaseDir string
}

func NewCompiler(baseDir string) *Compiler {
	return &Compiler{
		BaseDir: baseDir,
	}
}

func (c *Compiler) Compile(cfg *Config) (*RuntimeConfig, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}

	rt := &RuntimeConfig{
		Version: cfg.Version,

		DefaultWorkspace: cfg.DefaultWorkspace,
		DefaultNamespace: cfg.DefaultNamespace,

		Workspaces:        make(map[string]*RuntimeWorkspace),
		Kubeconfigs:       make(map[string]*RuntimeKubeconfig),
		KubeconfigAliases: make(map[string]*RuntimeKubeconfig),
		Contexts:          make(map[string]*RuntimeContextRef),
	}

	if err := c.compileKubeconfigs(rt, cfg); err != nil {
		return nil, err
	}

	if err := c.compileWorkspaces(rt, cfg); err != nil {
		return nil, err
	}

	if err := c.compileDefaults(rt, cfg); err != nil {
		return nil, err
	}

	return rt, nil
}

func (c *Compiler) compileKubeconfigs(rt *RuntimeConfig, cfg *Config) error {
	for kubeconfigName, kubeconfig := range cfg.Kubeconfigs {
		if kubeconfig == nil {
			return fmt.Errorf("kubeconfigs.%s is nil", kubeconfigName)
		}

		rkc := &RuntimeKubeconfig{
			Name:      kubeconfigName,
			Path:      c.resolvePath(kubeconfig.Path),
			Protected: kubeconfig.Protected,
			Aliases:   append([]string(nil), kubeconfig.Aliases...),

			Clusters:  make(map[string]*RuntimeCluster),
			AuthInfos: make(map[string]*RuntimeAuthInfo),
			Contexts:  make(map[string]*RuntimeContext),

			Config: api.NewConfig(),
		}

		if rkc.Path == "" {
			return fmt.Errorf("kubeconfigs.%s.path is required", kubeconfigName)
		}

		if err := compileClusters(rkc, kubeconfig); err != nil {
			return err
		}

		if err := compileAuthInfos(rkc, kubeconfig); err != nil {
			return err
		}

		if err := compileContexts(rkc, kubeconfig); err != nil {
			return err
		}

		if err := resolveRuntimeContexts(rkc); err != nil {
			return err
		}

		compileNativeKubeconfig(rkc)

		rt.Kubeconfigs[kubeconfigName] = rkc

		if err := indexKubeconfigAliases(rt, rkc); err != nil {
			return err
		}
	}

	return nil
}

func compileClusters(rkc *RuntimeKubeconfig, kc *Kubeconfig) error {
	for name, cluster := range kc.Clusters {
		if cluster == nil {
			return fmt.Errorf("kubeconfigs.%s.clusters.%s is nil", rkc.Name, name)
		}

		rkc.Clusters[name] = &RuntimeCluster{
			Name: name,
			Cluster: &api.Cluster{
				LocationOfOrigin:         cluster.LocationOfOrigin,
				Server:                   cluster.Server,
				TLSServerName:            cluster.TLSServerName,
				InsecureSkipTLSVerify:    cluster.InsecureSkipTLSVerify,
				CertificateAuthority:     cluster.CertificateAuthority,
				CertificateAuthorityData: cluster.CertificateAuthorityData,
				ProxyURL:                 cluster.ProxyURL,
				DisableCompression:       cluster.DisableCompression,
				Extensions:               cluster.Extensions,
			},
		}
	}

	return nil
}

func resolveExecEnvVars(src []ExecEnvVar) []api.ExecEnvVar {
	execEnvVar := make([]api.ExecEnvVar, len(src))
	for i, e := range src {
		execEnvVar[i] = api.ExecEnvVar{
			Name:  e.Name,
			Value: e.Value,
		}
	}
	return execEnvVar
}

func compileAuthInfos(rkc *RuntimeKubeconfig, kc *Kubeconfig) error {
	for name, ai := range kc.AuthInfos {
		if ai == nil {
			return fmt.Errorf("kubeconfigs.%s.authinfos.%s is nil", rkc.Name, name)
		}

		rkc.AuthInfos[name] = &RuntimeAuthInfo{
			Name: name,
			AuthInfo: &api.AuthInfo{
				LocationOfOrigin:      ai.LocationOfOrigin,
				ClientCertificate:     ai.ClientCertificate,
				ClientCertificateData: ai.ClientCertificateData,
				ClientKey:             ai.ClientKey,
				ClientKeyData:         ai.ClientKeyData,
				Token:                 ai.Token,

				TokenFile:      ai.TokenFile,
				Impersonate:    ai.Impersonate,
				ImpersonateUID: ai.ImpersonateUID,

				ImpersonateGroups:    ai.ImpersonateGroups,
				ImpersonateUserExtra: ai.ImpersonateUserExtra,
				Username:             ai.Username,
				Password:             ai.Password,
				AuthProvider:         (*api.AuthProviderConfig)(ai.AuthProvider),
				Exec:                 resolveAuthInfosExec(ai.Exec),
				Extensions:           ai.Extensions,
			},
			CredentialSource: resolveCredentialSource(ai.Login),
		}
	}

	return nil
}

func envSliceToMap(env []string) map[string]string {
	m := make(map[string]string, len(env))

	for _, e := range env {
		key, value, ok := strings.Cut(e, "=")
		if !ok {
			continue
		}
		m[key] = value
	}

	return m
}

func resolveCredentialSource(l *LoginAuth) RuntimeCredentialSource {
	if l == nil {
		return nil
	}

	return &RuntimeLoginCredentialSource{
		Command: l.Command,
		Args:    l.Args,
		Env:     envSliceToMap(l.Env),
		Import: RuntimeLoginImport{
			Context: l.CopyFromContextName,
		},
	}
}

func resolveAuthInfosExec(e *ExecConfig) *api.ExecConfig {
	if e == nil {
		return nil
	}
	return &api.ExecConfig{
		Command:                 e.Command,
		Args:                    e.Args,
		Env:                     resolveExecEnvVars(e.Env),
		APIVersion:              e.APIVersion,
		InstallHint:             e.InstallHint,
		ProvideClusterInfo:      e.ProvideClusterInfo,
		Config:                  e.Config,
		InteractiveMode:         api.ExecInteractiveMode(e.InteractiveMode),
		StdinUnavailable:        e.StdinUnavailable,
		StdinUnavailableMessage: e.StdinUnavailableMessage,
	}
}

func compileContexts(rkc *RuntimeKubeconfig, kc *Kubeconfig) error {
	for name, context := range kc.Contexts {
		if context == nil {
			return fmt.Errorf("kubeconfigs.%s.contexts.%s is nil", rkc.Name, name)
		}

		runtimeAuthInfo := rkc.AuthInfo(context.AuthInfo)
		runtimeCluster := rkc.Cluster(context.Cluster)

		rkc.Contexts[name] = &RuntimeContext{
			Name: name,

			AuthInfo: runtimeAuthInfo,
			Cluster:  runtimeCluster,

			ClusterKey:  context.Cluster,
			AuthInfoKey: context.AuthInfo,
			Namespace:   context.Namespace,

			Context: &api.Context{
				LocationOfOrigin: context.LocationOfOrigin,
				Cluster:          context.Cluster,
				AuthInfo:         context.AuthInfo,
				Namespace:        context.Namespace,
				Extensions:       context.Extensions,
			},
		}
	}

	return nil
}

func resolveRuntimeContexts(rkc *RuntimeKubeconfig) error {
	for contextKey, ctx := range rkc.Contexts {
		cluster, ok := rkc.Clusters[ctx.ClusterKey]
		if !ok {
			return fmt.Errorf(
				"kubeconfigs.%s.contexts.%s.cluster references missing cluster %q",
				rkc.Name,
				contextKey,
				ctx.ClusterKey,
			)
		}

		authInfo, ok := rkc.AuthInfos[ctx.AuthInfoKey]
		if !ok {
			return fmt.Errorf(
				"kubeconfigs.%s.contexts.%s.authinfo references missing authinfo %q",
				rkc.Name,
				contextKey,
				ctx.AuthInfoKey,
			)
		}

		ctx.Cluster = cluster
		ctx.AuthInfo = authInfo

		// Runtime context should point to rendered kubeconfig names.
		ctx.Context.Cluster = cluster.Name
		ctx.Context.AuthInfo = authInfo.Name
		ctx.Context.Namespace = ctx.Namespace
	}

	return nil
}

func compileNativeKubeconfig(rkc *RuntimeKubeconfig) {
	kcfg := api.NewConfig()

	for _, cluster := range rkc.Clusters {
		kcfg.Clusters[cluster.Name] = cluster.Cluster.DeepCopy()
	}

	for _, authInfo := range rkc.AuthInfos {
		kcfg.AuthInfos[authInfo.Name] = authInfo.AuthInfo.DeepCopy()
	}

	for _, ctx := range rkc.Contexts {
		kcfg.Contexts[ctx.Name] = ctx.Context.DeepCopy()
	}

	if rkc.CurrentContext != nil {
		kcfg.CurrentContext = rkc.CurrentContext.Name
	}

	rkc.Config = kcfg
}

func (c *Compiler) compileWorkspaces(rt *RuntimeConfig, cfg *Config) error {
	for workspaceName, workspace := range cfg.Workspaces {
		if workspace == nil {
			return fmt.Errorf("workspaces.%s is nil", workspaceName)
		}

		rw := &RuntimeWorkspace{
			Name: workspaceName,

			Description:      workspace.Description,
			DefaultNamespace: firstNonEmpty(workspace.DefaultNamespace, cfg.DefaultNamespace),

			Kubeconfigs: make(map[string]*RuntimeKubeconfig),
		}

		for _, kubeconfigName := range workspace.Kubeconfigs {
			rkc, ok := rt.Kubeconfigs[kubeconfigName]
			if !ok {
				return fmt.Errorf(
					"workspaces.%s.kubeconfigs references missing kubeconfig %q",
					workspaceName,
					kubeconfigName,
				)
			}

			rw.Kubeconfigs[kubeconfigName] = rkc

			indexWorkspaceContexts(rt, rw, rkc)
		}

		rt.Workspaces[workspaceName] = rw
	}

	return nil
}

func (c *Compiler) compileDefaults(rt *RuntimeConfig, cfg *Config) error {
	if cfg.DefaultWorkspace != "" {
		if _, ok := rt.Workspaces[cfg.DefaultWorkspace]; !ok {
			return fmt.Errorf("default_workspace references missing workspace %q", cfg.DefaultWorkspace)
		}
	}

	for workspaceName, workspace := range cfg.Workspaces {
		if workspace == nil {
			continue
		}

		if workspace.DefaultContext == "" {
			continue
		}

		rw := rt.Workspaces[workspaceName]
		ref, err := resolveWorkspaceDefaultContext(rt, rw, workspace.DefaultContext)
		if err != nil {
			return err
		}

		rw.DefaultContext = ref
		ref.Kubeconfig.CurrentContext = ref.Context

		compileNativeKubeconfig(ref.Kubeconfig)
	}

	return nil
}

func indexKubeconfigAliases(rt *RuntimeConfig, rkc *RuntimeKubeconfig) error {
	// The kubeconfig name itself is also a lookup alias.
	names := append([]string{rkc.Name}, rkc.Aliases...)

	for _, alias := range names {
		alias = strings.TrimSpace(alias)
		if alias == "" {
			continue
		}

		existing, exists := rt.KubeconfigAliases[alias]
		if exists && existing.Name != rkc.Name {
			return fmt.Errorf(
				"kubeconfig alias %q is used by both %q and %q",
				alias,
				existing.Name,
				rkc.Name,
			)
		}

		rt.KubeconfigAliases[alias] = rkc
	}

	return nil
}

func indexWorkspaceContexts(rt *RuntimeConfig, rw *RuntimeWorkspace, rkc *RuntimeKubeconfig) {
	for _, ctx := range rkc.Contexts {
		ref := &RuntimeContextRef{
			Workspace:  rw,
			Kubeconfig: rkc,
			Context:    ctx,
		}

		keys := []string{
			rw.Name + "/" + rkc.Name + "/" + ctx.Name,
			rkc.Name + "/" + ctx.Name,
		}

		for _, key := range keys {
			rt.Contexts[key] = ref
		}
	}
}

func resolveWorkspaceDefaultContext(
	rt *RuntimeConfig,
	rw *RuntimeWorkspace,
	value string,
) (*RuntimeContextRef, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}

	// Fully-qualified or kubeconfig-qualified lookup.
	if ref, ok := rt.Contexts[value]; ok {
		if ref.Workspace.Name == rw.Name {
			return ref, nil
		}
	}

	// Search by context key or rendered context name within this workspace.
	var matches []*RuntimeContextRef

	for _, rkc := range rw.Kubeconfigs {
		for _, ctx := range rkc.Contexts {
			if ctx.Name == value {
				matches = append(matches, &RuntimeContextRef{
					Workspace:  rw,
					Kubeconfig: rkc,
					Context:    ctx,
				})
			}
		}
	}

	switch len(matches) {
	case 0:
		return nil, fmt.Errorf(
			"workspaces.%s.default_context references missing context %q",
			rw.Name,
			value,
		)
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf(
			"workspaces.%s.default_context %q is ambiguous; use kubeconfig/context",
			rw.Name,
			value,
		)
	}
}

func (c *Compiler) resolvePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}

	if after, ok := strings.CutPrefix(path, "@"); ok {
		relative := after
		relative = strings.TrimPrefix(relative, "/")

		if c.BaseDir == "" {
			return relative
		}

		return filepath.Join(c.BaseDir, relative)
	}

	return path
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}

	return ""
}
