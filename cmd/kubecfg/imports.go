package main

import (
	"fmt"
	"strings"

	"github.com/amimof/kubecfg/pkg/config"
	"k8s.io/client-go/tools/clientcmd/api"
)

func applyImportedContexts(rk *config.RuntimeKubeconfig) error {
	if rk == nil {
		return fmt.Errorf("runtime kubeconfig is nil")
	}

	for _, ctx := range rk.Contexts {
		if ctx.Import == nil {
			continue
		}

		source, ok := rk.LoginSources[ctx.Import.LoginSourceName]
		if !ok {
			return fmt.Errorf(
				"kubeconfig %q context %q import login source %q is missing",
				rk.Name,
				ctx.Name,
				ctx.Import.LoginSourceName,
			)
		}

		if source.ImportedConfig == nil {
			return fmt.Errorf(
				"kubeconfig %q context %q import login source %q has no imported config; run login first",
				rk.Name,
				ctx.Name,
				ctx.Import.LoginSourceName,
			)
		}

		if err := applyImportedContext(rk, ctx, source.ImportedConfig); err != nil {
			return err
		}
	}

	return nil
}

func applyImportedContext(rk *config.RuntimeKubeconfig, ctx *config.RuntimeContext, imported *api.Config) error {
	importedContext, ok := imported.Contexts[ctx.Import.ContextName]
	if !ok {
		return fmt.Errorf(
			"kubeconfig %q context %q imports missing context %q from login source %q",
			rk.Name,
			ctx.Name,
			ctx.Import.ContextName,
			ctx.Import.LoginSourceName,
		)
	}

	clusterName := firstNonEmptyString(ctx.Import.ClusterName, importedContext.Cluster)
	if clusterName == "" {
		return fmt.Errorf(
			"kubeconfig %q context %q imported context %q has no cluster",
			rk.Name,
			ctx.Name,
			ctx.Import.ContextName,
		)
	}

	authInfoName := firstNonEmptyString(ctx.Import.AuthInfoName, importedContext.AuthInfo)
	if authInfoName == "" {
		return fmt.Errorf(
			"kubeconfig %q context %q imported context %q has no authinfo",
			rk.Name,
			ctx.Name,
			ctx.Import.ContextName,
		)
	}

	importedCluster, ok := imported.Clusters[clusterName]
	if !ok {
		return fmt.Errorf(
			"kubeconfig %q context %q imports missing cluster %q from login source %q",
			rk.Name,
			ctx.Name,
			clusterName,
			ctx.Import.LoginSourceName,
		)
	}

	importedAuthInfo, ok := imported.AuthInfos[authInfoName]
	if !ok {
		return fmt.Errorf(
			"kubeconfig %q context %q imports missing authinfo %q from login source %q",
			rk.Name,
			ctx.Name,
			authInfoName,
			ctx.Import.LoginSourceName,
		)
	}

	rk.Config.Clusters[clusterName] = importedCluster.DeepCopy()
	rk.Config.AuthInfos[authInfoName] = importedAuthInfo.DeepCopy()

	targetContext := rk.Config.Contexts[ctx.Name]
	if targetContext == nil {
		if ctx.Context != nil {
			targetContext = ctx.Context.DeepCopy()
		} else {
			targetContext = &api.Context{}
		}
		rk.Config.Contexts[ctx.Name] = targetContext
	}

	targetContext.Cluster = clusterName
	targetContext.AuthInfo = authInfoName
	targetContext.Namespace = ctx.Namespace

	ctx.ClusterKey = clusterName
	ctx.AuthInfoKey = authInfoName
	ctx.Context.Cluster = clusterName
	ctx.Context.AuthInfo = authInfoName
	ctx.Cluster = &config.RuntimeCluster{Name: clusterName, Cluster: rk.Config.Clusters[clusterName]}
	ctx.AuthInfo = &config.RuntimeAuthInfo{Name: authInfoName, AuthInfo: rk.Config.AuthInfos[authInfoName]}

	return nil
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}

	return ""
}
