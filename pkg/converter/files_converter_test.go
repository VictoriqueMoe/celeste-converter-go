package converter

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestFileConverterDataToPng(t *testing.T) {
	// Create temporary directories for test
	fromDir, err := os.MkdirTemp("", "celeste-test-from")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(fromDir)

	toDir, err := os.MkdirTemp("", "celeste-test-to")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(toDir)

	// Copy test files to fromDir
	setupTestDataFiles(t, fromDir)

	// Initialize converters
	graphicsConverter := NewGraphicsConverter()
	filesConverter := NewFilesConverter(graphicsConverter)

	// Run the conversion
	err = filesConverter.DataToPng(fromDir, toDir)
	if err != nil {
		t.Fatalf("DataToPng failed: %v", err)
	}

	// Verify output files
	smallTestImages := []string{
		"white", "red", "green", "blue", "cyan",
		"magenta", "yellow", "black", "transparent", "multi-color",
	}

	for _, imgName := range smallTestImages {
		outputPath := filepath.Join(toDir, imgName+".png")
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Errorf("Expected output file not found: %s", outputPath)
		}
	}
}

func TestFileConverterPngToData(t *testing.T) {
	// Create temporary directories for test
	fromDir, err := os.MkdirTemp("", "celeste-test-from")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(fromDir)

	toDir, err := os.MkdirTemp("", "celeste-test-to")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(toDir)

	// Copy test files to fromDir
	setupTestPngFiles(t, fromDir)

	// Initialize converters
	graphicsConverter := NewGraphicsConverter()
	filesConverter := NewFilesConverter(graphicsConverter)

	// Run the conversion
	err = filesConverter.PngToData(fromDir, toDir)
	if err != nil {
		t.Fatalf("PngToData failed: %v", err)
	}

	// Verify output files
	smallTestImages := []string{
		"white", "red", "green", "blue", "cyan",
		"magenta", "yellow", "black", "transparent", "multi-color",
	}

	for _, imgName := range smallTestImages {
		outputPath := filepath.Join(toDir, imgName+".data")
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Errorf("Expected output file not found: %s", outputPath)
		}
	}
}

func TestRoundTripConversion(t *testing.T) {
	// Create temporary directories for test
	dataDir, err := os.MkdirTemp("", "celeste-test-data")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(dataDir)

	pngDir, err := os.MkdirTemp("", "celeste-test-png")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(pngDir)

	dataDir2, err := os.MkdirTemp("", "celeste-test-data2")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(dataDir2)

	// Copy test files to dataDir
	setupTestDataFiles(t, dataDir)

	// Initialize converters
	graphicsConverter := NewGraphicsConverter()
	filesConverter := NewFilesConverter(graphicsConverter)

	// Run the first conversion: DATA -> PNG
	err = filesConverter.DataToPng(dataDir, pngDir)
	if err != nil {
		t.Fatalf("First DataToPng conversion failed: %v", err)
	}

	// Run the second conversion: PNG -> DATA
	err = filesConverter.PngToData(pngDir, dataDir2)
	if err != nil {
		t.Fatalf("Second PngToData conversion failed: %v", err)
	}

	// Compare original DATA files with the round-trip ones
	smallTestImages := []string{
		"white", "red", "green", "blue", "cyan",
		"magenta", "yellow", "black", "transparent", "multi-color",
	}

	for _, imgName := range smallTestImages {
		originalPath := filepath.Join(dataDir, imgName+".data")
		convertedPath := filepath.Join(dataDir2, imgName+".data")

		// Read both files
		originalData, err := os.ReadFile(originalPath)
		if err != nil {
			t.Fatalf("Failed to read original file %s: %v", originalPath, err)
		}

		convertedData, err := os.ReadFile(convertedPath)
		if err != nil {
			t.Fatalf("Failed to read converted file %s: %v", convertedPath, err)
		}

		// Convert both files to PNG and compare the images
		originalImage := bytesToImage(t, dataToPngBytes(t, graphicsConverter, originalData))
		convertedImage := bytesToImage(t, dataToPngBytes(t, graphicsConverter, convertedData))

		// Use tolerance of 3 for color comparison (slight variations are acceptable)
		assertImageEquals(t, originalImage, convertedImage, 3)
	}
}

// Helper functions for setting up test files

func setupTestDataFiles(t *testing.T, dir string) {
	smallTestImages := []string{
		"white", "red", "green", "blue", "cyan",
		"magenta", "yellow", "black", "transparent", "multi-color",
	}

	for _, imgName := range smallTestImages {
		sourcePath := filepath.Join("testdata", "data", imgName+".data")
		destPath := filepath.Join(dir, imgName+".data")

		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			t.Logf("Warning: Source file %s does not exist, skipping", sourcePath)
			continue
		}

		copyFile(t, sourcePath, destPath)
	}
}

func setupTestPngFiles(t *testing.T, dir string) {
	smallTestImages := []string{
		"white", "red", "green", "blue", "cyan",
		"magenta", "yellow", "black", "transparent", "multi-color",
	}

	for _, imgName := range smallTestImages {
		sourcePath := filepath.Join("testdata", "png", imgName+".png")
		destPath := filepath.Join(dir, imgName+".png")

		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			t.Logf("Warning: Source file %s does not exist, skipping", sourcePath)
			continue
		}

		copyFile(t, sourcePath, destPath)
	}
}

func copyFile(t *testing.T, sourcePath, destPath string) {
	source, err := os.Open(sourcePath)
	if err != nil {
		t.Fatalf("Failed to open source file %s: %v", sourcePath, err)
	}
	defer source.Close()

	dest, err := os.Create(destPath)
	if err != nil {
		t.Fatalf("Failed to create destination file %s: %v", destPath, err)
	}
	defer dest.Close()

	_, err = io.Copy(dest, source)
	if err != nil {
		t.Fatalf("Failed to copy file content from %s to %s: %v", sourcePath, destPath, err)
	}
}
