package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
)

func BuildFileTree(root string, c chan struct{}) (*Entry, error) {
	stat, err := os.Stat(root)
	if err != nil {
		return nil, err
	}

	if !stat.IsDir() {
		return nil, fmt.Errorf("'%s' is not a directory", root)
	}

	path, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	entry := newEntry()
	entry.IsDirectory = true
	entry.Name = filepath.Base(path)
	entry.Path = path

	wg := sync.WaitGroup{}
	wg.Add(1)
	go processChild(entry, c, &wg)
	go func() {
		wg.Wait()
		close(c)
	}()

	return entry, nil
}

func getEntries(path string) ([]os.DirEntry, error) {
	return os.ReadDir(path)
}

func processChild(parent *Entry, c chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()

	dirEntries, err := getEntries(parent.Path)
	if err != nil {
		parent.Err = err
		return
	}

	childFilesSize := uint64(0)
	for _, dirEntry := range dirEntries {
		entryName := dirEntry.Name()
		entryPath := filepath.Join(parent.Path, entryName)

		entry := newEntry()
		entry.Parent = parent
		entry.Name = entryName
		entry.Path = entryPath

		if dirEntry.IsDir() {
			entry.IsDirectory = true
			wg.Add(1)
			go processChild(entry, c, wg)
		} else {
			info, err := dirEntry.Info()
			if err != nil {
				entry.Err = err
			} else {
				entry.Size = uint64(info.Size())
				childFilesSize += entry.Size
			}
		}

		parent.Children = append(parent.Children, entry)
	}

	// Add children file sizes to parent and all ancestors
	// to properly reflect total size across all levels
	current := parent
	for current != nil {
		atomic.AddUint64(&current.Size, childFilesSize)
		current = current.Parent
	}

	// Update is not critical. Ignore if channel is currently busy
	select {
	case c <- struct{}{}:
	default:
	}
}
