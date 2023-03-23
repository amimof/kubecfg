package main

import (
	//"flag"

	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"k8s.io/client-go/tools/clientcmd"

	"github.com/amimof/kubecfg/pkg/cfg"
	"github.com/amimof/kubecfg/pkg/style"
	"github.com/amimof/kubecfg/pkg/view"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/pflag"
)

const (
	VIEW_DEFAULT = iota
	VIEW_CONFIG_EXISTS
	VIEW_ERROR

	stateShowDefault state = iota
	stateShowWarning
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

type state int

// Top level app model
type model struct {
	state            state
	backupConfigView view.BackupConfigView
	mainView         view.MainView
}

func New(c *cfg.Cfg) *model {
	r := &model{
		state:            stateShowDefault,
		backupConfigView: view.NewBackupConfigView(c),
		mainView:         view.NewMainView(c),
	}
	return r
}

func (r *model) Init() tea.Cmd {
	return nil
}

func (r *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := lipgloss.NewStyle().Margin(1, 2).GetFrameSize()
		r.mainView.List.SetSize(msg.Width-h, msg.Height-v)
		w := msg.Width - style.ListPaneWidth - h
		he := msg.Height - v
		style.DetailsPane = style.DetailsPane.Width(w)
		style.DetailsPane = style.DetailsPane.Height(he - 1)
	case view.ChangeViewMsg:
		r.state = stateShowDefault
		newDetaultModel, cmd := r.mainView.UpdateView(msg)
		r.mainView = newDetaultModel
		cmds = append(cmds, cmd)
	}

	switch r.state {
	case stateShowDefault:
		newDetaultModel, cmd := r.mainView.UpdateView(msg)
		r.mainView = newDetaultModel
		cmds = append(cmds, cmd)
	case stateShowWarning:
		newbackupConfigView, cmd := r.backupConfigView.UpdateView(msg)
		r.backupConfigView = newbackupConfigView
		cmds = append(cmds, cmd)
	}

	return r, tea.Batch(cmds...)
}

func (r *model) View() string {
	var s string
	switch r.state {
	case stateShowWarning:
		return r.backupConfigView.View()
	case stateShowDefault:
		return r.mainView.View()
	}

	return s

}
func usage() {
	fmt.Fprint(os.Stderr, "Usage:\n")
	fmt.Fprint(os.Stderr, "  kubecfg [PATH] <flags>\n\n")

	title := "kubecfg Kubernetes kubconfig manager"
	fmt.Fprint(os.Stderr, title+"\n\n")
	desc := "List, search and switch between multiple kubeconfig files within a directory"
	if desc != "" {
		fmt.Fprintf(os.Stderr, desc+"\n\n")
	}
	fmt.Fprintln(os.Stderr, pflag.CommandLine.FlagUsages())
}

func parseArgs() (string, error) {
	if len(os.Args) >= 2 {
		return os.Args[1], nil
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return path.Join(h, ".kube/"), nil
}

func main() {
	p, err := parseArgs()
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}

	showver := pflag.Bool("version", false, "Print version")
	pflag.Usage = usage

	// Parse CLI flags
	pflag.Parse()

	// Show version if requested
	if *showver {
		fmt.Printf("Version: %s\nCommit: %s\nBranch: %s\nGoVersion: %s\n", VERSION, COMMIT, BRANCH, GOVERSION)
		return
	}

	res := New(&cfg.Cfg{
		Path: p,
	})

	// Evaluate if symlink ~/.kube/config already exists since we don't want to overwrite the users config file
	if err = CheckConfig(path.Join(p, "config")); errors.Is(err, ErrExist) {
		res.state = stateShowWarning
	}

	err = filepath.Walk(p, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			config, err := clientcmd.LoadFromFile(filePath)
			if err != nil {
				return nil
			}
			i := view.NewItem(config, info, info.Name(), fmt.Sprintf("%d contexts", len(config.Contexts)))
			if pa, err := filepath.EvalSymlinks(path.Join(p, "config")); err == nil {
				if pa == filePath {
					i.IsSelected = true
				}
			}
			res.mainView.AddItem(i)
		}
		return nil
	})
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}

	prog := tea.NewProgram(res, tea.WithAltScreen())
	if _, err := prog.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

}

func CheckConfig(p string) error {
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
