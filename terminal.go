package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"golang.org/x/term"
)

type KeyCode int

const (
	KeyNone KeyCode = iota
	KeyChar
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
	KeyEnter
	KeyEsc
	KeyBackspace
	KeyDelete
	KeyHome
	KeyEnd
	KeyPageUp
	KeyPageDown
)

type KeyModifiers int

const (
	ModNone  KeyModifiers = 0
	ModShift KeyModifiers = 1 << iota
	ModAlt
	ModCtrl
)

type KeyEvent struct {
	Code      KeyCode
	Char      rune
	Modifiers KeyModifiers
}

var inputFile *os.File

func getInputFile() (*os.File, error) {
	if inputFile != nil {
		return inputFile, nil
	}
	if term.IsTerminal(int(os.Stdin.Fd())) {
		inputFile = os.Stdin
	} else {
		f, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
		if err != nil {
			return nil, err
		}
		inputFile = f
	}
	return inputFile, nil
}

func closeInput() {
	if inputFile != nil && inputFile != os.Stdin {
		inputFile.Close()
	}
	inputFile = nil
}

func enableRawMode() (*term.State, error) {
	f, err := getInputFile()
	if err != nil {
		return nil, err
	}
	return term.MakeRaw(int(f.Fd()))
}

func disableRawMode(state *term.State) error {
	f, err := getInputFile()
	if err != nil {
		return err
	}
	return term.Restore(int(f.Fd()), state)
}

func terminalSize() (int, int, error) {
	f, err := getInputFile()
	if err != nil {
		return 0, 0, err
	}
	w, h, err := term.GetSize(int(f.Fd()))
	if err != nil {
		return 0, 0, err
	}
	return w, h, nil
}

func cursorPosition() (int, int, error) {
	f, err := getInputFile()
	if err != nil {
		return 0, 0, err
	}

	// Temporarily enter raw mode to suppress echo during the query
	oldState, err := term.MakeRaw(int(f.Fd()))
	if err != nil {
		return 0, 0, err
	}
	defer term.Restore(int(f.Fd()), oldState)

	if f == os.Stdin {
		_, err = os.Stderr.Write([]byte("\x1B[6n"))
	} else {
		_, err = f.Write([]byte("\x1B[6n"))
	}
	if err != nil {
		return 0, 0, err
	}

	return readCursorPosition(f)
}

func readCursorPosition(r io.Reader) (int, int, error) {
	var buf []byte
	b := make([]byte, 1)
	for {
		_, err := r.Read(b)
		if err != nil {
			return 0, 0, err
		}
		buf = append(buf, b[0])
		if b[0] == 'R' {
			break
		}
	}

	response := string(buf)
	if !strings.HasPrefix(response, "\x1B[") {
		return 0, 0, fmt.Errorf("unexpected cursor position response: %q", response)
	}

	body := response[2 : len(response)-1]
	parts := strings.Split(body, ";")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("unexpected cursor position format: %q", response)
	}

	row := 0
	col := 0
	for _, c := range parts[0] {
		row = row*10 + int(c-'0')
	}
	for _, c := range parts[1] {
		col = col*10 + int(c-'0')
	}

	return col - 1, row - 1, nil
}

var keyBuffer []KeyEvent

func readKey() (KeyEvent, error) {
	if len(keyBuffer) > 0 {
		k := keyBuffer[0]
		keyBuffer = keyBuffer[1:]
		return k, nil
	}

	f, err := getInputFile()
	if err != nil {
		return KeyEvent{}, err
	}

	buf := make([]byte, 256)
	n, err := f.Read(buf)
	if err != nil {
		return KeyEvent{}, err
	}

	data := buf[:n]
	events := parseKeyEvents(data)
	if len(events) == 0 {
		return KeyEvent{Code: KeyNone}, nil
	}

	keyBuffer = append(keyBuffer, events[1:]...)
	return events[0], nil
}

func parseKeyEvents(data []byte) []KeyEvent {
	var events []KeyEvent
	i := 0
	for i < len(data) {
		if data[i] == 0x1B {
			// Find the end of the escape sequence
			if i+1 >= len(data) {
				events = append(events, KeyEvent{Code: KeyEsc})
				i++
				continue
			}
			if data[i+1] == '[' {
				// CSI sequence: find the terminator (letter)
				end := i + 2
				for end < len(data) && (data[end] < 0x40 || data[end] > 0x7E) {
					end++
				}
				if end < len(data) {
					end++ // include the terminator
				}
				events = append(events, parseKeyEvent(data[i:end]))
				i = end
				continue
			}
			if data[i+1] == 'O' && i+2 < len(data) {
				events = append(events, parseKeyEvent(data[i:i+3]))
				i += 3
				continue
			}
			// Alt+char
			events = append(events, parseKeyEvent(data[i:i+2]))
			i += 2
			continue
		}

		if data[i] < 0x20 || data[i] == 0x7F {
			events = append(events, parseKeyEvent(data[i:i+1]))
			i++
			continue
		}

		// Regular character (possibly multi-byte UTF-8)
		r, size := utf8.DecodeRune(data[i:])
		events = append(events, KeyEvent{Code: KeyChar, Char: r})
		i += size
	}
	return events
}

func parseKeyEvent(data []byte) KeyEvent {
	if len(data) == 0 {
		return KeyEvent{Code: KeyNone}
	}

	if data[0] == 0x1B {
		if len(data) == 1 {
			return KeyEvent{Code: KeyEsc}
		}

		// Alt+char (or Alt+Backspace)
		if len(data) == 2 && data[1] != '[' && data[1] != 'O' {
			if data[1] == 0x7F {
				return KeyEvent{Code: KeyBackspace, Modifiers: ModAlt}
			}
			if data[1] < 0x20 {
				c := rune(data[1] + 0x60)
				return KeyEvent{Code: KeyChar, Char: c, Modifiers: ModAlt}
			}
			r, _ := utf8.DecodeRune(data[1:])
			return KeyEvent{Code: KeyChar, Char: r, Modifiers: ModAlt}
		}

		if len(data) >= 3 && data[1] == '[' {
			return parseCSI(data[2:])
		}

		if len(data) >= 3 && data[1] == 'O' {
			switch data[2] {
			case 'H':
				return KeyEvent{Code: KeyHome}
			case 'F':
				return KeyEvent{Code: KeyEnd}
			}
		}

		return KeyEvent{Code: KeyEsc}
	}

	// Ctrl combinations
	if data[0] < 0x20 {
		switch data[0] {
		case 0x0D, 0x0A:
			return KeyEvent{Code: KeyEnter}
		case 0x7F:
			return KeyEvent{Code: KeyBackspace}
		case 0x09:
			return KeyEvent{Code: KeyChar, Char: '\t'}
		default:
			c := rune(data[0] + 0x60)
			return KeyEvent{Code: KeyChar, Char: c, Modifiers: ModCtrl}
		}
	}

	if data[0] == 0x7F {
		return KeyEvent{Code: KeyBackspace}
	}

	r, _ := utf8.DecodeRune(data)
	return KeyEvent{Code: KeyChar, Char: r}
}

func parseCSI(data []byte) KeyEvent {
	if len(data) == 0 {
		return KeyEvent{Code: KeyEsc}
	}

	switch data[0] {
	case 'A':
		return KeyEvent{Code: KeyUp}
	case 'B':
		return KeyEvent{Code: KeyDown}
	case 'C':
		return KeyEvent{Code: KeyRight}
	case 'D':
		return KeyEvent{Code: KeyLeft}
	case 'H':
		return KeyEvent{Code: KeyHome}
	case 'F':
		return KeyEvent{Code: KeyEnd}
	}

	params := string(data)
	suffix := data[len(data)-1]

	parts := strings.Split(params[:len(params)-1], ";")
	code := 0
	modifier := 0
	if len(parts) >= 1 {
		for _, c := range parts[0] {
			if c >= '0' && c <= '9' {
				code = code*10 + int(c-'0')
			}
		}
	}
	if len(parts) >= 2 {
		for _, c := range parts[1] {
			if c >= '0' && c <= '9' {
				modifier = modifier*10 + int(c-'0')
			}
		}
	}

	mods := decodeModifier(modifier)

	switch suffix {
	case '~':
		switch code {
		case 1:
			return KeyEvent{Code: KeyHome, Modifiers: mods}
		case 3:
			return KeyEvent{Code: KeyDelete, Modifiers: mods}
		case 4:
			return KeyEvent{Code: KeyEnd, Modifiers: mods}
		case 5:
			return KeyEvent{Code: KeyPageUp, Modifiers: mods}
		case 6:
			return KeyEvent{Code: KeyPageDown, Modifiers: mods}
		case 7:
			return KeyEvent{Code: KeyHome, Modifiers: mods}
		case 8:
			return KeyEvent{Code: KeyEnd, Modifiers: mods}
		}
	case 'A':
		return KeyEvent{Code: KeyUp, Modifiers: mods}
	case 'B':
		return KeyEvent{Code: KeyDown, Modifiers: mods}
	case 'C':
		return KeyEvent{Code: KeyRight, Modifiers: mods}
	case 'D':
		return KeyEvent{Code: KeyLeft, Modifiers: mods}
	case 'H':
		return KeyEvent{Code: KeyHome, Modifiers: mods}
	case 'F':
		return KeyEvent{Code: KeyEnd, Modifiers: mods}
	}

	return KeyEvent{Code: KeyNone}
}

func decodeModifier(m int) KeyModifiers {
	if m == 0 {
		return ModNone
	}
	m--
	var mods KeyModifiers
	if m&1 != 0 {
		mods |= ModShift
	}
	if m&2 != 0 {
		mods |= ModAlt
	}
	if m&4 != 0 {
		mods |= ModCtrl
	}
	return mods
}

// ANSI escape sequence helpers

func ansiMoveTo(x, y int) string {
	return fmt.Sprintf("\x1B[%d;%dH", y+1, x+1)
}

func ansiClearFromCursor() string {
	return "\x1B[J"
}

func ansiSetFgColor(color int) string {
	return fmt.Sprintf("\x1B[38;5;%dm", color)
}

func ansiBold() string {
	return "\x1B[1m"
}

func ansiReset() string {
	return "\x1B[0m"
}

func ansiResetFgColor() string {
	return "\x1B[39m"
}
