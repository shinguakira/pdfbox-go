package filetypedetector

import (
	"bufio"
	"errors"
)

// root holds every signature the detector knows.
//
// Port of the static initialiser of
// org.apache.pdfbox.util.filetypedetector.FileTypeDetector. Java reads its
// literals as ISO-8859-1 bytes, which for these ASCII strings is the same as
// the bytes of the Go string.
var root = func() *byteTrie {
	t := newByteTrie()
	t.setDefaultValue(Unknown)

	// https://en.wikipedia.org/wiki/List_of_file_signatures

	iiBytes := []byte("II")
	mmBytes := []byte("MM")
	t.addPath(JPEG, []byte{0xff, 0xd8})
	t.addPath(TIFF, iiBytes, []byte{0x2a, 0x00})
	t.addPath(TIFF, mmBytes, []byte{0x00, 0x2a})
	t.addPath(PSD, []byte("8BPS"))
	t.addPath(PNG, []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52})
	// TODO technically there are other very rare magic numbers for OS/2 BMP files...
	t.addPath(BMP, []byte("BM"))
	t.addPath(GIF, []byte("GIF87a"))
	t.addPath(GIF, []byte("GIF89a"))
	t.addPath(ICO, []byte{0x00, 0x00, 0x01, 0x00})
	// multiple PCX versions, explicitly listed
	t.addPath(PCX, []byte{0x0A, 0x00, 0x01})
	t.addPath(PCX, []byte{0x0A, 0x02, 0x01})
	t.addPath(PCX, []byte{0x0A, 0x03, 0x01})
	t.addPath(PCX, []byte{0x0A, 0x05, 0x01})
	t.addPath(RIFF, []byte("RIFF"))

	// https://github.com/drewnoakes/metadata-extractor/issues/217
	// t.addPath(ARW, []byte("II"), []byte{0x2a, 0x00, 0x08, 0x00})
	t.addPath(CRW, iiBytes, []byte{0x1a, 0x00, 0x00, 0x00}, []byte("HEAPCCDR"))
	t.addPath(CR2, iiBytes, []byte{0x2a, 0x00, 0x10, 0x00, 0x00, 0x00, 0x43, 0x52})
	t.addPath(NEF, mmBytes, []byte{0x00, 0x2a, 0x00, 0x00, 0x00, 0x80, 0x00})
	t.addPath(ORF, []byte("IIRO"), []byte{0x08, 0x00})
	t.addPath(ORF, []byte("IIRS"), []byte{0x08, 0x00})
	t.addPath(RAF, []byte("FUJIFILMCCD-RAW"))
	t.addPath(RW2, iiBytes, []byte{0x55, 0x00})
	return t
}()

// ErrStreamEndedBeforeMagicNumber is returned where a stream is empty.
//
// Port of the IOException FileTypeDetector throws for the same.
var ErrStreamEndedBeforeMagicNumber = errors.New(
	"Stream ended before file's magic number could be determined.")

// DetectFileType reads the leading bytes of a stream and names its format,
// leaving the stream where it found it.
//
// Port of detectFileType(BufferedInputStream). Java marks the stream, reads and
// resets, and refuses a stream that does not support mark; a bufio.Reader peeks
// instead, so there is no mark to support and no such refusal.
//
// Java searches the whole array it allocated, not the part it filled, so a file
// shorter than the longest signature is searched with trailing zeroes after it
// -- and a three byte file 00 00 01 is detected as an ICO, whose signature is
// 00 00 01 00. The port pads to the same length so that it reads the same
// files the same way.
func DetectFileType(inputStream *bufio.Reader) (FileType, error) {
	maxByteCount := root.getMaxDepth()

	peeked, _ := inputStream.Peek(maxByteCount)
	if len(peeked) == 0 {
		return Unknown, ErrStreamEndedBeforeMagicNumber
	}
	bytes := make([]byte, maxByteCount)
	copy(bytes, peeked)

	return root.find(bytes), nil
}

// DetectFileTypeOfBytes names the format of the bytes.
//
// Port of detectFileType(byte[]).
func DetectFileTypeOfBytes(fileBytes []byte) FileType {
	return root.find(fileBytes)
}
