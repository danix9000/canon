package main

type Item struct {
	Command             Command
	DescriptionLines    []string
	CurrentDescLine     int
	CmdScrollOffset     int
}

func (it *Item) GetCmd() string {
	return it.Command.Cmd
}

func (it *Item) GetDescriptionLines() []string {
	return it.DescriptionLines
}

func (it *Item) GetCurrentDescLine() int {
	return it.CurrentDescLine
}

func (it *Item) ScrollDescriptionDown() bool {
	if it.CurrentDescLine < len(it.DescriptionLines) {
		it.CurrentDescLine++
		return true
	}
	return false
}

func (it *Item) ScrollDescriptionUp() bool {
	if it.CurrentDescLine > 0 {
		it.CurrentDescLine--
		return true
	}
	return false
}

func (it *Item) ScrollCommandRight(windowWidth int) bool {
	cmdLen := len([]rune(it.Command.Cmd))
	displayWidth := windowWidth - 2
	// When scrolled, the left ellipsis takes 1 slot, leaving displayWidth-1 for text.
	maxOffset := cmdLen - (displayWidth - 1)
	if maxOffset < 0 {
		return false
	}
	if it.CmdScrollOffset < maxOffset {
		it.CmdScrollOffset++
		return true
	}
	return false
}

func (it *Item) ScrollCommandLeft() bool {
	if it.CmdScrollOffset > 0 {
		it.CmdScrollOffset--
		return true
	}
	return false
}

func (it *Item) ScrollCommandRightByWord(windowWidth int) bool {
	cmd := it.Command.Cmd
	cmdLen := len([]rune(cmd))
	displayWidth := windowWidth - 2
	maxOffset := cmdLen - (displayWidth - 1)
	if maxOffset < 0 || it.CmdScrollOffset >= maxOffset {
		return false
	}
	newOffset := moveRightByWord(it.CmdScrollOffset, cmd)
	if newOffset > maxOffset {
		newOffset = maxOffset
	}
	if newOffset == it.CmdScrollOffset {
		return false
	}
	it.CmdScrollOffset = newOffset
	return true
}

func (it *Item) ScrollCommandLeftByWord() bool {
	if it.CmdScrollOffset == 0 {
		return false
	}
	newOffset := moveLeftByWord(it.CmdScrollOffset, it.Command.Cmd)
	if newOffset == it.CmdScrollOffset {
		newOffset = 0
	}
	it.CmdScrollOffset = newOffset
	return true
}
