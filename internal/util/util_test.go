package util

import "testing"

func TestTruncate(t *testing.T) {
	cases := []struct {
		name   string
		in     string
		maxLen int
		want   string
	}{
		{name: "shorter than max", in: "hi", maxLen: 10, want: "hi"},
		{name: "equal to max", in: "hello", maxLen: 5, want: "hello"},
		{name: "longer than max", in: "hello world", maxLen: 8, want: "hello..."},
		{name: "multibyte runes", in: "héllo wörld", maxLen: 8, want: "héllo..."},
		{name: "emoji", in: "😀😀😀😀😀", maxLen: 4, want: "😀..."},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Truncate(tc.in, tc.maxLen); got != tc.want {
				t.Errorf("Truncate(%q, %d) = %q, want %q", tc.in, tc.maxLen, got, tc.want)
			}
		})
	}
}
