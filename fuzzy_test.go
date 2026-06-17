package main

import "testing"

func TestFuzzyMatch(t *testing.T) {
	items := []Item{
		{Command: Command{Cmd: "ls", Desc: "Lists all files in the current directory"}},
		{Command: Command{Cmd: "pwd", Desc: "Prints the current working directory"}},
		{Command: Command{Cmd: "cat server.log | grep -i error | wc -l", Desc: "Count the number of errors in the server log"}},
	}

	matches := fuzzyMatch(items, "ls")
	if len(matches) != 1 {
		t.Fatalf("expected 1 match for 'ls', got %d", len(matches))
	}
	if matches[0].ItemIdx != 0 {
		t.Errorf("expected match index 0, got %d", matches[0].ItemIdx)
	}

	matches = fuzzyMatch(items, "ca gr")
	if len(matches) != 1 {
		t.Fatalf("expected 1 match for 'ca gr', got %d", len(matches))
	}
	if matches[0].ItemIdx != 2 {
		t.Errorf("expected match index 2, got %d", matches[0].ItemIdx)
	}
}

func TestFuzzyMatchPrefersContiguous(t *testing.T) {
	items := []Item{
		{Command: Command{Cmd: "cargo clippy", Desc: "Run clippy linter"}},
	}

	matches := fuzzyMatch(items, "clippy")
	if len(matches) != 1 {
		t.Fatalf("expected 1 match for 'clippy', got %d", len(matches))
	}
	// Should highlight "clippy" at positions 6-11, not 'c' from "cargo" + "lippy"
	expected := []int{6, 7, 8, 9, 10, 11}
	if len(matches[0].CharsMatched) != len(expected) {
		t.Fatalf("expected %d matched chars, got %d: %v", len(expected), len(matches[0].CharsMatched), matches[0].CharsMatched)
	}
	for i, idx := range matches[0].CharsMatched {
		if idx != expected[i] {
			t.Errorf("CharsMatched[%d] = %d, want %d", i, idx, expected[i])
		}
	}
}

func TestFuzzyMatchDescription(t *testing.T) {
	items := []Item{
		{Command: Command{Cmd: "lorem ipsum", Desc: "Lorem ipsum dolor sit amet"}},
		{Command: Command{Cmd: "ls", Desc: "Lists all files"}},
		{Command: Command{Cmd: "pwd", Desc: "Prints the current working directory"}},
	}

	matches := fuzzyMatch(items, "dolor")
	if len(matches) != 1 {
		t.Fatalf("expected 1 match for 'dolor', got %d", len(matches))
	}
	if matches[0].ItemIdx != 0 {
		t.Errorf("expected match index 0, got %d", matches[0].ItemIdx)
	}
	if len(matches[0].CharsMatched) != 0 {
		t.Errorf("expected no command highlight indices for description match, got %v", matches[0].CharsMatched)
	}

	// Command match should rank higher than description-only match
	matches = fuzzyMatch(items, "ls")
	if len(matches) < 1 {
		t.Fatalf("expected at least 1 match for 'ls', got %d", len(matches))
	}
	if matches[0].ItemIdx != 1 {
		t.Errorf("expected 'ls' command (index 1) first, got index %d", matches[0].ItemIdx)
	}
	if len(matches[0].CharsMatched) == 0 {
		t.Error("expected command highlight indices for command match")
	}
}
