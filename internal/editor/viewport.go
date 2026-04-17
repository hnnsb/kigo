package editor

func (v *Viewport) Scroll(totalRows int, rows []DisplayLine, contentCols int) {
	v.rx = 0
	if v.cy < totalRows {
		v.rx = rows[v.cy].cxToRx(v.cx)
	}

	if contentCols < 1 {
		contentCols = 1
	}

	if v.cy < v.rowOffset {
		v.rowOffset = v.cy
	}
	if v.cy >= v.rowOffset+v.screenRows {
		v.rowOffset = v.cy - v.screenRows + 1
	}

	if v.rx < v.colOffset {
		v.colOffset = v.rx
	}
	if v.rx >= v.colOffset+contentCols {
		v.colOffset = v.rx - contentCols + 1
	}
}

func (e *Editor) Scroll() {
	e.ensureRenderer()
	contentCols := e.renderer.contentWidthForCurrentView(e)
	if e.mode == EXPLORER_MODE {
		e.scrollExplorerWithPinnedRows(contentCols)
		return
	}
	e.Viewport.Scroll(e.totalRows, e.rows, contentCols)
}

func (e *Editor) scrollExplorerWithPinnedRows(contentCols int) {
	e.rx = 0
	if e.cy < e.totalRows {
		e.rx = e.rows[e.cy].cxToRx(e.cx)
	}

	if contentCols < 1 {
		contentCols = 1
	}

	if e.rx < e.colOffset {
		e.colOffset = e.rx
	}
	if e.rx >= e.colOffset+contentCols {
		e.colOffset = e.rx - contentCols + 1
	}

	if e.screenRows <= explorerPinnedRows {
		e.rowOffset = 0
		return
	}

	firstScrollableRow := explorerPinnedRows
	visibleScrollableRows := max(e.screenRows-explorerPinnedRows, 1)

	if e.rowOffset < firstScrollableRow {
		e.rowOffset = firstScrollableRow
	}

	if e.cy < firstScrollableRow {
		e.rowOffset = firstScrollableRow
	} else {
		if e.cy < e.rowOffset {
			e.rowOffset = e.cy
		}
		if e.cy >= e.rowOffset+visibleScrollableRows {
			e.rowOffset = e.cy - visibleScrollableRows + 1
		}
	}

	maxRowOffset := max(e.totalRows-visibleScrollableRows, firstScrollableRow)
	if e.rowOffset > maxRowOffset {
		e.rowOffset = maxRowOffset
	}
}

func (v *Viewport) MoveCursor(key int, totalRows int, rows []DisplayLine) {
	var row *DisplayLine
	if v.cy >= totalRows {
		row = nil
	} else {
		row = &rows[v.cy]
	}

	switch key {
	case ARROW_LEFT:
		if v.cx != 0 {
			v.cx--
		} else if v.cy > 0 {
			v.cy--
			v.cx = len(rows[v.cy].chars)
		}
	case ARROW_RIGHT:
		if row != nil && v.cx < len(row.chars) {
			v.cx++
		} else if row != nil && v.cx == len(row.chars) {
			v.cy++
			v.cx = 0
		}
	case ARROW_UP:
		if v.cy != 0 {
			v.cy--
		}
	case ARROW_DOWN:
		if v.cy < totalRows {
			v.cy++
		}
	case CTRL_ARROW_LEFT:
		if row != nil {
			for v.cx > 0 && isSeparator(int(row.chars[v.cx-1])) {
				v.cx--
			}
			for v.cx > 0 && !isSeparator(int(row.chars[v.cx-1])) {
				v.cx--
			}
		}
	case CTRL_ARROW_RIGHT:
		if row != nil {
			rowlen := len(row.chars)
			for v.cx < rowlen && isSeparator(int(row.chars[v.cx])) {
				v.cx++
			}
			for v.cx < rowlen && !isSeparator(int(row.chars[v.cx])) {
				v.cx++
			}
		}
	}

	if v.cy >= totalRows {
		row = nil
	} else {
		row = &rows[v.cy]
	}
	rowlen := 0
	if row != nil {
		rowlen = len(row.chars)
	}
	if v.cx > rowlen {
		v.cx = rowlen
	}
}

func (e *Editor) MoveCursor(key int) {
	e.Viewport.MoveCursor(key, e.totalRows, e.rows)
}
