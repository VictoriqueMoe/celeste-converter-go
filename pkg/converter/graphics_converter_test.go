package converter

import (
	"bytes"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// List of test images for multiple conversion test
var testImages = []string{
	"white",
	"red",
	"green",
	"blue",
	"cyan",
	"magenta",
	"yellow",
	"black",
	"transparent",
	"multi-color",
}

// TestDataToPngRoundTrip tests that the image survives a round trip conversion
func TestDataToPngRoundTrip(t *testing.T) {
	graphicsConverter := NewGraphicsConverter()

	for _, imageName := range testImages {
		t.Run(imageName, func(t *testing.T) {
			// Skip files that don't exist
			dataPath := filepath.Join("testdata", "data", imageName+".data")
			if _, err := os.Stat(dataPath); os.IsNotExist(err) {
				t.Skipf("Skipping test for %s - file not found", imageName)
				return
			}

			// Start with DATA file
			dataBytes := readTestResource(t, filepath.Join("data", imageName+".data"))

			// First conversion: DATA -> PNG
			firstPngBytes := dataToPngBytes(t, graphicsConverter, dataBytes)
			firstPngImage := bytesToImage(t, firstPngBytes)

			// Second conversion: PNG -> DATA
			secondDataBytes := pngToDataBytes(t, graphicsConverter, firstPngBytes)

			// Third conversion: DATA -> PNG
			secondPngBytes := dataToPngBytes(t, graphicsConverter, secondDataBytes)
			secondPngImage := bytesToImage(t, secondPngBytes)

			// Compare first and second PNG images with tolerance
			assertImageEquals(t, firstPngImage, secondPngImage, 5)
		})
	}
}

// TestPngToDataRoundTrip tests that the image survives a round trip conversion
// PNG -> DATA -> PNG, comparing the first PNG to the second PNG
func TestPngToDataRoundTrip(t *testing.T) {
	graphicsConverter := NewGraphicsConverter()

	for _, imageName := range testImages {
		t.Run(imageName, func(t *testing.T) {
			// Skip files that don't exist
			pngPath := filepath.Join("testdata", "png", imageName+".png")
			if _, err := os.Stat(pngPath); os.IsNotExist(err) {
				t.Skipf("Skipping test for %s - file not found", imageName)
				return
			}

			// Start with PNG file
			originalPngBytes := readTestResource(t, filepath.Join("png", imageName+".png"))
			originalPngImage := bytesToImage(t, originalPngBytes)

			// First conversion: PNG -> DATA
			dataBytes := pngToDataBytes(t, graphicsConverter, originalPngBytes)

			// Second conversion: DATA -> PNG
			convertedPngBytes := dataToPngBytes(t, graphicsConverter, dataBytes)
			convertedPngImage := bytesToImage(t, convertedPngBytes)

			// Compare original and converted PNG images with tolerance
			assertImageEquals(t, originalPngImage, convertedPngImage, 5)
		})
	}
}

// TestFilesConverterRoundTrip tests the FilesConverter through a complete round trip
func TestFilesConverterRoundTrip(t *testing.T) {
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

	pngDir2, err := os.MkdirTemp("", "celeste-test-png2")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(pngDir2)

	// Copy test files to dataDir
	setupTestFiles(t, dataDir, ".data", "data")

	// Initialize converters
	graphicsConverter := NewGraphicsConverter()
	filesConverter := NewFilesConverter(graphicsConverter)

	// Run the conversions in sequence:
	// 1. DATA -> PNG
	if err := filesConverter.DataToPng(dataDir, pngDir); err != nil {
		t.Fatalf("First DataToPng conversion failed: %v", err)
	}

	// 2. PNG -> DATA
	if err := filesConverter.PngToData(pngDir, dataDir2); err != nil {
		t.Fatalf("PngToData conversion failed: %v", err)
	}

	// 3. DATA -> PNG (again)
	if err := filesConverter.DataToPng(dataDir2, pngDir2); err != nil {
		t.Fatalf("Second DataToPng conversion failed: %v", err)
	}

	// Compare the PNG files from the first and second conversion
	for _, imgName := range testImages {
		firstPngPath := filepath.Join(pngDir, imgName+".png")
		secondPngPath := filepath.Join(pngDir2, imgName+".png")

		// Skip files that don't exist
		if _, err := os.Stat(firstPngPath); os.IsNotExist(err) {
			continue
		}
		if _, err := os.Stat(secondPngPath); os.IsNotExist(err) {
			continue
		}

		// Read both files
		firstPngData, err := os.ReadFile(firstPngPath)
		if err != nil {
			t.Fatalf("Failed to read first PNG file %s: %v", firstPngPath, err)
		}

		secondPngData, err := os.ReadFile(secondPngPath)
		if err != nil {
			t.Fatalf("Failed to read second PNG file %s: %v", secondPngPath, err)
		}

		// Decode PNG images
		firstImage := bytesToImage(t, firstPngData)
		secondImage := bytesToImage(t, secondPngData)

		// Compare with tolerance
		assertImageEquals(t, firstImage, secondImage, 5)
	}
}

// Helper functions

// Helper function for test files
func setupTestFiles(t *testing.T, dir string, fileExtension string, resourceDir string) {
	// Create the directory if it doesn't exist
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("Failed to create directory %s: %v", dir, err)
	}

	for _, imgName := range testImages {
		sourcePath := filepath.Join("testdata", resourceDir, imgName+fileExtension)
		destPath := filepath.Join(dir, imgName+fileExtension)

		// Check if source file exists
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			t.Logf("Warning: Source file %s does not exist, skipping", sourcePath)
			continue
		}

		copyFile(t, sourcePath, destPath)
	}
}

// readTestResource reads a test resource file into a byte array
func readTestResource(t *testing.T, resource string) []byte {
	// Assuming test resources are in a 'testdata' directory
	filePath := filepath.Join("testdata", resource)
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read test resource %s: %v", resource, err)
	}
	return data
}

// dataToPngBytes converts DATA format bytes to PNG format bytes
func dataToPngBytes(t *testing.T, converter *GraphicsConverter, dataBytes []byte) []byte {
	input := bytes.NewReader(dataBytes)
	output := new(bytes.Buffer)

	err := converter.DataToPng(input, output)
	if err != nil {
		t.Fatalf("Failed to convert DATA to PNG: %v", err)
	}

	return output.Bytes()
}

// pngToDataBytes converts PNG format bytes to DATA format bytes
func pngToDataBytes(t *testing.T, converter *GraphicsConverter, pngBytes []byte) []byte {
	input := bytes.NewReader(pngBytes)
	output := new(bytes.Buffer)

	err := converter.PngToData(input, output)
	if err != nil {
		t.Fatalf("Failed to convert PNG to DATA: %v", err)
	}

	return output.Bytes()
}

// bytesToImage converts a byte array to an image.Image
func bytesToImage(t *testing.T, imgBytes []byte) image.Image {
	img, err := png.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		t.Fatalf("Failed to decode PNG: %v", err)
	}
	return img
}

// assertImageEquals asserts that two images are equal in dimensions and pixel data
// with a tolerance for color variations
func assertImageEquals(t *testing.T, expected, actual image.Image, tolerance int) {
	// Check dimensions
	expectedBounds := expected.Bounds()
	actualBounds := actual.Bounds()

	if expectedBounds.Dx() != actualBounds.Dx() || expectedBounds.Dy() != actualBounds.Dy() {
		t.Fatalf("Image dimensions don't match: expected %dx%d, got %dx%d",
			expectedBounds.Dx(), expectedBounds.Dy(), actualBounds.Dx(), actualBounds.Dy())
	}

	// Check pixel by pixel with tolerance
	for y := expectedBounds.Min.Y; y < expectedBounds.Max.Y; y++ {
		for x := expectedBounds.Min.X; x < expectedBounds.Max.X; x++ {
			expectedR, expectedG, expectedB, expectedA := expected.At(x, y).RGBA()
			actualR, actualG, actualB, actualA := actual.At(x, y).RGBA()

			// Check if colors are within tolerance (converting from 16-bit to 8-bit for comparison)
			if !colorsWithinTolerance(expectedR>>8, actualR>>8, tolerance) ||
				!colorsWithinTolerance(expectedG>>8, actualG>>8, tolerance) ||
				!colorsWithinTolerance(expectedB>>8, actualB>>8, tolerance) ||
				!colorsWithinTolerance(expectedA>>8, actualA>>8, tolerance) {
				t.Fatalf("Pixel mismatch at (%d,%d): expected rgba(%d,%d,%d,%d), got rgba(%d,%d,%d,%d)",
					x, y, expectedR>>8, expectedG>>8, expectedB>>8, expectedA>>8,
					actualR>>8, actualG>>8, actualB>>8, actualA>>8)
			}
		}
	}
}

// colorsWithinTolerance checks if two color values are within the specified tolerance
func colorsWithinTolerance(c1, c2 uint32, tolerance int) bool {
	diff := int(c1) - int(c2)
	return math.Abs(float64(diff)) <= float64(tolerance)
}
