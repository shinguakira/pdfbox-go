package tools_test

// The `export:images` command, which Java has no test for.
//
// `track/tools` wrote its eighteen commands from the source for the same
// reason: `tools` has four test classes and none of them covers this one. So
// the assertions are about what the Java's `write2file` decides -- which suffix
// a given image gets, that an image drawn twice is written once, and that the
// direct path copies the stream out untouched.
//
// The values are the Java's. `input/merge/jpegrgb.pdf` and
// `input/merge/multitiff.pdf` are checked in, and what PDFBox's own
// `PDFGraphicsStreamEngine` finds in them was read by running it: the first
// draws one 344x287 DeviceRGB JPEG twice, whose DCTDecode stream is 40336
// bytes, and the second draws three 344x287 one-bit DeviceGray images that
// `getSuffix` answers "tiff" for.
//
// Each PDF is copied into a temporary directory before the command runs, so
// that the files it writes land there and never beside the Java it read.

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/util/filetypedetector"
	"github.com/shinguakira/pdfbox-go/go/tools"
)

// The two documents, and what the Java finds in them.
const (
	jpegRGBPDF   = "../../pdfbox/src/test/resources/input/merge/jpegrgb.pdf"
	multiTIFFPDF = "../../pdfbox/src/test/resources/input/merge/multitiff.pdf"

	// jpegRGBStreamLength and jpegRGBStreamDigest are what
	// `pdImage.createInputStream(JPEG)` answers for the one image in
	// jpegrgb.pdf, which is what the direct path copies to the file.
	jpegRGBStreamLength = 40336
	jpegRGBStreamDigest = "62d1a60080d90150bc32ca86bbe4eb15f9eb2ea03096f571e9afa7802feeaad5"
)

// copiedResource copies a checked-in PDF into a temporary directory and answers
// its path there.
//
// The command writes its files beside the input when it is given no prefix, and
// the input is in the Java tree.
func copiedResource(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	copied := filepath.Join(t.TempDir(), filepath.Base(path))
	if err := os.WriteFile(copied, content, 0o644); err != nil {
		t.Fatalf("copying %s: %v", path, err)
	}
	return copied
}

// extractedFiles answers the files the command wrote beside the input, sorted.
func extractedFiles(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}
	var written []string
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".pdf") {
			written = append(written, entry.Name())
		}
	}
	sort.Strings(written)
	return written
}

// TestExtractImagesCopiesADeviceRGBJPEG is the direct path: a JPEG in
// DeviceRGB is copied out rather than decoded and encoded again, so the file is
// the stream that was in the PDF, byte for byte.
//
// It is also the default prefix, which Java takes from the input file with its
// extension removed, and the `seen` set: jpegrgb.pdf draws the one XObject
// twice and one file comes out.
func TestExtractImagesCopiesADeviceRGBJPEG(t *testing.T) {
	input := copiedResource(t, jpegRGBPDF)
	dir := filepath.Dir(input)

	code, stdout, stderr := runCommand(tools.NewExtractImages(), "-i", input)
	if code != 0 {
		t.Fatalf("exited %d; stderr is %q", code, stderr)
	}

	written := extractedFiles(t, dir)
	if len(written) != 1 || written[0] != "jpegrgb-1.jpg" {
		t.Fatalf("the command wrote %v, want [jpegrgb-1.jpg]: the image is drawn "+
			"twice and written once, and the name comes from the input", written)
	}
	if !strings.Contains(stdout, "Writing image: ") {
		t.Errorf("stdout is %q, want it to name the file it wrote", stdout)
	}

	content, err := os.ReadFile(filepath.Join(dir, written[0]))
	if err != nil {
		t.Fatal(err)
	}
	if got := filetypedetector.DetectFileTypeOfBytes(content); got != filetypedetector.JPEG {
		t.Errorf("the file is %v, want JPEG", got)
	}
	if len(content) != jpegRGBStreamLength {
		t.Fatalf("the file is %d bytes and the DCTDecode stream in the PDF is %d; "+
			"a DeviceRGB JPEG is copied, not re-encoded",
			len(content), jpegRGBStreamLength)
	}
	if got := hex.EncodeToString(sha256Of(content)); got != jpegRGBStreamDigest {
		t.Errorf("the file digests to %s, want %s", got, jpegRGBStreamDigest)
	}
}

// TestExtractImagesTakesThePrefix is the -prefix option, which replaces the
// name taken from the input and can put the files somewhere else.
func TestExtractImagesTakesThePrefix(t *testing.T) {
	input := copiedResource(t, jpegRGBPDF)
	into := t.TempDir()

	code, _, stderr := runCommand(tools.NewExtractImages(),
		"-i", input, "-prefix", filepath.Join(into, "picture"))
	if code != 0 {
		t.Fatalf("exited %d; stderr is %q", code, stderr)
	}
	if written := extractedFiles(t, into); len(written) != 1 ||
		written[0] != "picture-1.jpg" {
		t.Errorf("the command wrote %v, want [picture-1.jpg]", written)
	}
	if written := extractedFiles(t, filepath.Dir(input)); len(written) != 0 {
		t.Errorf("the command wrote %v beside the input as well", written)
	}
}

// TestExtractImagesWritesBitonalTIFFs is the `tiff` arm of writeImageBody: a
// CCITT-compressed image is DeviceGray, so it is converted to one bit per pixel
// and written as a TIFF.
//
// multitiff.pdf draws three of them, each 344x287 and one bit deep, and each is
// a separate XObject so all three are written.
func TestExtractImagesWritesBitonalTIFFs(t *testing.T) {
	input := copiedResource(t, multiTIFFPDF)
	dir := filepath.Dir(input)

	code, _, stderr := runCommand(tools.NewExtractImages(), "-i", input)
	if code != 0 {
		t.Fatalf("exited %d; stderr is %q", code, stderr)
	}

	written := extractedFiles(t, dir)
	want := []string{"multitiff-1.tiff", "multitiff-2.tiff", "multitiff-3.tiff"}
	if len(written) != len(want) {
		t.Fatalf("the command wrote %v, want %v", written, want)
	}
	for i, name := range want {
		if written[i] != name {
			t.Fatalf("the command wrote %v, want %v", written, want)
		}
	}

	for _, name := range written {
		content, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if got := filetypedetector.DetectFileTypeOfBytes(content); got != filetypedetector.TIFF {
			t.Errorf("%s is %v, want TIFF", name, got)
		}
		width, height, bits := tiffShape(t, content)
		if width != 344 || height != 287 {
			t.Errorf("%s is %dx%d, want 344x287", name, width, height)
		}
		if bits != 1 {
			t.Errorf("%s is %d bits per sample; a CCITT image is bitonal", name, bits)
		}
	}
}

// TestExtractImagesReportsAMissingFile is the catch, which names the exception
// class and answers 4.
func TestExtractImagesReportsAMissingFile(t *testing.T) {
	code, _, stderr := runCommand(tools.NewExtractImages(), "-i", "nosuch.pdf")
	if code != 4 {
		t.Errorf("exited %d, want 4", code)
	}
	if !strings.Contains(stderr, "Error extracting images") {
		t.Errorf("stderr is %q, want it to say what failed", stderr)
	}
}

// TestExtractImagesNeedsAnInput is the required option.
func TestExtractImagesNeedsAnInput(t *testing.T) {
	code, _, stderr := runCommand(tools.NewExtractImages())
	if code != 2 {
		t.Errorf("exited %d, want 2", code)
	}
	if !strings.Contains(stderr, "input") {
		t.Errorf("stderr is %q, want it to name the option", stderr)
	}
}

// sha256Of is the digest the driver that read these values from the Java
// printed.
func sha256Of(content []byte) []byte {
	sum := sha256.Sum256(content)
	return sum[:]
}

// tiffShape answers a TIFF's width, height and bits per sample, read out of the
// first directory.
func tiffShape(t *testing.T, content []byte) (int, int, int) {
	t.Helper()
	if len(content) < 8 {
		t.Fatal("the TIFF is too short to hold a header")
	}
	var order binary.ByteOrder
	switch string(content[0:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		t.Fatalf("the TIFF byte order mark is %q", content[0:2])
	}
	offset := int(order.Uint32(content[4:8]))
	if offset+2 > len(content) {
		t.Fatal("the TIFF directory is past the end of the file")
	}
	values := map[uint16]int{}
	count := int(order.Uint16(content[offset : offset+2]))
	for i := 0; i < count; i++ {
		at := offset + 2 + i*12
		if at+12 > len(content) {
			t.Fatal("a TIFF directory entry is past the end of the file")
		}
		tag := order.Uint16(content[at : at+2])
		switch order.Uint16(content[at+2 : at+4]) {
		case 3: // SHORT
			values[tag] = int(order.Uint16(content[at+8 : at+10]))
		case 4: // LONG
			values[tag] = int(order.Uint32(content[at+8 : at+12]))
		}
	}
	return values[0x0100], values[0x0101], values[0x0102]
}
