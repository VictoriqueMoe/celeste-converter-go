package main

import (
	"flag"
	"fmt"
	"github.com/VictoriqueMoe/celeste-converter-go/pkg/converter"
	"path/filepath"
	"runtime"
	"time"

	"github.com/sirupsen/logrus"
)

func main() {
	// Set up logging
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// Define command line flags
	workers := flag.Int("workers", runtime.NumCPU(), "Number of parallel workers (default: number of CPUs)")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")
	flag.Parse()

	// Set log level based on verbose flag
	if *verbose {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}

	// Process remaining arguments
	args := flag.Args()
	if len(args) < 3 {
		logrus.Fatal("Usage: celeste-converter [options] [data2png|png2data] <from_dir> <to_dir>\n\nOptions:\n  -workers N  Number of parallel workers (default: number of CPUs)\n  -verbose    Enable verbose logging")
	}

	command := args[0]
	from := args[1]
	to := args[2]

	// Create absolute paths
	fromPath, err := filepath.Abs(from)
	if err != nil {
		logrus.Fatalf("Invalid 'from' path: %v", err)
	}

	toPath, err := filepath.Abs(to)
	if err != nil {
		logrus.Fatalf("Invalid 'to' path: %v", err)
	}

	// Log configuration
	logrus.Infof("Workers: %d", *workers)
	logrus.Debugf("Verbose: %v", *verbose)

	// Initialize converters
	graphicsConverter := converter.NewGraphicsConverter()
	filesConverter := converter.NewFilesConverter(graphicsConverter)

	// Set number of workers
	if *workers > 0 {
		filesConverter.SetMaxWorkers(*workers)
	}

	// Execute command
	startTime := time.Now()

	switch command {
	case "data2png":
		if err := filesConverter.DataToPng(fromPath, toPath); err != nil {
			logrus.Fatalf("Conversion failed: %v", err)
		}
	case "png2data":
		if err := filesConverter.PngToData(fromPath, toPath); err != nil {
			logrus.Fatalf("Conversion failed: %v", err)
		}
	default:
		logrus.Fatalf("Unrecognized command: %s", command)
	}

	// Calculate elapsed time
	elapsed := time.Since(startTime)

	fmt.Printf("Conversion completed successfully in %v\n", elapsed)
}
