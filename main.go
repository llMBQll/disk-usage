package main

import (
	"fmt"
	"log"
	"os"

	"golang.design/x/clipboard"

	"github.com/alecthomas/kong"
)

func main() {
	kong.Parse(&CLI)

	if err := clipboard.Init(); err != nil {
		log.Fatalf("failed to init clipboard: %v", err)
	}

	repr := parseRepresentation(CLI.Representation)
	app, err := newApplication(CLI.Path, repr, CLI.NotifyOnReady)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	if err := app.Run(); err != nil {
		panic(err)
	}
}

var CLI struct {
	Path           string `arg:"" name:"path" help:"Path to analyse" type:"path" default:"."`
	Representation string `short:"r" name:"representation" help:"Representation of sizes" enum:"bytes,iec,si" default:"iec"`
	NotifyOnReady  bool   `short:"n" name:"notify-ready" help:"Show a notification when all files are processed"`
}

type Representation int

const (
	Bytes Representation = iota
	IEC
	SI
)

func parseRepresentation(representation string) Representation {
	switch representation {
	case "bytes":
		return Bytes
	case "iec":
		return IEC
	case "si":
		return SI
	default:
		// This should never happen
		fmt.Printf("Unknown representation '%s'", representation)
		os.Exit(1)
	}
	return Bytes // unreachable
}
