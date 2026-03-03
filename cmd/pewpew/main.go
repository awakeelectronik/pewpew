package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"pewpew/internal/app"
)

const version = "0.1.0-alpha"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version":
			fmt.Printf("pewpew %s\n", version)
			return
		case "start":
			// OK, continuar
		case "help":
			printHelp()
			return
		default:
			fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
			printHelp()
			os.Exit(1)
		}
	}

	// Crear aplicación
	application, err := app.New()
	if err != nil {
		log.Fatalf("failed to initialize app: %v", err)
	}

	// Context con cancel en signals
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Iniciar app
	if err := application.Start(ctx); err != nil {
		log.Fatalf("failed to start app: %v", err)
	}

	// Esperar signals (SIGINT, SIGTERM)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("shutting down...")
	application.Stop()
}

func printHelp() {
	fmt.Print(`
pewpew — security dashboard for VPS

Usage:
  pewpew [command]

Commands:
  start     Start pewpew (default)
  version   Print version
  help      Show this help

Examples:
  pewpew              # Start with defaults
  pewpew start        # Explicit start
  pewpew version      # Show version
`)
}
