package view

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/amimof/kubecfg/pkg/cfg"
	"github.com/amimof/kubecfg/pkg/style"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/client-go/tools/clientcmd/api"
)

type MainView struct {
	cfg      *cfg.Cfg
	items    []*Item
	List     list.Model
	selected int
	errMsg   string
}

type Item struct {
	config     *api.Config
	info       os.FileInfo
	title      string
	desc       string
	IsSelected bool
}

func (i Item) Title() string {
	s := i.title
	if i.IsSelected {
		s = lipgloss.NewStyle().Foreground(lipgloss.Color("#87ffaf")).Render(i.title + " •")
	}
	return s
}
func (i Item) Description() string {
	return i.desc
}
func (i Item) FilterValue() string {
	return i.title
}

func (r *MainView) Add(i *Item) {
	r.items = append(r.items, i)
}

func (r *MainView) ListItems() []list.Item {
	var items []list.Item
	for _, i := range r.items {
		items = append(items, i)
	}
	return items
}

func (r *MainView) NewItem(c *api.Config, info os.FileInfo, title, desc string) *Item {
	i := &Item{
		config: c,
		info:   info,
		title:  info.Name(),
		desc:   fmt.Sprintf("%d contexts", len(c.Contexts)),
	}
	//r.items = append(r.items, i)
	return i
}

func (r MainView) Init() tea.Cmd {
	return nil
}

func (r MainView) UpdateView(msg tea.Msg) (MainView, tea.Cmd) {
	var cmds []tea.Cmd
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
		case "enter":
			err := os.Remove(path.Join(r.cfg.Path, "config"))
			if err != nil {
				r.errMsg = err.Error()
			}
			err = os.Symlink(path.Join(r.cfg.Path, r.items[r.selected].info.Name()), path.Join(r.cfg.Path, "config"))
			if err != nil {
				r.errMsg = err.Error()
			}
			r.selectItem(r.selected)
		}
	case tea.WindowSizeMsg:
		h, v := lipgloss.NewStyle().Margin(1, 2).GetFrameSize()
		r.List.SetSize(msg.Width-h, msg.Height-v)
		w := msg.Width - style.ListPaneWidth - h
		he := msg.Height - v
		style.DetailsPane = style.DetailsPane.Width(w)
		style.DetailsPane = style.DetailsPane.Height(he - 1)
	}
	l, cmd := r.List.Update(msg)
	r.List = l
	cmds = append(cmds, cmd)
	return r, tea.Batch(cmds...)
}

func (r *MainView) selectItem(i int) {
	for _, i := range r.items {
		i.IsSelected = false
	}
	r.items[i].IsSelected = true
}

func (r MainView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return r.UpdateView(msg)
}

func (r MainView) View() string {
	curContext := lipgloss.JoinVertical(
		lipgloss.Top,
		style.DetailsHeader.Render("Current Context"),
		style.DetailsContent.Render(r.getCurrentContext()),
	)
	contextCount := lipgloss.JoinVertical(
		lipgloss.Left,
		style.DetailsHeader.Render("Contexts"),
		style.DetailsContent.Render(r.getContextCount()),
	)
	clusterCount := lipgloss.JoinVertical(
		lipgloss.Left,
		style.DetailsHeader.Render("Clusters"),
		style.DetailsContent.Render(r.getClusterCount()),
	)
	userCount := lipgloss.JoinVertical(
		lipgloss.Left,
		style.DetailsHeader.Render("Users"),
		style.DetailsContent.Render(r.getUserCount()),
	)
	counts := lipgloss.JoinHorizontal(
		lipgloss.Left,
		contextCount,
		clusterCount,
		userCount,
	)
	modified := lipgloss.JoinVertical(
		lipgloss.Left,
		style.DetailsHeader.Render("Last Modified"),
		style.DetailsContent.Render(r.getLastModified()),
	)
	sizeBytes := lipgloss.JoinVertical(
		lipgloss.Left,
		style.DetailsHeader.Render("Size"),
		style.DetailsContent.Render(r.getSize()),
	)

	details := lipgloss.JoinVertical(
		lipgloss.Top,
		counts,
		curContext,
		modified,
		sizeBytes,
		style.ErrorStyle.Render(r.errMsg),
	)
	return lipgloss.JoinHorizontal(lipgloss.Top, style.ListPane.Render(r.List.View()), style.DetailsPane.Render(details))
}

func (r *MainView) getContextCount() string {
	str := strconv.Itoa(len(r.items[r.selected].config.Contexts))
	return str
}

func (r *MainView) getClusterCount() string {
	str := strconv.Itoa(len(r.items[r.selected].config.Clusters))
	return str
}

func (r *MainView) getUserCount() string {
	str := strconv.Itoa(len(r.items[r.selected].config.AuthInfos))
	return str
}

func (r *MainView) getCurrentContext() string {
	return r.items[r.selected].config.CurrentContext
}

func (r *MainView) getLastModified() string {
	return duration.HumanDuration(time.Since(r.items[r.selected].info.ModTime())) + " ago"
}

func (r *MainView) getSize() string {
	return byteCountSI(int64(r.items[r.selected].info.Size()))
}

func byteCountSI(b int64) string {
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

func NewMainView(c *cfg.Cfg) MainView {
	return MainView{
		cfg: c,
	}
}
