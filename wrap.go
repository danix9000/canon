package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"math"
	"strings"
	"unicode"
)

//go:embed hyph-en-us.pat.txt
var hyphenPatternsData []byte

type hyphenator struct {
	patterns map[string][]int
}

var defaultHyphenator *hyphenator

func initHyphenator() {
	defaultHyphenator = newHyphenator(hyphenPatternsData)
}

func newHyphenator(patternsData []byte) *hyphenator {
	h := &hyphenator{
		patterns: make(map[string][]int),
	}

	scanner := bufio.NewScanner(bytes.NewReader(patternsData))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "%") {
			continue
		}
		h.addPattern(line)
	}

	return h
}

func (h *hyphenator) addPattern(pattern string) {
	var chars []rune
	var values []int

	lastWasDigit := false
	for _, r := range pattern {
		if r >= '0' && r <= '9' {
			values = append(values, int(r-'0'))
			lastWasDigit = true
		} else {
			if !lastWasDigit {
				values = append(values, 0)
			}
			chars = append(chars, r)
			lastWasDigit = false
		}
	}
	if !lastWasDigit {
		values = append(values, 0)
	}

	h.patterns[string(chars)] = values
}

func (h *hyphenator) hyphenate(word string) []int {
	runes := []rune(word)
	if len(runes) < 4 {
		return nil
	}

	work := "." + strings.ToLower(word) + "."
	workRunes := []rune(work)
	points := make([]int, len(workRunes)+1)

	for i := 0; i < len(workRunes); i++ {
		for j := i + 1; j <= len(workRunes); j++ {
			substr := string(workRunes[i:j])
			if vals, ok := h.patterns[substr]; ok {
				for k, v := range vals {
					pos := i + k
					if pos < len(points) && v > points[pos] {
						points[pos] = v
					}
				}
			}
		}
	}

	// lefthyphenmin=2, righthyphenmin=3 (standard TeX en-US values)
	var breakPoints []int
	for i := 2; i < len(runes)-2; i++ {
		if points[i+1]%2 == 1 {
			breakPoints = append(breakPoints, i)
		}
	}

	return breakPoints
}

func isAlphaWord(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

// fragment models a textwrap Fragment with width, whitespace, and penalty.
// This matches the textwrap Word struct behavior.
type wrapFragment struct {
	word        string  // the text content
	wordWidth   float64 // display width of text
	whitespaceW float64 // trailing whitespace width (space after this word)
	penaltyW    float64 // penalty width (1 for hyphen, 0 otherwise)
	penaltyText string  // "-" if hyphenated break, "" otherwise
}

func buildWrapFragments(text string, h *hyphenator) []wrapFragment {
	if strings.TrimSpace(text) == "" {
		return nil
	}

	// Split by whitespace, preserving trailing whitespace per word
	rawWords := strings.Fields(text)
	var frags []wrapFragment

	for i, word := range rawWords {
		trailingWS := float64(0)
		if i < len(rawWords)-1 {
			trailingWS = 1
		}

		if !isAlphaWord(word) || len([]rune(word)) < 4 {
			frags = append(frags, wrapFragment{
				word:        word,
				wordWidth:   float64(len([]rune(word))),
				whitespaceW: trailingWS,
			})
			continue
		}

		breaks := h.hyphenate(word)
		if len(breaks) == 0 {
			frags = append(frags, wrapFragment{
				word:        word,
				wordWidth:   float64(len([]rune(word))),
				whitespaceW: trailingWS,
			})
			continue
		}

		runes := []rune(word)
		prev := 0
		for _, bp := range breaks {
			frags = append(frags, wrapFragment{
				word:        string(runes[prev:bp]),
				wordWidth:   float64(bp - prev),
				whitespaceW: 0,
				penaltyW:    1,
				penaltyText: "-",
			})
			prev = bp
		}
		// Last fragment gets the trailing whitespace of the original word
		frags = append(frags, wrapFragment{
			word:        string(runes[prev:]),
			wordWidth:   float64(len(runes) - prev),
			whitespaceW: trailingWS,
		})
	}

	return frags
}

const (
	nlinePenalty          = 1000
	overflowPenalty       = 50 * 50
	shortLastLineFraction = 4
	shortLastLinePenalty  = 25
	hyphenPenalty         = 25
)

func wrapText(text string, width int) []string {
	if defaultHyphenator == nil {
		initHyphenator()
	}

	var allLines []string
	for _, paragraph := range strings.Split(text, "\n") {
		lines := wrapParagraph(paragraph, width, defaultHyphenator)
		allLines = append(allLines, lines...)
	}
	return allLines
}

func wrapParagraph(text string, width int, h *hyphenator) []string {
	frags := buildWrapFragments(text, h)
	n := len(frags)
	if n == 0 {
		return []string{""}
	}

	lineWidth := float64(width)

	// Precompute prefix sums: accumulated[i] = sum(w[k] + ws[k]) for k in 0..i
	accumulated := make([]float64, n+1)
	for i := 0; i < n; i++ {
		accumulated[i+1] = accumulated[i] + frags[i].wordWidth + frags[i].whitespaceW
	}

	// The displayed width of a line from fragment i to fragment j (inclusive) is:
	// sum of word widths + interior whitespace + penalty of last fragment
	// = accumulated[j+1] - accumulated[i] - frags[j].whitespaceW + frags[j].penaltyW
	// But for the first fragment on the line (i), we also need to strip its leading whitespace.
	// Wait -- in textwrap, whitespace is TRAILING, not leading. So:
	// displayed = accumulated[j+1] - accumulated[i] - ws[j] + penalty[j]
	// This works because accumulated includes trailing ws, and we subtract the last fragment's ws
	// (since trailing ws isn't displayed at line break) and add penalty (hyphen if needed).
	displayWidth := func(i, j int) float64 {
		return accumulated[j+1] - accumulated[i] - frags[j].whitespaceW + frags[j].penaltyW
	}

	// Optimal fit DP
	inf := math.MaxFloat64 / 2
	minCost := make([]float64, n+1)
	breakFrom := make([]int, n+1)
	for i := range minCost {
		minCost[i] = inf
	}
	minCost[0] = 0

	for i := 0; i < n; i++ {
		if minCost[i] >= inf {
			continue
		}

		for j := i; j < n; j++ {
			dw := displayWidth(i, j)
			if dw > lineWidth {
				break
			}

			isLast := j == n-1
			gap := lineWidth - dw

			var cost float64

			if isLast {
				// Last line: no gap penalty.
				// But check for short last line with a single fragment.
				fragCount := j - i + 1
				if fragCount == 1 && dw < lineWidth/float64(shortLastLineFraction) {
					cost = shortLastLinePenalty
				}
			} else {
				cost = gap * gap
			}

			// Hyphen penalty if line ends with a hyphenated fragment
			if frags[j].penaltyW > 0 {
				cost += hyphenPenalty
			}

			// nline penalty for each line beyond the first
			if i > 0 {
				cost += nlinePenalty
			}

			total := minCost[i] + cost
			if total < minCost[j+1] {
				minCost[j+1] = total
				breakFrom[j+1] = i
			}
		}
	}

	// Reconstruct lines
	var breaks []int
	pos := n
	for pos > 0 {
		breaks = append([]int{pos}, breaks...)
		pos = breakFrom[pos]
	}

	var lines []string
	start := 0
	for _, end := range breaks {
		line := buildLineFromWrapFragments(frags[start:end])
		lines = append(lines, line)
		start = end
	}

	return lines
}

func buildLineFromWrapFragments(lineFrags []wrapFragment) string {
	var sb strings.Builder
	for i, f := range lineFrags {
		sb.WriteString(f.word)
		if i < len(lineFrags)-1 {
			// Interior: add trailing whitespace
			if f.whitespaceW > 0 {
				sb.WriteString(" ")
			}
			// If this is a hyphenated fragment mid-line, don't add the hyphen
			// (hyphen is only added at line end)
		} else {
			// Last fragment on line: add penalty text (hyphen) if present
			sb.WriteString(f.penaltyText)
		}
	}
	return sb.String()
}
