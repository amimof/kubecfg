package main

import (
	//"flag"

	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/duration"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/charmbracelet/bubbles/list"
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

	// Colors
	subtle = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	body   = lipgloss.AdaptiveColor{Light: "#343433", Dark: "#C1C6B2"}

	// The list pane that displays the files on the left side
	listPaneWidth  = 60
	listPaneHeight = 20
	listPane       = lipgloss.NewStyle().
			Width(listPaneWidth).
			Height(listPaneHeight).
			PaddingTop(1)

	errorStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#d75f00")).
			Foreground(lipgloss.Color("#eeeeee")).
			MarginRight(2)

	// The pane on the right that displays details about the currently selected file
	detailsPane = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Foreground(lipgloss.Color("#FAFAFA")).
			PaddingLeft(1).
			PaddingRight(1).
			Margin(0).
			Align(lipgloss.Left).
			Width(24).
			Height(20)

	// This is a header usually display inside of the details pane
	detailsHeader = lipgloss.NewStyle().
			Foreground(subtle).
			MarginRight(2)

	// This is text displayed containing actual values. Usually used together with detailsHeader
	detailsContent = lipgloss.NewStyle().
			Foreground(body).
			PaddingBottom(1)
)

type state int

// Top level app model
type model struct {
	state        state
	warningModel warningModel
	defaultModel defaultModel
}

type defaultModel struct {
	items    []Item
	list     list.Model
	selected int
}

type Item struct {
	config *api.Config
	info   os.FileInfo
	title  string
	desc   string
	raw    []byte
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

func newDefaultModel() defaultModel {
	return defaultModel{}
}

func New() *model {
	r := &model{
		//activeView:          VIEW_DEFAULT,
		state:        stateShowWarning,
		defaultModel: newDefaultModel(),
		warningModel: newWarningModel(),
	}
	return r
}

func (r *defaultModel) Add(i Item) {
	r.items = append(r.items, i)
}

func (r *defaultModel) List() []list.Item {
	var items []list.Item
	for _, i := range r.items {
		items = append(items, i)
	}
	return items
}

func (r defaultModel) Init() tea.Cmd {
	return nil
}

func (r *model) Init() tea.Cmd {
	return nil
}

func (r *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := lipgloss.NewStyle().Margin(1, 2).GetFrameSize()
		r.defaultModel.list.SetSize(msg.Width-h, msg.Height-v)
		w := msg.Width - listPaneWidth - h
		he := msg.Height - v
		detailsPane = detailsPane.Width(w)
		detailsPane = detailsPane.Height(he - 1)
	case OverwriteConfigMsg:
		r.state = stateShowDefault
		newDetaultModel, cmd := r.defaultModel.update(msg)
		r.defaultModel = newDetaultModel
		cmds = append(cmds, cmd)

	}

	switch r.state {
	case stateShowDefault:
		newDetaultModel, cmd := r.defaultModel.update(msg)
		r.defaultModel = newDetaultModel
		cmds = append(cmds, cmd)
	case stateShowWarning:
		newWarningModel, cmd := r.warningModel.update(msg)
		r.warningModel = newWarningModel
		cmds = append(cmds, cmd)
	}

	return r, tea.Batch(cmds...)
}

func (r defaultModel) update(msg tea.Msg) (defaultModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c":
			return r, tea.Quit
		case "down":
			if r.selected < len(r.items)-1 {
				r.selected++
			}
		case "up":
			if r.selected > 0 {
				r.selected--
			}
		}
	case tea.WindowSizeMsg:
		h, v := lipgloss.NewStyle().Margin(1, 2).GetFrameSize()
		r.list.SetSize(msg.Width-h, msg.Height-v)
		w := msg.Width - listPaneWidth - h
		he := msg.Height - v
		detailsPane = detailsPane.Width(w)
		detailsPane = detailsPane.Height(he - 1)
	}
	var cmd tea.Cmd
	r.list, cmd = r.list.Update(msg)
	return r, cmd
}

func (r defaultModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return r.update(msg)
}

func (r defaultModel) View() string {
	curContext := lipgloss.JoinVertical(
		lipgloss.Top,
		detailsHeader.Render("Current Context"),
		detailsContent.Render(r.getCurrentContext()),
	)
	contextCount := lipgloss.JoinVertical(
		lipgloss.Left,
		detailsHeader.Render("Contexts"),
		detailsContent.Render(r.getContextCount()),
	)
	clusterCount := lipgloss.JoinVertical(
		lipgloss.Left,
		detailsHeader.Render("Clusters"),
		detailsContent.Render(r.getClusterCount()),
	)
	userCount := lipgloss.JoinVertical(
		lipgloss.Left,
		detailsHeader.Render("Users"),
		detailsContent.Render(r.getUserCount()),
	)
	counts := lipgloss.JoinHorizontal(
		lipgloss.Left,
		contextCount,
		clusterCount,
		userCount,
	)
	modified := lipgloss.JoinVertical(
		lipgloss.Left,
		detailsHeader.Render("Last Modified"),
		detailsContent.Render(r.getLastModified()),
	)
	sizeBytes := lipgloss.JoinVertical(
		lipgloss.Left,
		detailsHeader.Render("Size"),
		detailsContent.Render(r.getSize()),
	)
	details := lipgloss.JoinVertical(
		lipgloss.Top,
		counts,
		curContext,
		modified,
		sizeBytes,
	)
	return lipgloss.JoinHorizontal(lipgloss.Top, listPane.Render(r.list.View()), detailsPane.Render(details))
}

func (r *model) View() string {
	var s string
	switch r.state {
	case stateShowWarning:
		return r.warningModel.View()
	case stateShowDefault:
		return r.defaultModel.View()
	}

	return s

}

func (r *defaultModel) getContextCount() string {
	str := strconv.Itoa(len(r.items[r.selected].config.Contexts))
	return str
}

func (r *defaultModel) getClusterCount() string {
	str := strconv.Itoa(len(r.items[r.selected].config.Clusters))
	return str
}

func (r *defaultModel) getUserCount() string {
	str := strconv.Itoa(len(r.items[r.selected].config.AuthInfos))
	return str
}

func (r *defaultModel) getCurrentContext() string {
	return r.items[r.selected].config.CurrentContext
}

func (r *defaultModel) getLastModified() string {
	return duration.HumanDuration(time.Since(r.items[r.selected].info.ModTime())) + " ago"
}

func ByteCountSI(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(b)/float64(div), "kMGTPE"[exp])
}

func (r *defaultModel) getSize() string {
	return ByteCountSI(int64(r.items[r.selected].info.Size()))
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

	res := New()

	// Evaluate if symlink ~/.kube/config already exists since we don't want to overwrite the users config file
	if err = CheckConfig(path.Join(p, "config")); errors.Is(err, ErrExist) {
		res.state = stateShowWarning
	}

	err = filepath.Walk(p, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			config, err := clientcmd.LoadFromFile(path)
			if err != nil {
				return nil
			}
			// Read file so that we can reference raw data later
			b, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			i := Item{
				config: config,
				info:   info,
				title:  info.Name(),
				desc:   fmt.Sprintf("%d contexts", len(config.Contexts)),
				raw:    b,
			}
			res.defaultModel.Add(i)
		}
		return nil
	})
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}

	res.defaultModel.list = list.New(res.defaultModel.List(), list.NewDefaultDelegate(), 0, 0)
	res.defaultModel.list.Title = p

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

var (
	choices    = []string{"Make a backup and create a symlink to it", "Delete it. I want a fresh start", "Do nothing, I'm too paranoid to know what to do"}
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Margin(1, 2).
			Width(70)
	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("230")).
			MarginLeft(2)
	inactiveStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212")).
			MarginLeft(2)
)

type warningModel struct {
	cursor int
	choice string
	errMsg string
}

func newWarningModel() warningModel {
	return warningModel{}
}

func (m warningModel) Init() tea.Cmd {
	return nil
}

type OverwriteConfigMsg string

func OverwriteConfigCmd() tea.Cmd {
	return func() tea.Msg {
		return OverwriteConfigMsg("Overwriting config")
	}
}

func (m warningModel) update(msg tea.Msg) (warningModel, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		case "enter":
			m.choice = choices[m.cursor]
			err := m.executeChoice(m.cursor)
			if err != nil {
				m.errMsg = err.Error()
			} else {
				cmds = append(cmds, OverwriteConfigCmd())
			}

		case "down", "j":
			m.cursor++
			if m.cursor >= len(choices) {
				m.cursor = 0
			}
		case "up", "k":
			m.cursor--
			if m.cursor < 0 {
				m.cursor = len(choices) - 1
			}
		}
	}
	return m, tea.Batch(cmds...)
}

func (m *warningModel) executeChoice(i int) error {
	p, err := parseArgs()
	if err != nil {
		return err
	}
	switch i {
	case 0:
		err := m.copyFile(path.Join(p, "config"), path.Join(p, fmt.Sprintf("%s_kubecfg-backup", "config")))
		if err != nil {
			return err
		}
		err = os.Remove(path.Join(p, "config"))
		if err != nil {
			return err
		}
		err = m.createLink(fmt.Sprintf("%s_kubecfg-backup", "config"), path.Join(p, "config"))
		if err != nil {
			return err
		}
	case 1:
	case 2:
	default:
	}
	return nil
}

func (m *warningModel) copyFile(src, dst string) error {
	// srcStat, err := os.Stat(src)
	// if err != nil {
	// 	return err
	// }
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}
	return nil
}

func (m *warningModel) createLink(target, link string) error {
	return os.Symlink(target, link)
}

func (m warningModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m.update(msg)
}

func (m warningModel) View() string {
	s := strings.Builder{}
	s.WriteString(titleStyle.Render("Kubecfg uses symbolic links to switch between different kubeconfig files. However, an existing kubeconfig file was found. What do you want me to do with it?") + "\n")
	for i := 0; i < len(choices); i++ {
		if m.cursor == i {
			s.WriteString(selectedStyle.Render("➜ " + choices[i]))
		} else {
			s.WriteString(inactiveStyle.Render("  " + choices[i]))
		}
		s.WriteString("\n")
	}

	if len(m.errMsg) > 0 {
		s.WriteString("\n  " + errorStyle.Render(fmt.Sprintf("Error: %s", m.errMsg)))
	}

	return s.String()
}
