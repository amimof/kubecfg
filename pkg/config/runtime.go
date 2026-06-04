package config

import (
	"gopkg.in/yaml.v3"
	api "k8s.io/client-go/tools/clientcmd/api"
)

type RuntimeConfig struct {
	Version string
	BaseDir string

	Workspaces       map[string]*RuntimeWorkspace
	Kubeconfigs      map[string]*RuntimeKubeconfig
	DefaultWorkspace *RuntimeWorkspace

	// Lookup indexes for CLI ergonomics.
	KubeconfigAliases map[string]*RuntimeKubeconfig
	Contexts          map[string]*RuntimeContextRef
}

type RuntimeWorkspace struct {
	Name              string
	Description       string
	DefaultKubeconfig *RuntimeKubeconfig
	Kubeconfigs       map[string]*RuntimeKubeconfig
}

type RuntimeKubeconfig struct {
	Name string

	Path      string
	Protected bool
	Aliases   []string

	LoginSources map[string]*RuntimeLoginSource

	Clusters  map[string]*RuntimeCluster
	AuthInfos map[string]*RuntimeAuthInfo
	Contexts  map[string]*RuntimeContext

	CurrentContext   *RuntimeContext
	DefaultContext   *RuntimeContext
	DefaultNamespace string

	Config *api.Config
}

type RuntimeLoginSource struct {
	Command string
	Args    []string
	Env     map[string]string

	ImportedConfig *api.Config
}

type RuntimeImportRef struct {
	LoginSourceName string
	ContextName     string

	// Optional explicit names.
	ClusterName  string
	AuthInfoName string
}

func (rc *RuntimeConfig) Workspace(name string) *RuntimeWorkspace {
	if ws, ok := rc.Workspaces[name]; ok {
		return ws
	}
	return nil
}

func (rc *RuntimeConfig) WorkspaceExists(name string) bool {
	return rc.Workspace(name) != nil
}

func (rc *RuntimeConfig) KubeconfigExists(ws, name string) bool {
	if rc.Workspace(ws) == nil {
		return false
	}
	if rc.Workspace(ws).Kubeconfig(name) == nil {
		return false
	}
	return true
}

func (rc *RuntimeConfig) ContextExists(ws, k, name string) bool {
	if !rc.KubeconfigExists(ws, k) {
		return false
	}
	if rc.Workspace(ws).Kubeconfig(k).Context(name) == nil {
		return false
	}
	return true
}

func (rw *RuntimeWorkspace) Kubeconfig(name string) *RuntimeKubeconfig {
	if k, ok := rw.Kubeconfigs[name]; ok {
		return k
	}
	return nil
}

func (rk *RuntimeKubeconfig) Cluster(name string) *RuntimeCluster {
	if c, ok := rk.Clusters[name]; ok {
		return c
	}
	return nil
}

func (rk *RuntimeKubeconfig) AuthInfo(name string) *RuntimeAuthInfo {
	if a, ok := rk.AuthInfos[name]; ok {
		return a
	}
	return nil
}

func (rk *RuntimeKubeconfig) Context(name string) *RuntimeContext {
	if c, ok := rk.Contexts[name]; ok {
		return c
	}
	return nil
}

func (rk *RuntimeKubeconfig) Bytes() ([]byte, error) {
	return yaml.Marshal(rk.Config)
}

type RuntimeCluster struct {
	// Key  string
	Name string

	Cluster *api.Cluster
}

type RuntimeAuthInfo struct {
	// Key  string
	Name string

	AuthInfo *api.AuthInfo

	CredentialSource RuntimeCredentialSource
}

type RuntimeContext struct {
	// Key  string
	Name string

	ClusterKey  string
	AuthInfoKey string

	Cluster  *RuntimeCluster
	AuthInfo *RuntimeAuthInfo

	Namespace string

	Import *RuntimeImportRef

	Context *api.Context
}

type RuntimeContextRef struct {
	Workspace  *RuntimeWorkspace
	Kubeconfig *RuntimeKubeconfig
	Context    *RuntimeContext
}

type RuntimeCredentialSource interface {
	Type() string
}

type CredentialSourceType string

const (
	CredentialSourceNone  CredentialSourceType = "none"
	CredentialSourceExec  CredentialSourceType = "exec"
	CredentialSourceLogin CredentialSourceType = "login"
	CredentialSourceToken CredentialSourceType = "token"
)

type RuntimeLoginCredentialSource struct {
	Provider string

	Command string
	Args    []string
	Env     map[string]string

	Import RuntimeLoginImport
}

func (e *RuntimeLoginCredentialSource) Type() string {
	return string(CredentialSourceLogin)
}

type RuntimeLoginImport struct {
	Context  string
	Cluster  string
	AuthInfo string
}
