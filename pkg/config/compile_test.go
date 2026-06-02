package config

import (
	"os"
	"path/filepath"
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

func TestCompileExpandsBaseDirWithTilde(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	cfg := Config{
		BaseDir: "~/.kube",
		Kubeconfigs: map[string]*Kubeconfig{
			"demo": {
				Path: "@/target-kubeconfig.yaml",
			},
		},
	}

	runtime, err := NewCompiler().Compile(&cfg)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(homeDir, ".kube"), runtime.BaseDir)
	require.Equal(t, filepath.Join(homeDir, ".kube", "target-kubeconfig.yaml"), runtime.Kubeconfigs["demo"].Path)
}

func TestCompileExpandsKubeconfigPathWithTilde(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	cfg := Config{
		Kubeconfigs: map[string]*Kubeconfig{
			"demo": {
				Path: "~/target-kubeconfig.yaml",
			},
		},
	}

	runtime, err := NewCompiler().Compile(&cfg)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(homeDir, "target-kubeconfig.yaml"), runtime.Kubeconfigs["demo"].Path)
}

func TestCompileMergesLoginEnvFileWithoutMutatingProcessEnv(t *testing.T) {
	tempDir := t.TempDir()
	envFile := filepath.Join(tempDir, "login.env")
	err := os.WriteFile(envFile, []byte("FROM_FILE=file-value\nQUOTED=\"quoted value\"\n"), 0o600)
	require.NoError(t, err)

	_, existedBefore := os.LookupEnv("FROM_FILE")
	require.False(t, existedBefore)

	cfg := Config{
		Kubeconfigs: map[string]*Kubeconfig{
			"demo": {
				Path: "/tmp/demo",
				AuthInfos: map[string]*AuthInfo{
					"user": {
						Login: &LoginAuth{
							Command:             "login",
							Env:                 []string{"FROM_FILE=inline-value", "KEEP=keep-value"},
							EnvFile:             envFile,
							CopyFromContextName: "ctx",
						},
					},
				},
			},
		},
	}

	runtime, err := NewCompiler().Compile(&cfg)
	require.NoError(t, err)

	source, ok := runtime.Kubeconfigs["demo"].AuthInfos["user"].CredentialSource.(*RuntimeLoginCredentialSource)
	require.True(t, ok)
	require.Equal(t, "file-value", source.Env["FROM_FILE"])
	require.Equal(t, "keep-value", source.Env["KEEP"])
	require.Equal(t, "quoted value", source.Env["QUOTED"])
	_, existsAfter := os.LookupEnv("FROM_FILE")
	require.False(t, existsAfter)
}

func TestCompileMergesExecEnvFileWithEnvFileWinning(t *testing.T) {
	tempDir := t.TempDir()
	envFile := filepath.Join(tempDir, "exec.env")
	err := os.WriteFile(envFile, []byte("FROM_FILE=file-value\nQUOTED='quoted value'\n"), 0o600)
	require.NoError(t, err)

	cfg := Config{
		Kubeconfigs: map[string]*Kubeconfig{
			"demo": {
				Path: "/tmp/demo",
				AuthInfos: map[string]*AuthInfo{
					"user": {
						Exec: &ExecConfig{
							Command: "exec",
							Env: []ExecEnvVar{
								{Name: "FROM_FILE", Value: "inline-value"},
								{Name: "KEEP", Value: "keep-value"},
							},
							EnvFile: envFile,
						},
					},
				},
			},
		},
	}

	runtime, err := NewCompiler().Compile(&cfg)
	require.NoError(t, err)

	execCfg := runtime.Kubeconfigs["demo"].AuthInfos["user"].AuthInfo.Exec
	require.NotNil(t, execCfg)
	require.ElementsMatch(t, []map[string]string{
		{"name": "FROM_FILE", "value": "file-value"},
		{"name": "KEEP", "value": "keep-value"},
		{"name": "QUOTED", "value": "quoted value"},
	}, []map[string]string{
		{"name": execCfg.Env[0].Name, "value": execCfg.Env[0].Value},
		{"name": execCfg.Env[1].Name, "value": execCfg.Env[1].Value},
		{"name": execCfg.Env[2].Name, "value": execCfg.Env[2].Value},
	})
}

func TestCompileFailsWhenLoginEnvFileIsInvalid(t *testing.T) {
	tempDir := t.TempDir()
	envFile := filepath.Join(tempDir, "login.env")
	err := os.WriteFile(envFile, []byte("BROKEN_LINE\n"), 0o600)
	require.NoError(t, err)

	cfg := Config{
		Kubeconfigs: map[string]*Kubeconfig{
			"demo": {
				Path: "/tmp/demo",
				AuthInfos: map[string]*AuthInfo{
					"user": {
						Login: &LoginAuth{EnvFile: envFile},
					},
				},
			},
		},
	}

	_, err = NewCompiler().Compile(&cfg)
	require.EqualError(t, err, "authinfo \"user\" login env_file: invalid env line 1: missing '='")
}

func TestCompileFailsWhenExecEnvFileIsInvalid(t *testing.T) {
	tempDir := t.TempDir()
	envFile := filepath.Join(tempDir, "exec.env")
	err := os.WriteFile(envFile, []byte("BROKEN_LINE\n"), 0o600)
	require.NoError(t, err)

	cfg := Config{
		Kubeconfigs: map[string]*Kubeconfig{
			"demo": {
				Path: "/tmp/demo",
				AuthInfos: map[string]*AuthInfo{
					"user": {
						Exec: &ExecConfig{EnvFile: envFile},
					},
				},
			},
		},
	}

	_, err = NewCompiler().Compile(&cfg)
	require.EqualError(t, err, "authinfo \"user\" exec env_file: invalid env line 1: missing '='")
}
