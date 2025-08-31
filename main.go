package main

import (
	"fmt"
	"log"
	"math"
	"os"

	"golang.design/x/clipboard"
)

func main() {
	dirname, err := parseInput()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Println()
		fmt.Println("Usage: disk-usage [PATH]")
		os.Exit(1)
	}

	if err = clipboard.Init(); err != nil {
		log.Fatalf("failed to init clipboard: %v", err)
	}

	topLevel, err := getDirSize(dirname)
	if err != nil {
		log.Fatal(err)
	}

	app := newApplication(&topLevel)

	if err := app.Run(); err != nil {
		panic(err)
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

		return fmt.Sprintf("%d.%d %s", whole, fractional, suffixes[index])
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
