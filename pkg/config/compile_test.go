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

	_, err := NewCompiler("").Compile(&cfg)
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

	runtime, err := NewCompiler("", WithDecryptor(decryptor)).Compile(&cfg)
	require.NoError(t, err)
	require.Equal(t, "decrypted-token", runtime.Kubeconfigs["demo"].AuthInfos["user"].AuthInfo.Token)
}
