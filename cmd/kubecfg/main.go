package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/amimof/kubecfg/pkg/config"
)

const (
	VIEW_DEFAULT = iota
	VIEW_CONFIG_EXISTS
	VIEW_ERROR
)

var (
	// VERSION of the app. Is set when project is built and should never be set manually
	VERSION string
	// COMMIT is the Git commit currently used when compiling. Is set when project is built and should never be set manually
	COMMIT string
	// BRANCH is the Git branch currently used when compiling. Is set when project is built and should never be set manually
	BRANCH string
	// GOVERSION used to compile. Is set when project is built and should never be set manually
	GOVERSION string

	// Errors
	ErrExist    = errors.New("file already exists")
	ErrNotValid = errors.New("config not valid")

	cfg config.Config

	logLevel   string
	configFile string
	baseDir    string

	// Root command
	rootCmd = cobra.Command{
		Use:   "multikubectl",
		Short: "kubecfg Kubernetes kubconfig manager",
		Long:  "List, search and switch between multiple kubeconfig files within a directory",
	}
)

func init() {
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	viper.SetConfigFile(configFile)
	viper.SetConfigType("yaml")
	// viper.SetOptions(viper.WithEncoderRegistry)
}

func withConfig(run func(cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if err := loadConfig(true); err != nil {
			return err
		}
		return run(cmd, args)
	}
}

func loadConfig(validate bool) error {
	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) || errors.Is(err, os.ErrNotExist) {
			cfg = config.Config{
				Version:          "v1",
				DefaultWorkspace: "",
				DefaultNamespace: "",
				Kubeconfigs:      make(map[string]*config.Kubeconfig),
				Workspaces:       make(map[string]*config.Workspace),
			}
			if validate {
				if err := cfg.Validate(); err != nil {
					logrus.Fatalf("config validation error: %v", err)
					return err
				}
			}
			return nil
		}
		logrus.Fatalf("error reading config: %v", err)
		return err
	}
	if err := viper.Unmarshal(&cfg); err != nil {
		logrus.Fatalf("error decoding config into struct: %v", err)
		return err
	}
	if validate {
		if err := cfg.Validate(); err != nil {
			logrus.Fatalf("config validation error: %v", err)
			return err
		}
	}
	return nil
}

func main() {
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		lvl, err := logrus.ParseLevel(logLevel)
		if err != nil {
			return err
		}
		logrus.SetLevel(lvl)
		return nil
	}

	// Figure out path to default config file
	home, err := os.UserHomeDir()
	if err != nil {
		logrus.Fatalf("home directory cannot be determined: %v", err)
	}
	defaultConfigPath := filepath.Join(home, ".config", "kubecfg.yaml")
	defaultBaseDir := filepath.Join(home, ".kube")

	// Setup flags
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", defaultConfigPath, "config file")
	rootCmd.PersistentFlags().StringVarP(&baseDir, "base-dir", "b", defaultBaseDir, "kubeconfig base directory")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "number for the log level verbosity (debug, info, warn, error, fatal, panic)")

	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newWorkspacesCmd())
	rootCmd.AddCommand(newUseCmd())
	rootCmd.AddCommand(newLoginCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
