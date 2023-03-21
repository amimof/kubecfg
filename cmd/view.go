package main

// import (
// 	"fmt"
// 	"strings"

// 	tea "github.com/charmbracelet/bubbletea"
// 	"github.com/charmbracelet/lipgloss"
// )

// var (
// 	choices    = []string{"Make a backup and create a symlink to it", "Delete it. I want a fresh start", "Do nothing, I'm too paranoid to know what to do"}
// 	titleStyle = lipgloss.NewStyle().
// 			Foreground(lipgloss.Color("#7D56F4")).
// 			Margin(1, 2)
// 	selectedStyle = lipgloss.NewStyle().
// 			Foreground(lipgloss.Color("230")).
// 			MarginLeft(2)
// 	inactiveStyle = lipgloss.NewStyle().
// 			Foreground(lipgloss.Color("212")).
// 			MarginLeft(2)
// )

// type overwriteConfigModel struct {
// 	cursor int
// 	choice string
// }

// func (m overwriteConfigModel) Init() tea.Cmd {
// 	return nil
// }

// type OverwriteConfigMsg string

// func OverwriteConfigCmd() tea.Cmd {
// 	return func() tea.Msg {
// 		return OverwriteConfigMsg("Overwriting config")
// 	}
// }

// func (m overwriteConfigModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
// 	switch msg := msg.(type) {
// 	case tea.KeyMsg:
// 		switch msg.String() {
// 		case "ctrl+c", "q", "esc":
// 			return m, tea.Quit
// 		case "enter":
// 			fmt.Println("enter!")
// 			m.choice = choices[m.cursor]
// 			return m, OverwriteConfigCmd()
// 		case "down", "j":
// 			m.cursor++
// 			if m.cursor >= len(choices) {
// 				m.cursor = 0
// 			}
// 		case "up", "k":
// 			m.cursor--
// 			if m.cursor < 0 {
// 				m.cursor = len(choices) - 1
// 			}
// 		}
// 	}
// 	return m, nil
// }

// func (m overwriteConfigModel) View() string {
// 	s := strings.Builder{}
// 	s.WriteString(titleStyle.Render("Kubecfg uses symbolic links to switch between different kubeconfig files. However, an existing kubeconfig file was found. What do you want me to do with it?") + "\n")
// 	for i := 0; i < len(choices); i++ {
// 		if m.cursor == i {
// 			s.WriteString(selectedStyle.Render("➜ " + choices[i]))
// 		} else {
// 			s.WriteString(inactiveStyle.Render("  " + choices[i]))
// 		}
// 		//s.WriteString(choices[i])
// 		s.WriteString("\n")
// 	}

// 	return s.String()
// }
