package imageio_test

// Port of org.apache.pdfbox.tools.imageio.TestImageIOUtils.
//
// Its one `@Test` renders every PDF of a corpus with `PDFRenderer` and writes
// each page in six formats, then checks each file. The rendering half is
// `track/raster`'s -- nothing implements `rendering.Backend` -- and the checks
// are about the *writing*, which is this branch's:
//
//   - the resolution the file carries, which is `checkResolution` and
//     `checkBmpResolution`;
//   - what the file is, by its content rather than its name, which is
//     `checkFileTypeByContent` over the same `FileTypeDetector` the port has;
//   - that the image is not blank, which is `checkNotBlank`;
//   - and the TIFF compression, which is `checkTiffCompression`.
//
// So the pixels here are built rather than rendered. That is not a weaker test
// of a writer: a writer's contract is that the pixels and the resolution it is
// given come back out, and where they came from is the renderer's business.
// The values -- 36 dpi, the formats, the compression names -- are the Java's.

import (
	"bytes"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/util/filetypedetector"
	"github.com/shinguakira/pdfbox-go/go/tools/imageio"
)

// dpi is the Java's `float dpi = 36; // low DPI so that rendering is FAST`.
const dpi = 36

// sampleImage builds an image with more than one colour in it, which is what
// checkNotBlank insists on.
func sampleImage() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 8, 6))
	for y := 0; y < 6; y++ {
		for x := 0; x < 8; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 30), G: uint8(y * 40), B: 128, A: 255})
		}
	}
	return img
}

// bitonalImage builds a one-bit image, which is what ExtractImages converts a
// CCITT-compressed page to before writing it.
func bitonalImage() *image.Gray {
	img := image.NewGray(image.Rect(0, 0, 8, 6))
	for y := 0; y < 6; y++ {
		for x := 0; x < 8; x++ {
			if (x+y)%2 == 0 {
				img.SetGray(x, y, color.Gray{Y: 255})
			}
		}
	}
	return img
}

// TestWriteImageFormats is the Java's per-format block: write it, and the file
// is of the type its content says and carries the resolution it was given.
func TestWriteImageFormats(t *testing.T) {
	for _, c := range []struct {
		format     string
		want       filetypedetector.FileType
		resolution bool // whether the format carries one, as the Java tests it
	}{
		{"png", filetypedetector.PNG, true},
		{"jpg", filetypedetector.JPEG, true},
		{"jpeg", filetypedetector.JPEG, true},
		{"gif", filetypedetector.GIF, false},
		{"bmp", filetypedetector.BMP, true},
		{"tiff", filetypedetector.TIFF, true},
		{"tif", filetypedetector.TIFF, true},
	} {
		t.Run(c.format, func(t *testing.T) {
			var out bytes.Buffer
			written, err := imageio.WriteImageOfDPI(sampleImage(), c.format, &out, dpi)
			if err != nil {
				t.Fatalf("WriteImage: %v", err)
			}
			if !written {
				t.Fatal("WriteImage answered false")
			}
			if out.Len() == 0 {
				t.Fatal("Empty file")
			}

			if got := filetypedetector.DetectFileTypeOfBytes(out.Bytes()); got != c.want {
				t.Errorf("the content says the file is %v, want %v", got, c.want)
			}

			if !c.resolution {
				return
			}
			x, y := resolutionOf(t, c.format, out.Bytes())
			if x != dpi || y != dpi {
				t.Errorf("the resolution is %dx%d, want %dx%d", x, y, dpi, dpi)
			}
		})
	}
}

// TestWriteImageRefusesAnUnknownFormat is the `LOG.error("No ImageWriter found
// for '{}' format")` arm, which answers false rather than throwing.
func TestWriteImageRefusesAnUnknownFormat(t *testing.T) {
	var out bytes.Buffer
	written, err := imageio.WriteImageOfDPI(sampleImage(), "webp", &out, dpi)
	if err != nil {
		t.Fatalf("an unknown format is answered false, not an error: %v", err)
	}
	if written {
		t.Error("an unknown format was accepted")
	}
	if out.Len() != 0 {
		t.Errorf("%d bytes were written for a format that cannot be written", out.Len())
	}
}

// TestWriteImageToFile is the overload that takes a filename and picks the
// format off the suffix.
func TestWriteImageToFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "page-1.png")
	written, err := imageio.WriteImageToFile(sampleImage(), path, dpi)
	if err != nil {
		t.Fatalf("WriteImageToFile: %v", err)
	}
	if !written {
		t.Fatal("WriteImageToFile answered false")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading it back: %v", err)
	}
	if got := filetypedetector.DetectFileTypeOfBytes(content); got != filetypedetector.PNG {
		t.Errorf("the file is %v, want PNG", got)
	}
}

// TestWrittenImageReadsBack is `checkImageFileSizeAndContent`: the file has the
// size it was given and more than one colour in it.
func TestWrittenImageReadsBack(t *testing.T) {
	for _, format := range []string{"png", "gif", "bmp", "tiff"} {
		t.Run(format, func(t *testing.T) {
			var out bytes.Buffer
			if _, err := imageio.WriteImageOfDPI(sampleImage(), format, &out, dpi); err != nil {
				t.Fatalf("WriteImage: %v", err)
			}
			decoded, err := decodeAny(t, format, out.Bytes())
			if err != nil {
				t.Fatalf("the file this port wrote did not read back: %v", err)
			}
			bounds := decoded.Bounds()
			if bounds.Dx() != 8 || bounds.Dy() != 6 {
				t.Errorf("the image read back is %dx%d, want 8x6", bounds.Dx(), bounds.Dy())
			}
			colours := map[color.Color]bool{}
			for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
				for x := bounds.Min.X; x < bounds.Max.X; x++ {
					colours[decoded.At(x, y)] = true
				}
			}
			if len(colours) < 2 {
				t.Errorf("the image read back has %d colours; it has less than two",
					len(colours))
			}
		})
	}
}

// TestBitonalTiffStaysOneBit is what ExtractImages converts an image for.
//
// Java converts a CCITT-compressed page to TYPE_BYTE_BINARY "so that a G4
// compressed TIFF image is created". This port writes the TIFF uncompressed --
// see the A0 record in migration/STATUS.md -- but the depth is the half of that
// which does not depend on the compression, and it is the half that decides
// whether the file is a bitonal image at all.
func TestBitonalTiffStaysOneBit(t *testing.T) {
	var out bytes.Buffer
	if _, err := imageio.WriteImageOfDPI(bitonalImage(), "tiff", &out, dpi); err != nil {
		t.Fatalf("WriteImage: %v", err)
	}
	if got := bitsPerSampleOfTIFF(t, out.Bytes()); got != 1 {
		t.Errorf("the TIFF is %d bits per sample, want 1", got)
	}
	if got := compressionOfTIFF(t, out.Bytes()); got != 1 {
		t.Errorf("the TIFF compression tag is %d, want 1 (none). Java writes "+
			"CCITT T.6 here; see migration/STATUS.md", got)
	}
}

// TestWriteImageToFileLeavesAnEmptyFileForAFormatItCannotWrite is the order
// Java's filename overload does two things in.
//
// `writeImage(image, filename, dpi, quality)` opens the file --
// `new BufferedOutputStream(new FileOutputStream(filename))` -- and only then
// takes the format off the name and goes looking for a writer. A format
// nothing can write therefore leaves an empty file behind and answers false,
// and the port does the same rather than tidying up after the Java.
func TestWriteImageToFileLeavesAnEmptyFileForAFormatItCannotWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "page-1.webp")
	written, err := imageio.WriteImageToFile(sampleImage(), path, dpi)
	if err != nil {
		t.Fatalf("an unknown format is answered false, not an error: %v", err)
	}
	if written {
		t.Error("an unknown format was accepted")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("the file was not created: %v", err)
	}
	if info.Size() != 0 {
		t.Errorf("the file is %d bytes, want none written into it", info.Size())
	}
}

// TestFilenameOverloadPicksTheSameQuality is the quality Java's filename
// overload chooses, which is the one the format overload chooses: 0 for PNG --
// "PDFBOX-4655: prevent huge PNG files on jdk11 / jdk12 / jdk13" -- and 1 for
// everything else.
//
// The two are the same code in Java and are two functions here, so the check is
// that the files come out identical.
func TestFilenameOverloadPicksTheSameQuality(t *testing.T) {
	for _, format := range []string{"png", "jpg", "gif", "bmp", "tiff"} {
		t.Run(format, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "page-1."+format)
			if _, err := imageio.WriteImageToFile(sampleImage(), path, dpi); err != nil {
				t.Fatalf("WriteImageToFile: %v", err)
			}
			viaName, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}

			var viaFormat bytes.Buffer
			if _, err := imageio.WriteImageOfDPI(sampleImage(), format, &viaFormat, dpi); err != nil {
				t.Fatalf("WriteImageOfDPI: %v", err)
			}
			if !bytes.Equal(viaName, viaFormat.Bytes()) {
				t.Errorf("the file written by name is %d bytes and the one written "+
					"by format is %d; the two overloads pick the same quality",
					len(viaName), viaFormat.Len())
			}
		})
	}
}

// TestWBMPLengthsRunToMoreThanOneByte is the format's variable-length integer,
// which the eight-pixel images above never reach the second byte of.
//
// A page rendered at the Java test's 36 dpi is about 300 pixels across, so
// every real WBMP takes two bytes for its width.
func TestWBMPLengthsRunToMoreThanOneByte(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 300, 130))
	img.SetGray(0, 0, color.Gray{Y: 255})

	var out bytes.Buffer
	if _, err := imageio.WriteImageOfDPI(img, "wbmp", &out, dpi); err != nil {
		t.Fatalf("WriteImage: %v", err)
	}
	content := out.Bytes()
	// 300 is 0b10_0101100: 0x82 0x2c, the high bit set on all but the last.
	// 130 is 0b1_0000010: 0x81 0x02.
	want := []byte{0, 0, 0x82, 0x2c, 0x81, 0x02}
	if len(content) < len(want) || !bytes.Equal(content[:len(want)], want) {
		t.Fatalf("the header is % x, want % x", content[:min(len(want), len(content))], want)
	}
	if got := len(content) - len(want); got != (300+7)/8*130 {
		t.Errorf("the rows are %d bytes, want %d", got, (300+7)/8*130)
	}
}

// TestGreyTIFFStaysEightBit is the middle arm of tiffSamples: an image that is
// grey but uses more than two values is one channel and eight bits, not one
// bit.
func TestGreyTIFFStaysEightBit(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 8, 6))
	for y := 0; y < 6; y++ {
		for x := 0; x < 8; x++ {
			img.SetGray(x, y, color.Gray{Y: uint8(x*30 + y)})
		}
	}

	var out bytes.Buffer
	if _, err := imageio.WriteImageOfDPI(img, "tiff", &out, dpi); err != nil {
		t.Fatalf("WriteImage: %v", err)
	}
	content := out.Bytes()
	if got := bitsPerSampleOfTIFF(t, content); got != 8 {
		t.Errorf("BitsPerSample is %d, want 8", got)
	}
	if got := tiffTagValue(t, content, 0x0115); got != 1 {
		t.Errorf("SamplesPerPixel is %d, want 1", got)
	}
	if got := tiffTagValue(t, content, 0x0106); got != 1 {
		t.Errorf("PhotometricInterpretation is %d, want 1 (BlackIsZero); "+
			"WhiteIsZero is the bitonal case and this is not one", got)
	}
}
