package main

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var currentRoot *Entry = nil
var currentIndex = 0
var previousIndices []int = []int{}

func newApplication(root *Entry) *tview.Application {
	app := tview.NewApplication()

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'q':
			app.Stop()
			return nil
		}
		return event
	})

	setNewState(app, root, 0)

	return app
}

func createList(app *tview.Application, newRoot *Entry) *tview.List {
	sortChildren(newRoot)

	var fieldLength = 0
	for _, entry := range newRoot.children {
		fieldLength = max(fieldLength, utf8.RuneCountInString(entry.name))
	}

	list := tview.NewList().ShowSecondaryText(false)
	for _, entry := range newRoot.children {
		addListEntry(list, &entry, fieldLength)
	}

	list.SetChangedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		currentIndex = index
	})

	list.SetSelectedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		tryPushRoot(app, index)
	})

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyLeft:
			tryPopRoot(app)
			return nil
		case tcell.KeyRight:
			tryPushRoot(app, currentIndex)
			return nil
		}
		return event
	})

	list.SetBorder(true).SetTitle(newRoot.fullName).SetTitleAlign(tview.AlignLeft)

	return list
}

func sortChildren(parent *Entry) {
	if parent.childrenSorted {
		return
	}

	slices.SortFunc(parent.children, func(lhs Entry, rhs Entry) int {
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
	parent.childrenSorted = true
}

func addListEntry(list *tview.List, entry *Entry, fieldLenght int) {
	nameLen := utf8.RuneCountInString(entry.name)
	padding := strings.Repeat(" ", fieldLenght-nameLen)
	size := toHumanReadableSize(entry.size)

	pushFormat := "[::b"
	popFormat := "[::B"
	if entry.isDirectory {
		pushFormat += "u"
		popFormat += "U"
	}
	pushFormat += "]"
	popFormat += "]"

	text := fmt.Sprintf("%s%s%s%s %s", pushFormat, entry.name, popFormat, padding, size)

	list.AddItem(text, size, 0, nil)
}

func tryPushRoot(app *tview.Application, index int) {
	if index >= len(currentRoot.children) {
		return
	}

	if !currentRoot.children[index].isDirectory {
		return
	}

	previousIndices = append(previousIndices, index)
	newRoot := &currentRoot.children[index]
	setNewState(app, newRoot, index)
}

func tryPopRoot(app *tview.Application) {
	newRoot := currentRoot.parent
	if newRoot == nil {
		return
	}

	index := previousIndices[len(previousIndices)-1]
	previousIndices = previousIndices[:len(previousIndices)-1]
	setNewState(app, newRoot, index)
}

func setNewState(app *tview.Application, newRoot *Entry, newIndex int) {
	currentRoot = newRoot
	currentIndex = 0

	list := createList(app, newRoot)
	app.SetRoot(list, true)
	if len(newRoot.children) > newIndex {
		list.SetCurrentItem(newIndex)
		currentIndex = newIndex
	}
}
