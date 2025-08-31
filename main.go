package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var currentRoot *Entry = nil
var currentIndex = 0

func main() {
	dirname, err := parseInput()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Println()
		fmt.Println("Usage: disk-visualizer [PATH]")
		os.Exit(1)
	}

	topLevel, err := getDirSize(dirname)
	if err != nil {
		log.Fatal(err)
	}

	app := tview.NewApplication()

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'q':
			app.Stop()
			return nil
		}
		return event
	})

	setList(app, &topLevel)

	if err := app.Run(); err != nil {
		panic(err)
	}
}

func setList(app *tview.Application, topLevel *Entry) {
	list := tview.NewList()

	slices.SortFunc(topLevel.children, func(lhs Entry, rhs Entry) int {
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

	var maxLen = 0
	for _, entry := range topLevel.children {
		maxLen = max(maxLen, utf8.RuneCountInString(entry.name))
	}

	list.Clear()
	for _, entry := range topLevel.children {
		nameLen := utf8.RuneCountInString(entry.name)
		padding := strings.Repeat(" ", maxLen-nameLen)
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
	list.ShowSecondaryText(false)

	currentRoot = topLevel

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

	list.SetBorder(true).SetTitle(topLevel.fullName).SetTitleAlign(tview.AlignLeft)
	if len(currentRoot.children) > 0 {
		list.SetCurrentItem(0)
		currentIndex = 0
	}

	app.SetRoot(list, true)
}

func tryPushRoot(app *tview.Application, index int) {
	if index >= len(currentRoot.children) {
		return
	}

	if !currentRoot.children[index].isDirectory {
		return
	}

	currentRoot = &currentRoot.children[index]
	setList(app, currentRoot)
}

func tryPopRoot(app *tview.Application) {
	if currentRoot.parent != nil {
		setList(app, currentRoot.parent)
	}
}

func toHumanReadableSize(bytes uint64) string {
	const step uint64 = 1024 // TODO handle both decimal and IEC suffixes
	suffixes := []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB", "ZiB", "YiB"}

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

		return fmt.Sprintf("%d.%d %s (%d)", whole, fractional, suffixes[index], bytes)
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
