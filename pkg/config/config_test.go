package config

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestViperUnmarshalEncryptedTokenCamelCase(t *testing.T) {
	v := viper.New()
	v.SetConfigType("yaml")

	err := v.ReadConfig(strings.NewReader(`
version: v1
kubeconfigs:
  demo:
    path: /tmp/demo
    auth_infos:
      user:
        encryptedToken: secret-value
`))
	require.NoError(t, err)

	var cfg Config
	require.NoError(t, v.Unmarshal(&cfg))
	require.Equal(t, "secret-value", cfg.Kubeconfig("demo").AuthInfo("user").EncryptedToken)
}
