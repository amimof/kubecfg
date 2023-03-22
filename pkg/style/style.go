package style

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	Subtle = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	Body   = lipgloss.AdaptiveColor{Light: "#343433", Dark: "#C1C6B2"}

	// The list pane that displays the files on the left side
	ListPaneWidth  = 60
	ListPaneHeight = 20
	ListPane       = lipgloss.NewStyle().
			Width(ListPaneWidth).
			Height(ListPaneHeight).
			PaddingTop(1)

	ErrorStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#d75f00")).
			Foreground(lipgloss.Color("#eeeeee")).
			MarginRight(2)

	// The pane on the right that displays details about the currently selected file
	DetailsPane = lipgloss.NewStyle().
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
	DetailsHeader = lipgloss.NewStyle().
			Foreground(Subtle).
			MarginRight(2)

	// This is text displayed containing actual values. Usually used together with detailsHeader
	DetailsContent = lipgloss.NewStyle().
			Foreground(Body).
			PaddingBottom(1)
)
