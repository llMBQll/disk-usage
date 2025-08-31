package main

import (
	"log"
	"os"
	"path/filepath"
)

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
