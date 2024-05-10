package viewport

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) helpView() string {
	defaultKeys := strings.Join([]string{
		headerStyle.Render("General"),
		separatorStyle.Render("⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯"),
		"[f]         filter",
		"[/]         search",
		"[ctrl+u]    scroll up",
		"[ctrl+d]    scroll down",
		"[ctrl+r]    restart the command",
		"[ctrl+c]    quit",
		"",
		"[n]         go to next search match",
		"[N]         go to previous search match",
	}, "\n")

	inputKeys := strings.Join([]string{
		headerStyle.Render("Search/Filter"),
		separatorStyle.Render("⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯"),
		"[esc]       cancel",
		"[enter]     accept",
		"[ctrl+u]    clear field",
		"",
		"",
		"",
		headerStyle.Render("This help"),
		separatorStyle.Render("⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯"),
		"[esc]  exit",
	}, "\n")

	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		paragraphStyle.Render(defaultKeys),
		paragraphStyle.Render(inputKeys),
	)
	// place the content in a block with a background color
	content = lipgloss.Place(
		lipgloss.Width(content), lipgloss.Height(content),
		lipgloss.Center, lipgloss.Center,
		content,
		lipgloss.WithWhitespaceBackground(softBackground),
	)

	content = blockStyle.Render(content)

	// center the content in the viewport
	return lipgloss.Place(
		m.viewport.Width, m.viewport.Height,
		lipgloss.Center, lipgloss.Center,
		content,
	)
}
