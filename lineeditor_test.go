package main

import "testing"

func TestDeletePrevWord(t *testing.T) {
	tests := []struct {
		index   int
		input   string
		wantIdx int
		wantStr string
	}{
		{0, "", 0, ""},
		{4, "foo bar baz", 0, "bar baz"},
		{3, "foo bar baz", 0, " bar baz"},
		{8, "foo bar baz", 4, "foo baz"},
		{11, "foo bar baz", 8, "foo bar "},
		{5, " foo bar baz", 1, " bar baz"},
		{4, " foo bar baz", 1, "  bar baz"},
		{11, "foo bar:baz", 8, "foo bar:"},
		{1, " f", 0, "f"},
		{2, "f ", 0, ""},
	}

	for _, tt := range tests {
		gotIdx, gotStr := deletePrevWord(tt.index, tt.input)
		if gotIdx != tt.wantIdx || gotStr != tt.wantStr {
			t.Errorf("deletePrevWord(%d, %q) = (%d, %q), want (%d, %q)",
				tt.index, tt.input, gotIdx, gotStr, tt.wantIdx, tt.wantStr)
		}
	}
}
