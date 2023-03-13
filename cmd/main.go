package main

import (
	//"flag"

	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/list"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/pflag"
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
	cfgPath   string
)

type Item struct {
	config *api.Config
	info   os.FileInfo
	title  string
	desc   string
}

func (i Item) Title() string {
	return i.title
}
func (i Item) Description() string {
	return i.desc
}
func (i Item) FilterValue() string {
	return i.title
}

type Result struct {
	items []Item
	list  list.Model
}

func (r *Result) Add(i Item) {
	r.items = append(r.items, i)
}

func (r *Result) List() []list.Item {
	var items []list.Item
	for _, i := range r.items {
		items = append(items, i)
	}
	return items
}

func (r *Result) Init() tea.Cmd {
	return nil
}

func (r *Result) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return r, tea.Quit
		}
	case tea.WindowSizeMsg:
		h, v := lipgloss.NewStyle().Margin(1, 2).GetFrameSize()
		r.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	r.list, cmd = r.list.Update(msg)
	return r, cmd
}

func (r *Result) View() string {
	return lipgloss.NewStyle().Margin(1, 2).Render(r.list.View())
}

func init() {
	pflag.StringVar(&cfgPath, "path", "~/.kube/", "Path to a folder containing kubeconfig files. The program will search within directories for valid kubeconfigs.")
}

func usage() {
	fmt.Fprint(os.Stderr, "Usage:\n")
	fmt.Fprint(os.Stderr, "  kubecfg [OPTIONS]\n\n")

	title := "kubecfg Kubernetes kubconfig manager"
	fmt.Fprint(os.Stderr, title+"\n\n")
	desc := "List, search and switch between multiple kubeconfig files within a directory"
	if desc != "" {
		fmt.Fprintf(os.Stderr, desc+"\n\n")
	}
	fmt.Fprintln(os.Stderr, pflag.CommandLine.FlagUsages())
}

func main() {

	showver := pflag.Bool("version", false, "Print version")
	pflag.Usage = usage

	// Parse CLI flags
	pflag.Parse()

	// Show version if requested
	if *showver {
		fmt.Printf("Version: %s\nCommit: %s\nBranch: %s\nGoVersion: %s\n", VERSION, COMMIT, BRANCH, GOVERSION)
		return
	}

	res := &Result{}

	err := filepath.Walk(cfgPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			config, err := clientcmd.LoadFromFile(path)
			if err != nil {
				return nil
			}
			i := Item{
				config: config,
				info:   info,
				title:  info.Name(),
				desc:   fmt.Sprintf("%d contexts", len(config.Contexts)),
			}
			res.Add(i)
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	res.list = list.New(res.List(), list.NewDefaultDelegate(), 0, 0)
	res.list.Title = fmt.Sprintf("%d kubeconfigs in %s", len(res.items), cfgPath)

	p := tea.NewProgram(res, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

	// log.Printf("Found %d kubeconfigs in %s", len(results), cfgPath)
	// for _, r := range results {
	// 	log.Printf("\t%s", r.info.Name())
	// }
}
