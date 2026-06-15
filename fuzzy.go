package main

type MatchResult struct {
	ItemIdx      int
	CharsMatched []int
}

func fuzzyMatch(items []Item, query string) []MatchResult {
	if query == "" {
		results := make([]MatchResult, len(items))
		for i := range items {
			results[i] = MatchResult{ItemIdx: i, CharsMatched: []int{}}
		}
		return results
	}

	queryRunes := []rune(query)

	type scored struct {
		index   int
		score   int
		matched []int
	}
	var matches []scored

	for idx, item := range items {
		s, matched := fuzzyScore([]rune(item.GetCmd()), queryRunes)
		if s >= 0 {
			matches = append(matches, scored{index: idx, score: s, matched: matched})
		}
	}

	// Sort by score descending, then by original index ascending for stability
	for i := 1; i < len(matches); i++ {
		for j := i; j > 0; j-- {
			if matches[j].score > matches[j-1].score ||
				(matches[j].score == matches[j-1].score && matches[j].index < matches[j-1].index) {
				matches[j], matches[j-1] = matches[j-1], matches[j]
			} else {
				break
			}
		}
	}

	results := make([]MatchResult, len(matches))
	for i, m := range matches {
		results[i] = MatchResult{
			ItemIdx:      m.index,
			CharsMatched: m.matched,
		}
	}

	return results
}

// fuzzyScore implements a case-sensitive fuzzy matching algorithm.
// Returns (score, matchedIndices) or (-1, nil) if no match.
// Higher score = better match.
func fuzzyScore(text, pattern []rune) (int, []int) {
	if len(pattern) == 0 {
		return 0, nil
	}

	// Check if pattern matches at all and find the best match positions
	matched := fuzzyMatchPositions(text, pattern)
	if matched == nil {
		return -1, nil
	}

	score := 0

	for i, pos := range matched {
		// Bonus for consecutive matches
		if i > 0 && pos == matched[i-1]+1 {
			score += 8
		}

		// Bonus for match after separator (space, /, |, etc.)
		if pos > 0 && isFuzzySeparator(text[pos-1]) {
			score += 6
		}

		// Bonus for match at start
		if pos == 0 {
			score += 6
		}

		// Bonus for match at end of text
		if pos == len(text)-1 {
			score += 2
		}

		// Base score for matching
		score += 1
	}

	return score, matched
}

func fuzzyMatchPositions(text, pattern []rune) []int {
	// Find the best match positions using a greedy forward scan
	// that prefers matches at separators and consecutive positions
	positions := make([]int, 0, len(pattern))
	pi := 0
	for ti := 0; ti < len(text) && pi < len(pattern); ti++ {
		if text[ti] == pattern[pi] {
			positions = append(positions, ti)
			pi++
		}
	}
	if pi < len(pattern) {
		return nil
	}
	return positions
}

func isFuzzySeparator(r rune) bool {
	switch r {
	case ' ', '/', '\\', '-', '_', '.', '|', '&', ';', ':':
		return true
	}
	return false
}
