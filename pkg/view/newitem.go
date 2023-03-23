package view

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/amimof/kubecfg/pkg/cfg"
	"github.com/amimof/kubecfg/pkg/style"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"k8s.io/client-go/tools/clientcmd"
)

type newItemModel struct {
	cfg       *cfg.Cfg
	textInput textinput.Model
	err       error
}

type NewItemCreatedMsg struct {
	Item *Item
}

func NewItemCreatedCmd(i *Item) tea.Cmd {
	return func() tea.Msg {
		return NewItemCreatedMsg{Item: i}
	}
}

func (r newItemModel) UpdateView(msg tea.Msg) (newItemModel, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c":
			return r, tea.Quit
		case "enter":
			i, err := r.NewKubeconfig(path.Join(r.cfg.Path, r.textInput.Value()))
			if err != nil {
				r.err = err
				return r, nil
			}
			cmd = NewItemCreatedCmd(i)
			r.textInput.SetValue("")
			return r, cmd
		case "esc":
			cmd = NewItemCreatedCmd(nil)
			r.textInput.SetValue("")
			r.err = nil
			return r, cmd
		}
	}
	r.textInput, cmd = r.textInput.Update(msg)
	return r, cmd
}

func (r newItemModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return r.UpdateView(msg)
}

func (r newItemModel) View() string {
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("Create a new empty kubeconfig in %s\n\n", r.cfg.Path) + r.textInput.View())
	if r.err != nil {
		s.WriteString("\n\n" + style.ErrorStyle.Render(r.err.Error()))
	}
	return s.String()
}

func (r newItemModel) Init() tea.Cmd {

	return textinput.Blink
}

func (r *newItemModel) NewKubeconfig(p string) (*Item, error) {
	body := []byte(`apiVersion: v1
clusters: []
contexts: []
current-context: ""
kind: Config
preferences: {}
users: []
`)
	if _, err := os.Stat(p); !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("file %s already exists", p)
	}
	err := os.WriteFile(p, body, 0666)
	if err != nil {
		return nil, err
	}
	config, err := clientcmd.LoadFromFile(p)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	return NewItem(config, info, info.Name(), fmt.Sprintf("%d contexts", len(config.Contexts))), nil
}

func NewNewItemView(c *cfg.Cfg) newItemModel {
	ti := textinput.New()
	ti.Placeholder = "Enter filename"
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20

	return newItemModel{
		cfg:       c,
		textInput: ti,
		err:       nil,
	}
}
