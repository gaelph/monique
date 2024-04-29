package viewport

// TODO: Separation of concerns!
// Currently, we hold all the lines in a variable,
// and then we filter them and put that in another,
// and then we decorate and pout that elsewhere
// and then we draw that.
// There should be a function to return the indices of the lines
// matching the filter.
// And another function to return the search matches.
// And another function to render the filtered lines (using then indices and
// the search matches).
// Only then we can render the viewport.
// Once all of this is done, we can decide if we jump to a search match
// or to the bottom of the viewport.

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// You generally won't need this unless you're processing stuff with
// complicated ANSI escape sequences. Turn it on if you notice flickering.
//
// Also keep in mind that high performance rendering only works for programs
// that use the full size of the terminal. We're enabling that below with
// tea.EnterAltScreen().
const useHighPerformanceRenderer = true

var (
	titleStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Right = "├"
		return lipgloss.NewStyle().BorderStyle(b).Padding(0, 1)
	}()

	infoStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Left = "┤"
		return titleStyle.Copy().BorderStyle(b)
	}()

	searchMatchStyle lipgloss.Style                   = lipgloss.NewStyle().
				Background(lipgloss.Color("9")). // red
				Foreground(lipgloss.Color("15")) // white

	activeMatchStyle                                   = lipgloss.NewStyle().
				Background(lipgloss.Color("10")). // green
				Foreground(lipgloss.Color("15"))  // white
)

type SetContentMsg struct {
	Content string
}

type AppendContentMsg struct {
	Content string
}

type ClearContentMsg struct{}

type fieldStatus int8

type searchMatch struct {
	text  string
	line  int
	start int
	end   int
}

const (
	FILTER fieldStatus = 0
	SEARCH fieldStatus = 1
)

type model struct {
	activeMatch   int
	content       string
	filterString  string
	searchString  string
	viewport      viewport.Model
	keyMap        KeyMap
	searchResults []searchMatch
	textinput     textinput.Model
	fieldStatus   fieldStatus
	ready         bool
}

func NewModel() model {
	m := model{
		activeMatch: -1,
		content:     string(""),
		textinput:   textinput.New(),
		keyMap:      DefaultKeyBinding(),
	}

	m.textinput.Focus()
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

	m.textinput.Prompt = m.inputPrompt()
	m.textinput, cmd = m.textinput.Update(msg)
	cmds = append(cmds, cmd)
	m = m.updateStrings()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		// Quit
		case key.Matches(msg, m.keyMap.Quit):
			return m.quit()

		// Reject the current search/filter
		case key.Matches(msg, m.keyMap.Blur):
			m = m.blur()

		// Accept the current search/filter
		case key.Matches(msg, m.keyMap.Accept):
			m = m.accept()

		// Start filter
		case key.Matches(msg, m.keyMap.Filter):
			if !m.hasFocus() {
				m = m.startFilter()
			}

		// Start search
		case key.Matches(msg, m.keyMap.Search):
			if !m.hasFocus() {
				m = m.startSearch()
			}

		// Move to the next search match
		case key.Matches(msg, m.keyMap.NextMatch):
			if !m.hasFocus() && m.hasSearchResults() {
				line := m.getNextSearchMatchLine()
				cmds = m.goToLine(line, cmds)
			}

		// Move to the previous search match
		case key.Matches(msg, m.keyMap.PreviousMatch):
			if !m.hasFocus() && m.hasSearchResults() {
				line := m.getPreviousMatchLine()
				cmds = m.goToLine(line, cmds)
			}
		}
		// Sets the content with filter and search highlights if any
		m.viewport.SetContent(m.decorateSearch(m.filterContent(m.content)))

		// Move to the next search match as typing
		if m.isSearching() && m.hasSearchResults() {
			if line := m.getActiveMatchLine(); line >= 0 {
				cmds = m.goToLine(line, cmds)
			}
		} else {
			// Move to the bottom of the viewport
			cmds = m.goToBottom(cmds)
		}

		// Sets the whole content at once
	case SetContentMsg:
		m.content = msg.Content
		m.viewport.SetContent(m.decorateSearch(m.filterContent(m.content)))
		if !m.isSearching() {
			cmds = m.goToBottom(cmds)
		}

		// Appends to the current content
	case AppendContentMsg:
		m.content += msg.Content
		m.viewport.SetContent(m.decorateSearch(m.filterContent(m.content)))
		if !m.isSearching() {
			cmds = m.goToBottom(cmds)
		}

		// Clears the whole content
	case ClearContentMsg:
		m.content = ""
		m.viewport.SetContent(m.decorateSearch(m.filterContent(m.content)))
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
	return fmt.Sprintf("%s\n%s", m.viewport.View(), m.footerView())
}

// MARK - Key Map Handlers

func (m model) quit() (model, tea.Cmd) {
	return m, tea.Quit
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
	m.textinput.SetValue(m.filterString)
	m.textinput.Prompt = m.inputPrompt()
	m.textinput.Focus()

	return m
}

func (m model) startSearch() model {
	m.fieldStatus = SEARCH
	m.textinput.SetValue(m.searchString)
	m.textinput.Prompt = m.inputPrompt()
	m.textinput.Focus()

	return m
}

// MARK - Search Privates

func (m *model) getActiveMatchLine() int {
	if len(m.searchResults) == 0 {
		return -1
	}

	if m.activeMatch < 0 {
		m.activeMatch = 0
	} else if m.activeMatch >= len(m.searchResults) {
		m.activeMatch = len(m.searchResults) - 1
	}
	return m.searchResults[m.activeMatch].line
}

func (m *model) getNextSearchMatchLine() int {
	if m.activeMatch < 0 {
		m.activeMatch = len(m.searchResults) - 1
	}
	m.activeMatch -= 1
	m.activeMatch = boundLoop(m.activeMatch, 0, len(m.searchResults)-1)

	return m.searchResults[m.activeMatch].line
}

func (m *model) getPreviousMatchLine() int {
	if m.activeMatch < 0 {
		m.activeMatch = len(m.searchResults) - 1
	}
	m.activeMatch += 1
	m.activeMatch = boundLoop(m.activeMatch, 0, len(m.searchResults)-1)

	return m.searchResults[m.activeMatch].line
}

func boundLoop(val, min, max int) int {
	if val < min {
		val = max
	}
	if val > max {
		val = min
	}

	return val
}

// MARK - Viewport Navigation

func (m model) goToLine(line int, cmds []tea.Cmd) []tea.Cmd {
	m.viewport.SetYOffset(line)
	m.viewport.YPosition = 0
	if useHighPerformanceRenderer {
		// Render (or re-render) the whole viewport. Necessary both to
		// initialize the viewport and when the window is resized.
		//
		// This is needed for high-performance rendering only.
		cmds = append(cmds, viewport.Sync(m.viewport))
	}

	return cmds
}

func (m model) goToTop(cmds []tea.Cmd) []tea.Cmd {
	m.viewport.GotoTop()
	m.viewport.YPosition = 0
	if useHighPerformanceRenderer {
		// Render (or re-render) the whole viewport. Necessary both to
		// initialize the viewport and when the window is resized.
		//
		// This is needed for high-performance rendering only.
		cmds = append(cmds, viewport.Sync(m.viewport))
	}

	return cmds
}

func (m model) goToBottom(cmds []tea.Cmd) []tea.Cmd {
	m.viewport.GotoBottom()
	m.viewport.YPosition = 0
	if useHighPerformanceRenderer {
		// Render (or re-render) the whole viewport. Necessary both to
		// initialize the viewport and when the window is resized.
		//
		// This is needed for high-performance rendering only.
		cmds = append(cmds, viewport.Sync(m.viewport))
	}

	return cmds
}

func (m model) resize(msg tea.WindowSizeMsg, cmds []tea.Cmd) (model, []tea.Cmd) {
	footerHeight := lipgloss.Height(m.footerView())
	verticalMarginHeight := footerHeight

	if !m.ready {
		// Since this program is using the full size of the viewport we
		// need to wait until we've received the window dimensions before
		// we can initialize the viewport. The initial dimensions come in
		// quickly, though asynchronously, which is why we wait for them
		// here.
		m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
		m.viewport.YPosition = 0
		m.viewport.HighPerformanceRendering = useHighPerformanceRenderer
		m.viewport.SetContent(m.decorateSearch(m.filterContent(m.content)))
		m.ready = true

		// This is only necessary for high performance rendering, which in
		// most cases you won't need.
		//
		// Render the viewport one line below the header.
		m.viewport.YPosition = 0
	} else {
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - verticalMarginHeight
	}
	if useHighPerformanceRenderer {
		// Render (or re-render) the whole viewport. Necessary both to
		// initialize the viewport and when the window is resized.
		//
		// This is needed for high-performance rendering only.
		cmds = append(cmds, viewport.Sync(m.viewport))
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

func (m model) hasSearchResults() bool {
	return len(m.searchResults) > 0 && m.activeMatch != -1
}

func (m model) inputPrompt() string {
	switch m.fieldStatus {
	case FILTER:
		return fmt.Sprintf("%s > ", m.fieldStatus.String())
	case SEARCH:
		// active := len(m.searchResults) - (m.activeMatch + 1)
		return fmt.Sprintf("%s [%d/%d] > ", m.fieldStatus.String(), m.activeMatch, len(m.searchResults))
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

func (f fieldStatus) String() string {
	switch f {
	case FILTER:
		return "Filter"
	case SEARCH:
		return "Search"
	}

	return ""
}

func (m model) footerView() string {
	return m.textinput.View()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// TODO: Only decorate what is visible
func (m *model) decorateSearch(content string) string {
	// lineStart := m.viewport.YOffset
	// lineEnd := lineStart + m.viewport.VisibleLineCount()

	if m.searchString == "" {
		m.activeMatch = -1
		m.searchResults = []searchMatch{}
		return content
	}

	reg, err := regexp.Compile(m.searchString)
	if err != nil {
		return content
	}

	m.searchResults = make([]searchMatch, 0)

	lines := strings.Split(content, "\n")

	for l, line := range lines {
		// Don't decorate lines outside of the viewport.
		// if l < lineStart || l >= lineEnd {
		// 	lines = append(lines, line)
		// 	continue
		// }

		locations := reg.FindAllStringIndex(line, -1)
		for _, location := range locations {
			searchResult := searchMatch{
				line:  l,
				start: location[0],
				end:   location[1],
				text:  line[location[0]:location[1]],
			}
			m.searchResults = append(m.searchResults, searchResult)
		}
	}

	if len(m.searchResults) == 0 {
		m.activeMatch = -1
		return content
	}

	if m.activeMatch == -1 {
		m.activeMatch = len(m.searchResults) - 1
	}

	for i, searchResult := range m.searchResults {
		builder := strings.Builder{}
		currentLine := lines[searchResult.line]
		// from 0 or end of previous match
		builder.WriteString(currentLine[0:searchResult.start])
		// match
		if m.activeMatch > 0 && m.activeMatch == i {
			styled := activeMatchStyle.Render(searchResult.text)
			builder.WriteString(styled)
		} else {
			styled := searchMatchStyle.Render(searchResult.text)
			builder.WriteString(styled)
		}
		builder.WriteString(currentLine[searchResult.end:])
		currentLine = builder.String()

		lines[searchResult.line] = currentLine
	}

	return strings.Join(lines, "\n")
}

func (m model) filterContent(content string) string {
	if m.filterString == "" {
		return content
	}

	reg, err := regexp.Compile(m.filterString)
	if err != nil {
		return content
	}

	lines := make([]string, 0)
	for _, line := range strings.Split(content, "\n") {
		if reg.Match([]byte(line)) {
			lines = append(lines, line)
		}
	}

	return strings.Join(lines, "\n")
}
