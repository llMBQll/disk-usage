package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func main() {
	dirname, err := parseInput()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Println()
		fmt.Println("Usage: disk-visualizer [PATH]")
		os.Exit(1)
	}

	entry, err := getDirSize(dirname)
	if err != nil {
		log.Fatal(err)
	}

	slices.SortFunc(entry.children, func(lhs Entry, rhs Entry) int {
		if lhs.size < rhs.size {
			return 1
		} else if lhs.size == rhs.size {
			if lhs.err != nil && rhs.err == nil {
				return 1
			} else if lhs.err == nil && rhs.err != nil {
				return -1
			}
			return strings.Compare(lhs.name, rhs.name)
		} else {
			return -1
		}
	})

	fmt.Printf("%s: %d\n", entry.name, entry.size)
	for _, child := range entry.children {
		fmt.Printf("  %s: %d", child.name, child.size)
		if child.err != nil {
			fmt.Printf(" - %v", child.err)
		}
		fmt.Printf("\n")
	}
}

func parseInput() (string, error) {
	argc := len(os.Args)
	switch argc {
	case 1:
		return ".", nil
	case 2:
		return os.Args[1], nil
	default:
		return "", fmt.Errorf("expected 0 or 1 arguments, got %d", argc)
	}
}

type Entry struct {
	isDirectory bool
	children    []Entry
	size        uint64
	name        string
	fullName    string
	err         error
}

func getDirSize(directory string) (Entry, error) {
	entry := Entry{}
	entry.isDirectory = true

	fullName, err := filepath.Abs(directory)
	if err != nil {
		return entry, err
	}

	entry.name = filepath.Base(fullName)
	entry.fullName = fullName

	err = getSubDirSize(&entry)
	return entry, err
}

func getSubDirSize(parent *Entry) error {
	entries, err := os.ReadDir(parent.fullName)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		entryName := entry.Name()
		fullEntryName := filepath.Join(parent.fullName, entryName)

		child := Entry{
			isDirectory: false,
			children:    []Entry{},
			size:        0,
			name:        entryName,
			fullName:    fullEntryName,
		}

		if entry.IsDir() {
			child.isDirectory = true
			err = getSubDirSize(&child)
			if err != nil {
				log.Print(err)
				child.err = err
			}
		} else {
			info, err := entry.Info()
			if err != nil {
				log.Print(err)
				child.err = err
			} else {
				child.size = uint64(info.Size())
			}
		}

		parent.children = append(parent.children, child)
		parent.size += child.size
	}

	return nil
}
