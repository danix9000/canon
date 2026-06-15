package main

const (
	height   = 10
	maxItems = height - 4
)

type State struct {
	terminalSize     [2]int
	promptStart      bool
	start            [2]int
	appY             int
	cursorY          int
	cursorIndex      int
	currentSelection int
	windowStartIndex int
	windowX          int
	windowWidth      int
	descX            int
	descWidth        int
	descHeight       int
	query            string
	queryStartIndex  int
	items            []Item
	filteredMatches  []MatchResult
}

func NewState(terminalSize [2]int, start [2]int, commands []Command) *State {
	startX := start[0]
	startY := start[1]

	rightWidth := terminalSize[0] / 2
	leftWidth := terminalSize[0] - rightWidth

	windowX := 2
	windowWidth := leftWidth - 4

	descX := leftWidth + 1
	descWidth := rightWidth - 3

	promptStart := startX != 0
	appY := startY

	if promptStart {
		appY++
	}

	if appY+height > terminalSize[1] {
		offset := (appY + height) - terminalSize[1]
		appY -= offset
		startY -= offset
	}

	items := buildItems(commands, descWidth)

	filteredMatches := make([]MatchResult, len(items))
	for i := range items {
		filteredMatches[i] = MatchResult{ItemIdx: i, CharsMatched: []int{}}
	}

	return &State{
		terminalSize:     terminalSize,
		promptStart:      promptStart,
		start:            [2]int{startX, startY},
		appY:             appY,
		windowX:          windowX,
		windowWidth:      windowWidth,
		descX:            descX,
		descWidth:        descWidth,
		descHeight:       maxItems,
		cursorIndex:      0,
		cursorY:          appY + 1,
		currentSelection: 0,
		windowStartIndex: 0,
		query:            "",
		queryStartIndex:  0,
		filteredMatches:  filteredMatches,
		items:            items,
	}
}

func buildItems(commands []Command, descWidth int) []Item {
	items := make([]Item, len(commands))
	for i, cmd := range commands {
		lines := wrapText(cmd.Desc, descWidth)
		items[i] = Item{
			Command:          cmd,
			DescriptionLines: lines,
			CurrentDescLine:  0,
		}
	}
	return items
}

func (s *State) PromptStart() bool {
	return s.promptStart
}

func (s *State) GetDescX() int {
	return s.descX
}

func (s *State) GetDescWidth() int {
	return s.descWidth
}

func (s *State) GetDescH() int {
	return s.descHeight
}

func (s *State) HasQuery() bool {
	return s.query != ""
}

func (s *State) GetQuery() string {
	return s.query
}

func (s *State) GetQueryStartIndex() int {
	return s.queryStartIndex
}

func (s *State) SetQuery(query string, cursorIndex int) {
	s.query = query
	s.currentSelection = 0
	s.windowStartIndex = 0
	s.SetCursorIndex(cursorIndex)
	s.filteredMatches = fuzzyMatch(s.items, s.query)
}

func (s *State) setQueryStartIndex() {
	cursorIndex := s.GetCursorIndex()
	promptWidth := s.GetPromptWidth()
	start := s.queryStartIndex
	end := start + promptWidth

	if cursorIndex >= start && cursorIndex < end {
		// cursor is visible, keep start
	} else if cursorIndex >= end {
		s.queryStartIndex = cursorIndex - promptWidth + 1
	} else {
		s.queryStartIndex = cursorIndex
	}
}

func (s *State) GetSelectedCommand() *string {
	if len(s.filteredMatches) == 0 {
		return nil
	}
	idx := s.filteredMatches[s.windowStartIndex+s.currentSelection].ItemIdx
	if idx >= len(s.items) {
		return nil
	}
	cmd := s.items[idx].GetCmd()
	return &cmd
}

func (s *State) GetTerminalWidth() int {
	return s.terminalSize[0]
}

func (s *State) GetTerminalHeight() int {
	return s.terminalSize[1]
}

func (s *State) GetPromptWidth() int {
	return s.terminalSize[0] - 5
}

func (s *State) GetWindowX() int {
	return s.windowX
}

func (s *State) GetWindowWidth() int {
	return s.windowWidth
}

func (s *State) GetStart() (int, int) {
	return s.start[0], s.start[1]
}

func (s *State) GetAppY() int {
	return s.appY
}

func (s *State) GetCursorX() int {
	pw := s.GetPromptWidth()
	qsi := s.GetQueryStartIndex()

	if s.cursorIndex >= qsi+pw-1 {
		return pw - 1
	} else if s.cursorIndex > qsi {
		return s.cursorIndex - qsi
	}
	return 0
}

func (s *State) GetCursorY() int {
	return s.cursorY
}

func (s *State) SetCursorIndex(idx int) {
	s.cursorIndex = idx
	s.setQueryStartIndex()
}

func (s *State) GetCursorIndex() int {
	return s.cursorIndex
}

func (s *State) MoveCursorRight() bool {
	chars := []rune(s.query)
	if s.cursorIndex < len(chars) {
		s.SetCursorIndex(s.cursorIndex + 1)
		return true
	}
	return false
}

func (s *State) MoveCursorLeft() bool {
	if s.cursorIndex > 0 {
		s.SetCursorIndex(s.cursorIndex - 1)
		return true
	}
	return false
}

func (s *State) MoveToEnd() bool {
	chars := []rune(s.query)
	if s.cursorIndex < len(chars) {
		s.SetCursorIndex(len(chars))
		return true
	}
	return false
}

func (s *State) MoveCursorToStart() bool {
	if s.cursorIndex > 0 {
		s.SetCursorIndex(0)
		return true
	}
	return false
}

func (s *State) MoveLeftByWord() bool {
	newIdx := moveLeftByWord(s.GetCursorIndex(), s.query)
	if newIdx == s.GetCursorIndex() {
		return false
	}
	s.SetCursorIndex(newIdx)
	return true
}

func (s *State) MoveRightByWord() bool {
	newIdx := moveRightByWord(s.GetCursorIndex(), s.query)
	if newIdx == s.cursorIndex {
		return false
	}
	s.SetCursorIndex(newIdx)
	return true
}

func (s *State) GetCurrentSelection() int {
	return s.currentSelection
}

func (s *State) GetCurrentItem() *Item {
	if len(s.filteredMatches) == 0 {
		return nil
	}
	idx := s.filteredMatches[s.windowStartIndex+s.currentSelection].ItemIdx
	if idx >= len(s.items) {
		return nil
	}
	return &s.items[idx]
}

func (s *State) GetCurrentItemMut() *Item {
	return s.GetCurrentItem()
}

func (s *State) SelectPrevious() bool {
	if len(s.filteredMatches) == 0 {
		return false
	}
	if s.currentSelection > 0 {
		s.currentSelection--
		return true
	}
	if s.windowStartIndex > 0 {
		s.windowStartIndex--
		return true
	}
	return false
}

func (s *State) SelectNext() bool {
	if len(s.filteredMatches) == 0 {
		return false
	}
	windowSize := len(s.filteredMatches)
	if windowSize > maxItems {
		windowSize = maxItems
	}

	if s.currentSelection < windowSize-1 {
		s.currentSelection++
		return true
	}
	if s.currentSelection == windowSize-1 &&
		(s.windowStartIndex+windowSize) < len(s.filteredMatches) {
		s.windowStartIndex++
		return true
	}
	return false
}

func (s *State) SelectPrevPage() bool {
	if s.currentSelection > 0 {
		s.currentSelection = 0
		return true
	}
	if s.windowStartIndex > 0 {
		sub := maxItems
		if s.windowStartIndex < sub {
			sub = s.windowStartIndex
		}
		s.windowStartIndex -= sub
		s.currentSelection = 0
		return true
	}
	return false
}

func (s *State) SelectNextPage() bool {
	remaining := len(s.filteredMatches) - s.windowStartIndex
	windowSize := remaining
	if windowSize > maxItems {
		windowSize = maxItems
	}
	if s.currentSelection < windowSize-1 {
		s.currentSelection = windowSize - 1
		return true
	}
	if s.windowStartIndex+s.currentSelection < len(s.filteredMatches)-1 {
		s.windowStartIndex = s.windowStartIndex + s.currentSelection + 1
		remaining = len(s.filteredMatches) - s.windowStartIndex
		ws := remaining
		if ws > maxItems {
			ws = maxItems
		}
		s.currentSelection = ws - 1
		return true
	}
	return false
}

func (s *State) Backspace() bool {
	if s.GetCursorIndex() == 0 {
		return false
	}
	chars := []rune(s.query)
	idx := s.cursorIndex - 1
	chars = append(chars[:idx], chars[idx+1:]...)
	s.SetQuery(string(chars), s.cursorIndex-1)
	return true
}

func (s *State) DeleteNextWord() bool {
	chars := []rune(s.query)
	if s.cursorIndex == len(chars) {
		return false
	}
	newIdx, newQuery := deleteNextWord(s.cursorIndex, s.query)
	s.SetQuery(newQuery, newIdx)
	return true
}

func (s *State) DeletePrevWord() bool {
	if s.cursorIndex == 0 {
		return false
	}
	newIdx, newQuery := deletePrevWord(s.cursorIndex, s.query)
	s.SetQuery(newQuery, newIdx)
	return true
}

func (s *State) DeleteToStart() bool {
	if s.cursorIndex == 0 {
		return false
	}
	chars := []rune(s.query)
	remaining := chars[s.GetCursorIndex():]
	s.SetQuery(string(remaining), 0)
	return true
}

func (s *State) DeleteToEnd() bool {
	chars := []rune(s.query)
	if s.cursorIndex == len(chars) {
		return false
	}
	s.SetQuery(string(chars[:s.GetCursorIndex()]), s.GetCursorIndex())
	return true
}

func (s *State) DeleteAll() bool {
	if s.query == "" {
		return false
	}
	s.SetQuery("", 0)
	return true
}

func (s *State) DeleteChar() bool {
	chars := []rune(s.query)
	if s.cursorIndex == len(chars) {
		return false
	}
	chars = append(chars[:s.cursorIndex], chars[s.cursorIndex+1:]...)
	s.SetQuery(string(chars), s.cursorIndex)
	return true
}

func (s *State) Insert(input string) {
	chars := []rune(s.query)
	for _, c := range input {
		newChars := make([]rune, 0, len(chars)+1)
		newChars = append(newChars, chars[:s.cursorIndex]...)
		newChars = append(newChars, c)
		newChars = append(newChars, chars[s.cursorIndex:]...)
		chars = newChars
		s.SetCursorIndex(s.cursorIndex + 1)
	}
	s.SetQuery(string(chars), s.cursorIndex)
}

type FilteredItem struct {
	Item         *Item
	CharsMatched []int
}

func (s *State) GetFilteredItems() []FilteredItem {
	end := s.windowStartIndex + maxItems
	if end > len(s.filteredMatches) {
		end = len(s.filteredMatches)
	}

	var result []FilteredItem
	for _, m := range s.filteredMatches[s.windowStartIndex:end] {
		result = append(result, FilteredItem{
			Item:         &s.items[m.ItemIdx],
			CharsMatched: m.CharsMatched,
		})
	}
	return result
}

func (s *State) ScrollDescriptionDown() bool {
	item := s.GetCurrentItemMut()
	if item == nil {
		return false
	}
	if len(item.GetDescriptionLines())-item.GetCurrentDescLine() <= s.GetDescH() {
		return false
	}
	return item.ScrollDescriptionDown()
}

func (s *State) ScrollCommandRight() bool {
	item := s.GetCurrentItemMut()
	if item == nil {
		return false
	}
	return item.ScrollCommandRight(s.windowWidth)
}

func (s *State) ScrollCommandLeft() bool {
	item := s.GetCurrentItemMut()
	if item == nil {
		return false
	}
	return item.ScrollCommandLeft()
}

func (s *State) ScrollCommandRightByWord() bool {
	item := s.GetCurrentItemMut()
	if item == nil {
		return false
	}
	return item.ScrollCommandRightByWord(s.windowWidth)
}

func (s *State) ScrollCommandLeftByWord() bool {
	item := s.GetCurrentItemMut()
	if item == nil {
		return false
	}
	return item.ScrollCommandLeftByWord()
}

func (s *State) ScrollDescriptionUp() bool {
	item := s.GetCurrentItemMut()
	if item == nil {
		return false
	}
	if len(item.GetDescriptionLines()) < s.GetDescH() {
		return false
	}
	return item.ScrollDescriptionUp()
}
