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

func (c *Cfg) Run() error {
	// TODO: Explore if we can use preview in fzf
	cmd := exec.Command("/opt/homebrew/bin/fzf", "--ansi", "--height=~10")
	var out bytes.Buffer
	var in bytes.Buffer
	cmd.Stdin = &in
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr

	// Assemble items in Cfg
	err := c.readInKubeConfig()
	if err != nil {
		return err
	}

	// Write each kubeconfig to stdin
	for i, it := range c.ListItemsStr() {
		sym, err := filepath.EvalSymlinks(c.Path)
		if err != nil {
			return err
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

	if err := cmd.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			return err
		}
	}

	selected := strings.TrimSpace(out.String())
	if selected == "" {
		return errors.New("nothing selected")
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

	fmt.Printf("Switched context to %s", out.String())

	return nil
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
