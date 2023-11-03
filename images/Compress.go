package images

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"sync"
)

type Compressor struct {
	QualityStep int // Quality adjustment step
	MinQuality  int // Minimum quality
	MaxQuality  int // Maximum quality
	ThresholdKB int // Size threshold in KB
}

func NewCompressor(thresholdKB, qualityStep, minQuality, maxQuality int) *Compressor {
	return &Compressor{
		QualityStep: qualityStep,
		MinQuality:  minQuality,
		MaxQuality:  maxQuality,
		ThresholdKB: thresholdKB,
	}
}

// CompressImage handles image compression based on format.
func (c *Compressor) CompressImage(imageData io.Reader) ([]byte, error) {
	if c.ThresholdKB == 0 {
		return io.ReadAll(imageData)
	}

	// Create a buffer to copy the imageData and determine the image format.
	buffer := bytes.NewBuffer(nil)
	tee := io.TeeReader(imageData, buffer)

	// Decode image using the buffer so we don't lose the initial bytes.
	img, format, err := image.Decode(tee)
	if err != nil {
		return nil, fmt.Errorf("decoding image failed: %w", err)
	}

	// For GIFs, use the buffer which contains all bytes read from the original imageData.
	if format == "gif" {
		return c.handleGIF(buffer)
	}

	// For non-GIFs, check the initial size using a fresh buffer.
	buf := &bytes.Buffer{}
	switch format {
	case "jpeg":
		err = jpeg.Encode(buf, img, nil)
	case "png":
		err = png.Encode(buf, img)
	default:
		return nil, fmt.Errorf("unsupported image format: %s", format)
	}

	if err != nil {
		return nil, fmt.Errorf("encoding image failed: %w", err)
	}

	if buf.Len() <= c.ThresholdKB*1024 {
		// If the image is already below the threshold, return the original encoded bytes.
		return buf.Bytes(), nil
	}

	// Apply format-specific compression.
	switch format {
	case "jpeg":
		return c.compressJPEG(img)
	case "png":
		return c.compressPNG(img)
	}

	return nil, fmt.Errorf("unsupported image format: %s", format)
}

// handleGIF decodes and processes a GIF image.
func (c *Compressor) handleGIF(imageData io.Reader) ([]byte, error) {
	gifImg, err := gif.DecodeAll(imageData)
	if err != nil {
		return nil, fmt.Errorf("decoding GIF image failed: %w", err)
	}
	return c.compressGIF(gifImg)
}

func (c *Compressor) compressJPEG(img image.Image) ([]byte, error) {
	quality := c.MaxQuality
	buf := &bytes.Buffer{}

	for {
		opts := jpeg.Options{Quality: quality}
		buf.Reset()
		err := jpeg.Encode(buf, img, &opts)
		if err != nil {
			return nil, fmt.Errorf("JPEG encoding failed at quality %d: %w", quality, err)
		}

		if buf.Len() <= c.ThresholdKB*1024 || quality <= c.MinQuality {
			break
		}

		quality -= c.QualityStep
		if quality < c.MinQuality {
			quality = c.MinQuality
		}
	}

	return buf.Bytes(), nil
}

func (c *Compressor) compressPNG(img image.Image) ([]byte, error) {
	// Convert PNG to JPEG with a white background
	b := img.Bounds()
	whiteBg := image.NewRGBA(b)
	draw.Draw(whiteBg, b, image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.Draw(whiteBg, b, img, b.Min, draw.Over)
	return c.compressJPEG(whiteBg)
}

func (c *Compressor) compressGIF(originalGIF *gif.GIF) ([]byte, error) {
	// Create a new GIF to hold the compressed frames
	var compressedGIF gif.GIF
	compressedGIF.LoopCount = originalGIF.LoopCount
	compressedGIF.Disposal = originalGIF.Disposal
	compressedGIF.Config = originalGIF.Config
	compressedGIF.BackgroundIndex = originalGIF.BackgroundIndex

	for i, srcFrame := range originalGIF.Image {
		// Convert frame to RGBA to avoid paletted color issues
		b := srcFrame.Bounds()
		frame := image.NewRGBA(b)
		draw.Draw(frame, b, srcFrame, b.Min, draw.Over)

		// Create a white image same size of the frame
		whiteImage := image.NewRGBA(b)
		draw.Draw(whiteImage, b, &image.Uniform{color.White}, image.ZP, draw.Src)

		// Draw the frame onto the white image to remove transparency
		draw.Draw(whiteImage, b, frame, b.Min, draw.Over)

		// Compress the frame
		compressedFrame, err := c.compressJPEG(whiteImage)
		if err != nil {
			return nil, fmt.Errorf("compressing GIF frame failed: %w", err)
		}

		// Decode the compressed frame back to image
		jpgFrame, _, err := image.Decode(bytes.NewReader(compressedFrame))
		if err != nil {
			return nil, fmt.Errorf("decoding JPEG frame failed: %w", err)
		}

		// Convert back to paletted image for GIF
		palettedFrame := image.NewPaletted(b, srcFrame.Palette)
		draw.FloydSteinberg.Draw(palettedFrame, b, jpgFrame, b.Min)

		compressedGIF.Image = append(compressedGIF.Image, palettedFrame)
		compressedGIF.Delay = append(compressedGIF.Delay, originalGIF.Delay[i])
	}

	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, &compressedGIF); err != nil {
		return nil, fmt.Errorf("encoding compressed GIF failed: %w", err)
	}

	return buf.Bytes(), nil
}

func ProcessImages(imageData []io.Reader, compressor *Compressor) ([][]byte, error) {
	var wg sync.WaitGroup
	mu := &sync.Mutex{}
	compressedImages := make([][]byte, len(imageData))
	errChan := make(chan error, len(imageData)) // 错误通道，缓冲以避免阻塞

	wg.Add(len(imageData))
	for i, data := range imageData {
		go func(idx int, imgData io.Reader) {
			defer wg.Done()
			compressed, err := compressor.CompressImage(imgData)
			mu.Lock()
			compressedImages[idx] = compressed
			mu.Unlock()
			if err != nil {
				errChan <- fmt.Errorf("compressing image at index %d failed: %w", idx, err)
			}
		}(i, data)
	}

	wg.Wait()
	close(errChan) // 处理完所有goroutine后关闭错误通道

	// 检查错误通道中是否有错误
	for err := range errChan {
		if err != nil {
			return nil, err // 可以返回第一个错误或累积所有错误
		}
	}

	return compressedImages, nil
}
