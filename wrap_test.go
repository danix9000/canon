package main

import (
	"strings"
	"testing"
)

func TestWrapText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		width    int
		expected []string
	}{
		{
			name:  "directory hyphenation",
			input: "Lists all files in the current directory",
			width: 37,
			expected: []string{
				"Lists all files in the current di-",
				"rectory",
			},
		},
		{
			name:  "truncated hyphenation",
			input: "A long command, which would be truncated and whose match would not be visible",
			width: 37,
			expected: []string{
				"A long command, which would be trun-",
				"cated and whose match would not be",
				"visible",
			},
		},
		{
			name:  "no wrapping needed",
			input: "Prints the current working directory",
			width: 37,
			expected: []string{
				"Prints the current working directory",
			},
		},
		{
			name:  "short text",
			input: "Prints those magic words",
			width: 37,
			expected: []string{
				"Prints those magic words",
			},
		},
		{
			name:  "multiline description",
			input: "The description for this commmand is\nreally long and spans over many lines!\nIt is really surprisingly long.\nBut if you press on Shift and Down,\nyou'll be able to read it fully.\nIncluding this very last line!",
			width: 37,
			expected: []string{
				"The description for this commmand is",
				"really long and spans over many",
				"lines!",
				"It is really surprisingly long.",
				"But if you press on Shift and Down,",
				"you'll be able to read it fully.",
				"Including this very last line!",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := wrapText(tt.input, tt.width)
			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d lines, got %d:\n  got:  %v\n  want: %v",
					len(tt.expected), len(result), result, tt.expected)
			}
			for i, line := range result {
				if line != tt.expected[i] {
					t.Errorf("line %d mismatch:\n  got:  %q (len=%d)\n  want: %q (len=%d)",
						i, line, len(line), tt.expected[i], len(tt.expected[i]))
				}
			}
		})
	}
}

func TestHyphenate(t *testing.T) {
	initHyphenator()

	breaks := defaultHyphenator.hyphenate("directory")
	word := []rune("directory")
	var parts []string
	prev := 0
	for _, bp := range breaks {
		parts = append(parts, string(word[prev:bp])+"-")
		prev = bp
	}
	parts = append(parts, string(word[prev:]))

	result := strings.Join(parts, " ")
	expected := "di- rec- tory"
	if result != expected {
		t.Errorf("hyphenation of 'directory': got %q, want %q", result, expected)
	}
}
