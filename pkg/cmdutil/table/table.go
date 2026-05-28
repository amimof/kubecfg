package table

import (
	"fmt"
	"io"
	"regexp"
	"strings"
)

type Option func(*Table)

type Alignment int

const (
	AlignLeft Alignment = iota
	AlignRight
)

type Column struct {
	Header string
	Align  Alignment
}

type Table struct {
	columns    []Column
	rows       [][]string
	hasHeaders bool
}

func WithHeaders() Option {
	return func(t *Table) {
		t.hasHeaders = true
	}
}

func WithoutHeaders() Option {
	return func(t *Table) {
		t.hasHeaders = false
	}
}

func NewTable(columns []Column, opts ...Option) *Table {
	t := &Table{
		columns:    append([]Column(nil), columns...),
		hasHeaders: true,
	}

	for _, opt := range opts {
		opt(t)
	}

	return t
}

func (t *Table) AddRow(cells ...string) error {
	if len(cells) != len(t.columns) {
		return fmt.Errorf("expected %d cells, got %d", len(t.columns), len(cells))
	}

	row := append([]string(nil), cells...)
	t.rows = append(t.rows, row)
	return nil
}

func (t *Table) Render() []string {
	if len(t.columns) == 0 {
		return nil
	}

	widths := t.columnWidths()
	lines := make([]string, 0, len(t.rows)+1)

	if t.hasHeaders {
		header := make([]string, len(t.columns))
		for i, col := range t.columns {
			header[i] = cell(col.Header, widths[i], col.Align, i == len(t.columns)-1)
		}
		lines = append(lines, strings.Join(header, "  "))
	}

	for _, row := range t.rows {
		cells := make([]string, len(row))
		for i, value := range row {
			cells[i] = cell(value, widths[i], t.columns[i].Align, i == len(row)-1)
		}
		lines = append(lines, strings.Join(cells, "  "))
	}

	return lines
}

func (t *Table) WriteTo(w io.Writer) (int64, error) {
	var written int64
	for _, line := range t.Render() {
		d, err := fmt.Fprintln(w, line)
		if err != nil {
			return 0, err
		}
		written += int64(d)
	}

	return written, nil
}

func (t *Table) columnWidths() []int {
	widths := make([]int, len(t.columns))

	for i, col := range t.columns {
		if t.hasHeaders {
			widths[i] = visibleLength(col.Header)
		}
	}

	for _, row := range t.rows {
		for i, cell := range row {
			if width := visibleLength(cell); width > widths[i] {
				widths[i] = width
			}
		}
	}

	return widths
}

func cell(s string, width int, align Alignment, isLast bool) string {
	if isLast {
		if align == AlignRight {
			return padLeft(s, width)
		}
		return s
	}

	return pad(s, width, align)
}

func pad(s string, width int, align Alignment) string {
	visible := visibleLength(s)
	if visible >= width {
		return s
	}

	padding := strings.Repeat(" ", width-visible)
	if align == AlignRight {
		return padding + s
	}

	return s + padding
}

func padLeft(s string, width int) string {
	visible := visibleLength(s)
	if visible >= width {
		return s
	}

	return strings.Repeat(" ", width-visible) + s
}

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func visibleLength(s string) int {
	return len(ansiPattern.ReplaceAllString(s, ""))
}
