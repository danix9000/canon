package main

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// dedent strips the common leading whitespace from all non-empty lines,
// mimicking Rust's indoc! macro.
func dedent(s string) string {
	lines := strings.Split(s, "\n")

	// Drop leading empty line (from opening backtick)
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	// Drop trailing empty line (from closing backtick)
	if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}

	minIndent := math.MaxInt
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if indent < minIndent {
			minIndent = indent
		}
	}

	if minIndent == math.MaxInt {
		minIndent = 0
	}

	for i, line := range lines {
		if len(line) >= minIndent {
			lines[i] = line[minIndent:]
		}
	}

	return strings.Join(lines, "\n")
}

func runTmux(args ...string) (string, error) {
	cmd := exec.Command("./script/tmux.sh", args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func setupTmux(t *testing.T, name string) {
	t.Helper()
	out, err := runTmux("create", "-s", name)
	if err != nil {
		t.Fatalf("Failed to create tmux session: %s\n%s", err, out)
	}
}

func teardownTmux(name string) {
	runTmux("kill", "-s", name)
}

func createCanonFile(t *testing.T, name string, content string) {
	t.Helper()
	runnerDir := os.Getenv("RUNNER_TEMP")
	if runnerDir == "" {
		runnerDir = "/tmp/canon"
	}
	home := fmt.Sprintf("%s/%s", runnerDir, name)
	path := fmt.Sprintf("%s/.canon", home)
	err := os.WriteFile(path, []byte(content+"\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create .canon: %s", err)
	}
}

func createLocalCanonFile(t *testing.T, name string, dir string, content string) {
	t.Helper()
	runnerDir := os.Getenv("RUNNER_TEMP")
	if runnerDir == "" {
		runnerDir = "/tmp/canon"
	}
	home := fmt.Sprintf("%s/%s", runnerDir, name)
	dirPath := fmt.Sprintf("%s/%s", home, dir)
	err := os.MkdirAll(dirPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create directory: %s", err)
	}
	path := fmt.Sprintf("%s/.canon", dirPath)
	err = os.WriteFile(path, []byte(content+"\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create local .canon: %s", err)
	}
}

func sendKeys(t *testing.T, name string, keys ...string) {
	t.Helper()
	args := append([]string{"send", "-s", name}, keys...)
	out, err := runTmux(args...)
	if err != nil {
		t.Fatalf("Failed to send keys: %s\n%s", err, out)
	}
}

func capturePane(t *testing.T, name string) string {
	t.Helper()
	out, err := runTmux("capture", "-s", name)
	if err != nil {
		t.Fatalf("Failed to capture: %s\n%s", err, out)
	}
	return out
}

func expectScreen(t *testing.T, name string, expected string) {
	t.Helper()
	expected = strings.TrimSpace(expected)
	retries := 10

	actual := strings.TrimSpace(capturePane(t, name))

	for retries > 0 {
		if actual == expected {
			return
		}
		retries--
		time.Sleep(200 * time.Millisecond)
		actual = strings.TrimSpace(capturePane(t, name))
	}

	t.Errorf("Screen mismatch\nExpected:\n%s\n\nActual:\n%s", expected, actual)
}

func sendAndExpectScreen(t *testing.T, name string, keys []string, expected string) {
	t.Helper()
	sendKeys(t, name, keys...)
	expectScreen(t, name, expected)
}

func runIntegrationTest(t *testing.T, name string, testFn func(t *testing.T, name string)) {
	t.Helper()

	setupTmux(t, name)
	defer teardownTmux(name)

	expectScreen(t, name, ">")

	testFn(t, name)
}

func TestDirectCommand(t *testing.T) {
	runIntegrationTest(t, "direct_command", func(t *testing.T, name string) {
		createCanonFile(t, name, dedent(`
			# Lists all files in the current directory
			ls
			# Prints the current working directory
			pwd
			# Prints those magic words
			echo oh hi
		`))

		sendAndExpectScreen(t, name, []string{"canon", "Enter"}, dedent(`
			> canon
			┌──────────────────────────────────────────────────────────────────────────────╮
			│ >                                                                            │
			├──────────────────────────────────────┬───────────────────────────────────────┤
			│ > ls                                 │ Lists all files in the current di-    │
			│   pwd                                │ rectory                               │
			│   echo oh hi                         │                                       │
			│                                      │                                       │
			│                                      │                                       │
			│                                      │                                       │
			╰──────────────────────────────────────┴─[ ~/.canon ]──────────────────────────╯
		`))

		sendAndExpectScreen(t, name, []string{"Down"}, dedent(`
			> canon
			┌──────────────────────────────────────────────────────────────────────────────╮
			│ >                                                                            │
			├──────────────────────────────────────┬───────────────────────────────────────┤
			│   ls                                 │ Prints the current working directory  │
			│ > pwd                                │                                       │
			│   echo oh hi                         │                                       │
			│                                      │                                       │
			│                                      │                                       │
			│                                      │                                       │
			╰──────────────────────────────────────┴─[ ~/.canon ]──────────────────────────╯
		`))

		sendAndExpectScreen(t, name, []string{"ech"}, dedent(`
			> canon
			┌──────────────────────────────────────────────────────────────────────────────╮
			│ > ech                                                                        │
			├──────────────────────────────────────┬───────────────────────────────────────┤
			│ > echo oh hi                         │ Prints those magic words              │
			│                                      │                                       │
			│                                      │                                       │
			│                                      │                                       │
			│                                      │                                       │
			│                                      │                                       │
			╰──────────────────────────────────────┴─[ ~/.canon ]──────────────────────────╯
		`))

		sendAndExpectScreen(t, name, []string{"Enter"}, dedent(`
			> canon
			echo oh hi
			>
		`))
	})
}

func TestZshWidget(t *testing.T) {
	runIntegrationTest(t, "zsh_widget", func(t *testing.T, name string) {
		createCanonFile(t, name, dedent(`
			# Prints those magic words
			echo oh hi
			# Lists all files in the current directory
			ls
			# Prints the current working directory
			pwd
		`))

		sendAndExpectScreen(t, name, []string{"C-]"}, dedent(`
			>
			┌──────────────────────────────────────────────────────────────────────────────╮
			│ >                                                                            │
			├──────────────────────────────────────┬───────────────────────────────────────┤
			│ > echo oh hi                         │ Prints those magic words              │
			│   ls                                 │                                       │
			│   pwd                                │                                       │
			│                                      │                                       │
			│                                      │                                       │
			│                                      │                                       │
			╰──────────────────────────────────────┴─[ ~/.canon ]──────────────────────────╯
		`))

		sendAndExpectScreen(t, name, []string{"Enter"}, dedent(`
			> echo oh hi
		`))

		sendAndExpectScreen(t, name, []string{"Enter"}, dedent(`
			> echo oh hi
			oh hi
			>
		`))
	})
}

func TestMatching(t *testing.T) {
	runIntegrationTest(t, "truncated_command_match", func(t *testing.T, name string) {
		createCanonFile(t, name, dedent(`
			# A long command, which would be truncated and whose match would not be visible
			echo a really really long message! && echo foo

			# Short command
			echo foo
		`))

		sendAndExpectScreen(t, name, []string{"canon", "Enter"}, dedent(`
			> canon
			┌──────────────────────────────────────────────────────────────────────────────╮
			│ >                                                                            │
			├──────────────────────────────────────┬───────────────────────────────────────┤
			│ > echo a really really long message… │ A long command, which would be trun-  │
			│   echo foo                           │ cated and whose match would not be    │
			│                                      │ visible                               │
			│                                      │                                       │
			│                                      │                                       │
			│                                      │                                       │
			╰──────────────────────────────────────┴─[ ~/.canon ]──────────────────────────╯
		`))

		sendAndExpectScreen(t, name, []string{"foo"}, dedent(`
			> canon
			┌──────────────────────────────────────────────────────────────────────────────╮
			│ > foo                                                                        │
			├──────────────────────────────────────┬───────────────────────────────────────┤
			│ > echo a really really long message… │ A long command, which would be trun-  │
			│   echo foo                           │ cated and whose match would not be    │
			│                                      │ visible                               │
			│                                      │                                       │
			│                                      │                                       │
			│                                      │                                       │
			╰──────────────────────────────────────┴─[ ~/.canon ]──────────────────────────╯
		`))
	})
}

func TestQueryScrolling(t *testing.T) {
	runIntegrationTest(t, "query_scrolling", func(t *testing.T, name string) {
		createCanonFile(t, name, dedent(`
			# Lists all files in the current directory
			ls
			# Prints the current working directory
			pwd
			# Prints those magic words
			echo oh hi
		`))

		sendAndExpectScreen(t, name, []string{"canon", "Enter"}, dedent(`
			> canon
			┌──────────────────────────────────────────────────────────────────────────────╮
			│ >                                                                            │
			├──────────────────────────────────────┬───────────────────────────────────────┤
			│ > ls                                 │ Lists all files in the current di-    │
			│   pwd                                │ rectory                               │
			│   echo oh hi                         │                                       │
			│                                      │                                       │
			│                                      │                                       │
			│                                      │                                       │
			╰──────────────────────────────────────┴─[ ~/.canon ]──────────────────────────╯
		`))

		sendAndExpectScreen(t, name,
			[]string{"A really really long query, one that would not fit on the screen and would require scrolling"},
			dedent(`
				> canon
				┌──────────────────────────────────────────────────────────────────────────────╮
				│ > ng query, one that would not fit on the screen and would require scrolling │
				├──────────────────────────────────────┬───────────────────────────────────────┤
				│                                      │                                       │
				│                                      │                                       │
				│                                      │                                       │
				│                                      │                                       │
				│                                      │                                       │
				│                                      │                                       │
				╰──────────────────────────────────────┴───────────────────────────────────────╯
			`))

		leftKeys := make([]string, 75)
		for i := range leftKeys {
			leftKeys[i] = "Left"
		}
		sendAndExpectScreen(t, name, leftKeys, dedent(`
			> canon
			┌──────────────────────────────────────────────────────────────────────────────╮
			│ > ong query, one that would not fit on the screen and would require scrollin │
			├──────────────────────────────────────┬───────────────────────────────────────┤
			│                                      │                                       │
			│                                      │                                       │
			│                                      │                                       │
			│                                      │                                       │
			│                                      │                                       │
			│                                      │                                       │
			╰──────────────────────────────────────┴───────────────────────────────────────╯
		`))
	})
}

func TestCommandHorizontalScrolling(t *testing.T) {
	runIntegrationTest(t, "command_horizontal_scrolling", func(t *testing.T, name string) {
		createCanonFile(t, name, dedent(`
			# A long command, which would be truncated and whose match would not be visible
			echo a really really long message! && echo foo

			# Short command
			echo foo
		`))

		sendAndExpectScreen(t, name, []string{"canon", "Enter"}, dedent(`
			> canon
			┌──────────────────────────────────────────────────────────────────────────────╮
			│ >                                                                            │
			├──────────────────────────────────────┬───────────────────────────────────────┤
			│ > echo a really really long message… │ A long command, which would be trun-  │
			│   echo foo                           │ cated and whose match would not be    │
			│                                      │ visible                               │
			│                                      │                                       │
			│                                      │                                       │
			│                                      │                                       │
			╰──────────────────────────────────────┴─[ ~/.canon ]──────────────────────────╯
		`))

		// Shift+Right scrolls right: left ellipsis appears, right ellipsis stays
		sendAndExpectScreen(t, name, []string{"S-Right"}, dedent(`
			> canon
			┌──────────────────────────────────────────────────────────────────────────────╮
			│ >                                                                            │
			├──────────────────────────────────────┬───────────────────────────────────────┤
			│ > …cho a really really long message… │ A long command, which would be trun-  │
			│   echo foo                           │ cated and whose match would not be    │
			│                                      │ visible                               │
			│                                      │                                       │
			│                                      │                                       │
			│                                      │                                       │
			╰──────────────────────────────────────┴─[ ~/.canon ]──────────────────────────╯
		`))

		// Continue scrolling right
		sendAndExpectScreen(t, name, []string{"S-Right"}, dedent(`
			> canon
			┌──────────────────────────────────────────────────────────────────────────────╮
			│ >                                                                            │
			├──────────────────────────────────────┬───────────────────────────────────────┤
			│ > …ho a really really long message!… │ A long command, which would be trun-  │
			│   echo foo                           │ cated and whose match would not be    │
			│                                      │ visible                               │
			│                                      │                                       │
			│                                      │                                       │
			│                                      │                                       │
			╰──────────────────────────────────────┴─[ ~/.canon ]──────────────────────────╯
		`))

		// Shift+Left scrolls back left
		sendAndExpectScreen(t, name, []string{"S-Left"}, dedent(`
			> canon
			┌──────────────────────────────────────────────────────────────────────────────╮
			│ >                                                                            │
			├──────────────────────────────────────┬───────────────────────────────────────┤
			│ > …cho a really really long message… │ A long command, which would be trun-  │
			│   echo foo                           │ cated and whose match would not be    │
			│                                      │ visible                               │
			│                                      │                                       │
			│                                      │                                       │
			│                                      │                                       │
			╰──────────────────────────────────────┴─[ ~/.canon ]──────────────────────────╯
		`))

		// Scroll all the way back: left ellipsis disappears, right ellipsis returns
		sendAndExpectScreen(t, name, []string{"S-Left"}, dedent(`
			> canon
			┌──────────────────────────────────────────────────────────────────────────────╮
			│ >                                                                            │
			├──────────────────────────────────────┬───────────────────────────────────────┤
			│ > echo a really really long message… │ A long command, which would be trun-  │
			│   echo foo                           │ cated and whose match would not be    │
			│                                      │ visible                               │
			│                                      │                                       │
			│                                      │                                       │
			│                                      │                                       │
			╰──────────────────────────────────────┴─[ ~/.canon ]──────────────────────────╯
		`))
	})
}

func TestCommandWordScrolling(t *testing.T) {
	runIntegrationTest(t, "command_word_scrolling", func(t *testing.T, name string) {
		createCanonFile(t, name, dedent(`
			# A long command, which would be truncated and whose match would not be visible
			echo a really really long message! && echo foo

			# Short command
			echo foo
		`))

		sendAndExpectScreen(t, name, []string{"canon", "Enter"}, dedent(`
			> canon
			┌──────────────────────────────────────────────────────────────────────────────╮
			│ >                                                                            │
			├──────────────────────────────────────┬───────────────────────────────────────┤
			│ > echo a really really long message… │ A long command, which would be trun-  │
			│   echo foo                           │ cated and whose match would not be    │
			│                                      │ visible                               │
			│                                      │                                       │
			│                                      │                                       │
			│                                      │                                       │
			╰──────────────────────────────────────┴─[ ~/.canon ]──────────────────────────╯
		`))

		// Shift+Alt+Right jumps forward by one word
		sendAndExpectScreen(t, name, []string{"M-S-Right"}, dedent(`
			> canon
			┌──────────────────────────────────────────────────────────────────────────────╮
			│ >                                                                            │
			├──────────────────────────────────────┬───────────────────────────────────────┤
			│ > …a really really long message! &&… │ A long command, which would be trun-  │
			│   echo foo                           │ cated and whose match would not be    │
			│                                      │ visible                               │
			│                                      │                                       │
			│                                      │                                       │
			│                                      │                                       │
			╰──────────────────────────────────────┴─[ ~/.canon ]──────────────────────────╯
		`))

		// Another word jump forward
		sendAndExpectScreen(t, name, []string{"M-S-Right"}, dedent(`
			> canon
			┌──────────────────────────────────────────────────────────────────────────────╮
			│ >                                                                            │
			├──────────────────────────────────────┬───────────────────────────────────────┤
			│ > … really long message! && echo foo │ A long command, which would be trun-  │
			│   echo foo                           │ cated and whose match would not be    │
			│                                      │ visible                               │
			│                                      │                                       │
			│                                      │                                       │
			│                                      │                                       │
			╰──────────────────────────────────────┴─[ ~/.canon ]──────────────────────────╯
		`))

		// Shift+Alt+Left jumps back by one word
		sendAndExpectScreen(t, name, []string{"M-S-Left"}, dedent(`
			> canon
			┌──────────────────────────────────────────────────────────────────────────────╮
			│ >                                                                            │
			├──────────────────────────────────────┬───────────────────────────────────────┤
			│ > …really really long message! && e… │ A long command, which would be trun-  │
			│   echo foo                           │ cated and whose match would not be    │
			│                                      │ visible                               │
			│                                      │                                       │
			│                                      │                                       │
			│                                      │                                       │
			╰──────────────────────────────────────┴─[ ~/.canon ]──────────────────────────╯
		`))

		// Jump back further
		sendAndExpectScreen(t, name, []string{"M-S-Left"}, dedent(`
			> canon
			┌──────────────────────────────────────────────────────────────────────────────╮
			│ >                                                                            │
			├──────────────────────────────────────┬───────────────────────────────────────┤
			│ > …a really really long message! &&… │ A long command, which would be trun-  │
			│   echo foo                           │ cated and whose match would not be    │
			│                                      │ visible                               │
			│                                      │                                       │
			│                                      │                                       │
			│                                      │                                       │
			╰──────────────────────────────────────┴─[ ~/.canon ]──────────────────────────╯
		`))

		// Jump back to start
		sendAndExpectScreen(t, name, []string{"M-S-Left"}, dedent(`
			> canon
			┌──────────────────────────────────────────────────────────────────────────────╮
			│ >                                                                            │
			├──────────────────────────────────────┬───────────────────────────────────────┤
			│ > echo a really really long message… │ A long command, which would be trun-  │
			│   echo foo                           │ cated and whose match would not be    │
			│                                      │ visible                               │
			│                                      │                                       │
			│                                      │                                       │
			│                                      │                                       │
			╰──────────────────────────────────────┴─[ ~/.canon ]──────────────────────────╯
		`))
	})
}

func TestAltBackspaceDeletesWord(t *testing.T) {
	runIntegrationTest(t, "alt_backspace", func(t *testing.T, name string) {
		createCanonFile(t, name, dedent(`
			# Lists all files in the current directory
			ls
			# Prints the current working directory
			pwd
			# Prints those magic words
			echo oh hi
		`))

		sendAndExpectScreen(t, name, []string{"canon", "Enter"}, dedent(`
			> canon
			┌──────────────────────────────────────────────────────────────────────────────╮
			│ >                                                                            │
			├──────────────────────────────────────┬───────────────────────────────────────┤
			│ > ls                                 │ Lists all files in the current di-    │
			│   pwd                                │ rectory                               │
			│   echo oh hi                         │                                       │
			│                                      │                                       │
			│                                      │                                       │
			│                                      │                                       │
			╰──────────────────────────────────────┴─[ ~/.canon ]──────────────────────────╯
		`))

		// Type a multi-word query
		sendAndExpectScreen(t, name, []string{"hello world"}, dedent(`
			> canon
			┌──────────────────────────────────────────────────────────────────────────────╮
			│ > hello world                                                                │
			├──────────────────────────────────────┬───────────────────────────────────────┤
			│                                      │                                       │
			│                                      │                                       │
			│                                      │                                       │
			│                                      │                                       │
			│                                      │                                       │
			│                                      │                                       │
			╰──────────────────────────────────────┴───────────────────────────────────────╯
		`))

		// Alt+Backspace deletes the previous word
		sendAndExpectScreen(t, name, []string{"M-BSpace"}, dedent(`
			> canon
			┌──────────────────────────────────────────────────────────────────────────────╮
			│ > hello                                                                      │
			├──────────────────────────────────────┬───────────────────────────────────────┤
			│                                      │                                       │
			│                                      │                                       │
			│                                      │                                       │
			│                                      │                                       │
			│                                      │                                       │
			│                                      │                                       │
			╰──────────────────────────────────────┴───────────────────────────────────────╯
		`))

		// Alt+Backspace again deletes the remaining word, items reappear
		sendAndExpectScreen(t, name, []string{"M-BSpace"}, dedent(`
			> canon
			┌──────────────────────────────────────────────────────────────────────────────╮
			│ >                                                                            │
			├──────────────────────────────────────┬───────────────────────────────────────┤
			│ > ls                                 │ Lists all files in the current di-    │
			│   pwd                                │ rectory                               │
			│   echo oh hi                         │                                       │
			│                                      │                                       │
			│                                      │                                       │
			│                                      │                                       │
			╰──────────────────────────────────────┴─[ ~/.canon ]──────────────────────────╯
		`))
	})
}

func TestDescriptionVerticalScrolling(t *testing.T) {
	runIntegrationTest(t, "description_vertical_scrolling", func(t *testing.T, name string) {
		createCanonFile(t, name, dedent(`
			# The description for this commmand is
			# really long and spans over many lines!
			# It is really surprisingly long.
			# But if you press on Shift and Down,
			# you'll be able to read it fully.
			# Including this very last line!
			ls

			# Prints the current working directory
			pwd

			# Prints those magic words
			echo oh hi
		`))

		sendAndExpectScreen(t, name, []string{"canon", "Enter"}, dedent(`
			> canon
			┌──────────────────────────────────────────────────────────────────────────────╮
			│ >                                                                            │
			├──────────────────────────────────────┬───────────────────────────────────────┤
			│ > ls                                 │ The description for this commmand is  │
			│   pwd                                │ really long and spans over many       │
			│   echo oh hi                         │ lines!                                │
			│                                      │ It is really surprisingly long.       │
			│                                      │ But if you press on Shift and Down,   │
			│                                      │ you'll be able to read it fully.      │
			╰──────────────────────────────────────┴─[ ~/.canon ]──────────────────────────╯
		`))

		sendAndExpectScreen(t, name, []string{"S-Down"}, dedent(`
			> canon
			┌──────────────────────────────────────────────────────────────────────────────╮
			│ >                                                                            │
			├──────────────────────────────────────┬───────────────────────────────────────┤
			│ > ls                                 │ really long and spans over many       │
			│   pwd                                │ lines!                                │
			│   echo oh hi                         │ It is really surprisingly long.       │
			│                                      │ But if you press on Shift and Down,   │
			│                                      │ you'll be able to read it fully.      │
			│                                      │ Including this very last line!        │
			╰──────────────────────────────────────┴─[ ~/.canon ]──────────────────────────╯
		`))

		sendAndExpectScreen(t, name, []string{"S-Up"}, dedent(`
			> canon
			┌──────────────────────────────────────────────────────────────────────────────╮
			│ >                                                                            │
			├──────────────────────────────────────┬───────────────────────────────────────┤
			│ > ls                                 │ The description for this commmand is  │
			│   pwd                                │ really long and spans over many       │
			│   echo oh hi                         │ lines!                                │
			│                                      │ It is really surprisingly long.       │
			│                                      │ But if you press on Shift and Down,   │
			│                                      │ you'll be able to read it fully.      │
			╰──────────────────────────────────────┴─[ ~/.canon ]──────────────────────────╯
		`))
	})
}

func TestSourceDisplay(t *testing.T) {
	runIntegrationTest(t, "source_display", func(t *testing.T, name string) {
		createCanonFile(t, name, dedent(`
			# Global command from home
			echo global
		`))

		createLocalCanonFile(t, name, "project", dedent(`
			# Local command from project
			echo local
		`))

		// cd into the project subdirectory and run canon
		sendAndExpectScreen(t, name, []string{"cd project && canon", "Enter"}, dedent(`
			> cd project && canon
			┌──────────────────────────────────────────────────────────────────────────────╮
			│ >                                                                            │
			├──────────────────────────────────────┬───────────────────────────────────────┤
			│ > echo global                        │ Global command from home              │
			│   echo local                         │                                       │
			│                                      │                                       │
			│                                      │                                       │
			│                                      │                                       │
			│                                      │                                       │
			╰──────────────────────────────────────┴─[ ~/.canon ]──────────────────────────╯
		`))

		// Navigate to the local command — source changes to project/.canon
		sendAndExpectScreen(t, name, []string{"Down"}, dedent(`
			> cd project && canon
			┌──────────────────────────────────────────────────────────────────────────────╮
			│ >                                                                            │
			├──────────────────────────────────────┬───────────────────────────────────────┤
			│   echo global                        │ Local command from project            │
			│ > echo local                         │                                       │
			│                                      │                                       │
			│                                      │                                       │
			│                                      │                                       │
			│                                      │                                       │
			╰──────────────────────────────────────┴─[ project/.canon ]────────────────────╯
		`))
	})
}
