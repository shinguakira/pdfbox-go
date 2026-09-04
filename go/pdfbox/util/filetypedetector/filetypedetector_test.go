package filetypedetector

import (
	"bufio"
	"bytes"
	"errors"
	"os"
	"testing"
)

// org.apache.pdfbox.util.filetypedetector has no Java tests, so these are
// written from the source, which slice 6's A4 asks to be named first.
//
// What they cover: every signature FileTypeDetector registers, the trie's
// longest-match rule, the default value it falls back to, and the empty stream.
// The image files come from the Java test resources rather than from bytes
// written here, so that the signatures are checked against real files.

// imageFixture is where the image test resources of pdmodel/graphics/image
// live, relative to this package.
const imageFixture = "../../../../pdfbox/src/test/resources/org/apache/pdfbox/pdmodel/graphics/image/"

// TestDetectRealFiles names the format of files the Java test resources carry.
func TestDetectRealFiles(t *testing.T) {
	cases := []struct {
		file string
		want FileType
	}{
		{"jpeg.jpg", JPEG},
		{"jpegcmyk.jpg", JPEG},
		{"png.png", PNG},
		{"png_alpha_rgb.png", PNG},
		{"gif.gif", GIF},
		{"ccittg4.tif", TIFF},
		{"lzw.tif", TIFF},
	}
	for _, c := range cases {
		t.Run(c.file, func(t *testing.T) {
			content, err := os.ReadFile(imageFixture + c.file)
			if err != nil {
				t.Fatalf("reading %s: %v", c.file, err)
			}
			if got := DetectFileTypeOfBytes(content); got != c.want {
				t.Errorf("DetectFileTypeOfBytes(%s) = %v, want %v", c.file, got, c.want)
			}
			got, err := DetectFileType(bufio.NewReader(bytes.NewReader(content)))
			if err != nil {
				t.Fatalf("DetectFileType(%s): %v", c.file, err)
			}
			if got != c.want {
				t.Errorf("DetectFileType(%s) = %v, want %v", c.file, got, c.want)
			}
		})
	}
}

// TestDetectSignatures runs every signature the static initialiser registers,
// each as the bytes the Java lists.
func TestDetectSignatures(t *testing.T) {
	cases := []struct {
		name  string
		magic []byte
		want  FileType
	}{
		{"jpeg", []byte{0xff, 0xd8}, JPEG},
		{"tiff little endian", []byte("II\x2a\x00"), TIFF},
		{"tiff big endian", []byte("MM\x00\x2a"), TIFF},
		{"psd", []byte("8BPS"), PSD},
		{"png", []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
			0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52}, PNG},
		{"bmp", []byte("BM"), BMP},
		{"gif87a", []byte("GIF87a"), GIF},
		{"gif89a", []byte("GIF89a"), GIF},
		{"ico", []byte{0x00, 0x00, 0x01, 0x00}, ICO},
		{"pcx 0", []byte{0x0A, 0x00, 0x01}, PCX},
		{"pcx 2", []byte{0x0A, 0x02, 0x01}, PCX},
		{"pcx 3", []byte{0x0A, 0x03, 0x01}, PCX},
		{"pcx 5", []byte{0x0A, 0x05, 0x01}, PCX},
		{"riff", []byte("RIFF"), RIFF},
		{"crw", append([]byte("II"), append([]byte{0x1a, 0x00, 0x00, 0x00},
			[]byte("HEAPCCDR")...)...), CRW},
		{"cr2", []byte("II\x2a\x00\x10\x00\x00\x00CR"), CR2},
		{"nef", []byte{'M', 'M', 0x00, 0x2a, 0x00, 0x00, 0x00, 0x80, 0x00}, NEF},
		{"orf ro", []byte("IIRO\x08\x00"), ORF},
		{"orf rs", []byte("IIRS\x08\x00"), ORF},
		{"raf", []byte("FUJIFILMCCD-RAW"), RAF},
		{"rw2", []byte("II\x55\x00"), RW2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// with data after the signature, which is what a real file has
			data := append(append([]byte{}, c.magic...), 0x11, 0x22, 0x33)
			if got := DetectFileTypeOfBytes(data); got != c.want {
				t.Errorf("DetectFileTypeOfBytes = %v, want %v", got, c.want)
			}
		})
	}
}

// TestDetectLongestMatchWins pins the trie's rule: CR2 and RW2 both start with
// the TIFF signature, and the deeper match is the answer.
func TestDetectLongestMatchWins(t *testing.T) {
	if got := DetectFileTypeOfBytes([]byte("II\x2a\x00")); got != TIFF {
		t.Errorf("the bare TIFF signature gave %v", got)
	}
	if got := DetectFileTypeOfBytes([]byte("II\x2a\x00\x10\x00\x00\x00CR")); got != CR2 {
		t.Errorf("a CR2 gave %v, want the deeper match", got)
	}
	if got := DetectFileTypeOfBytes([]byte("II\x55\x00")); got != RW2 {
		t.Errorf("an RW2 gave %v", got)
	}
}

// TestDetectUnknown checks the default value the root carries.
func TestDetectUnknown(t *testing.T) {
	for _, data := range [][]byte{
		nil,
		{},
		[]byte("not an image at all"),
		{0xff},             // half a JPEG signature
		{0x0A, 0x01, 0x01}, // a PCX version that is not listed
	} {
		if got := DetectFileTypeOfBytes(data); got != Unknown {
			t.Errorf("DetectFileTypeOfBytes(%v) = %v, want UNKNOWN", data, got)
		}
	}
}

// TestDetectEmptyStream checks the one error the detector reports.
func TestDetectEmptyStream(t *testing.T) {
	_, err := DetectFileType(bufio.NewReader(bytes.NewReader(nil)))
	if !errors.Is(err, ErrStreamEndedBeforeMagicNumber) {
		t.Errorf("an empty stream gave %v", err)
	}
}

// TestFileTypeString checks that the constants print the names Java's enum
// prints, since PDFBox puts them in exception messages.
func TestFileTypeString(t *testing.T) {
	cases := map[FileType]string{
		Unknown: "UNKNOWN", JPEG: "JPEG", TIFF: "TIFF", PSD: "PSD", PNG: "PNG",
		BMP: "BMP", GIF: "GIF", ICO: "ICO", PCX: "PCX", RIFF: "RIFF",
		ARW: "ARW", CRW: "CRW", CR2: "CR2", NEF: "NEF", ORF: "ORF",
		RAF: "RAF", RW2: "RW2",
	}
	for value, want := range cases {
		if got := value.String(); got != want {
			t.Errorf("FileType(%d).String() = %q, want %q", int(value), got, want)
		}
	}
}

// TestByteTrieRejectsADuplicateValue pins the one unchecked exception the trie
// raises: Java's ByteTrieNode.setValue throws IllegalStateException where a
// node already has a value, so the port panics.
func TestByteTrieRejectsADuplicateValue(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("setting a trie value twice should panic")
		}
	}()
	trie := newByteTrie()
	trie.addPath(PNG, []byte("ab"))
	trie.addPath(GIF, []byte("ab"))
}
