package editor

func (e *Editor) FindCallback(query []byte, key int) {
	if e.searchState.savedHl != nil && e.searchState.savedHlLine >= 0 && e.searchState.savedHlLine < len(e.rows) {
		copy(e.rows[e.searchState.savedHlLine].hl, e.searchState.savedHl)
		e.searchState.savedHl = nil
	}

	switch key {
	case '\r', '\x1b':
		e.searchState.lastMatch = -1
		e.searchState.direction = 1
		return
	case ARROW_RIGHT, ARROW_DOWN:
		e.searchState.direction = 1
	case ARROW_LEFT, ARROW_UP:
		e.searchState.direction = -1
	default:
		e.searchState.lastMatch = -1
		e.searchState.direction = 1
	}

	if e.searchState.lastMatch == -1 {
		e.searchState.direction = 1
	}
	current := e.searchState.lastMatch
	queryRunes := []rune(string(query))

	for range e.totalRows {
		current += e.searchState.direction
		switch current {
		case -1:
			current = e.totalRows - 1
		case e.totalRows:
			current = 0
		}

		row := &e.rows[current]
		match := runeIndexOf(row.render, queryRunes)
		if match != -1 {
			e.searchState.lastMatch = current
			e.cy = current
			e.cx = row.rxToCx(match)
			e.rowOffset = e.totalRows

			e.searchState.savedHlLine = current
			e.searchState.savedHl = make([]int, len(row.hl))
			copy(e.searchState.savedHl, row.hl)
			// Highlight the match
			for k := match; k < match+len(queryRunes) && k < len(row.hl); k++ {
				row.hl[k] = HL_MATCH
			}
			break
		}
	}
}

func (e *Editor) Find() {
	savedCx := e.cx
	savedCy := e.cy
	savedColOffset := e.colOffset
	savedRowOffset := e.rowOffset

	query := e.Prompt("Search: %s (Use ESC/Arrows/Enter)", e.FindCallback)

	if query == "" {
		e.cx = savedCx
		e.cy = savedCy
		e.colOffset = savedColOffset
		e.rowOffset = savedRowOffset
	}
}
