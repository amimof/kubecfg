package view

import (
	"github.com/amimof/kubecfg/pkg/cfg"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type newItemModel struct {
	textInput textinput.Model
	err       error
}

type NewItemCreatedMsg struct {
	Name string
}

func NewItemCreatedCmd(s string) tea.Cmd {
	return func() tea.Msg {
		return NewItemCreatedMsg{Name: s}
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
			return r, NewItemCreatedCmd(r.textInput.Value())
		}
	}
	r.textInput, cmd = r.textInput.Update(msg)
	return r, cmd
}

func (r newItemModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return r.UpdateView(msg)
}

func (r newItemModel) View() string {
	return r.textInput.View()
}

func (r newItemModel) Init() tea.Cmd {
	return textinput.Blink
}

func NewNewItemView(c *cfg.Cfg) newItemModel {
	ti := textinput.New()
	ti.Placeholder = "Pikachu"
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20

	return newItemModel{
		textInput: ti,
		err:       nil,
	}
}
