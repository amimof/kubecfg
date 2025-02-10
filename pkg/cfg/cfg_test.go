package cfg

import (
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/tools/clientcmd/api"
)

func Test_RemoveContext(t *testing.T) {
	inputKubeConfig := &api.Config{
		CurrentContext: "my-context-1",
		Contexts: map[string]*api.Context{
			"my-context-1": {
				Cluster:  "my-cluster-1",
				AuthInfo: "my-authinfo-1",
			},
			"my-context-2": {
				Cluster:  "my-cluster-2",
				AuthInfo: "my-authinfo-2",
			},
		},
		AuthInfos: map[string]*api.AuthInfo{
			"my-authinfo-1": {
				Token: "some-token",
			},
			"my-authinfo-2": {
				Token: "some-token",
			},
			"my-authinfo-3": {
				Token: "some-token",
			},
			"my-authinfo-4": {
				Token: "some-token",
			},
		},
		Clusters: map[string]*api.Cluster{
			"my-cluster-1": {
				Server:                   "https://localhost:8443",
				CertificateAuthorityData: []byte{},
				InsecureSkipTLSVerify:    true,
			},
			"my-cluster-2": {
				Server:                   "https://localhost:8443",
				CertificateAuthorityData: []byte{},
				InsecureSkipTLSVerify:    true,
			},
			"my-cluster-3": {
				Server:                   "https://localhost:8443",
				CertificateAuthorityData: []byte{},
				InsecureSkipTLSVerify:    true,
			},
		},
	}

	expectedKubeConfig := &api.Config{
		CurrentContext: "my-context-1",
		Contexts: map[string]*api.Context{
			"my-context-2": {
				Cluster:  "my-cluster-2",
				AuthInfo: "my-authinfo-2",
			},
		},
		AuthInfos: map[string]*api.AuthInfo{
			"my-authinfo-2": {
				Token: "some-token",
			},
			"my-authinfo-3": {
				Token: "some-token",
			},
			"my-authinfo-4": {
				Token: "some-token",
			},
		},
		Clusters: map[string]*api.Cluster{
			"my-cluster-2": {
				Server:                   "https://localhost:8443",
				CertificateAuthorityData: []byte{},
				InsecureSkipTLSVerify:    true,
			},
			"my-cluster-3": {
				Server:                   "https://localhost:8443",
				CertificateAuthorityData: []byte{},
				InsecureSkipTLSVerify:    true,
			},
		},
	}
	res, err := RemoveContextFromKubeConfig("my-context-1", inputKubeConfig)
	if err != nil {
		log.Fatal(err)
	}
	assert.Equal(t, expectedKubeConfig, res)
}

func Test_PruneOrphanedClusters(t *testing.T) {
	inputKubeConfig := &api.Config{
		CurrentContext: "my-context-1",
		Contexts: map[string]*api.Context{
			"my-context-1": {
				Cluster:  "my-cluster-1",
				AuthInfo: "my-authinfo-1",
			},
			"my-context-2": {
				Cluster:  "my-cluster-2",
				AuthInfo: "my-authinfo-2",
			},
		},
		AuthInfos: map[string]*api.AuthInfo{
			"my-authinfo-1": {
				Token: "some-token",
			},
			"my-authinfo-2": {
				Token: "some-token",
			},
			"my-authinfo-3": {
				Token: "some-token",
			},
			"my-authinfo-4": {
				Token: "some-token",
			},
		},
		Clusters: map[string]*api.Cluster{
			"my-cluster-1": {
				Server:                   "https://localhost:8443",
				CertificateAuthorityData: []byte{},
				InsecureSkipTLSVerify:    true,
			},
			"my-cluster-2": {
				Server:                   "https://localhost:8443",
				CertificateAuthorityData: []byte{},
				InsecureSkipTLSVerify:    true,
			},
			"my-cluster-3": {
				Server:                   "https://localhost:8443",
				CertificateAuthorityData: []byte{},
				InsecureSkipTLSVerify:    true,
			},
		},
	}

	expected := []string{"my-cluster-3"}
	res, err := RemoveOrphanedClustersFromKubeConfig(inputKubeConfig)
	if err != nil {
		log.Fatal(err)
	}
	assert.Equal(t, expected, res)
}

func Test_PruneOrphanedUsers(t *testing.T) {
	inputKubeConfig := &api.Config{
		CurrentContext: "my-context-1",
		Contexts: map[string]*api.Context{
			"my-context-1": {
				Cluster:  "my-cluster-1",
				AuthInfo: "my-authinfo-1",
			},
			"my-context-2": {
				Cluster:  "my-cluster-2",
				AuthInfo: "my-authinfo-2",
			},
		},
		AuthInfos: map[string]*api.AuthInfo{
			"my-authinfo-1": {
				Token: "some-token",
			},
			"my-authinfo-2": {
				Token: "some-token",
			},
			"my-authinfo-3": {
				Token: "some-token",
			},
			"my-authinfo-4": {
				Token: "some-token",
			},
		},
		Clusters: map[string]*api.Cluster{
			"my-cluster-1": {
				Server:                   "https://localhost:8443",
				CertificateAuthorityData: []byte{},
				InsecureSkipTLSVerify:    true,
			},
			"my-cluster-2": {
				Server:                   "https://localhost:8443",
				CertificateAuthorityData: []byte{},
				InsecureSkipTLSVerify:    true,
			},
			"my-cluster-3": {
				Server:                   "https://localhost:8443",
				CertificateAuthorityData: []byte{},
				InsecureSkipTLSVerify:    true,
			},
		},
	}

	expected := []string{"my-authinfo-3", "my-authinfo-4"}
	res, err := RemoveOrphanedUsersFromKubeConfig(inputKubeConfig)
	if err != nil {
		log.Fatal(err)
	}
	assert.Equal(t, expected, res)
}
