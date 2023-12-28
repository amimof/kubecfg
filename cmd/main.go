package main

import (
	//"flag"

	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"k8s.io/client-go/tools/clientcmd"

	"github.com/amimof/kubecfg/pkg/cfg"
	"github.com/amimof/kubecfg/pkg/style"
	"github.com/amimof/kubecfg/pkg/view"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
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
	Cfg              *cfg.Cfg
}

func New(c *cfg.Cfg) *model {
	r := &model{
		Cfg:              c,
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
	// p, err := parseArgs()
	// if err != nil {
	// 	fmt.Printf("%s", err)
	// 	os.Exit(1)
	// }

	h, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("error trying to determine home dir", err)
		os.Exit(1)
	}
	kubecfgdir := pflag.StringP("dir", "d", path.Join(h, ".kube/"), "Directory containing kubeconfig files")
	showver := pflag.Bool("version", false, "Print version")
	interactive := pflag.BoolP("interactive", "i", false, "Interactive TUI mode 😎")
	pflag.Usage = usage

	// Parse CLI flags
	pflag.Parse()

	// Show version if requested
	if *showver {
		fmt.Printf("Version: %s\nCommit: %s\nBranch: %s\nGoVersion: %s\n", VERSION, COMMIT, BRANCH, GOVERSION)
		return
	}

	res := New(&cfg.Cfg{
		Path: *kubecfgdir,
	})

	// Evaluate if symlink ~/.kube/config already exists since we don't want to overwrite the users config file
	if err = CheckConfig(path.Join(*kubecfgdir, "config")); errors.Is(err, ErrExist) {
		res.state = stateShowWarning
		if !*interactive {
			fmt.Printf("regular file %s exists\n", path.Join(*kubecfgdir, "config"))
			return
		}
	}

	err = filepath.Walk(*kubecfgdir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			config, err := clientcmd.LoadFromFile(filePath)
			if err != nil {
				return nil
			}
			i := view.NewItem(config, info, info.Name(), fmt.Sprintf("%d contexts", len(config.Contexts)))
			if pa, err := filepath.EvalSymlinks(path.Join(*kubecfgdir, "config")); err == nil {
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

	if err = res.Run(*interactive); err != nil {
		fmt.Println(err)
	}

}

func (r *model) Run(interactive bool) error {
	if interactive {
		prog := tea.NewProgram(r, tea.WithAltScreen())
		if _, err := prog.Run(); err != nil {
			return err
		}
		os.Exit(0)
	}

	stdcfg := path.Join(r.Cfg.Path, "config")

	// TODO: Explore if we can use preview in fzf
	cmd := exec.Command("/opt/homebrew/bin/fzf", "--ansi", "--height=~10")
	var out bytes.Buffer
	var in bytes.Buffer
	cmd.Stdin = &in
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr

	// Write each kubeconfig to stdin ignoring 'config' which is a symlink
	for _, i := range r.mainView.ListItemsStr() {
		if i != "config" {
			sym, err := filepath.EvalSymlinks(stdcfg)
			if err != nil {
				return err
			}
			if path.Base(sym) == i {
				in.WriteString(color.GreenString(i + "\n"))
				continue
			}
			in.WriteString(i + "\n")
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
	if _, err := os.Stat(stdcfg); !os.IsNotExist(err) {
		err := os.Remove(stdcfg)
		if err != nil {
			return err
		}
	}

	// Create the symlink to config
	err := os.Symlink(path.Join(r.Cfg.Path, selected), stdcfg)
	if err != nil {
		return err
	}

	fmt.Printf("Switched context to %s", out.String())

	return nil
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
