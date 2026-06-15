package main

import "strings"

func isWordBoundary(c rune) bool {
	if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
		return true
	}
	return strings.ContainsRune("!@#$%^&*()-_=+[]{}|;:',.<>?/\\`~", c)
}

func deletePrevWord(index int, s string) (int, string) {
	if index == 0 {
		return 0, s
	}

	chars := []rune(s)
	newIndex := index
	inWord := false

	for i := index - 1; i >= 0; i-- {
		if !isWordBoundary(chars[i]) {
			inWord = true
		} else if inWord {
			newIndex = i + 1
			break
		}

		if i == 0 {
			newIndex = 0
			break
		}
	}

	result := make([]rune, 0, len(chars)-(index-newIndex))
	result = append(result, chars[:newIndex]...)
	result = append(result, chars[index:]...)

	return newIndex, string(result)
}

func deleteNextWord(index int, s string) (int, string) {
	chars := []rune(s)

	if index >= len(chars) {
		return index, s
	}

	endIndex := index
	inWord := false

	for i := index; i < len(chars); i++ {
		if !isWordBoundary(chars[i]) {
			inWord = true
		} else if inWord {
			endIndex = i
			break
		}

		if i == len(chars)-1 {
			endIndex = len(chars)
			break
		}
	}

	result := make([]rune, 0, len(chars)-(endIndex-index))
	result = append(result, chars[:index]...)
	result = append(result, chars[endIndex:]...)

	return index, string(result)
}

func moveRightByWord(index int, s string) int {
	chars := []rune(s)
	length := len(chars)
	if index > length-1 {
		return index
	}

	nonBoundaryFound := false
	for i, c := range chars[index+1:] {
		if !isWordBoundary(c) {
			nonBoundaryFound = true
		} else if nonBoundaryFound {
			return index + i + 2
		}
	}

	return length
}

func moveLeftByWord(index int, s string) int {
	if index == 0 {
		return index
	}

	chars := []rune(s)
	sub := chars[:index]

	nonBoundaryFound := false
	for i := len(sub) - 1; i >= 0; i-- {
		if !isWordBoundary(sub[i]) {
			nonBoundaryFound = true
		} else if nonBoundaryFound {
			return index - (len(sub) - 1 - i)
		}
	}

	return 0
}
