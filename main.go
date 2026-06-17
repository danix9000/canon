package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/mattn/go-runewidth"
)

func main() {
	result, err := run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
	if result != nil {
		fmt.Println(*result)
	}
}

func run() (*string, error) {
	initLog()

	out := os.Stderr
	state, err := initState(out)
	if err != nil {
		return nil, err
	}

	startX, startY := state.GetStart()
	oldState, err := enableRawMode()
	if err != nil {
		return nil, err
	}

	defer func() {
		fmt.Fprint(out, ansiMoveTo(startX, startY))
		fmt.Fprint(out, ansiClearFromCursor())
		disableRawMode(oldState)
		closeInput()
	}()

	return doLoop(out, state)
}

func initLog() {
	logFile := os.Getenv("LOG")
	if logFile == "" {
		// Discard all log output when no log file is configured
		slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
		return
	}
	f, err := os.Create(logFile)
	if err != nil {
		return
	}
	handler := slog.NewTextHandler(f, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(handler))
}

func initState(out io.Writer) (*State, error) {
	startX, startY, err := cursorPosition()
	if err != nil {
		return nil, err
	}

	tw, th, err := terminalSize()
	if err != nil {
		return nil, err
	}

	state := NewState([2]int{tw, th}, [2]int{startX, startY}, mustReadCommands())

	if state.PromptStart() {
		fmt.Fprint(out, "\n\r")
	}
	if state.GetTerminalHeight() < height || state.GetTerminalWidth() < 10 {
		return nil, fmt.Errorf("terminal too small")
	}
	for i := 0; i < height-1; i++ {
		fmt.Fprintln(out)
	}

	return state, nil
}

func mustReadCommands() []Command {
	cmds, err := readCommands()
	if err != nil {
		return nil
	}
	return cmds
}

func doLoop(out io.Writer, state *State) (*string, error) {
	drawBorders(out, state)
	draw(out, state)

	for {
		key, err := readKey()
		if err != nil {
			return nil, err
		}

		switch {
		case key.Code == KeyUp && key.Modifiers == ModNone:
			if state.SelectPrevious() {
				drawItems(out, state)
			}
		case key.Code == KeyDown && key.Modifiers == ModNone:
			if state.SelectNext() {
				drawItems(out, state)
			}
		case key.Code == KeyPageUp && key.Modifiers == ModNone:
			if state.SelectPrevPage() {
				drawItems(out, state)
			}
		case key.Code == KeyPageDown && key.Modifiers == ModNone:
			if state.SelectNextPage() {
				drawItems(out, state)
			}
		case (key.Code == KeyLeft && key.Modifiers == ModNone) ||
			(key.Code == KeyChar && key.Char == 'b' && key.Modifiers == ModCtrl):
			if state.MoveCursorLeft() {
				drawQuery(out, state)
			}
		case (key.Code == KeyRight && key.Modifiers == ModNone) ||
			(key.Code == KeyChar && key.Char == 'f' && key.Modifiers == ModCtrl):
			if state.MoveCursorRight() {
				drawQuery(out, state)
			}
		case key.Code == KeyEsc ||
			(key.Code == KeyChar && (key.Char == 'c' || key.Char == 'z') && key.Modifiers == ModCtrl):
			return nil, nil
		case key.Code == KeyEnter:
			return state.GetSelectedCommand(), nil
		case (key.Code == KeyChar && key.Char == 'w' && key.Modifiers == ModCtrl) ||
			(key.Code == KeyBackspace && (key.Modifiers&ModAlt != 0 || key.Modifiers&ModCtrl != 0)):
			if state.DeletePrevWord() {
				draw(out, state)
			}
		case (key.Code == KeyChar && key.Char == 'k' && key.Modifiers == ModCtrl) ||
			(key.Code == KeyEnd && key.Modifiers&ModShift != 0):
			if state.DeleteToEnd() {
				draw(out, state)
			}
		case key.Code == KeyHome && key.Modifiers&ModShift != 0:
			if state.DeleteToStart() {
				draw(out, state)
			}
		case key.Code == KeyBackspace && key.Modifiers == ModNone:
			if state.Backspace() {
				draw(out, state)
			}
		case (key.Code == KeyChar && key.Char == 'a' && key.Modifiers == ModCtrl) ||
			key.Code == KeyHome:
			if state.MoveCursorToStart() {
				drawQuery(out, state)
			}
		case (key.Code == KeyChar && key.Char == 'e' && key.Modifiers == ModCtrl) ||
			(key.Code == KeyEnd && key.Modifiers&ModShift == 0):
			if state.MoveToEnd() {
				drawQuery(out, state)
			}
		case key.Code == KeyChar && key.Char == 'f' && key.Modifiers == ModAlt:
			if state.MoveRightByWord() {
				drawQuery(out, state)
			}
		case key.Code == KeyChar && key.Char == 'b' && key.Modifiers == ModAlt:
			if state.MoveLeftByWord() {
				drawQuery(out, state)
			}
		case (key.Code == KeyChar && key.Char == 'd' && key.Modifiers == ModAlt) ||
			(key.Code == KeyDelete && key.Modifiers&ModCtrl != 0):
			if state.DeleteNextWord() {
				draw(out, state)
			}
		case key.Code == KeyChar && key.Char == 'u' && key.Modifiers == ModCtrl:
			if state.DeleteAll() {
				draw(out, state)
			}
		case key.Code == KeyChar && key.Char == 'd' && key.Modifiers == ModCtrl:
			if !state.HasQuery() {
				return nil, nil
			}
			if state.DeleteChar() {
				draw(out, state)
			}
		case key.Code == KeyDelete && key.Modifiers == ModNone:
			if state.DeleteChar() {
				draw(out, state)
			}
		case key.Code == KeyChar && (key.Modifiers == ModNone || key.Modifiers == ModShift):
			state.Insert(string(key.Char))
			draw(out, state)
		case key.Code == KeyRight && key.Modifiers == ModShift|ModAlt:
			if state.ScrollCommandRightByWord() {
				drawItems(out, state)
			}
		case key.Code == KeyLeft && key.Modifiers == ModShift|ModAlt:
			if state.ScrollCommandLeftByWord() {
				drawItems(out, state)
			}
		case key.Code == KeyRight && key.Modifiers&ModShift != 0:
			if state.ScrollCommandRight() {
				drawItems(out, state)
			}
		case key.Code == KeyLeft && key.Modifiers&ModShift != 0:
			if state.ScrollCommandLeft() {
				drawItems(out, state)
			}
		case key.Code == KeyDown && key.Modifiers&ModShift != 0:
			if state.ScrollDescriptionDown() {
				draw(out, state)
			}
		case key.Code == KeyUp && key.Modifiers&ModShift != 0:
			if state.ScrollDescriptionUp() {
				draw(out, state)
			}
		default:
			slog.Info("Unsupported key", "code", key.Code, "char", key.Char, "modifiers", key.Modifiers)
		}
	}
}

func drawBorders(out io.Writer, state *State) {
	tw := state.GetTerminalWidth()

	topBorder := "┌" + strings.Repeat("─", tw-2) + "╮"

	hdRunes := []rune("├" + strings.Repeat("─", tw-2) + "┤")
	hdRunes[state.GetDescX()-2] = '┬'

	btRunes := []rune("╰" + strings.Repeat("─", tw-2) + "╯")
	btRunes[state.GetDescX()-2] = '┴'

	appY := state.GetAppY()

	fmt.Fprint(out, ansiMoveTo(0, appY))
	fmt.Fprint(out, ansiClearFromCursor())
	fmt.Fprint(out, topBorder)

	fmt.Fprint(out, ansiMoveTo(0, appY+2))
	fmt.Fprint(out, string(hdRunes))

	fmt.Fprint(out, ansiMoveTo(0, appY+height-1))
	fmt.Fprint(out, string(btRunes))

	// Prompt line
	fmt.Fprint(out, ansiMoveTo(0, appY+1))
	fmt.Fprint(out, "│ ")
	fmt.Fprint(out, ansiSetFgColor(117))
	fmt.Fprint(out, ansiBold())
	fmt.Fprint(out, "> ")
	fmt.Fprint(out, ansiReset())
	fmt.Fprint(out, ansiMoveTo(tw-1, appY+1))
	fmt.Fprint(out, "│")
	fmt.Fprint(out, ansiMoveTo(4, appY+1))

	// Vertical borders for item rows
	for i := 0; i < maxItems; i++ {
		fmt.Fprint(out, ansiMoveTo(0, appY+3+i))
		fmt.Fprint(out, "│")
		fmt.Fprint(out, ansiMoveTo(state.GetDescX()-2, appY+3+i))
		fmt.Fprint(out, "│")
		fmt.Fprint(out, ansiMoveTo(tw-1, appY+3+i))
		fmt.Fprint(out, "│")
	}
}

func draw(out io.Writer, state *State) {
	drawItems(out, state)
	drawQuery(out, state)
}

func drawItems(out io.Writer, state *State) {
	items := state.GetFilteredItems()
	appY := state.GetAppY()

	for i := 0; i < maxItems; i++ {
		var cmd string
		scrollOffset := 0
		if i < len(items) {
			cmd = items[i].Item.GetCmd()
			scrollOffset = items[i].Item.CmdScrollOffset
		}

		cmd = strings.ReplaceAll(cmd, "\t", " ")

		displayWidth := state.GetWindowWidth() - 2
		cmdRunes := []rune(cmd)
		fullLen := len(cmdRunes)
		clippedLeft := scrollOffset > 0
		remaining := fullLen - scrollOffset
		if remaining < 0 {
			remaining = 0
		}

		// Determine how many chars of actual command text we can show
		textSlots := displayWidth
		if clippedLeft {
			textSlots--
		}
		clippedRight := remaining > textSlots
		if clippedRight {
			textSlots--
		}

		// Extract the visible slice of command runes
		start := scrollOffset
		end := scrollOffset + textSlots
		if start > fullLen {
			start = fullLen
		}
		if end > fullLen {
			end = fullLen
		}
		visibleRunes := cmdRunes[start:end]
		visible := string(visibleRunes)

		// Compute match indices mapped to positions in the final display string
		var matchIndices []int
		if i < len(items) {
			ellipsisOffset := 0
			if clippedLeft {
				ellipsisOffset = 1
			}
			for _, idx := range items[i].CharsMatched {
				pos := idx - scrollOffset
				if pos >= 0 && pos < len(visibleRunes) {
					matchIndices = append(matchIndices, pos+ellipsisOffset)
				}
			}
		}

		// Assemble display string with ellipsis indicators and pad to width
		var display string
		if clippedLeft {
			display = "…" + visible
		} else {
			display = visible
		}
		if clippedRight {
			display += "…"
		}
		for runewidth.StringWidth(display) < displayWidth {
			display += " "
		}

		if i < len(items) {
			display = highlightMatches(display, matchIndices)
		}

		marker := " "
		if len(items) > 0 && i == state.GetCurrentSelection() {
			marker = ">"
		}
		line := marker + " " + display

		cursorXPos := 4 + state.GetCursorX()
		cursorYPos := state.GetCursorY()

		if i == state.GetCurrentSelection() {
			fmt.Fprint(out, ansiMoveTo(state.GetWindowX(), appY+3+i))
			fmt.Fprint(out, ansiBold())
			fmt.Fprint(out, line)
			fmt.Fprint(out, ansiReset())
			fmt.Fprint(out, ansiMoveTo(cursorXPos, cursorYPos))
		} else {
			fmt.Fprint(out, ansiMoveTo(state.GetWindowX(), appY+3+i))
			fmt.Fprint(out, line)
			fmt.Fprint(out, ansiMoveTo(cursorXPos, cursorYPos))
		}
	}

	var currentItem *Item
	var descMatchIndices []int
	if len(items) > 0 && state.GetCurrentSelection() < len(items) {
		sel := items[state.GetCurrentSelection()]
		currentItem = sel.Item
		descMatchIndices = sel.DescCharsMatched
	}
	drawPreview(out, state, currentItem, descMatchIndices)
}

func drawPreview(out io.Writer, state *State, currentItem *Item, descMatchIndices []int) {
	var lines []string
	var lineOffsets []int // character offset of each line within the original description
	if currentItem != nil {
		allLines := currentItem.GetDescriptionLines()
		lineOffsets = descriptionLineOffsets(currentItem.Command.Desc, allLines)

		startLine := currentItem.GetCurrentDescLine()
		if startLine < len(allLines) {
			lines = allLines[startLine:]
			lineOffsets = lineOffsets[startLine:]
		}
	}

	appY := state.GetAppY()
	cursorXPos := 4 + state.GetCursorX()
	cursorYPos := state.GetCursorY()

	descIndexSet := make(map[int]bool, len(descMatchIndices))
	for _, idx := range descMatchIndices {
		descIndexSet[idx] = true
	}

	for i := 0; i < state.GetDescH(); i++ {
		var line string
		if i < len(lines) {
			line = lines[i]
		}
		line = fitWidth(line, state.GetDescWidth(), true)

		if i < len(lines) && i < len(lineOffsets) {
			lineMatchIndices := mapDescIndicesToLine(lineOffsets[i], len([]rune(lines[i])), descIndexSet)
			if len(lineMatchIndices) > 0 {
				line = highlightMatches(line, lineMatchIndices)
			}
		}

		fmt.Fprint(out, ansiMoveTo(state.GetDescX(), appY+3+i))
		fmt.Fprint(out, line)
		fmt.Fprint(out, ansiMoveTo(cursorXPos, cursorYPos))
	}
}

// descriptionLineOffsets computes the starting character offset of each
// wrapped line within the original description text.
func descriptionLineOffsets(originalDesc string, wrappedLines []string) []int {
	offsets := make([]int, len(wrappedLines))
	pos := 0
	descRunes := []rune(originalDesc)

	for i, line := range wrappedLines {
		offsets[i] = pos
		lineRunes := []rune(line)

		// Skip through the original description to find where this line's
		// content ends. Wrapped lines may omit trailing spaces and hyphens
		// may be inserted, so we match character by character.
		matched := 0
		for pos < len(descRunes) && matched < len(lineRunes) {
			if descRunes[pos] == lineRunes[matched] {
				matched++
				pos++
			} else if lineRunes[matched] == '-' && matched == len(lineRunes)-1 {
				// Hyphen inserted by word wrapping — not in the original
				matched++
			} else if descRunes[pos] == '\n' {
				pos++
			} else {
				pos++
			}
		}
		// Skip trailing whitespace/newlines between wrapped lines
		for pos < len(descRunes) && (descRunes[pos] == ' ' || descRunes[pos] == '\n') {
			pos++
		}
	}

	return offsets
}

func mapDescIndicesToLine(lineOffset int, lineLen int, descIndexSet map[int]bool) []int {
	var indices []int
	for j := 0; j < lineLen; j++ {
		if descIndexSet[lineOffset+j] {
			indices = append(indices, j)
		}
	}
	return indices
}

func drawQuery(out io.Writer, state *State) {
	promptWidth := state.GetPromptWidth()
	queryStartIdx := state.GetQueryStartIndex()
	query := strings.ReplaceAll(state.GetQuery(), "\t", " ")
	runes := []rune(query)

	var visible string
	if queryStartIdx < len(runes) {
		visible = string(runes[queryStartIdx:])
	}
	filteredQuery := fitWidth(visible, promptWidth, false)

	cursorXPos := 4 + state.GetCursorX()
	cursorYPos := state.GetCursorY()

	fmt.Fprint(out, ansiMoveTo(4, cursorYPos))
	fmt.Fprint(out, filteredQuery)
	fmt.Fprint(out, ansiMoveTo(cursorXPos, cursorYPos))
}

func highlightMatches(word string, indices []int) string {
	if len(indices) == 0 {
		return word
	}

	color := 120
	runes := []rune(word)
	var result strings.Builder
	currentIdx := 0

	indexSet := make(map[int]bool, len(indices))
	for _, idx := range indices {
		indexSet[idx] = true
	}

	for i, r := range runes {
		_ = currentIdx
		if indexSet[i] {
			result.WriteString(ansiSetFgColor(color))
			result.WriteRune(r)
			result.WriteString(ansiResetFgColor())
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}

func fitWidth(s string, width int, ellipsis bool) string {
	runes := []rune(s)
	result := s
	truncated := runewidth.StringWidth(result) > width

	for runewidth.StringWidth(result) >= width {
		if len(runes) == 0 {
			break
		}
		runes = runes[:len(runes)-1]
		result = string(runes)
	}

	if ellipsis && truncated {
		result += "…"
	}

	for runewidth.StringWidth(result) < width {
		result += " "
	}

	return result
}
