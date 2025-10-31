package imageprocessor

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"

	"golang.org/x/image/draw"
)

// ImageSize represents different image sizes
type ImageSize struct {
	Name   string
	Width  int
	Height int
}

var (
	// Predefined image sizes
	SizeThumbnail = ImageSize{Name: "thumbnail", Width: 150, Height: 150}
	SizeSmall     = ImageSize{Name: "small", Width: 400, Height: 400}
	SizeMedium    = ImageSize{Name: "medium", Width: 800, Height: 800}
	SizeLarge     = ImageSize{Name: "large", Width: 1600, Height: 1600}
)

// Processor handles image processing operations
type Processor struct {
	quality int // JPEG quality (1-100)
}

// NewProcessor creates a new image processor
func NewProcessor(quality int) *Processor {
	if quality <= 0 || quality > 100 {
		quality = 85 // Default quality
	}
	return &Processor{
		quality: quality,
	}
}

// ProcessImage processes an image: decodes, resizes, and encodes
func (p *Processor) ProcessImage(reader io.Reader, size ImageSize, format string) (io.Reader, error) {
	// Decode image
	img, imgFormat, err := image.Decode(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Resize image
	resized := p.resize(img, size.Width, size.Height)

	// Encode image
	var buf bytes.Buffer
	switch format {
	case "jpeg", "jpg":
		if err := jpeg.Encode(&buf, resized, &jpeg.Options{Quality: p.quality}); err != nil {
			return nil, fmt.Errorf("failed to encode JPEG: %w", err)
		}
	case "png":
		if err := png.Encode(&buf, resized); err != nil {
			return nil, fmt.Errorf("failed to encode PNG: %w", err)
		}
	default:
		// Use original format
		switch imgFormat {
		case "jpeg", "jpg":
			if err := jpeg.Encode(&buf, resized, &jpeg.Options{Quality: p.quality}); err != nil {
				return nil, fmt.Errorf("failed to encode JPEG: %w", err)
			}
		case "png":
			if err := png.Encode(&buf, resized); err != nil {
				return nil, fmt.Errorf("failed to encode PNG: %w", err)
			}
		default:
			return nil, fmt.Errorf("unsupported image format: %s", imgFormat)
		}
	}

	return &buf, nil
}

// resize resizes an image maintaining aspect ratio
func (p *Processor) resize(img image.Image, maxWidth, maxHeight int) image.Image {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Calculate new dimensions maintaining aspect ratio
	ratio := float64(width) / float64(height)
	newWidth := maxWidth
	newHeight := maxHeight

	if float64(maxWidth)/float64(maxHeight) > ratio {
		newWidth = int(float64(maxHeight) * ratio)
	} else {
		newHeight = int(float64(maxWidth) / ratio)
	}

	// Create new image
	dst := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))

	// Resize using high-quality algorithm
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)

	return dst
}

// GetImageDimensions returns the dimensions of an image
func GetImageDimensions(reader io.Reader) (width, height int, err error) {
	img, _, err := image.Decode(reader)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to decode image: %w", err)
	}

	bounds := img.Bounds()
	return bounds.Dx(), bounds.Dy(), nil
}

// IsValidImage checks if the reader contains a valid image
func IsValidImage(reader io.Reader) bool {
	_, _, err := image.Decode(reader)
	return err == nil
}
