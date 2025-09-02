package main

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"golang.design/x/clipboard"
)

var currentRoot *Entry = nil
var currentIndex = 0
var previousIndices []int = []int{}
var toByteRepresentation func(uint64) string

func newApplication(root *Entry, representation Representation) *tview.Application {
	toByteRepresentation = makeToByteRepresentationFunc(representation)

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

		entry := &currentRoot.children[currentIndex]
		if entry.err != nil {
			// TODO show notification
		}
	})

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyLeft:
			fallthrough
		case tcell.KeyEscape:
			tryPopRoot(app)
			return nil
		case tcell.KeyRight:
			fallthrough
		case tcell.KeyEnter:
			tryPushRoot(app, currentIndex)
			return nil
		case tcell.KeyCtrlL:
			clipboard.Write(clipboard.FmtText, []byte(currentRoot.fullName))
			// TODO show notification
			return nil
		}
		return event
	})

	title := fmt.Sprintf(" %s %s ", newRoot.fullName, toByteRepresentation(newRoot.size))
	draw := func(screen tcell.Screen, x int, y int, width int, height int) (int, int, int, int) {
		help := ""
		appendHelp := func(keys string, text string) {
			help += fmt.Sprintf(" [blue]%s [white]%s ", keys, text)
		}

		appendHelp("q", "Quit")
		appendHelp("↑/↓", "Select file")
		appendHelp("→/Enter", "Enter directory")
		appendHelp("←/Escape", "Exit directory")
		appendHelp("Ctrl-l", "Copy Current Path to Clipboard")

		tview.Print(screen, help, x+1, y+height-1, width-2, tview.AlignLeft, tcell.ColorWhite)

		// Space for inner content
		return x + 1, y + 1, width - 2, height - 2
	}
	list.SetBorder(true).SetTitle(title).SetTitleAlign(tview.AlignLeft).SetDrawFunc(draw)

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
	size := toByteRepresentation(entry.size)

	pushForeground := ""
	popForeground := ""
	pushBackground := ""
	popBackground := ""
	pushAttributes := "b"
	popAttributes := "B"

	if entry.err != nil {
		pushForeground = "red"
		popForeground = "white"
	}

	if entry.isDirectory {
		pushAttributes += "u"
		popAttributes += "U"
	}

	pushFormat := fmt.Sprintf("[%s:%s:%s]", pushForeground, pushBackground, pushAttributes)
	popFormat := fmt.Sprintf("[%s:%s:%s]", popForeground, popBackground, popAttributes)

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
	setNewState(app, newRoot, 0)
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

func makeToByteRepresentationFunc(repr Representation) func(uint64) string {
	var step uint64
	var suffixes []string

	switch repr {
	case Bytes:
		return func(bytes uint64) string { return fmt.Sprintf("%d", bytes) }
	case IEC:
		step = 1024
		suffixes = []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB", "ZiB", "YiB"}
	case SI:
		step = 1000
		suffixes = []string{"B", "kB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"}
	}

	return func(bytes uint64) string {
		index := 0
		current := uint64(1)
		for current*step < bytes {
			current *= step
			index += 1
		}

		if index == 0 {
			return fmt.Sprintf("%d %s", bytes, suffixes[index])
		} else {
			whole := bytes / current
			fractional := uint64(math.Floor(float64(bytes-whole*current) / float64(current) * 10))

			return fmt.Sprintf("%d.%d %s", whole, fractional, suffixes[index])
		}
	}
}
