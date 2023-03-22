package view

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/amimof/kubecfg/pkg/cfg"
	"github.com/amimof/kubecfg/pkg/style"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	choices = []string{
		"Make a backup and create a symlink to it",
		"Delete it. I want a fresh start",
		"Do nothing, I'm too paranoid to know what to do",
	}
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

type BackupConfigView struct {
	cfg    *cfg.Cfg
	cursor int
	choice string
	errMsg string
}

func NewBackupConfigView(c *cfg.Cfg) BackupConfigView {
	return BackupConfigView{
		cfg: c,
	}
}

func (m BackupConfigView) Init() tea.Cmd {
	return nil
}

func (m BackupConfigView) UpdateView(msg tea.Msg) (BackupConfigView, tea.Cmd) {
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
				cmds = append(cmds, ChangeViewCmd(0))
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

func (m *BackupConfigView) executeChoice(i int) error {
	switch i {
	case 0:
		err := m.copyFile(path.Join(m.cfg.Path, "config"), path.Join(m.cfg.Path, fmt.Sprintf("%s_kubecfg-backup", "config")))
		if err != nil {
			return err
		}
		err = os.Remove(path.Join(m.cfg.Path, "config"))
		if err != nil {
			return err
		}
		err = m.createLink(fmt.Sprintf("%s_kubecfg-backup", "config"), path.Join(m.cfg.Path, "config"))
		if err != nil {
			return err
		}
	case 1:
	case 2:
	default:
	}
	return nil
}

func (m *BackupConfigView) copyFile(src, dst string) error {
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

func (m *BackupConfigView) createLink(target, link string) error {
	return os.Symlink(target, link)
}

func (m BackupConfigView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m.UpdateView(msg)
}

func (m BackupConfigView) View() string {
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
		s.WriteString("\n  " + style.ErrorStyle.Render(fmt.Sprintf("Error: %s", m.errMsg)))
	}

	return s.String()
}
