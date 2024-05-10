package viewport

import "github.com/charmbracelet/lipgloss"

const (
	Black        = "0"
	Red          = "1"
	Green        = "2"
	Orange       = "3"
	Blue         = "4"
	Purple       = "5"
	Cyan         = "6"
	Gray         = "7" // Should be lighter than 8
	BrightBlack  = "8" // Darker than 7
	BrightRed    = "9"
	BrightGreen  = "10"
	BrightOrange = "11"
	BrightBlue   = "12"
	BrightPurple = "13"
	BrightCyan   = "14"
	BrightGray   = "15"
)

var (
	softBackground lipgloss.AdaptiveColor = lipgloss.AdaptiveColor{Light: BrightGray, Dark: BrightBlack}
	softForeground lipgloss.AdaptiveColor = lipgloss.AdaptiveColor{Light: Gray, Dark: BrightGray}

	titleBackground lipgloss.Color = lipgloss.Color(Purple)
	titleForeground lipgloss.Color = lipgloss.Color(BrightGray)

	// Top bar with Monique: <command>
	titleStyle = func() lipgloss.Style {
		return lipgloss.NewStyle().
			Background(titleBackground). // magenta
			Foreground(titleForeground)  // white
	}()

	helpLineStyle lipgloss.Style = lipgloss.NewStyle().
			Background(softBackground).
			Foreground(softForeground)

	// Style for a non-active search match
	searchMatchStyle lipgloss.Style                         = lipgloss.NewStyle().
				Background(lipgloss.Color(Orange)).    // red
				Foreground(lipgloss.Color(BrightGray)) // white

		// Style for the active search match
	activeMatchStyle                                        = lipgloss.NewStyle().
				Background(lipgloss.Color(Red)).       // green
				Foreground(lipgloss.Color(BrightGray)) // white

	// Help View Styles
	paragraphStyle = lipgloss.NewStyle().
			Background(softBackground).
			PaddingTop(1).
			PaddingBottom(1).
			PaddingLeft(2).
			PaddingRight(2)

	blockStyle = lipgloss.NewStyle().
			Background(softBackground).
			PaddingLeft(1).
			PaddingRight(1)

	headerStyle = lipgloss.NewStyle().
			Bold(true)

	separatorStyle = lipgloss.NewStyle().
			Foreground(softForeground)
)
