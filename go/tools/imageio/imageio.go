// Package imageio writes a raster out to a file.
//
// Port of org.apache.pdfbox.tools.imageio, and **a substitution rather than a
// transliteration**. `ImageIOUtil` is a shell around `javax.imageio`: it asks a
// registry for a writer by format name, takes an `ImageWriteParam` and an
// `IIOMetadata` tree off it, sets a compression type by string and edits the
// metadata as a DOM. Go has none of that machinery, so what is ported is the
// decisions the Java makes — which format, which quality, which resolution goes
// into the file — over writers that are the standard library where it has one
// and are here where it has not.
//
// The A0 record in migration/STATUS.md says which is which, and names the two
// things this cannot do: TIFF is written uncompressed where Java compresses it,
// and JPEG 2000 cannot be written at all.
package imageio

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"log/slog"
	"os"
	"strings"
)

// DefaultResolution is the dpi the overloads that do not take one use.
//
// Java's `writeImage(image, formatName, output)` passes 72.
const DefaultResolution = 72

// WriteImage writes the image in the given format at the default resolution.
//
// Port of writeImage(BufferedImage, String, OutputStream).
func WriteImage(img image.Image, formatName string, output io.Writer) (bool, error) {
	return WriteImageOfDPI(img, formatName, output, DefaultResolution)
}

// WriteImageOfDPI writes the image in the given format at the given resolution.
//
// Port of writeImage(BufferedImage, String, OutputStream, int), including the
// quality it chooses: 1 for everything, and 0 for PNG, which Java does to
// "prevent huge PNG files on jdk11 / jdk12 / jdk13" (PDFBOX-4655).
func WriteImageOfDPI(img image.Image, formatName string, output io.Writer,
	dpi int) (bool, error) {
	quality := float32(1)
	if strings.EqualFold(formatName, "png") {
		quality = 0
	}
	return WriteImageOfQuality(img, formatName, output, dpi, quality)
}

// WriteImageOfQuality writes the image with the given compression quality,
// which is a fraction from 0 to 1.
//
// Port of writeImage(BufferedImage, String, OutputStream, int, float). Java's
// last overload also takes a compression type by name, which is a
// `javax.imageio` string this port has nothing to do with: what compression
// each format gets is decided in the writers below and recorded in
// migration/STATUS.md.
func WriteImageOfQuality(img image.Image, formatName string, output io.Writer,
	dpi int, compressionQuality float32) (bool, error) {
	writer := writerFor(formatName)
	if writer == nil {
		// Java logs "No ImageWriter found for '{}' format" and the formats it
		// does have, and answers false. It does not throw.
		slogNoWriter(formatName)
		return false, nil
	}
	if err := writer(img, output, dpi, compressionQuality); err != nil {
		return false, err
	}
	return true, nil
}

// WriteImageToFile writes the image to a file, taking the format from the name
// after the last dot.
//
// Port of writeImage(BufferedImage, String, int), whose filename carries the
// format the same way.
func WriteImageToFile(img image.Image, filename string, dpi int) (bool, error) {
	return WriteImageToFileOfQuality(img, filename, dpi, qualityFor(filename))
}

// WriteImageToFileOfQuality is the same with the quality given.
//
// Port of writeImage(BufferedImage, String, int, float).
func WriteImageToFileOfQuality(img image.Image, filename string, dpi int,
	compressionQuality float32) (bool, error) {
	// Java opens the file before it goes looking for a writer -- the
	// try-with-resources is `new BufferedOutputStream(new
	// FileOutputStream(filename))` and the format name is taken inside it -- so
	// a format nothing can write still leaves an empty file behind. This does
	// the same.
	file, err := os.Create(filename)
	if err != nil {
		return false, err
	}
	defer file.Close()

	formatName := filename[strings.LastIndex(filename, ".")+1:]
	return WriteImageOfQuality(img, formatName, file, dpi, compressionQuality)
}

// qualityFor is the quality Java's filename overload picks, which is the one
// WriteImageOfDPI picks.
func qualityFor(filename string) float32 {
	if strings.EqualFold(filename[strings.LastIndex(filename, ".")+1:], "png") {
		return 0
	}
	return 1
}

// imageWriter writes one format.
type imageWriter func(img image.Image, output io.Writer, dpi int, quality float32) error

// writerFor answers the writer of a format, or nil where there is none.
//
// This is `ImageIO.getImageWritersByFormatName`, reduced to what it can answer
// here. The names are the ones PDFToImage and ExtractImages pass.
func writerFor(formatName string) imageWriter {
	switch strings.ToLower(formatName) {
	case "png":
		return writePNG
	case "jpg", "jpeg":
		return writeJPEG
	case "gif":
		return writeGIF
	case "bmp":
		return writeBMP
	case "wbmp":
		return writeWBMP
	case "tif", "tiff":
		return writeTIFF
	}
	return nil
}

// writePNG writes a PNG with its resolution in a pHYs chunk.
func writePNG(img image.Image, output io.Writer, dpi int, quality float32) error {
	encoder := png.Encoder{CompressionLevel: png.DefaultCompression}
	if quality == 0 {
		// PDFBOX-4655: Java passes 0 to keep the file from taking a long time
		// and coming out large. Go's nearest is the fastest level.
		encoder.CompressionLevel = png.BestSpeed
	}
	var buffer bytes.Buffer
	if err := encoder.Encode(&buffer, img); err != nil {
		return fmt.Errorf("imageio: writing a PNG: %w", err)
	}
	withResolution, err := pngWithResolution(buffer.Bytes(), dpi)
	if err != nil {
		return err
	}
	_, err = output.Write(withResolution)
	return err
}

// writeJPEG writes a JPEG with its resolution in the JFIF density fields.
func writeJPEG(img image.Image, output io.Writer, dpi int, quality float32) error {
	// Java's compressionQuality is a fraction and Go's Quality is 1 to 100.
	options := &jpeg.Options{Quality: int(quality*100 + 0.5)}
	if options.Quality < 1 {
		options.Quality = 1
	}
	var buffer bytes.Buffer
	if err := jpeg.Encode(&buffer, img, options); err != nil {
		return fmt.Errorf("imageio: writing a JPEG: %w", err)
	}
	withResolution, err := jpegWithResolution(buffer.Bytes(), dpi)
	if err != nil {
		return err
	}
	_, err = output.Write(withResolution)
	return err
}

// writeGIF writes a GIF, which carries no resolution.
//
// Java writes none either: "no META data possible for GIF, thus no dpi test".
func writeGIF(img image.Image, output io.Writer, dpi int, quality float32) error {
	if err := gif.Encode(output, img, nil); err != nil {
		return fmt.Errorf("imageio: writing a GIF: %w", err)
	}
	return nil
}

// slogNoWriter is Java's two error lines when no writer answers to a format
// name, which it logs and then returns false.
func slogNoWriter(formatName string) {
	slog.Error("imageio: no image writer found for format", "format", formatName)
	slog.Error("imageio: supported formats", "formats",
		[]string{"png", "jpg", "jpeg", "gif", "bmp", "wbmp", "tif", "tiff"})
}
