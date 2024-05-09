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

const (
	FILTER fieldStatus = 0
	SEARCH fieldStatus = 1
)

type searchMatch struct {
	text  string // The text that matched
	id    int    // An identifier for the match
	line  int    // The line in whole content where the match was found
	start int    // Start column of the match
	end   int    // End column of the match
}

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

	m.textinput.Focus()
	m.textinput.Prompt = m.inputPrompt()
	m.textinput.PromptStyle = m.textinput.PromptStyle.
		Bold(true)

	return m
}

func (m *model) SetMediator(mediator mediator.Mediator) {
	m.mediator = mediator
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
	nextLine := -1

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		// Quit
		case key.Matches(msg, m.keyMap.Quit):
			return m.quit()

		case key.Matches(msg, m.keyMap.Restart):
			m.restart()
			return m, tea.Batch(cmds...)

		// Reject the current search/filter
		case key.Matches(msg, m.keyMap.Blur):
			m = m.blur()
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
			}
			cmds = m.goToBottom(cmds)

			return m, tea.Batch(cmds...)

		// Start search
		case key.Matches(msg, m.keyMap.Search):
			if !m.hasFocus() {
				m = m.startSearch()
			}
			cmds = m.goToBottom(cmds)

			return m, tea.Batch(cmds...)

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
			if !m.hasFocus() && m.hasSearchResults() {
				m.activeMatch = m.getNextActiveMatch()
				nextLine = m.getActiveMatchLine()

				m.renderedLines = m.renderContent()
				m.viewport.SetContent(strings.Join(m.renderedLines, "\n"))
				cmds = m.goToLine(nextLine, cmds)
			}

			return m, tea.Batch(cmds...)

		// Move to the previous search match
		case key.Matches(msg, m.keyMap.PreviousMatch):
			if !m.hasFocus() && m.hasSearchResults() {
				m.activeMatch = m.getPreviousActiveMatch()
				nextLine = m.getActiveMatchLine()

				m.renderedLines = m.renderContent()
				m.viewport.SetContent(strings.Join(m.renderedLines, "\n"))
				cmds = m.goToLine(nextLine, cmds)
			}

			return m, tea.Batch(cmds...)
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
		// TODO: Find a more efficient way to do this
		content := strings.Join(m.allLines, "\n") + msg.Content
		m.allLines = strings.Split(content, "\n")

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
	return fmt.Sprintf("%s\n%s\n%s", m.headerView(), m.viewport.View(), m.footerView())
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

func (m *model) getNextActiveMatch() int {
	if m.activeMatch < 0 {
		m.activeMatch = len(m.searchResults) - 1
	}
	m.activeMatch -= 1
	m.activeMatch = boundLoop(m.activeMatch, 0, len(m.searchResults)-1)

	return m.activeMatch
}

func (m *model) getPreviousActiveMatch() int {
	if m.activeMatch < 0 {
		m.activeMatch = len(m.searchResults) - 1
	}
	m.activeMatch += 1
	m.activeMatch = boundLoop(m.activeMatch, 0, len(m.searchResults)-1)

	return m.activeMatch
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

func (m *model) goToMatch(match int, cmds []tea.Cmd) []tea.Cmd {
	line := m.searchResults[match].line
	return m.goToLine(line, cmds)
}

func clamp(val, min, max int) int {
	if val < min {
		return min
	}
	if val > max {
		return max
	}

	return val
}

func (m *model) goToLine(line int, cmds []tea.Cmd) []tea.Cmd {
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

func (m model) hasSearchResults() bool {
	return len(m.searchResults) > 0 && m.activeMatch != -1
}

func (m model) inputPrompt() string {
	switch m.fieldStatus {
	case FILTER:
		return fmt.Sprintf("%s > ", m.fieldStatus.String())
	case SEARCH:
		active := m.activeMatch
		if active < 0 {
			active = 0
		}
		active = len(m.searchResults) - active
		return fmt.Sprintf("%s [%d/%d] > ", m.fieldStatus.String(), active, len(m.searchResults))
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

func (m model) headerView() string {
	title := titleStyle.Render(fmt.Sprintf(" Monique: %s", m.command))
	line := strings.Repeat(" ", max(0, m.viewport.Width-lipgloss.Width(title)))
	line = titleStyle.Render(line)
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
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

func (m model) search() ([]searchMatch, int) {
	if m.searchString == "" {
		return []searchMatch{}, -1
	}

	reg, err := regexp.Compile(m.searchString)
	if err != nil {
		return []searchMatch{}, -1
	}

	searchResults := make([]searchMatch, 0)

	for _, lineNr := range m.filteredIndices {
		// Don't decorate lines outside of the viewport.
		// if l < lineStart || l >= lineEnd {
		// 	lines = append(lines, line)
		// 	continue
		// }

		line := m.allLines[lineNr]
		locations := reg.FindAllStringIndex(line, -1)
		for _, location := range locations {
			searchResult := searchMatch{
				id:    len(searchResults),
				line:  lineNr,
				start: location[0],
				end:   location[1],
				text:  line[location[0]:location[1]],
			}
			searchResults = append(searchResults, searchResult)
		}
	}

	nextActiveMatch := m.activeMatch
	if nextActiveMatch == -1 && len(searchResults) > 0 {
		nextActiveMatch = len(searchResults) - 1
	}

	return searchResults, nextActiveMatch
}

func (m model) renderContent() []string {
	content := make([]string, len(m.filteredIndices))

	for i, lineNr := range m.filteredIndices {
		matches := m.searchResultsAtLine(lineNr)
		content[i] = decorateLine(m.allLines[lineNr], matches, m.activeMatch)
	}

	return content
}

func (m model) searchResultsAtLine(lineNr int) []searchMatch {
	searchResults := make([]searchMatch, 0)

	for _, match := range m.searchResults {
		if match.line == lineNr {
			searchResults = append(searchResults, match)
		}
	}

	return searchResults
}

func decorateLine(line string, searchResults []searchMatch, activeMatch int) string {
	offsets := make(map[int]int)
	for _, searchResult := range searchResults {
		builder := strings.Builder{}
		offset := offsets[searchResult.line]
		start := searchResult.start + offset
		end := searchResult.end + offset

		// from 0 or end of previous match
		builder.WriteString(line[0:start])
		// match
		if activeMatch >= 0 && activeMatch == searchResult.id {
			styled := activeMatchStyle.Render(searchResult.text)
			builder.WriteString(styled)
		} else {
			styled := searchMatchStyle.Render(searchResult.text)
			builder.WriteString(styled)
		}
		builder.WriteString(line[end:])
		newLine := builder.String()
		offset = len(newLine) - len(line)
		offsets[searchResult.line] += offset
		line = newLine
	}

	return line
}
