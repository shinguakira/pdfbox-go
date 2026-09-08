package imageio_test

// What the Java actually writes, byte for byte.
//
// `TestImageIOUtils` checks the resolution through `javax.imageio`'s readers,
// which answer a metadata tree rather than the bytes; this port has no such
// readers, so the values here were read off the files the Java writes. The
// four classes of `tools/imageio` were compiled and run over the same two
// images `imageio_test.go` builds -- an 8x6 `TYPE_INT_RGB` and an 8x6
// `TYPE_BYTE_BINARY` -- and every number below came out of that run.
//
// They are the assertions the Java test could not make, and they are what says
// the writers here put the Java's decisions into the file rather than merely
// putting something into the file.

import (
	"bytes"
	"encoding/hex"
	"image"
	"image/color"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/tools/imageio"
)

// TestPNGResolutionIsTheJavasPixelsPerMetre pins the pHYs values the Java
// writes.
//
// `ImageIOUtil.setDPI` gives the PNG writer `dpi / 25.4f` pixels per
// millimetre -- "PNG writer doesn't conform to the spec ... but instead counts
// the pixels per millimeter" -- and the writer turns that into pixels per
// metre. These three are what came out.
func TestPNGResolutionIsTheJavasPixelsPerMetre(t *testing.T) {
	for _, c := range []struct{ dpi, perMetre int }{
		{36, 1417},
		{72, 2835},
		{300, 11811},
	} {
		var out bytes.Buffer
		if _, err := imageio.WriteImageOfDPI(sampleImage(), "png", &out, c.dpi); err != nil {
			t.Fatalf("WriteImage: %v", err)
		}
		chunk := pngChunk(t, out.Bytes(), "pHYs")
		x := int(bigEndianUint32(chunk[0:4]))
		y := int(bigEndianUint32(chunk[4:8]))
		if x != c.perMetre || y != c.perMetre {
			t.Errorf("at %d dpi the pHYs chunk says %dx%d pixels per metre, "+
				"want %d", c.dpi, x, y, c.perMetre)
		}
	}
}

// TestPNGPutsTheResolutionBeforeThePixels is the chunk order the Java writes,
// which is also what the specification requires: pHYs is an ancillary chunk
// that has to come before the first IDAT.
func TestPNGPutsTheResolutionBeforeThePixels(t *testing.T) {
	var out bytes.Buffer
	if _, err := imageio.WriteImageOfDPI(sampleImage(), "png", &out, dpi); err != nil {
		t.Fatalf("WriteImage: %v", err)
	}
	kinds := pngChunkNames(t, out.Bytes())
	want := []string{"IHDR", "pHYs", "IDAT", "IEND"}
	for i, kind := range want {
		if i >= len(kinds) || kinds[i] != kind {
			t.Fatalf("the chunks are %v, want them to start %v", kinds, want)
		}
	}
}

// TestJPEGCarriesTheJavasJFIFSegment pins the eighteen bytes.
//
// Java's writer puts a JFIF APP0 in every file and `JPEGUtil.updateMetadata`
// edits it: majorVersion 1, minorVersion 2, resUnits 1, Xdensity and Ydensity
// the dpi, thumbWidth and thumbHeight 0. Go's `image/jpeg` writes no APP0 at
// all, so the port writes this segment; these are the bytes the Java produced
// at 36 dpi.
func TestJPEGCarriesTheJavasJFIFSegment(t *testing.T) {
	var out bytes.Buffer
	if _, err := imageio.WriteImageOfDPI(sampleImage(), "jpg", &out, dpi); err != nil {
		t.Fatalf("WriteImage: %v", err)
	}
	const want = "ffd8ffe000104a46494600010201002400240000"
	got := hex.EncodeToString(out.Bytes()[:len(want)/2])
	if got != want {
		t.Errorf("the file starts %s, want %s: SOI, then a JFIF APP0 of 16 "+
			"bytes at version 1.02, one inch, 36 by 36, no thumbnail", got, want)
	}
}

// TestWBMPIsTheBytesTheJavaWrites pins the whole file.
//
// A WBMP is a type byte, a fixed-header byte, two variable-length lengths and
// the rows, one bit each with 1 for white. The Java wrote exactly this for the
// bitonal image.
func TestWBMPIsTheBytesTheJavaWrites(t *testing.T) {
	var out bytes.Buffer
	written, err := imageio.WriteImageOfDPI(bitonalImage(), "wbmp", &out, dpi)
	if err != nil {
		t.Fatalf("WriteImage: %v", err)
	}
	if !written {
		t.Fatal("WriteImage answered false")
	}
	const want = "00000806aa55aa55aa55"
	if got := hex.EncodeToString(out.Bytes()); got != want {
		t.Errorf("the WBMP is %s, want %s", got, want)
	}
}

// TestTIFFCarriesTheTagsTheJavaWrites is the directory of an RGB image.
//
// The Java's file has thirteen entries and these are twelve of their values;
// the thirteenth is the Compression, which this port writes as 1 where the
// Java writes 5 for LZW. That one is asserted in TestBitonalTiffStaysOneBit,
// with the reason.
func TestTIFFCarriesTheTagsTheJavaWrites(t *testing.T) {
	var out bytes.Buffer
	if _, err := imageio.WriteImageOfDPI(sampleImage(), "tiff", &out, dpi); err != nil {
		t.Fatalf("WriteImage: %v", err)
	}
	content := out.Bytes()

	for _, c := range []struct {
		tag  uint16
		name string
		want int
	}{
		{0x0100, "ImageWidth", 8},
		{0x0101, "ImageLength", 6},
		{0x0106, "PhotometricInterpretation", 2}, // RGB
		{0x0115, "SamplesPerPixel", 3},
		{0x0116, "RowsPerStrip", 6}, // the whole image, which is its height
		{0x0128, "ResolutionUnit", 2},
	} {
		if got := tiffTagValue(t, content, c.tag); got != c.want {
			t.Errorf("%s is %d, want %d", c.name, got, c.want)
		}
	}

	if got := bitsPerSampleOfTIFF(t, content); got != 8 {
		t.Errorf("BitsPerSample is %d, want 8", got)
	}
	if got := tiffTagCount(t, content, 0x0102); got != 3 {
		t.Errorf("BitsPerSample carries %d values, want 3, one per sample", got)
	}
	if x, y := resolutionOf(t, "tiff", content); x != dpi || y != dpi {
		t.Errorf("the resolution is %dx%d, want %dx%d", x, y, dpi, dpi)
	}
	if got := tiffASCII(t, content, 0x0131); got != "PDFBOX" {
		t.Errorf("Software is %q, want %q, which is what TIFFUtil writes",
			got, "PDFBOX")
	}
}

// TestBitonalTIFFIsWhiteIsZero is the one tag TIFFUtil sets only for a bitonal
// image: PhotometricInterpretation 0, "because of bug in Windows XP preview".
//
// It decides what the bits mean, so the pixels go with it: with WhiteIsZero a
// clear bit is white. The Java's file round-trips to the image it was given,
// and so does this one -- the check below is that the corner the sample image
// makes white is a zero bit.
func TestBitonalTIFFIsWhiteIsZero(t *testing.T) {
	var out bytes.Buffer
	if _, err := imageio.WriteImageOfDPI(bitonalImage(), "tiff", &out, dpi); err != nil {
		t.Fatalf("WriteImage: %v", err)
	}
	content := out.Bytes()

	if got := tiffTagValue(t, content, 0x0106); got != 0 {
		t.Errorf("PhotometricInterpretation is %d, want 0 (WhiteIsZero)", got)
	}
	if got := tiffTagValue(t, content, 0x0115); got != 1 {
		t.Errorf("SamplesPerPixel is %d, want 1", got)
	}
	if got := tiffASCII(t, content, 0x0131); got != "PDFBOX" {
		t.Errorf("Software is %q, want %q", got, "PDFBOX")
	}

	// bitonalImage makes (0,0) white and (1,0) black, so the first row of bits
	// is 01010101.
	entry, found := tiffEntries(t, content)[0x0111] // StripOffsets
	if !found {
		t.Fatal("the TIFF carries no StripOffsets")
	}
	at := int(tiffOrder.Uint32(entry[8:12]))
	if got := content[at]; got != 0x55 {
		t.Errorf("the first row of bits is %08b, want 01010101: with WhiteIsZero "+
			"the white pixel at the origin is a clear bit", got)
	}
}

// TestPNGQualityZeroCompressesHardest is what compressionQuality means.
//
// `ImageWriteParam.setCompressionQuality` documents 0 as "high compression is
// important" and 1 as "high image quality is important", and `ImageIOUtil`
// passes 0 for PNG "to prevent huge PNG files on jdk11 / jdk12 / jdk13"
// (PDFBOX-4655) -- so 0 is the *small* file. The JDK's PNG writer turns the
// fraction into a deflate level with
//
//	deflaterLevel = 9 - Math.round(9.0f * quality)
//
// read off `com.sun.imageio.plugins.png.PNGImageWriter`: quality 0 is level 9
// and quality 1 is level 0, which stores.
//
// Measured on the 600x400 image below, Java writes 559673 bytes at quality 0
// and 720846 at quality 1, against 720000 bytes of raw samples. So the numbers
// asserted here are the direction and the fact that quality 1 does not
// compress at all; the byte counts themselves cannot be, because Java's
// Deflater and Go's compress/zlib do not agree to the byte.
func TestPNGQualityZeroCompressesHardest(t *testing.T) {
	img := gradientImage()
	raw := img.Bounds().Dx() * img.Bounds().Dy() * 3

	var hard, soft bytes.Buffer
	if _, err := imageio.WriteImageOfQuality(img, "png", &hard, dpi, 0); err != nil {
		t.Fatalf("WriteImage: %v", err)
	}
	if _, err := imageio.WriteImageOfQuality(img, "png", &soft, dpi, 1); err != nil {
		t.Fatalf("WriteImage: %v", err)
	}

	if hard.Len() >= soft.Len() {
		t.Errorf("quality 0 gives %d bytes and quality 1 gives %d; 0 is the "+
			"quality that compresses, and PDFBOX-4655 passes it to keep PNGs small",
			hard.Len(), soft.Len())
	}
	if soft.Len() < raw {
		t.Errorf("quality 1 gives %d bytes for %d bytes of samples; it is "+
			"deflate level 0, which stores", soft.Len(), raw)
	}

	// And the overload ExtractImages and PDFToImage call is the small one.
	var byFormat bytes.Buffer
	if _, err := imageio.WriteImageOfDPI(img, "png", &byFormat, dpi); err != nil {
		t.Fatalf("WriteImage: %v", err)
	}
	if byFormat.Len() != hard.Len() {
		t.Errorf("writeImage(image, \"png\", out, dpi) gives %d bytes and quality "+
			"0 gives %d; the overload passes 0", byFormat.Len(), hard.Len())
	}
}

// gradientImage is something big enough for a compression level to show, and
// not so uniform that every level gives the same answer.
func gradientImage() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 600, 400))
	seed := uint32(42)
	next := func() uint32 {
		seed = seed*1664525 + 1013904223
		return seed >> 16
	}
	for y := 0; y < 400; y++ {
		for x := 0; x < 600; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8(uint32(x/3) + next()%8),
				G: uint8(uint32(y/2) + next()%8),
				B: uint8((x + y) / 4),
				A: 255,
			})
		}
	}
	return img
}

// TestCMYKTIFFKeepsItsFourSamples is the raster ExtractImages picks TIFF for.
//
// `-noColorConvert` writes `getRawImage()`, and where the raster has more than
// three data elements the suffix becomes "tiff" -- "More than 3 channels:
// That's likely CMYK. We use tiff here". Java hands that four-band image
// straight to the TIFF writer, which keeps the four samples.
//
// Measured, by building a four-component ComponentColorModel over a four-band
// raster the way `PDColorSpace.toRawImage` does and writing it through
// `ImageIOUtil`: 13 entries, BitsPerSample 8,8,8,8, SamplesPerPixel 4, and
// PhotometricInterpretation 5, which is Separated.
func TestCMYKTIFFKeepsItsFourSamples(t *testing.T) {
	img := image.NewCMYK(image.Rect(0, 0, 8, 6))
	for y := 0; y < 6; y++ {
		for x := 0; x < 8; x++ {
			img.SetCMYK(x, y, color.CMYK{
				C: uint8(x * 30), M: uint8(y * 40), Y: 128, K: 20,
			})
		}
	}

	var out bytes.Buffer
	if _, err := imageio.WriteImageOfDPI(img, "tiff", &out, dpi); err != nil {
		t.Fatalf("WriteImage: %v", err)
	}
	content := out.Bytes()

	if got := tiffTagValue(t, content, 0x0115); got != 4 {
		t.Errorf("SamplesPerPixel is %d, want 4", got)
	}
	if got := tiffTagCount(t, content, 0x0102); got != 4 {
		t.Errorf("BitsPerSample carries %d values, want 4", got)
	}
	if got := bitsPerSampleOfTIFF(t, content); got != 8 {
		t.Errorf("BitsPerSample is %d, want 8", got)
	}
	if got := tiffTagValue(t, content, 0x0106); got != 5 {
		t.Errorf("PhotometricInterpretation is %d, want 5 (Separated)", got)
	}

	// The samples are the ones that went in, not three converted ones.
	entry, found := tiffEntries(t, content)[0x0111] // StripOffsets
	if !found {
		t.Fatal("the TIFF carries no StripOffsets")
	}
	at := int(tiffOrder.Uint32(entry[8:12]))
	want := []byte{0, 0, 128, 20, 30, 0, 128, 20}
	if got := content[at : at+len(want)]; !bytes.Equal(got, want) {
		t.Errorf("the first two pixels are % d, want % d", got, want)
	}
}

// TestBitonalSubImageIsStillBitonal is the padding an image.Gray can carry.
//
// A `SubImage` shares its parent's `Pix` and `Stride`, and the slice runs to
// the end of the parent's buffer -- so reading `Pix` straight through sees
// pixels that are not in the image. Where those are neither black nor white,
// a bitonal sub-image would be written at eight bits per pixel.
func TestBitonalSubImageIsStillBitonal(t *testing.T) {
	parent := image.NewGray(image.Rect(0, 0, 16, 6))
	for y := 0; y < 6; y++ {
		for x := 0; x < 16; x++ {
			switch {
			case x >= 8:
				parent.SetGray(x, y, color.Gray{Y: 0x7F}) // outside, and grey
			case (x+y)%2 == 0:
				parent.SetGray(x, y, color.Gray{Y: 255})
			}
		}
	}
	img := parent.SubImage(image.Rect(0, 0, 8, 6))

	var out bytes.Buffer
	if _, err := imageio.WriteImageOfDPI(img, "tiff", &out, dpi); err != nil {
		t.Fatalf("WriteImage: %v", err)
	}
	content := out.Bytes()
	if got := bitsPerSampleOfTIFF(t, content); got != 1 {
		t.Errorf("BitsPerSample is %d, want 1: every pixel inside the image is "+
			"black or white, and the grey ones are outside it", got)
	}
	if got := tiffTagValue(t, content, 0x0106); got != 0 {
		t.Errorf("PhotometricInterpretation is %d, want 0 (WhiteIsZero)", got)
	}
}

// TestCMYKTIFFReadsBack is checkImageFileSizeAndContent for the four-sample
// file: the strip is a TIFF a reader can take, and the colours come back.
func TestCMYKTIFFReadsBack(t *testing.T) {
	img := image.NewCMYK(image.Rect(0, 0, 8, 6))
	for y := 0; y < 6; y++ {
		for x := 0; x < 8; x++ {
			img.SetCMYK(x, y, color.CMYK{
				C: uint8(x * 30), M: uint8(y * 40), Y: 128, K: 20,
			})
		}
	}

	var out bytes.Buffer
	if _, err := imageio.WriteImageOfDPI(img, "tiff", &out, dpi); err != nil {
		t.Fatalf("WriteImage: %v", err)
	}
	decoded, err := decodeAny(t, "tiff", out.Bytes())
	if err != nil {
		t.Fatalf("the file this port wrote did not read back: %v", err)
	}
	for y := 0; y < 6; y++ {
		for x := 0; x < 8; x++ {
			// Eight bits a channel: the file holds eight and the reader puts
			// them back into an image.RGBA, so the low byte of the sixteen the
			// colour models answer with is not carried either way.
			wantR, wantG, wantB := rgb8Of(img, x, y)
			gotR, gotG, gotB := rgb8Of(decoded, x, y)
			if wantR != gotR || wantG != gotG || wantB != gotB {
				t.Fatalf("the pixel at %d,%d read back as %v, want %v",
					x, y, decoded.At(x, y), img.At(x, y))
			}
		}
	}
}

// rgb8Of answers a pixel as three eight-bit channels.
func rgb8Of(img image.Image, x, y int) (uint8, uint8, uint8) {
	r, g, b, _ := img.At(x, y).RGBA()
	return uint8(r >> 8), uint8(g >> 8), uint8(b >> 8)
}
