package main

import (
	"fmt"
	"math"
	"os"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"disk-usage/filesystem"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"golang.design/x/clipboard"
)

var currentRoot *filesystem.Entry = nil
var currentIndex = 0
var previousIndices []int = []int{}
var toByteRepresentation func(uint64) string
var notificationText = ""
var timer *time.Timer = nil

func newApplication(path string, representation Representation, notifyOnReady bool) *tview.Application {
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

	setupNotificationBox(app)
	updates := make(chan struct{})
	go func() {
		for range updates {
			setNewState(app, currentRoot, currentIndex)
			app.Draw()
		}

		setNewState(app, currentRoot, currentIndex)
		if notifyOnReady {
			setNotification(app, "Ready")
		}
		app.Draw()
	}()

	root, err := filesystem.BuildFileTree(path, updates)
	if err != nil {
		// TODO error handling
		os.Exit(1)
	}

	setNewState(app, root, 0)

	return app
}

func setupNotificationBox(app *tview.Application) {
	timer = time.AfterFunc(2*time.Second, func() {
		notificationText = ""
		app.Draw()
	})
	// Stop immediately after creation so it doesn't go off without an actual notfication
	timer.Stop()

	app.SetAfterDrawFunc(func(screen tcell.Screen) {
		if notificationText == "" {
			return
		}

		width, height := screen.Size()
		tview.Print(screen, notificationText, 1, height-2, width-2, tview.AlignCenter, tcell.ColorWhite)
	})
}

func setNotification(app *tview.Application, text string) {
	notificationText = text

	if notificationText == "" {
		wasRunning := timer.Stop()
		if wasRunning {
			// Redraw only if the timer hasn't expired - if expired then
			// we already cleared the notification
			// Requesting a redraw is always safe from a goroutine
			go app.Draw()
		}
		return
	}

	timer.Stop()
	timer.Reset(2 * time.Second)
}

func createList(app *tview.Application, newRoot *filesystem.Entry) *tview.List {
	sortChildren(newRoot)

	var fieldLength = 0
	for _, entry := range newRoot.Children {
		fieldLength = max(fieldLength, utf8.RuneCountInString(entry.Name))
	}

	list := tview.NewList().ShowSecondaryText(false)
	for _, entry := range newRoot.Children {
		addListEntry(list, entry, fieldLength)
	}

	list.SetChangedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		currentIndex = index

		entry := currentRoot.Children[currentIndex]
		if entry.Err == nil {
			setNotification(app, "")
		} else {
			setNotification(app, fmt.Sprintf("[red]%v", entry.Err))
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
			clipboard.Write(clipboard.FmtText, []byte(currentRoot.Path))
			setNotification(app, fmt.Sprintf("[blue]Copied '%s' to clipboard", currentRoot.Path))
			return nil
		}
		return event
	})

	title := fmt.Sprintf(" %s %s ", newRoot.Path, toByteRepresentation(newRoot.Size))
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

func sortChildren(parent *filesystem.Entry) {
	slices.SortFunc(parent.Children, func(lhs *filesystem.Entry, rhs *filesystem.Entry) int {
		if lhs.Size < rhs.Size {
			return 1
		} else if lhs.Size == rhs.Size {
			if lhs.Err != nil && rhs.Err == nil {
				return 1
			} else if lhs.Err == nil && rhs.Err != nil {
				return -1
			}
			return strings.Compare(lhs.Name, rhs.Name)
		} else {
			return -1
		}
	})
}

func addListEntry(list *tview.List, entry *filesystem.Entry, fieldLenght int) {
	nameLen := utf8.RuneCountInString(entry.Name)
	padding := strings.Repeat(" ", max(fieldLenght-nameLen, 0))
	size := toByteRepresentation(entry.Size)

	pushForeground := ""
	popForeground := ""
	pushBackground := ""
	popBackground := ""
	pushAttributes := "b"
	popAttributes := "B"

	if entry.Err != nil {
		pushForeground = "red"
		popForeground = "white"
	}

	if entry.IsDirectory {
		pushAttributes += "u"
		popAttributes += "U"
	}

	pushFormat := fmt.Sprintf("[%s:%s:%s]", pushForeground, pushBackground, pushAttributes)
	popFormat := fmt.Sprintf("[%s:%s:%s]", popForeground, popBackground, popAttributes)

	text := fmt.Sprintf("%s%s%s%s %s", pushFormat, entry.Name, popFormat, padding, size)

	list.AddItem(text, size, 0, nil)
}

func tryPushRoot(app *tview.Application, index int) {
	if index >= len(currentRoot.Children) {
		return
	}

	if !currentRoot.Children[index].IsDirectory {
		return
	}

	previousIndices = append(previousIndices, index)
	newRoot := currentRoot.Children[index]
	setNewState(app, newRoot, 0)
}

func tryPopRoot(app *tview.Application) {
	newRoot := currentRoot.Parent
	if newRoot == nil {
		return
	}

	index := previousIndices[len(previousIndices)-1]
	previousIndices = previousIndices[:len(previousIndices)-1]
	setNewState(app, newRoot, index)
}

func setNewState(app *tview.Application, newRoot *filesystem.Entry, newIndex int) {
	currentRoot = newRoot
	currentIndex = 0

	list := createList(app, newRoot)
	app.SetRoot(list, true)
	if len(newRoot.Children) > newIndex {
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
