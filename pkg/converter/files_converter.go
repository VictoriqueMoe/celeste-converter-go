package converter

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

// FilesConverter handles batch conversion of files between formats
type FilesConverter struct {
	graphicsConverter *GraphicsConverter
	log               *logrus.Logger
	maxWorkers        int // Number of concurrent workers
}

// NewFilesConverter creates a new FilesConverter instance
func NewFilesConverter(graphicsConverter *GraphicsConverter) *FilesConverter {
	numCPU := runtime.NumCPU()
	maxWorkers := numCPU
	if maxWorkers > 8 {
		maxWorkers = 8
	}

	return &FilesConverter{
		graphicsConverter: graphicsConverter,
		log:               logrus.StandardLogger(),
		maxWorkers:        maxWorkers,
	}
}

// SetMaxWorkers allows overriding the default number of workers
func (f *FilesConverter) SetMaxWorkers(workers int) {
	if workers > 0 {
		f.maxWorkers = workers
	}
}

// DataToPng converts all .data files in the source directory to .png files in the target directory
func (f *FilesConverter) DataToPng(fromDir, toDir string) error {
	f.log.Info("Converting DATA -> PNG")
	return f.convert(fromDir, toDir, ".data", ".png", f.graphicsConverter.DataToPng)
}

// PngToData converts all .png files in the source directory to .data files in the target directory
func (f *FilesConverter) PngToData(fromDir, toDir string) error {
	f.log.Info("Converting PNG -> DATA")
	return f.convert(fromDir, toDir, ".png", ".data", f.graphicsConverter.PngToData)
}

// ConversionTask represents a single file conversion task
type ConversionTask struct {
	index      int
	totalFiles int
	relPath    string
	inputPath  string
	outputPath string
}

// convert does the actual conversion between file formats using goroutines for parallelism
func (f *FilesConverter) convert(
	fromDir, toDir string,
	fromExt, toExt string,
	convertFunc func(io.Reader, io.Writer) error,
) error {
	f.log.Infof("From directory: %s", fromDir)
	f.log.Infof("To directory: %s", toDir)

	var files []string
	err := filepath.Walk(fromDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(path), strings.ToLower(fromExt)) {
			relPath, err := filepath.Rel(fromDir, path)
			if err != nil {
				return err
			}
			files = append(files, relPath)
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("error scanning directory: %w", err)
	}

	f.log.Infof("%d files to convert", len(files))

	if len(files) == 0 {
		return nil // No files to convert
	}

	var wg sync.WaitGroup

	errChan := make(chan error, len(files))

	// Create task queue
	taskQueue := make(chan ConversionTask, len(files))

	if err := os.MkdirAll(toDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory '%s': %w", toDir, err)
	}

	for i, relPath := range files {
		inputPath := filepath.Join(fromDir, relPath)
		outputDir := filepath.Join(toDir, filepath.Dir(relPath))
		outputPath := filepath.Join(outputDir, strings.TrimSuffix(filepath.Base(relPath), fromExt)+toExt)

		taskQueue <- ConversionTask{
			index:      i + 1,
			totalFiles: len(files),
			relPath:    relPath,
			inputPath:  inputPath,
			outputPath: outputPath,
		}
	}
	close(taskQueue) // No more tasks will be added

	// Create a mutex for synchronized logging
	var logMutex sync.Mutex

	// Start worker goroutines
	for w := 0; w < f.maxWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for task := range taskQueue {
				logMutex.Lock()
				f.log.Infof("[%d/%d] converting %s", task.index, task.totalFiles, task.relPath)
				logMutex.Unlock()

				outputDir := filepath.Dir(task.outputPath)
				if err := os.MkdirAll(outputDir, 0755); err != nil {
					errChan <- fmt.Errorf("failed to create output directory '%s': %w", outputDir, err)
					continue
				}

				inputFile, err := os.Open(task.inputPath)
				if err != nil {
					errChan <- fmt.Errorf("failed to open input file '%s': %w", task.inputPath, err)
					continue
				}

				outputFile, err := os.Create(task.outputPath)
				if err != nil {
					inputFile.Close()
					errChan <- fmt.Errorf("failed to create output file '%s': %w", task.outputPath, err)
					continue
				}

				err = convertFunc(inputFile, outputFile)
				if err != nil {
					errChan <- fmt.Errorf("failed to convert file '%s': %w", task.relPath, err)
					continue
				}

				err = inputFile.Close()
				if err != nil {
					return
				}

				err = outputFile.Close()
				if err != nil {
					return
				}

			}
		}()
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		return err
	}

	return nil
}
