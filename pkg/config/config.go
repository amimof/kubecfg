package config

import (
	"k8s.io/apimachinery/pkg/runtime"
)

type Config struct {
	Version          string                 `mapstructure:"version,omitempty" json:"version,omitempty" yaml:"version,omitempty"`
	DefaultWorkspace string                 `mapstructure:"default_workspace,omitempty" json:"default_workspace,omitempty" yaml:"default_workspace,omitempty"`
	Workspaces       map[string]*Workspace  `mapstructure:"workspaces,omitempty" json:"workspaces,omitempty" yaml:"workspaces,omitempty"`
	Kubeconfigs      map[string]*Kubeconfig `mapstructure:"kubeconfigs,omitempty" json:"kubeconfigs,omitempty" yaml:"kubeconfigs,omitempty"`
	BaseDir          string                 `mapstructure:"base_dir,omitempty" json:"base_dir,omitempty" yaml:"base_dir,omitempty"`
}

type Workspace struct {
	Description       string   `mapstructure:"description,omitempty" json:"description,omitempty" yaml:"description,omitempty"`
	Kubeconfigs       []string `mapstructure:"kubeconfigs,omitempty" json:"kubeconfigs,omitempty" yaml:"kubeconfigs,omitempty"`
	DefaultKubeconfig string   `mapstructure:"default_kubeconfig,omitempty" json:"default_kubeconfig,omitempty" yaml:"default_kubeconfig,omitempty"`
}

type Kubeconfig struct {
	Path           string   `json:"path,omitempty"`
	Protected      bool     `json:"protected,omitempty"`
	Aliases        []string `json:"aliases,omitempty"`
	CurrentContext string   `mapstructure:"current_context,omitempty" json:"current_context,omitempty" yaml:"current_context,omitempty"`

	DefaultContext   string `mapstructure:"default_context,omitempty" json:"default_context,omitempty" yaml:"default_context,omitempty"`
	DefaultNamespace string `mapstructure:"default_namespace,omitempty" json:"default_namespace,omitempty" yaml:"default_namespace,omitempty"`

	Clusters  map[string]*Cluster  `mapstructure:"clusters,omitempty" json:"clusters,omitempty" yaml:"clusters,omitempty"`
	AuthInfos map[string]*AuthInfo `mapstructure:"auth_infos,omitempty" json:"auth_infos,omitempty" yaml:"auth_infos,omitempty"`
	Contexts  map[string]*Context  `mapstructure:"contexts,omitempty" json:"contexts,omitempty" yaml:"contexts,omitempty"`
}

type Cluster struct {
	LocationOfOrigin         string                    `mapstructure:"location_of_origin,omitempty" json:"location_of_origin,omitempty" yaml:"location_of_origin,omitempty"`
	Server                   string                    `mapstructure:"server,omitempty" json:"server,omitempty" yaml:"server,omitempty"`
	TLSServerName            string                    `mapstructure:"tls_server_name,omitempty" json:"tls_server_name,omitempty" yaml:"tls_server_name,omitempty"`
	InsecureSkipTLSVerify    bool                      `mapstructure:"insecure_skip_tls_verify,omitempty" json:"insecure_skip_tls_verify,omitempty" yaml:"insecure_skip_tls_verify,omitempty"`
	CertificateAuthority     string                    `mapstructure:"certificate_authority,omitempty" json:"certificate_authority,omitempty" yaml:"certificate_authority,omitempty"`
	CertificateAuthorityData []byte                    `mapstructure:"certificate_authority_data,omitempty" json:"certificate_authority_data,omitempty" yaml:"certificate_authority_data,omitempty"`
	ProxyURL                 string                    `mapstructure:"proxy_url,omitempty" json:"proxy_url,omitempty" yaml:"proxy_url,omitempty"`
	DisableCompression       bool                      `mapstructure:"disable_compression,omitempty" json:"disable_compression,omitempty" yaml:"disable_compression,omitempty"`
	Extensions               map[string]runtime.Object `mapstructure:"extensions" json:"extensions,omitempty" yaml:"extensions,omitempty"`
}

type AuthInfo struct {
	Login *LoginAuth `mapstructure:"login" json:"login" yaml:"login"`

	LocationOfOrigin               string
	ClientCertificate              string                    `json:"client-certificate,omitempty"`
	ClientCertificateData          []byte                    `json:"client-certificate-data,omitempty"`
	EncryptedClientCertificate     []byte                    `mapstructure:"encryptedClientCertificate,omitempty" json:"encryptedClientCertificate,omitempty" yaml:"encryptedClientCertificate,omitempty"`
	EncryptedClientCertificateData []byte                    `mapstructure:"encryptedClientCertificateData,omitempty" json:"encryptedClientCertificateData,omitempty" yaml:"encryptedClientCertificateData,omitempty"`
	ClientKey                      string                    `json:"client-key,omitempty"`
	ClientKeyData                  []byte                    `json:"client-key-data,omitempty" datapolicy:"security-key"`
	EncryptedClientKey             []byte                    `mapstructure:"encryptedClientKey,omitempty" json:"encryptedClientKey,omitempty" yaml:"encryptedClientKey,omitempty"`
	EncryptedClientKeyData         []byte                    `mapstructure:"encryptedClientKeyData,omitempty" json:"encryptedClientKeyData,omitempty" yaml:"encryptedClientKeyData,omitempty"`
	Token                          string                    `json:"token,omitempty" datapolicy:"token"`
	TokenFile                      string                    `json:"tokenFile,omitempty"`
	EncryptedToken                 string                    `mapstructure:"encryptedToken,omitempty" json:"encryptedToken,omitempty" yaml:"encryptedToken,omitempty"`
	Impersonate                    string                    `json:"act-as,omitempty"`
	ImpersonateUID                 string                    `json:"act-as-uid,omitempty"`
	ImpersonateGroups              []string                  `json:"act-as-groups,omitempty"`
	ImpersonateUserExtra           map[string][]string       `json:"act-as-user-extra,omitempty"`
	Username                       string                    `json:"username,omitempty"`
	Password                       string                    `json:"password,omitempty" datapolicy:"password"`
	EncryptedPassword              string                    `mapstructure:"encryptedPassword,omitempty" json:"encryptedPassword,omitempty" yaml:"encryptedPassword,omitempty" datapolicy:"password"`
	AuthProvider                   *AuthProviderConfig       `json:"auth-provider,omitempty"`
	Exec                           *ExecConfig               `json:"exec,omitempty"`
	Extensions                     map[string]runtime.Object `json:"extensions,omitempty"`
}

type Auth struct {
	Login *LoginAuth `mapstructure:"login" json:"login" yaml:"login"`
	Token *TokenAuth `mapstructure:"token" json:"token" yaml:"token"`
}

type LoginAuth struct {
	Command             string   `json:"command"`
	Args                []string `json:"args"`
	OutputMode          string   `json:"output_mode"`
	Env                 []string `json:"env"`
	CopyFromContextName string   `mapstructure:"copy_from_context_name" json:"copy_from_context_name" yaml:"copy_from_context_name"`
}

type TokenAuth struct {
	Command    string
	Args       []string
	OutputMode string
	Env        []string
}

type AuthProviderConfig struct {
	Name   string            `json:"name"`
	Config map[string]string `json:"config,omitempty"`
}

type ExecConfig struct {
	Command                 string       `json:"command"`
	Args                    []string     `json:"args"`
	Env                     []ExecEnvVar `json:"env"`
	APIVersion              string       `json:"apiVersion,omitempty"`
	InstallHint             string       `json:"installHint,omitempty"`
	ProvideClusterInfo      bool         `json:"provideClusterInfo"`
	Config                  runtime.Object
	InteractiveMode         ExecInteractiveMode
	StdinUnavailable        bool
	StdinUnavailableMessage string
}

type ExecEnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type ExecInteractiveMode string

type Context struct {
	LocationOfOrigin string
	Cluster          string                    `json:"cluster"`
	AuthInfo         string                    `json:"user"`
	Namespace        string                    `json:"namespace,omitempty"`
	Extensions       map[string]runtime.Object `json:"extensions,omitempty"`
}

func (c *Config) Validate() error {
	return nil
}

func (c *Config) Workspace(name string) *Workspace {
	if ws, ok := c.Workspaces[name]; ok {
		return ws
	}
	return nil
}

func (c *Config) Kubeconfig(name string) *Kubeconfig {
	if k, ok := c.Kubeconfigs[name]; ok {
		return k
	}
	return nil
}

func (k *Kubeconfig) Cluster(name string) *Cluster {
	if c, ok := k.Clusters[name]; ok {
		return c
	}
	return nil
}

func (k *Kubeconfig) AuthInfo(name string) *AuthInfo {
	if a, ok := k.AuthInfos[name]; ok {
		return a
	}
	return nil
}

func (k *Kubeconfig) Context(name string) *Context {
	if c, ok := k.Contexts[name]; ok {
		return c
	}
	return nil
}

func (c *Config) HasEncryptedAuthInfos() bool {
	for _, kubeconfig := range c.Kubeconfigs {
		if kubeconfig == nil {
			continue
		}

		for _, authInfo := range kubeconfig.AuthInfos {
			if authInfo != nil && authInfo.HasEncryptedFields() {
				return true
			}
		}
	}

	return false
}

func (a *AuthInfo) HasEncryptedFields() bool {
	if a == nil {
		return false
	}

	return a.EncryptedToken != "" ||
		a.EncryptedPassword != "" ||
		len(a.EncryptedClientKeyData) > 0 ||
		len(a.EncryptedClientCertificateData) > 0 ||
		len(a.EncryptedClientKey) > 0 ||
		len(a.EncryptedClientCertificate) > 0
}
