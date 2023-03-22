package view

import tea "github.com/charmbracelet/bubbletea"

type ChangeViewMsg int

func ChangeViewCmd(i int) tea.Cmd {
	return func() tea.Msg {
		return ChangeViewMsg(i)
	}
}
