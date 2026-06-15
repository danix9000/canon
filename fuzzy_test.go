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
