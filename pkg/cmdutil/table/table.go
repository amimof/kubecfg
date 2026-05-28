package table

type Option func(*Table)

type Column struct{}

type Table struct {
	columns     []Column
	hasHeader   bool
	headerStr   string
	maxServices int
}

// WithHeader sets a header line that will be rendered once at the start
// The header can use template syntax if it contains {{ }}, otherwise it's treated as plain text
func WithHeader(header string) Option {
	return func(t *Table) {
		t.hasHeader = true
		t.headerStr = header
	}
}

// WithMaxRows sets a limit of how many rows can be displayed on each render frame.
func WithMaxRows(max int) Option {
	return func(t *Table) {
		t.maxServices = max
	}
}

func NewTable(opts ...Option) *Table {
	return &Table{
		headerStr:   "",
		hasHeader:   false,
		maxServices: 0,
	}
}
