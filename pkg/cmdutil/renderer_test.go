package cmdutil

import (
	"strings"
	"testing"

	"github.com/fatih/color"
)

func TestContainerRenderLinesTruncatesLongPlainText(t *testing.T) {
	container := NewContainer(nil,
		NewElement(`{{ .Value }}`),
	).WithLayout(Layout{Dimensions: [2]int{10, 0}})

	lines := container.RenderLines(Data{"Value": "abcdefghijklmnopqrstuvwxyz"}, 0)
	if len(lines) != 1 {
		t.Fatalf("expected 1 rendered line, got %d", len(lines))
	}

	got := stripANSI(lines[0])
	if visibleLength(got) != 10 {
		t.Fatalf("expected visible width 10, got %d (%q)", visibleLength(got), got)
	}
	if !strings.Contains(got, "…") {
		t.Fatalf("expected ellipsis in %q", got)
	}
	if got != "  abcde…  " {
		t.Fatalf("unexpected truncation result %q", got)
	}
}

func TestContainerRenderLinesTruncatesAnsiTextByVisibleWidth(t *testing.T) {
	oldNoColor := color.NoColor
	color.NoColor = false
	t.Cleanup(func() {
		color.NoColor = oldNoColor
	})

	container := NewContainer(nil,
		NewElement(`{{ .Value | FgRed }}`),
	).WithLayout(Layout{Dimensions: [2]int{10, 0}})

	lines := container.RenderLines(Data{"Value": "abcdefghijklmnopqrstuvwxyz"}, 0)
	if len(lines) != 1 {
		t.Fatalf("expected 1 rendered line, got %d", len(lines))
	}

	got := lines[0]
	if visibleLength(got) != 10 {
		t.Fatalf("expected visible width 10, got %d (%q)", visibleLength(got), stripANSI(got))
	}
	if !strings.Contains(stripANSI(got), "…") {
		t.Fatalf("expected ellipsis in %q", stripANSI(got))
	}
	if !strings.Contains(got, "\x1b[") {
		t.Fatalf("expected ANSI escape sequences in %q", got)
	}
	if stripANSI(got) != "  abcde…  " {
		t.Fatalf("unexpected truncation result %q", stripANSI(got))
	}
}

func TestContainerRenderLinesTruncatesMultibyteTextByRunes(t *testing.T) {
	container := NewContainer(nil,
		NewElement(`{{ .Value }}`),
	).WithLayout(Layout{Dimensions: [2]int{8, 0}})

	lines := container.RenderLines(Data{"Value": "世界世界世界世界"}, 0)
	if len(lines) != 1 {
		t.Fatalf("expected 1 rendered line, got %d", len(lines))
	}

	got := stripANSI(lines[0])
	if visibleLength(got) != 8 {
		t.Fatalf("expected visible width 8, got %d (%q)", visibleLength(got), got)
	}
	if got != "  世界世…  " {
		t.Fatalf("unexpected truncation result %q", got)
	}
}

func TestNewDashboardUsesDefaultWidthWhenWriterIsNotATerminal(t *testing.T) {
	dash, err := NewDashboard([]string{"service"}, WithWriter(&strings.Builder{}))
	if err != nil {
		t.Fatalf("NewDashboard returned error: %v", err)
	}

	if len(dash.services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(dash.services))
	}

	if got := dash.services[0].container.Dimensions[0]; got != defaultDashboardWidth {
		t.Fatalf("expected fallback width %d, got %d", defaultDashboardWidth, got)
	}
}
