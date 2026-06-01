package config

import (
	"testing"

	"filippo.io/age"
	decryptpkg "github.com/amimof/kubecfg/pkg/decrypt"
	"github.com/stretchr/testify/require"
)

func TestCompileFailsWhenEncryptedFieldsExistWithoutDecryptor(t *testing.T) {
	cfg := Config{
		Kubeconfigs: map[string]*Kubeconfig{
			"demo": {
				Path: "/tmp/demo",
				AuthInfos: map[string]*AuthInfo{
					"user": {EncryptedToken: "ciphertext"},
				},
			},
		},
	}

	_, err := NewCompiler().Compile(&cfg)
	require.EqualError(t, err, "authinfo \"user\" contains encrypted fields; provide --identity-file or a passphrase")
}

func TestCompileEncryptedTokenOverridesPlaintextToken(t *testing.T) {
	identity, err := age.NewScryptIdentity("password")
	require.NoError(t, err)

	recipient, err := age.NewScryptRecipient("password")
	require.NoError(t, err)

	encryptor, err := decryptpkg.NewAgeEncryptor(recipient)
	require.NoError(t, err)

	encryptedToken, err := encryptor.EncryptString("decrypted-token")
	require.NoError(t, err)

	decryptor, err := decryptpkg.NewAgeDecryptor(identity)
	require.NoError(t, err)

	cfg := Config{
		Kubeconfigs: map[string]*Kubeconfig{
			"demo": {
				Path: "/tmp/demo",
				AuthInfos: map[string]*AuthInfo{
					"user": {
						Token:          "plain-token",
						EncryptedToken: encryptedToken,
					},
				},
			},
		},
	}

	runtime, err := NewCompiler(WithDecryptor(decryptor)).Compile(&cfg)
	require.NoError(t, err)
	require.Equal(t, "decrypted-token", runtime.Kubeconfigs["demo"].AuthInfos["user"].AuthInfo.Token)
}

func TestCompileResolvesKubeconfigCurrentContext(t *testing.T) {
	cfg := Config{
		Kubeconfigs: map[string]*Kubeconfig{
			"demo": {
				Path:           "/tmp/demo",
				CurrentContext: "admin",
				Clusters: map[string]*Cluster{
					"cluster": {Server: "https://example.com"},
				},
				AuthInfos: map[string]*AuthInfo{
					"user": {},
				},
				Contexts: map[string]*Context{
					"admin": {
						Cluster:  "cluster",
						AuthInfo: "user",
					},
				},
			},
		},
	}

	runtime, err := NewCompiler().Compile(&cfg)
	require.NoError(t, err)
	require.Equal(t, "admin", runtime.Kubeconfigs["demo"].CurrentContext.Name)
	require.Equal(t, "admin", runtime.Kubeconfigs["demo"].Config.CurrentContext)
}

func TestCompileFailsWhenKubeconfigCurrentContextIsMissing(t *testing.T) {
	cfg := Config{
		Kubeconfigs: map[string]*Kubeconfig{
			"demo": {
				Path:           "/tmp/demo",
				CurrentContext: "missing",
				Clusters: map[string]*Cluster{
					"cluster": {Server: "https://example.com"},
				},
				AuthInfos: map[string]*AuthInfo{
					"user": {},
				},
				Contexts: map[string]*Context{
					"admin": {
						Cluster:  "cluster",
						AuthInfo: "user",
					},
				},
			},
		},
	}

	_, err := NewCompiler().Compile(&cfg)
	require.EqualError(t, err, "kubeconfigs.demo.current_context references missing context \"missing\"")
}

func TestCompileKubeconfigDefaultContextOverridesCurrentContext(t *testing.T) {
	cfg := Config{
		DefaultWorkspace: "work",
		Workspaces: map[string]*Workspace{
			"work": {
				Kubeconfigs:       []string{"demo"},
				DefaultKubeconfig: "demo",
			},
		},
		Kubeconfigs: map[string]*Kubeconfig{
			"demo": {
				Path:           "/tmp/demo",
				CurrentContext: "admin",
				DefaultContext: "ops",
				Clusters: map[string]*Cluster{
					"cluster": {Server: "https://example.com"},
				},
				AuthInfos: map[string]*AuthInfo{
					"user": {},
				},
				Contexts: map[string]*Context{
					"admin": {
						Cluster:  "cluster",
						AuthInfo: "user",
					},
					"ops": {
						Cluster:  "cluster",
						AuthInfo: "user",
					},
				},
			},
		},
	}

	runtime, err := NewCompiler().Compile(&cfg)
	require.NoError(t, err)
	require.Equal(t, "work", runtime.DefaultWorkspace.Name)
	require.Equal(t, "demo", runtime.Workspaces["work"].DefaultKubeconfig.Name)
	require.Equal(t, "ops", runtime.Kubeconfigs["demo"].CurrentContext.Name)
	require.Equal(t, "ops", runtime.Kubeconfigs["demo"].Config.CurrentContext)
	require.Equal(t, "ops", runtime.Kubeconfigs["demo"].DefaultContext.Name)
}

func TestCompileFailsWhenKubeconfigDefaultContextIsMissing(t *testing.T) {
	cfg := Config{
		Kubeconfigs: map[string]*Kubeconfig{
			"demo": {
				Path:           "/tmp/demo",
				DefaultContext: "missing",
				Clusters: map[string]*Cluster{
					"cluster": {Server: "https://example.com"},
				},
				AuthInfos: map[string]*AuthInfo{
					"user": {},
				},
				Contexts: map[string]*Context{
					"admin": {
						Cluster:  "cluster",
						AuthInfo: "user",
					},
				},
			},
		},
	}

	_, err := NewCompiler().Compile(&cfg)
	require.EqualError(t, err, "kubeconfigs.demo.default_context references missing context \"missing\"")
}

func TestCompileKubeconfigDefaultNamespaceAppliesToContextsWithoutNamespace(t *testing.T) {
	cfg := Config{
		Kubeconfigs: map[string]*Kubeconfig{
			"demo": {
				Path:             "/tmp/demo",
				DefaultNamespace: "team-a",
				Clusters: map[string]*Cluster{
					"cluster": {Server: "https://example.com"},
				},
				AuthInfos: map[string]*AuthInfo{
					"user": {},
				},
				Contexts: map[string]*Context{
					"admin": {
						Cluster:  "cluster",
						AuthInfo: "user",
					},
					"ops": {
						Cluster:   "cluster",
						AuthInfo:  "user",
						Namespace: "team-b",
					},
				},
			},
		},
	}

	runtime, err := NewCompiler().Compile(&cfg)
	require.NoError(t, err)
	require.Equal(t, "team-a", runtime.Kubeconfigs["demo"].DefaultNamespace)
	require.Equal(t, "team-a", runtime.Kubeconfigs["demo"].Contexts["admin"].Namespace)
	require.Equal(t, "team-a", runtime.Kubeconfigs["demo"].Config.Contexts["admin"].Namespace)
	require.Equal(t, "team-b", runtime.Kubeconfigs["demo"].Contexts["ops"].Namespace)
	require.Equal(t, "team-b", runtime.Kubeconfigs["demo"].Config.Contexts["ops"].Namespace)
}

func TestCompileResolvesKubeconfigPathAgainstBaseDir(t *testing.T) {
	cfg := Config{
		BaseDir: "/tmp/kube",
		Kubeconfigs: map[string]*Kubeconfig{
			"demo": {
				Path: "@/target-kubeconfig.yaml",
			},
		},
	}

	runtime, err := NewCompiler().Compile(&cfg)
	require.NoError(t, err)
	require.Equal(t, "/tmp/kube/target-kubeconfig.yaml", runtime.Kubeconfigs["demo"].Path)
}
