package tui

import (
	"testing"
	"unicode/utf8"
)

func TestTruncateRuneSafe(t *testing.T) {
	cases := []struct {
		in     string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"hello world", 8, "hello..."},
		{"abc", 3, "abc"},
		{"abcd", 3, "abc"},
		{"", 5, ""},
		{"anything", 0, ""},
	}
	for _, c := range cases {
		if got := truncate(c.in, c.maxLen); got != c.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", c.in, c.maxLen, got, c.want)
		}
	}
}

func TestTruncateNeverCorruptsUTF8(t *testing.T) {
	// A run of multi-byte runes truncated at every length must stay valid UTF-8.
	s := "éàùœçñ — accents"
	for n := 0; n <= utf8.RuneCountInString(s)+2; n++ {
		got := truncate(s, n)
		if !utf8.ValidString(got) {
			t.Fatalf("truncate(%q, %d) = %q is not valid UTF-8", s, n, got)
		}
	}
}

func TestFormatBytes(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
	}
	for _, c := range cases {
		if got := formatBytes(c.in); got != c.want {
			t.Errorf("formatBytes(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}
