package edit

import (
	"testing"

	"github.com/mike-ward/go-edit/edit/buffer"
)

func TestLeadingWhitespace(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"hello", ""},
		{"\thello", "\t"},
		{"  hello", "  "},
		{"\t  mixed", "\t  "},
		{"   ", "   "},
	}
	for _, tt := range tests {
		got := string(leadingWhitespace([]byte(tt.in)))
		if got != tt.want {
			t.Errorf("leadingWhitespace(%q)=%q want %q",
				tt.in, got, tt.want)
		}
	}
}

func TestIndentUnit_Tabs(t *testing.T) {
	u := indentUnit(buffer.IndentStyle{UseTabs: true, Width: 4})
	if string(u) != "\t" {
		t.Errorf("got %q want tab", u)
	}
}

func TestIndentUnit_Spaces(t *testing.T) {
	u := indentUnit(buffer.IndentStyle{UseTabs: false, Width: 2})
	if string(u) != "  " {
		t.Errorf("got %q want 2 spaces", u)
	}
}

func TestIndentUnit_ZeroWidth(t *testing.T) {
	u := indentUnit(buffer.IndentStyle{UseTabs: false, Width: 0})
	if len(u) != 4 {
		t.Errorf("got %d bytes want 4 (default)", len(u))
	}
}

func TestDedentLine_Tab(t *testing.T) {
	buf := buffer.FromBytes([]byte("\thello"))
	removed := dedentLine(buf, 0)
	if removed != 1 {
		t.Errorf("removed=%d want 1", removed)
	}
	if buf.String() != "hello" {
		t.Errorf("buf=%q", buf.String())
	}
}

func TestDedentLine_Spaces(t *testing.T) {
	buf := buffer.FromBytes([]byte("    hello"))
	buf.Props.IndentStyle.Width = 4
	removed := dedentLine(buf, 0)
	if removed != 4 {
		t.Errorf("removed=%d want 4", removed)
	}
	if buf.String() != "hello" {
		t.Errorf("buf=%q", buf.String())
	}
}

func TestDedentLine_NoIndent(t *testing.T) {
	buf := buffer.FromBytes([]byte("hello"))
	removed := dedentLine(buf, 0)
	if removed != 0 {
		t.Errorf("removed=%d want 0", removed)
	}
}

func TestDedentLine_EmptyLine(t *testing.T) {
	buf := buffer.FromBytes([]byte(""))
	removed := dedentLine(buf, 0)
	if removed != 0 {
		t.Errorf("removed=%d want 0", removed)
	}
}
