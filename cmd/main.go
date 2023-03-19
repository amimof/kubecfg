package main

import (
	//"flag"

	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/charmbracelet/bubbles/list"
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

var (

	// Colors
	subtle = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	body   = lipgloss.AdaptiveColor{Light: "#343433", Dark: "#C1C6B2"}

	// The list pane that displays the files on the left side
	listPaneWidth  = 60
	listPaneHeight = 20
	listPane       = lipgloss.NewStyle().
			Width(listPaneWidth).
			Height(listPaneHeight)

	// The pane on the right that displays details about the currently selected file
	detailsPane = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Foreground(lipgloss.Color("#FAFAFA")).
		//Background(lipgloss.Color("#7D56F4")).
		//PaddingTop(2).
		PaddingLeft(1).
		PaddingRight(1).
		Margin(0).
		//Padding(2).
		Align(lipgloss.Left).
		Width(24).
		Height(20)

	//
	detailsHeader = lipgloss.NewStyle().
			Foreground(subtle).
			MarginRight(2)

	detailsContent = lipgloss.NewStyle().
			Foreground(body).
			PaddingBottom(1)
)

type Result struct {
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
		switch keypress := msg.String(); keypress {
		case "ctrl+c":
			return r, tea.Quit
		case "down":
			r.selected++
		case "up":
			r.selected--
		}
	case tea.WindowSizeMsg:
		h, v := lipgloss.NewStyle().Margin(1, 2).GetFrameSize()
		r.list.SetSize(msg.Width-h, msg.Height-v)
		w := msg.Width - listPaneWidth - h
		detailsPane = detailsPane.Width(w)
	}

	var cmd tea.Cmd
	r.list, cmd = r.list.Update(msg)
	return r, cmd
}

func (r *Result) View() string {

	//const width = 70

	// vp := viewport.New(0, 20)
	// vp.Style = lipgloss.NewStyle().
	// 	BorderStyle(lipgloss.RoundedBorder()).
	// 	BorderForeground(lipgloss.Color("62")).
	// 	PaddingRight(2)

	// renderer, _ := glamour.NewTermRenderer(
	// 	glamour.WithAutoStyle(),
	// 	//glamour.WithWordWrap(width),
	// )

	// str, _ := renderer.Render("```yaml\n" + string(r.items[r.selected].raw) + "\n```")

	//vp.SetContent(str)

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

	//return lipgloss.NewStyle().Margin(1, 2).Render()
}

func (r *Result) getContextCount() string {
	str := strconv.Itoa(len(r.items[r.selected].config.Contexts))
	return str
}

func (r *Result) getClusterCount() string {
	str := strconv.Itoa(len(r.items[r.selected].config.Clusters))
	return str
}

func (r *Result) getUserCount() string {
	str := strconv.Itoa(len(r.items[r.selected].config.AuthInfos))
	return str
}

func (r *Result) getCurrentContext() string {
	return r.items[r.selected].config.CurrentContext
}

func (r *Result) getLastModified() string {
	return r.items[r.selected].info.ModTime().String()
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

func (r *Result) getSize() string {
	return ByteCountSI(int64(r.items[r.selected].info.Size()))
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
