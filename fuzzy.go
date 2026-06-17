package main

type MatchResult struct {
	ItemIdx          int
	CharsMatched     []int
	DescCharsMatched []int
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
		index       int
		score       int
		matched     []int
		descMatched []int
	}
	var matches []scored

	for idx, item := range items {
		cmdScore, cmdMatched := fuzzyScore([]rune(item.GetCmd()), queryRunes)
		_, descMatched := fuzzyScore([]rune(item.Command.Desc), queryRunes)

		if cmdScore >= 0 {
			matches = append(matches, scored{index: idx, score: cmdScore, matched: cmdMatched, descMatched: descMatched})
			continue
		}
		// Fall back to description-only match for ranking
		descScore, _ := fuzzyScore([]rune(item.Command.Desc), queryRunes)
		if descScore >= 0 {
			matches = append(matches, scored{index: idx, score: descScore, matched: nil, descMatched: descMatched})
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
			ItemIdx:          m.index,
			CharsMatched:     m.matched,
			DescCharsMatched: m.descMatched,
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
	var bestPositions []int
	bestScore := -1

	for start := 0; start <= len(text)-len(pattern); start++ {
		if text[start] != pattern[0] {
			continue
		}

		positions := make([]int, 0, len(pattern))
		positions = append(positions, start)
		pi := 1
		for ti := start + 1; ti < len(text) && pi < len(pattern); ti++ {
			if text[ti] == pattern[pi] {
				positions = append(positions, ti)
				pi++
			}
		}
		if pi < len(pattern) {
			continue
		}

		score := positionScore(text, positions)
		if score > bestScore {
			bestScore = score
			bestPositions = positions
		}
	}

	return bestPositions
}

func positionScore(text []rune, positions []int) int {
	score := 0
	for i, pos := range positions {
		if i > 0 && pos == positions[i-1]+1 {
			score += 8
		}
		if pos > 0 && isFuzzySeparator(text[pos-1]) {
			score += 6
		}
		if pos == 0 {
			score += 6
		}
	}
	return score
}

func isFuzzySeparator(r rune) bool {
	switch r {
	case ' ', '/', '\\', '-', '_', '.', '|', '&', ';', ':':
		return true
	}
	return false
}
