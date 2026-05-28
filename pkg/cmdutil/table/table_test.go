package table

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderWithHeaders(t *testing.T) {
	tbl := NewTable([]Column{
		{Header: "NAME"},
		{Header: "DESCRIPTION"},
		{Header: "COUNT", Align: AlignRight},
	})

	require.NoError(t, tbl.AddRow("default", "Main workspace", "2"))
	require.NoError(t, tbl.AddRow("secondary", "Ops", "12"))

	require.Equal(t, []string{
		"NAME       DESCRIPTION     COUNT",
		"default    Main workspace      2",
		"secondary  Ops                12",
	}, tbl.Render())
}

func TestRenderWithoutHeaders(t *testing.T) {
	tbl := NewTable([]Column{
		{Header: "NAME"},
		{Header: "VALUE", Align: AlignRight},
	}, WithoutHeaders())

	require.NoError(t, tbl.AddRow("alpha", "1"))
	require.NoError(t, tbl.AddRow("beta", "22"))

	require.Equal(t, []string{
		"alpha   1",
		"beta   22",
	}, tbl.Render())
}

func TestWriteTo(t *testing.T) {
	tbl := NewTable([]Column{{Header: "NAME"}})
	require.NoError(t, tbl.AddRow("alpha"))

	var out bytes.Buffer

	_, err := tbl.WriteTo(&out)
	require.NoError(t, err)
	require.Equal(t, "NAME\nalpha\n", out.String())
}

func TestAddRowReturnsErrorForWrongCellCount(t *testing.T) {
	tbl := NewTable([]Column{{Header: "NAME"}, {Header: "VALUE"}})

	err := tbl.AddRow("alpha")
	require.EqualError(t, err, "expected 2 cells, got 1")
}

func TestRenderUsesVisibleWidthForANSICells(t *testing.T) {
	tbl := NewTable([]Column{
		{Header: "NAME"},
		{Header: "STATUS"},
		{Header: "NEXT"},
	})

	green := "\x1b[32mok\x1b[0m"
	require.NoError(t, tbl.AddRow("alpha", green, "after"))
	require.NoError(t, tbl.AddRow("beta", "pending", "after"))

	rendered := tbl.Render()
	require.Len(t, rendered, 3)
	require.Equal(t, "NAME   STATUS   NEXT", stripANSI(rendered[0]))
	require.Equal(t, "alpha  ok       after", stripANSI(rendered[1]))
	require.Equal(t, "beta   pending  after", stripANSI(rendered[2]))
}

func stripANSI(s string) string {
	return ansiPattern.ReplaceAllString(s, "")
}
