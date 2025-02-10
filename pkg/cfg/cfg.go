package cfg

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/homedir"
)

type Cfg struct {
	// Cfg   *api.Config
	Path         string
	Globs        []string
	items        map[string]*Item
	SelectedItem string
}

type Item struct {
	config     *api.Config
	info       os.FileInfo
	title      string
	desc       string
	IsSelected bool
	fullPath   string
}

func (c *Cfg) GetItem(name string) *Item {
	if i, ok := c.items[name]; ok {
		return i
	}
	return nil
}

func (c *Cfg) AddItem(name string, i *Item) {
	c.items[name] = i
}

func (c *Cfg) ListItemsStr() []string {
	var items []string
	for _, i := range c.items {
		items = append(items, i.info.Name())
	}
	return items
}

func (c *Cfg) readInKubeConfig() error {
	kubeConfigs := []string{}
	for _, iglobs := range c.Globs {
		matches, err := filepath.Glob(iglobs)
		if err != nil {
			continue
		}
		kubeConfigs = append(kubeConfigs, matches...)
	}

	for _, kubeConfig := range kubeConfigs {
		if info, err := os.Stat(kubeConfig); err == nil {
			if !info.IsDir() {

				config, err := clientcmd.LoadFromFile(kubeConfig)
				if err != nil {
					return nil
				}
				i := NewItem(kubeConfig, config, info, info.Name(), fmt.Sprintf("%d contexts", len(config.Contexts)))
				if pa, err := filepath.EvalSymlinks(c.Path); err == nil {
					if pa == kubeConfig {
						i.IsSelected = true
						c.SelectedItem = info.Name()
					}
				}

				c.AddItem(info.Name(), i)
			}
		}
	}
	return nil
}

// Fzf uses fzf to display a fuzzy finder in stdout.
// Returns a string containing the selected item in the fzf result
func Fzf(in bytes.Buffer) (string, error) {
	// TODO: Explore if we can use preview in fzf
	cmd := exec.Command("/opt/homebrew/bin/fzf", "--ansi", "--height=~10")
	var out bytes.Buffer
	// var in bytes.Buffer
	cmd.Stdin = &in
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			return "", err
		}
	}

	res := strings.TrimSpace(out.String())
	if res == "" {
		return "", errors.New("nothing selected")
	}

	return res, nil
}

func FzfListKubeconfigs(c *Cfg) (string, error) {
	var in bytes.Buffer

	// Assemble items in Cfg
	err := c.readInKubeConfig()
	if err != nil {
		return "", err
	}

	// Write each kubeconfig to stdin
	for i, it := range c.ListItemsStr() {
		sym, err := filepath.EvalSymlinks(c.Path)
		if err != nil {
			return "", err
		}
		if path.Base(sym) == it {
			in.WriteString(color.GreenString(it))
		} else {
			in.WriteString(it)
		}
		if i != len(c.ListItemsStr())-1 {
			in.WriteString("\n")
		}
	}

	return Fzf(in)
}

// Set creates a new symlink to a kubeconfigfile overwriting any existing one
func (c *Cfg) Set() error {
	selected, err := FzfListKubeconfigs(c)
	if err != nil {
		return err
	}
	// Remove existing symlink to config so we don't run into an error
	if _, err := os.Stat(c.Path); !os.IsNotExist(err) {
		err := os.Remove(c.Path)
		if err != nil {
			return err
		}
	}

	i := c.GetItem(selected)

	// Create the symlink to config
	err = os.Symlink(i.fullPath, c.Path)
	if err != nil {
		return err
	}

	fmt.Printf("Switched kubeconfig to %s", selected)

	return nil
}

func FzfListContexts() (string, error) {
	var in bytes.Buffer

	kubeconfigPath := filepath.Join(homedir.HomeDir(), ".kube", "config")
	cfg, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return "", err
	}

	// Write each context to stdin
	for k := range cfg.Contexts {
		if k == cfg.CurrentContext {
			in.WriteString(color.GreenString(k))
		} else {
			in.WriteString(k)
		}
		in.WriteString("\n")
	}

	return Fzf(in)
}

func (c *Cfg) Delete() error {
	selected, err := FzfListContexts()
	if err != nil {
		return err
	}

	kubeconfigPath := filepath.Join(homedir.HomeDir(), ".kube", "config")
	cfg, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return err
	}

	cfg, err = RemoveContextFromKubeConfig(selected, cfg)
	if err != nil {
		return err
	}

	if err := clientcmd.WriteToFile(*cfg, kubeconfigPath); err != nil {
		return err
	}

	fmt.Printf("Deleted context %s\n", selected)

	return nil
}

func (c *Cfg) Prune() error {
	kubeconfigPath := filepath.Join(homedir.HomeDir(), ".kube", "config")
	cfg, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return err
	}

	removedClusters, err := RemoveOrphanedClustersFromKubeConfig(cfg)
	if err != nil {
		return err
	}

	removedUsers, err := RemoveOrphanedUsersFromKubeConfig(cfg)
	if err != nil {
		return err
	}

	if err := clientcmd.WriteToFile(*cfg, kubeconfigPath); err != nil {
		return err
	}

	fmt.Printf("Removed %d orphaned clusters\n", len(removedClusters))
	fmt.Printf("Removed %d orphaned users\n", len(removedUsers))

	return nil
}

func userIsUsed(user string, contexts map[string]*api.Context) bool {
	for _, v := range contexts {
		if user == v.AuthInfo {
			return true
		}
	}
	return false
}

func clusterIsUsed(cluster string, contexts map[string]*api.Context) bool {
	for _, v := range contexts {
		if cluster == v.Cluster {
			return true
		}
	}
	return false
}

// Remove a context from provided kubeconfig, including it's referenced user and context
func RemoveContextFromKubeConfig(context string, kubeconfig *api.Config) (*api.Config, error) {
	// Delete user and cluster associated with the context
	if _, ok := kubeconfig.Contexts[context]; ok {
		ai := kubeconfig.Contexts[context].AuthInfo
		cl := kubeconfig.Contexts[context].Cluster
		delete(kubeconfig.AuthInfos, ai)
		delete(kubeconfig.Clusters, cl)
		delete(kubeconfig.Contexts, context)
	}

	return kubeconfig, nil
}

// Remove orphaned clusters from provided kubeconfig
func RemoveOrphanedClustersFromKubeConfig(kubeconfig *api.Config) ([]string, error) {
	// Purge orphaned users & clusters
	delClusters := []string{}
	for k := range kubeconfig.Clusters {
		if !clusterIsUsed(k, kubeconfig.Contexts) {
			delClusters = append(delClusters, k)
		}
	}
	for _, delCluster := range delClusters {
		delete(kubeconfig.Clusters, delCluster)
	}

	return delClusters, nil
}

// Remove orphaned users from provided kubeconfig
func RemoveOrphanedUsersFromKubeConfig(kubeconfig *api.Config) ([]string, error) {
	delUsers := []string{}
	for k := range kubeconfig.AuthInfos {
		if !userIsUsed(k, kubeconfig.Contexts) {
			delUsers = append(delUsers, k)
		}
	}
	for _, delUser := range delUsers {
		delete(kubeconfig.AuthInfos, delUser)
	}

	return delUsers, nil
}

func New(p string, globs ...string) *Cfg {
	r := &Cfg{
		Path:  p,
		Globs: globs,
		items: map[string]*Item{},
	}
	return r
}

func NewItem(fullPath string, c *api.Config, info os.FileInfo, title, desc string) *Item {
	i := &Item{
		fullPath: fullPath,
		config:   c,
		info:     info,
		title:    info.Name(),
		desc:     fmt.Sprintf("%d contexts", len(c.Contexts)),
	}
	return i
}
