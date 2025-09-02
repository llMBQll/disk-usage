package main

import (
	"fmt"
	"log"
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
