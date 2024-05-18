package viewport

import (
	"regexp"
	"strings"
)

type searchMatch struct {
	text  string // The text that matched
	id    int    // An identifier for the match
	line  int    // The line in whole content where the match was found
	start int    // Start column of the match
	end   int    // End column of the match
}

func (m model) search(lines []string, indices []int) ([]searchMatch, int) {
	if m.searchString == "" {
		return []searchMatch{}, -1
	}

	reg, err := regexp.Compile(m.searchString)
	if err != nil {
		return []searchMatch{}, -1
	}

	searchResults := make([]searchMatch, 0)

	for _, lineNr := range indices {
		line := lines[lineNr]
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
	if (nextActiveMatch == -1 && len(searchResults) > 0) ||
		nextActiveMatch > len(searchResults)-1 {
		nextActiveMatch = len(searchResults) - 1
	}

	return searchResults, nextActiveMatch
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

func (m model) hasSearchResults() bool {
	return len(m.searchResults) > 0 && m.activeMatch != -1
}

func decorateLine(
	line string,
	searchResults []searchMatch,
	activeMatch int,
	lineNr, maxLine int,
) string {
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

	// This add line numbers. Should this be an option ?
	// lineWidth := len(strconv.Itoa(maxLine))
	// lineNrStr := strconv.Itoa(lineNr + 1)
	// if lineWidth > len(lineNrStr) {
	// 	lineNrStr = strings.Repeat(" ", lineWidth-len(lineNrStr)) + lineNrStr
	// }
	//
	// lineNrStr = lipgloss.NewStyle().
	// 	Foreground(softForeground).
	// 	PaddingLeft(1).
	// 	PaddingRight(2).
	// 	Render(lineNrStr)
	//
	// return lineNrStr + line

	return line
}

func (m *model) getActiveMatchLine() int {
	if len(m.searchResults) == 0 {
		return -1
	}

	if m.activeMatch < 0 {
		m.activeMatch = 0
	} else if m.activeMatch >= len(m.searchResults) {
		m.activeMatch = len(m.searchResults) - 1
	}

	actualLine := m.searchResults[m.activeMatch].line

	lineInBuffer := actualLine
	for i := 0; i < len(m.filteredIndices); i++ {
		if m.filteredIndices[i] == actualLine {
			lineInBuffer = i
			break
		}
	}

	return lineInBuffer
}

func (m *model) getNextActiveMatch() int {
	if m.activeMatch < 0 {
		m.activeMatch = len(m.searchResults) - 1
	}
	m.activeMatch -= 1
	m.activeMatch = clampLoop(m.activeMatch, 0, len(m.searchResults)-1)

	return m.activeMatch
}

func (m *model) getPreviousActiveMatch() int {
	if m.activeMatch < 0 {
		m.activeMatch = len(m.searchResults) - 1
	}
	m.activeMatch += 1
	m.activeMatch = clampLoop(m.activeMatch, 0, len(m.searchResults)-1)

	return m.activeMatch
}
