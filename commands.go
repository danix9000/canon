package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

type Command struct {
	Cmd    string
	Desc   string
	Source string
}

func readCommands() ([]Command, error) {
	items, err := readUserFile()
	if err != nil {
		return nil, err
	}

	home, _ := os.UserHomeDir()
	cwd, _ := os.Getwd()

	if home == "" || cwd != home {
		local, err := readLocalFile()
		if err == nil {
			items = append(items, local...)
		}
	}

	return items, nil
}

func readLocalFile() ([]Command, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return []Command{}, nil
	}
	source := filepath.Base(cwd) + "/.canon"
	return readFile(".canon", source)
}

func readUserFile() ([]Command, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return []Command{}, nil
	}
	return readFile(filepath.Join(home, ".canon"), "~/.canon")
}

func readFile(path string, source string) ([]Command, error) {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return []Command{}, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var comments []string
	var commands []Command

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "#") {
			comments = append(comments, line[1:])
		} else {
			allStartWithSpace := true
			for _, c := range comments {
				if !strings.HasPrefix(c, " ") && c != "" {
					allStartWithSpace = false
					break
				}
			}
			if allStartWithSpace {
				for i, c := range comments {
					if len(c) > 0 {
						comments[i] = c[1:]
					}
				}
			}

			commands = append(commands, Command{
				Cmd:    line,
				Desc:   strings.Join(comments, "\n"),
				Source: source,
			})
			comments = nil
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return commands, nil
}
