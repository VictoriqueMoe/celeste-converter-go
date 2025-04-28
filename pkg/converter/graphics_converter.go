package converter

import (
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"

	"github.com/sirupsen/logrus"
)

// GraphicsConverter handles the conversion between the Celeste DATA format and PNG images
type GraphicsConverter struct {
	log *logrus.Logger
}

// NewGraphicsConverter creates a new GraphicsConverter instance
func NewGraphicsConverter() *GraphicsConverter {
	return &GraphicsConverter{
		log: logrus.StandardLogger(),
	}
}

// DataToPng converts from Celeste's DATA format to a PNG image
func (g *GraphicsConverter) DataToPng(input io.Reader, output io.Writer) error {
	// Read image header (width, height, alpha flag)
	var width, height int32
	var alphaFlag int32 // Changed to int32 to match binary format

	if err := binary.Read(input, binary.LittleEndian, &width); err != nil {
		return err
	}
	if err := binary.Read(input, binary.LittleEndian, &height); err != nil {
		return err
	}
	if err := binary.Read(input, binary.LittleEndian, &alphaFlag); err != nil {
		return err
	}

	hasAlpha := alphaFlag != 0 // Convert integer flag to boolean

	g.log.Infof("DATA image parameters: %dx%d, %s", width, height,
		boolToFormat(hasAlpha))

	if width <= 0 || height <= 0 || width > 8192 || height > 8192 {
		return errors.New("invalid image dimensions")
	}

	img := image.NewRGBA(image.Rect(0, 0, int(width), int(height)))

	for y := 0; y < int(height); y++ {
		for x := 0; x < int(width); x++ {
			if hasAlpha {
				img.SetRGBA(x, y, color.RGBA{0, 0, 0, 0})
			} else {
				img.SetRGBA(x, y, color.RGBA{0, 0, 0, 255})
			}
		}
	}

	i := 0
	for i < int(width*height) {
		// Read RLE count
		var countBuf [1]byte
		n, err := input.Read(countBuf[:])
		if err != nil {
			if err == io.EOF {
				// If we've reached EOF, we'll just use what we have so far
				g.log.Warnf("Reached end of file with %d/%d pixels processed", i, int(width*height))
				break
			}
			return err
		}
		if n != 1 {
			return errors.New("failed to read count byte")
		}

		count := int(countBuf[0])
		if count == 0 {
			count = 256 // Treat 0 as 256
		}

		var r, g, b, a byte = 0, 0, 0, 255 // Default to opaque black

		if hasAlpha {
			var alphaBuf [1]byte
			n, err := input.Read(alphaBuf[:])
			if err != nil {
				if err == io.EOF {
					break
				}
				return err
			}
			if n != 1 {
				return errors.New("failed to read alpha byte")
			}

			a = alphaBuf[0]

			// Only read RGB if alpha is non-zero
			if a != 0 {
				var rgbBuf [3]byte
				n, err := io.ReadFull(input, rgbBuf[:])
				if err != nil {
					if err == io.EOF {
						break
					}
					return err
				}
				if n != 3 {
					return errors.New("failed to read RGB bytes")
				}

				b, g, r = rgbBuf[0], rgbBuf[1], rgbBuf[2]
			}
		} else {
			// Always read RGB for non-alpha images
			var rgbBuf [3]byte
			n, err := io.ReadFull(input, rgbBuf[:])
			if err != nil {
				if err == io.EOF {
					break
				}
				return err
			}
			if n != 3 {
				return errors.New("failed to read RGB bytes")
			}

			b, g, r = rgbBuf[0], rgbBuf[1], rgbBuf[2]
		}

		// Make sure we don't exceed image bounds
		pixelsLeft := int(width*height) - i
		if count > pixelsLeft {
			count = pixelsLeft
		}

		// Apply the run-length encoding
		c := color.RGBA{R: r, G: g, B: b, A: a}
		for j := 0; j < count; j++ {
			x := (i + j) % int(width)
			y := (i + j) / int(width)
			img.SetRGBA(x, y, c)
		}

		i += count
	}

	// Encode to PNG even if we didn't fill all pixels
	return png.Encode(output, img)
}

// PngToData converts from a PNG image to Celeste's DATA format
func (g *GraphicsConverter) PngToData(input io.Reader, output io.Writer) error {
	// Decode the PNG
	img, err := png.Decode(input)
	if err != nil {
		return err
	}

	bounds := img.Bounds()
	width := bounds.Max.X - bounds.Min.X
	height := bounds.Max.Y - bounds.Min.Y

	// Determine if we need to handle alpha
	hasAlpha := hasAlphaChannel(img)

	g.log.Infof("PNG image parameters: %dx%d, %s", width, height,
		boolToFormat(hasAlpha))

	// Write image header
	if err := binary.Write(output, binary.LittleEndian, int32(width)); err != nil {
		return err
	}
	if err := binary.Write(output, binary.LittleEndian, int32(height)); err != nil {
		return err
	}

	// Write alpha flag as int32 to match the binary format expected
	var alphaFlag int32 = 0
	if hasAlpha {
		alphaFlag = 1
	}
	if err := binary.Write(output, binary.LittleEndian, alphaFlag); err != nil {
		return err
	}

	// Compress and write pixel data
	i := 0
	for i < width*height {
		// Get current pixel
		x := i % width
		y := i / width
		r, g, b, a := getRGBA(img, x, y)

		// Calculate run length by looking ahead
		count := 1
		for {
			// Don't step out of bounds
			if i+count >= width*height {
				break
			}

			// Compare with next pixel color
			x2 := (i + count) % width
			y2 := (i + count) / width
			r2, g2, b2, a2 := getRGBA(img, x2, y2)

			if r != r2 || g != g2 || b != b2 || a != a2 {
				break
			}

			// Increment, but don't exceed maximum 8-bit value
			count++
			if count >= 256 {
				count = 256
				break
			}
		}

		// Write RLE count (0 for 256)
		countByte := uint8(count)
		if count == 256 {
			countByte = 0
		}
		if err := binary.Write(output, binary.LittleEndian, countByte); err != nil {
			return err
		}

		// Write pixel data
		if hasAlpha {
			// Write alpha value
			if err := binary.Write(output, binary.LittleEndian, a); err != nil {
				return err
			}

			// Only write color channels for non-transparent pixels
			if a != 0 {
				if err := binary.Write(output, binary.LittleEndian, b); err != nil {
					return err
				}
				if err := binary.Write(output, binary.LittleEndian, g); err != nil {
					return err
				}
				if err := binary.Write(output, binary.LittleEndian, r); err != nil {
					return err
				}
			}
		} else {
			// Always write color channels for non-alpha images
			if err := binary.Write(output, binary.LittleEndian, b); err != nil {
				return err
			}
			if err := binary.Write(output, binary.LittleEndian, g); err != nil {
				return err
			}
			if err := binary.Write(output, binary.LittleEndian, r); err != nil {
				return err
			}
		}

		i += count
	}

	return nil
}

// Helper function to get RGBA values from any image type
func getRGBA(img image.Image, x, y int) (r, g, b, a uint8) {
	c := img.At(x, y)
	r16, g16, b16, a16 := c.RGBA()
	return uint8(r16 >> 8), uint8(g16 >> 8), uint8(b16 >> 8), uint8(a16 >> 8)
}

// Helper function to convert boolean to image format string
func boolToFormat(hasAlpha bool) string {
	if hasAlpha {
		return "RGBA"
	}
	return "RGB"
}

// Helper function to detect if an image has an alpha channel with non-255 values
func hasAlphaChannel(img image.Image) bool {
	switch img.(type) {
	case *image.RGBA, *image.NRGBA:
		// These types have alpha channels, but verify if any pixel actually uses it
		bounds := img.Bounds()
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				_, _, _, a := img.At(x, y).RGBA()
				if a < 0xffff { // Check if any alpha value is less than fully opaque
					return true
				}
			}
		}
	}
	return false
}
