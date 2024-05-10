package viewport

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/gaelph/monique/mediator"
)

var (
	// Top bar with Monique: <command>
	titleStyle = func() lipgloss.Style {
		return lipgloss.NewStyle().
			Background(lipgloss.Color("5")). // magenta
			Foreground(lipgloss.Color("15")) // white
	}()

	helpLineStyle lipgloss.Style                    = lipgloss.NewStyle().
			Background(lipgloss.Color("15")). // magenta
			Foreground(lipgloss.Color("8"))

	// Style for a non-active search match
	searchMatchStyle lipgloss.Style                   = lipgloss.NewStyle().
				Background(lipgloss.Color("9")). // red
				Foreground(lipgloss.Color("15")) // white

		// Style for the active search match
	activeMatchStyle                                   = lipgloss.NewStyle().
				Background(lipgloss.Color("10")). // green
				Foreground(lipgloss.Color("15"))  // white
)

// Set the whole content at once
type SetContentMsg struct {
	Content string
}

// Append lines to the current content
type AppendContentMsg struct {
	Content string
}

// Clear the whole content
type ClearContentMsg struct{}

// Type of text input
type fieldStatus int8

func (f fieldStatus) String() string {
	switch f {
	case FILTER:
		return "Filter"
	case SEARCH:
		return "Search"
	}

	return ""
}

const (
	FILTER fieldStatus = 0
	SEARCH fieldStatus = 1
)

// Model holding the state of the application
type model struct {
	mediator        mediator.Mediator // Communication Hub
	command         string            // the command that was run
	searchString    string            // the string to search for (displays matches)
	filterString    string            // the string fo filter the results by (displays only matching lines)
	keyMap          KeyMap            // key bindings
	viewport        viewport.Model    // inner viewport component
	searchResults   []searchMatch     // the search results
	allLines        []string          // the whole content
	filteredIndices []int             // indices of the lines that match the filter string
	renderedLines   []string          // the rendered content (filtered with search decorations)
	textinput       textinput.Model   // inner text input component
	scrollPos       int               // current scroll position (although, it should match viewport.YOffset)
	activeMatch     int               // the index of the active search match (in searchResults)
	fieldStatus     fieldStatus       // current kind of input (filter or search)
	ready           bool              // whether the model is ready to be rendered
	showingHelp     bool
}

func NewModel(command string, mediator mediator.Mediator) model {
	m := model{
		command:     command,
		mediator:    mediator,
		activeMatch: -1,
		viewport:    viewport.New(0, 0),
		textinput:   textinput.New(),
		keyMap:      DefaultKeyBinding(),
	}

	// m.textinput.Focus()
	m.textinput.Prompt = m.inputPrompt()
	m.textinput.PromptStyle = m.textinput.PromptStyle.
		Bold(true)

	return m
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	// Set the prompt matching the current field status
	m.textinput.Prompt = m.inputPrompt()
	// Handle keyboard events on the text input
	m.textinput, cmd = m.textinput.Update(msg)
	cmds = append(cmds, cmd)
	// Update the search/filter strings with user input
	m = m.updateStrings()

	// TODO: check this behavior
	// we should only go to bottom appending if we already are there
	shouldBottom := m.viewport.ScrollPercent() < 1

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.ShowHelp):
			if !m.hasFocus() {
				m.showingHelp = true

				return m, tea.Batch(cmds...)
			}
		// Quit
		case key.Matches(msg, m.keyMap.Quit):
			return m.quit()

		// Restart the command
		case key.Matches(msg, m.keyMap.Restart):
			m.restart()
			return m, tea.Batch(cmds...)

		// Reject the current search/filter
		case key.Matches(msg, m.keyMap.Blur):
			if m.showingHelp {
				m.showingHelp = false

				return m, tea.Batch(cmds...)
			}

			m = m.blur()
			m.filteredIndices = m.applyFilter()
			m.renderedLines = m.renderContent()
			m.viewport.SetContent(strings.Join(m.renderedLines, "\n"))
			cmds = m.goToBottom(cmds)

			return m, tea.Batch(cmds...)

		// Accept the current search/filter
		case key.Matches(msg, m.keyMap.Accept):
			m = m.accept()

			return m, tea.Batch(cmds...)

		// Start filter
		case key.Matches(msg, m.keyMap.Filter):
			if !m.hasFocus() {
				m = m.startFilter()
				cmds = m.goToBottom(cmds)

				return m, tea.Batch(cmds...)
			}

		// Start search
		case key.Matches(msg, m.keyMap.Search):
			if !m.hasFocus() {
				m = m.startSearch()
				cmds = m.goToBottom(cmds)

				return m, tea.Batch(cmds...)
			}

		case key.Matches(msg, m.keyMap.HalfPageDown):
			if !m.viewport.AtBottom() {
				cmds = m.halfPageDown(cmds)
			}
			return m, tea.Batch(cmds...)

		case key.Matches(msg, m.keyMap.HalfPageUp):
			if !m.viewport.AtTop() {
				cmds = m.halfPageUp(cmds)
			}
			return m, tea.Batch(cmds...)

		// Move to the next search match
		case key.Matches(msg, m.keyMap.NextMatch):
			log.Printf("Next Match %+v\n", msg)
			if !m.hasFocus() && m.hasSearchResults() {
				cmds = m.goToNextMatch(cmds)
				return m, tea.Batch(cmds...)
			}

		// Move to the previous search match
		case key.Matches(msg, m.keyMap.PreviousMatch):
			log.Printf("Previous Match: %+v\n", msg)
			if !m.hasFocus() && m.hasSearchResults() {
				cmds = m.goToPreviousMatch(cmds)
				return m, tea.Batch(cmds...)
			}
		}

		m.filteredIndices = m.applyFilter()
		m.searchResults, m.activeMatch = m.search()
		m.renderedLines = m.renderContent()

		// Sets the content with filter and search highlights if any
		m.viewport.SetContent(strings.Join(m.renderedLines, "\n"))
		cmds = m.goToBottom(cmds)

	// Sets the whole content at once
	case SetContentMsg:
		m.allLines = strings.Split(msg.Content, "\n")

		m.filteredIndices = m.applyFilter()
		m.searchResults, m.activeMatch = m.search()
		m.renderedLines = m.renderContent()
		m.viewport.SetContent(strings.Join(m.renderedLines, "\n"))

		if shouldBottom {
			cmds = m.goToBottom(cmds)
		}

	// Appends to the current content
	case AppendContentMsg:
		lastLine := ""
		// pop the last line off
		if l := pop(&m.allLines); l != nil {
			lastLine = *l
		}

		lastLine += msg.Content
		newLines := strings.Split(lastLine, "\n")

		push(&m.allLines, newLines...)

		m.filteredIndices = m.applyFilter()
		m.searchResults, m.activeMatch = m.search()
		m.renderedLines = m.renderContent()
		m.viewport.SetContent(strings.Join(m.renderedLines, "\n"))

		if shouldBottom {
			cmds = m.goToBottom(cmds)
		}

	// Clears the whole content
	case ClearContentMsg:
		m.allLines = []string{}
		m.viewport.SetContent("")
		cmds = m.goToTop(cmds)

	// Resize the viewport
	case tea.WindowSizeMsg:
		m, cmds = m.resize(msg, cmds)
	}

	// Handle keyboard and mouse events in the viewport
	m.viewport, cmd = m.viewport.Update(msg)

	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}
	content := ""
	if m.showingHelp {
		content = m.helpView()
	} else {
		content = m.viewport.View()
	}
	return fmt.Sprintf("%s\n%s\n%s", m.headerView(), content, m.footerView())
}

// Returns all the line indices
func (m model) everything() []int {
	indices := make([]int, len(m.allLines))
	for i := range m.allLines {
		indices[i] = i
	}
	return indices
}

// Apply the filter and return the matching indices
func (m model) applyFilter() (indices []int) {
	if m.filterString == "" {
		return m.everything()
	}

	reg, err := regexp.Compile(m.filterString)
	if err != nil {
		return m.everything()
	}

	indices = make([]int, 0)
	for i, line := range m.allLines {
		if reg.Match([]byte(line)) {
			indices = append(indices, i)
		}
	}

	return indices
}

// MARK - Key Map Handlers

func (m model) quit() (model, tea.Cmd) {
	return m, tea.Quit
}

func (m model) restart() {
	if m.mediator != nil {
		m.mediator.SendRequestRestart()
	}
}

func (m model) blur() model {
	m = m.clearCurrentString()
	m.textinput.Blur()

	return m
}

func (m model) accept() model {
	m.textinput.Blur()

	return m
}

func (m model) startFilter() model {
	m.fieldStatus = FILTER
	m.textinput.Focus()
	m.textinput.SetValue(m.filterString)
	m.textinput.Prompt = m.inputPrompt()

	return m
}

func (m model) startSearch() model {
	m.fieldStatus = SEARCH
	m.textinput.Focus()
	m.textinput.SetValue(m.searchString)
	m.textinput.Prompt = m.inputPrompt()

	return m
}

func (m *model) goToNextMatch(cmds []tea.Cmd) []tea.Cmd {
	m.activeMatch = m.getNextActiveMatch()
	nextLine := m.getActiveMatchLine()

	m.renderedLines = m.renderContent()
	m.viewport.SetContent(strings.Join(m.renderedLines, "\n"))
	return m.goToLine(nextLine, cmds)
}

func (m *model) goToPreviousMatch(cmds []tea.Cmd) []tea.Cmd {
	m.activeMatch = m.getPreviousActiveMatch()
	nextLine := m.getActiveMatchLine()

	m.renderedLines = m.renderContent()
	m.viewport.SetContent(strings.Join(m.renderedLines, "\n"))
	return m.goToLine(nextLine, cmds)
}

// MARK - Viewport Navigation

func (m *model) goToMatch(match int, cmds []tea.Cmd) []tea.Cmd {
	line := m.searchResults[match].line
	return m.goToLine(line, cmds)
}

func (m *model) goToLine(line int, cmds []tea.Cmd) []tea.Cmd {
	log.Printf("🚀  ~ m/v/viewport.go:373 ~ line: %+v\n", line)
	log.Printf("🚀  ~ m/v/viewport.go:375 ~ m.scrollPos: %+v\n", m.scrollPos)
	if line < m.scrollPos {
		diffUp := m.scrollPos - line
		m.viewport.LineUp(diffUp)
		m.scrollPos -= diffUp
		m.viewport.SetYOffset(m.scrollPos)
	} else if line > m.scrollPos+m.viewport.Height {
		diffDown := line - (m.scrollPos + m.viewport.Height)
		m.viewport.LineDown(diffDown)
		m.scrollPos += diffDown
		m.viewport.SetYOffset(m.scrollPos)
	}
	log.Printf("🚀  ~ m/v/viewport.go:387 ~ m.viewport.YOffset: %+v\n", m.viewport.YOffset)
	log.Printf("🚀  ~ m/v/viewport.go:384 ~ m.scrollPos: %+v\n", m.scrollPos)

	log.Println(cmds)

	return cmds
}

func (m *model) halfPageUp(cmds []tea.Cmd) []tea.Cmd {
	m.viewport.HalfViewUp()
	m.scrollPos = m.viewport.YOffset

	return cmds
}

func (m *model) halfPageDown(cmds []tea.Cmd) []tea.Cmd {
	m.viewport.HalfViewDown()
	m.scrollPos = m.viewport.YOffset

	return cmds
}

func (m *model) goToTop(cmds []tea.Cmd) []tea.Cmd {
	m.scrollPos = 0
	m.viewport.GotoTop()

	return cmds
}

func (m *model) goToBottom(cmds []tea.Cmd) []tea.Cmd {
	m.viewport.GotoBottom()
	m.scrollPos = m.viewport.YOffset

	return cmds
}

func (m model) resize(msg tea.WindowSizeMsg, cmds []tea.Cmd) (model, []tea.Cmd) {
	headerHeight := lipgloss.Height(m.headerView())
	footerHeight := lipgloss.Height(m.footerView())
	verticalMarginHeight := footerHeight + headerHeight

	if !m.ready {
		// Since this program is using the full size of the viewport we
		// need to wait until we've received the window dimensions before
		// we can initialize the viewport. The initial dimensions come in
		// quickly, though asynchronously, which is why we wait for them
		// here.
		m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
		m.viewport.YPosition = headerHeight + 1
		m.viewport.HighPerformanceRendering = false

		m.ready = true

		// This is only necessary for high performance rendering, which in
		// most cases you won't need.
		//
		// Render the viewport one line below the header.
		m.viewport.YPosition = headerHeight + 1
	} else {
		m.viewport.Width = msg.Width
		m.viewport.YPosition = headerHeight + 1
		m.viewport.Height = msg.Height - verticalMarginHeight
	}

	return m, cmds
}

// MARK: - Utilities

func (m model) hasFocus() bool {
	return m.textinput.Focused()
}

func (m model) isSearching() bool {
	return m.fieldStatus == SEARCH
}

func (m model) inputPrompt() string {
	if !m.hasFocus() {
		return ""
	}

	switch m.fieldStatus {
	case FILTER:
		return fmt.Sprintf("%s > ", m.fieldStatus.String())
	case SEARCH:
		return fmt.Sprintf("%s > ", m.fieldStatus.String())
	}
	return "> "
}

func (m model) clearCurrentString() model {
	switch m.fieldStatus {
	case FILTER:
		m.filterString = ""
		m.textinput.SetValue(m.filterString)
	case SEARCH:
		m.searchString = ""
		m.textinput.SetValue(m.searchString)
		m.searchResults = []searchMatch{}
	}

	return m
}

func (m model) updateStrings() model {
	switch m.fieldStatus {
	case FILTER:
		m.filterString = m.textinput.Value()
	case SEARCH:
		if m.searchString != m.textinput.Value() {
			m.activeMatch = -1
			m.searchString = m.textinput.Value()
		}
	}

	return m
}

func (m model) headerView() string {
	title := fmt.Sprintf(" Monique: %s", m.command)
	helpText := "help [?] "
	space := strings.Repeat(" ", max(0, m.viewport.Width-lipgloss.Width(title)-lipgloss.Width(helpText)))

	return titleStyle.Render(fmt.Sprintf("%s%s%s", title, space, helpText))
}

func (m model) footerView() string {
	statusLine := ""
	if m.filterString != "" {
		statusLine = fmt.Sprintf("Filter: %s | ", m.filterString)
	}
	if m.searchString != "" {
		statusLine += fmt.Sprintf("Search: %s |", m.searchString)
	}

	help := ""
	if m.hasFocus() {
		help += "[esc] to cancel | [enter] to accept"
	} else {
		if m.hasSearchResults() {
			help += "[n/N] next/previous match | "
		}
		help += "[/] to search | [f] to filter"
	}

	space := strings.Repeat(" ", max(0, m.viewport.Width-lipgloss.Width(help)-lipgloss.Width(statusLine)))

	helpLine := helpLineStyle.Render(fmt.Sprintf("%s%s%s", statusLine, space, help))
	input := ""
	if m.hasFocus() {
		input = m.textinput.View()
	} else if m.hasSearchResults() {
		count := len(m.searchResults)
		n := count - (m.activeMatch)
		input = fmt.Sprintf("[%d of %d]", n, count)
	}

	return fmt.Sprintf("%s\n%s", helpLine, input)
}

func (m model) renderContent() []string {
	content := make([]string, len(m.filteredIndices))

	for i, lineNr := range m.filteredIndices {
		matches := m.searchResultsAtLine(lineNr)
		content[i] = decorateLine(m.allLines[lineNr], matches, m.activeMatch)
	}

	return content
}

func (m model) helpView() string {
	paragraphStyle := lipgloss.NewStyle().
		Background(lipgloss.AdaptiveColor{Light: "15", Dark: "8"}).
		PaddingTop(1).
		PaddingBottom(1).
		PaddingLeft(2).
		PaddingRight(2)

	blockStyle := lipgloss.NewStyle().
		Background(lipgloss.AdaptiveColor{Light: "15", Dark: "8"}).
		PaddingLeft(1).
		PaddingRight(1)

	headerStyle := lipgloss.NewStyle().
		Bold(true)

	separatorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "8", Dark: "7"})

	defaultKeys := strings.Join([]string{
		headerStyle.Render("General"),
		separatorStyle.Render("⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺"),
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
		separatorStyle.Render("⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺"),
		"[esc]       cancel",
		"[enter]     accept",
		"[ctrl+u]    clear field",
		"",
		"",
		"",
		headerStyle.Render("This help"),
		separatorStyle.Render("⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺"),
		"[esc]  exit",
	}, "\n")

	content := lipgloss.JoinHorizontal(lipgloss.Top, paragraphStyle.Render(defaultKeys), paragraphStyle.Render(inputKeys))
	content = lipgloss.Place(lipgloss.Width(content), lipgloss.Height(content), lipgloss.Center, lipgloss.Center, content, lipgloss.WithWhitespaceBackground(lipgloss.AdaptiveColor{Light: "15", Dark: "8"}))
	content = blockStyle.Render(content)

	return lipgloss.Place(
		m.viewport.Width, m.viewport.Height,
		lipgloss.Center, lipgloss.Center,
		content,
		// lipgloss.WithWhitespaceChars("⣿"), lipgloss.WithWhitespaceForeground(lipgloss.Color("8"))
	)
}
