package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/client-go/tools/clientcmd/api"
)

type Compiler struct {
	Decryptor SecretDecryptor
}

type CompilerOption func(*Compiler)

func WithDecryptor(d SecretDecryptor) CompilerOption {
	return func(c *Compiler) {
		c.Decryptor = d
	}
}

func NewCompiler(opts ...CompilerOption) *Compiler {
	c := &Compiler{}

	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *Compiler) Compile(cfg *Config) (*RuntimeConfig, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}

	rt := &RuntimeConfig{
		Version: cfg.Version,
		BaseDir: cfg.BaseDir,

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
			Name:             kubeconfigName,
			Path:             c.resolvePath(rt, kubeconfig.Path),
			Protected:        kubeconfig.Protected,
			Aliases:          append([]string(nil), kubeconfig.Aliases...),
			DefaultNamespace: strings.TrimSpace(kubeconfig.DefaultNamespace),

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

		if err := c.compileAuthInfos(rkc, kubeconfig); err != nil {
			return err
		}

		if err := compileContexts(rkc, kubeconfig); err != nil {
			return err
		}

		if err := resolveRuntimeContexts(rkc); err != nil {
			return err
		}

		if err := resolveKubeconfigCurrentContext(rkc, kubeconfig.CurrentContext); err != nil {
			return err
		}

		if err := resolveKubeconfigDefaultContext(rkc, kubeconfig.DefaultContext); err != nil {
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

func (c *Compiler) compileAuthInfos(rkc *RuntimeKubeconfig, kc *Kubeconfig) error {
	for name, ai := range kc.AuthInfos {
		compiler := &AuthInfoCompiler{Decryptor: c.Decryptor}
		rai, err := compiler.Compile(name, ai)
		if err != nil {
			return err
		}
		rkc.AuthInfos[name] = rai
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

		namespace := firstNonEmpty(context.Namespace, kc.DefaultNamespace)

		rkc.Contexts[name] = &RuntimeContext{
			Name: name,

			AuthInfo: runtimeAuthInfo,
			Cluster:  runtimeCluster,

			ClusterKey:  context.Cluster,
			AuthInfoKey: context.AuthInfo,
			Namespace:   namespace,

			Context: &api.Context{
				LocationOfOrigin: context.LocationOfOrigin,
				Cluster:          context.Cluster,
				AuthInfo:         context.AuthInfo,
				Namespace:        namespace,
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

func resolveKubeconfigCurrentContext(rkc *RuntimeKubeconfig, currentContext string) error {
	currentContext = strings.TrimSpace(currentContext)
	if currentContext == "" {
		return nil
	}

	ctx, ok := rkc.Contexts[currentContext]
	if !ok {
		return fmt.Errorf(
			"kubeconfigs.%s.current_context references missing context %q",
			rkc.Name,
			currentContext,
		)
	}

	rkc.CurrentContext = ctx
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

			Description: workspace.Description,

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

		if workspace.DefaultKubeconfig != "" {
			rw.DefaultKubeconfig = rw.Kubeconfig(workspace.DefaultKubeconfig)
			if rw.DefaultKubeconfig == nil {
				return fmt.Errorf(
					"workspaces.%s.default_kubeconfig references missing kubeconfig %q",
					workspaceName,
					workspace.DefaultKubeconfig,
				)
			}
		}

		rt.Workspaces[workspaceName] = rw
	}

	return nil
}

func (c *Compiler) compileDefaults(rt *RuntimeConfig, cfg *Config) error {
	if cfg.DefaultWorkspace != "" {
		rw, ok := rt.Workspaces[cfg.DefaultWorkspace]
		if !ok {
			return fmt.Errorf("default_workspace references missing workspace %q", cfg.DefaultWorkspace)
		}

		rt.DefaultWorkspace = rw
	}

	if rt.BaseDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		defaultBaseDir := filepath.Join(home, ".kube")
		rt.BaseDir = defaultBaseDir
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

func resolveKubeconfigDefaultContext(rkc *RuntimeKubeconfig, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	ctx, ok := rkc.Contexts[value]
	if !ok {
		return fmt.Errorf(
			"kubeconfigs.%s.default_context references missing context %q",
			rkc.Name,
			value,
		)
	}

	rkc.DefaultContext = ctx
	rkc.CurrentContext = ctx
	return nil
}

func (c *Compiler) resolvePath(rt *RuntimeConfig, path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}

	if after, ok := strings.CutPrefix(path, "@"); ok {
		relative := after
		relative = strings.TrimPrefix(relative, "/")

		if rt.BaseDir == "" {
			return relative
		}

		return filepath.Join(rt.BaseDir, relative)
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

type AuthInfoCompiler struct {
	Decryptor SecretDecryptor
}

type SecretDecryptor interface {
	DecryptString(ciphertext string) (string, error)
	DecryptBytes(ciphertext []byte) ([]byte, error)
}

func (c *AuthInfoCompiler) Compile(name string, in *AuthInfo) (*RuntimeAuthInfo, error) {
	if in == nil {
		return nil, fmt.Errorf("authinfo %q is nil", name)
	}

	if in.HasEncryptedFields() && c.Decryptor == nil {
		return nil, fmt.Errorf("authinfo %q contains encrypted fields; provide --identity-file or a passphrase", name)
	}

	rai := &RuntimeAuthInfo{
		Name: name,
		AuthInfo: &api.AuthInfo{
			LocationOfOrigin:      in.LocationOfOrigin,
			ClientCertificate:     in.ClientCertificate,
			ClientCertificateData: in.ClientCertificateData,
			ClientKey:             in.ClientKey,
			ClientKeyData:         in.ClientKeyData,
			Token:                 in.Token,

			TokenFile:      in.TokenFile,
			Impersonate:    in.Impersonate,
			ImpersonateUID: in.ImpersonateUID,

			ImpersonateGroups:    in.ImpersonateGroups,
			ImpersonateUserExtra: in.ImpersonateUserExtra,
			Username:             in.Username,
			Password:             in.Password,
			AuthProvider:         (*api.AuthProviderConfig)(in.AuthProvider),
			Exec:                 resolveAuthInfosExec(in.Exec),
			Extensions:           in.Extensions,
		},
		CredentialSource: resolveCredentialSource(in.Login),
	}

	if in.EncryptedToken != "" {
		token, err := c.Decryptor.DecryptString(in.EncryptedToken)
		if err != nil {
			return nil, fmt.Errorf("decrypt authinfo %q token: %w", name, err)
		}
		rai.AuthInfo.Token = token
	}

	if in.EncryptedPassword != "" {
		password, err := c.Decryptor.DecryptString(in.EncryptedPassword)
		if err != nil {
			return nil, fmt.Errorf("decrypt authinfo %q password: %w", name, err)
		}
		rai.AuthInfo.Password = password
	}

	if len(in.EncryptedClientKeyData) > 0 {
		data, err := c.Decryptor.DecryptBytes(in.EncryptedClientKeyData)
		if err != nil {
			return nil, fmt.Errorf("decrypt authinfo %q client key data: %w", name, err)
		}
		rai.AuthInfo.ClientKeyData = data
	}

	if len(in.EncryptedClientCertificateData) > 0 {
		data, err := c.Decryptor.DecryptBytes(in.EncryptedClientCertificateData)
		if err != nil {
			return nil, fmt.Errorf("decrypt authinfo %q client certificate data: %w", name, err)
		}
		rai.AuthInfo.ClientCertificateData = data
	}

	if len(in.EncryptedClientKey) > 0 {
		data, err := c.Decryptor.DecryptBytes(in.EncryptedClientKey)
		if err != nil {
			return nil, fmt.Errorf("decrypt authinfo %q client key: %w", name, err)
		}
		rai.AuthInfo.ClientKey = string(data)
	}

	if len(in.EncryptedClientCertificate) > 0 {
		data, err := c.Decryptor.DecryptBytes(in.EncryptedClientCertificate)
		if err != nil {
			return nil, fmt.Errorf("decrypt authinfo %q client certificate: %w", name, err)
		}
		rai.AuthInfo.ClientCertificate = string(data)
	}

	return rai, nil
}
