package main

import (
	//"flag"

	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/amimof/kubecfg/pkg/cfg"
	"github.com/spf13/pflag"
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
	ErrExist = errors.New("file already exists")
)

func usage() {
	fmt.Fprint(os.Stderr, "Usage:\n")
	fmt.Fprint(os.Stderr, "  kubecfg [PATH] <flags>\n\n")

	title := "kubecfg Kubernetes kubconfig manager"
	fmt.Fprintf(os.Stderr, "%s\n\n", title)
	desc := "List, search and switch between multiple kubeconfig files within a directory"
	if desc != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", desc)
	}
	fmt.Fprintln(os.Stderr, pflag.CommandLine.FlagUsages())
}

func main() {
	h, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("error trying to determine home dir", err)
		os.Exit(1)
	}
	kubeconfig := pflag.StringP("dir", "d", path.Join(h, ".kube/config"), "The symlink kubeconfig")
	globs := pflag.StringSliceP("glob", "g", []string{path.Join(h, ".kube/*.yaml")}, "List files matching a pattern to include. This flag can be used multiple times.")
	remove := pflag.BoolP("remove", "r", false, "Removes a context and it's referenced users and clusters")
	prune := pflag.BoolP("prune", "p", false, "Prunes orphaned users & clusters from kubeconfig")
	showver := pflag.Bool("version", false, "Print version")
	pflag.Usage = usage

	// Parse CLI flags
	pflag.Parse()

	// Show version if requested
	if *showver {
		fmt.Printf("Version: %s\nCommit: %s\nBranch: %s\nGoVersion: %s\n", VERSION, COMMIT, BRANCH, GOVERSION)
		return
	}

	c := cfg.New(*kubeconfig, *globs...)

	// Evaluate if symlink ~/.kube/config already exists since we don't want to overwrite the users config file
	if err = checkConfig(*kubeconfig); errors.Is(err, ErrExist) {
		fmt.Printf("regular file %s exists\n", *kubeconfig)
		return
	}

	// Remove operation
	if *remove {
		if err = c.Delete(); err != nil {
			fmt.Println(err)
		}
		return
	}

	// Pune operation
	if *prune {
		if err = c.Prune(); err != nil {
			fmt.Println(err)
		}
		return
	}

	if err = c.Set(); err != nil {
		fmt.Println(err)
	}
}

func checkConfig(p string) error {
	sPath, err := filepath.EvalSymlinks(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	// If target file is same as symlink then it is probably not a symlink
	if sPath == p {
		return ErrExist
	}
	return nil
}
