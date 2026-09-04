package ttf

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// Ported from
// fontbox/src/test/java/org/apache/fontbox/ttf/RandomAccessReadBufferDataStreamTest.java.

// ttfFixture is the directory the Java test fonts live in, relative to this
// package.
const ttfFixture = "../../../fontbox/src/test/resources/ttf/"

func newTestStream(t *testing.T, data []byte) *RandomAccessReadDataStream {
	t.Helper()
	stream, err := NewRandomAccessReadDataStream(pdfio.NewReadBufferBytes(data))
	if err != nil {
		t.Fatalf("NewRandomAccessReadDataStream: %v", err)
	}
	t.Cleanup(func() { stream.Close() })
	return stream
}

// TestEOF pins that reading past the end reports -1 rather than failing, for
// as long as a caller keeps asking.
func TestEOF(t *testing.T) {
	stream := newTestStream(t, make([]byte, 10))

	value, err := stream.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	for value > -1 {
		if value, err = stream.Read(); err != nil {
			t.Fatalf("Read: %v", err)
		}
	}
	// and it keeps reporting -1 rather than failing
	for i := 0; i < 3; i++ {
		if value, err = stream.Read(); err != nil || value != -1 {
			t.Fatalf("Read past the end = %d, %v, want -1, nil", value, err)
		}
	}
}

func TestEOFUnsignedShort(t *testing.T) {
	stream := newTestStream(t, make([]byte, 3))

	if _, err := stream.ReadUnsignedShort(); err != nil {
		t.Fatalf("ReadUnsignedShort: %v", err)
	}
	if _, err := stream.ReadUnsignedShort(); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("ReadUnsignedShort past the end = %v, want an unexpected EOF", err)
	}
}

func TestEOFUnsignedInt(t *testing.T) {
	stream := newTestStream(t, make([]byte, 5))

	if _, err := stream.ReadUnsignedInt(); err != nil {
		t.Fatalf("ReadUnsignedInt: %v", err)
	}
	if _, err := stream.ReadUnsignedInt(); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("ReadUnsignedInt past the end = %v, want an unexpected EOF", err)
	}
}

func TestEOFUnsignedByte(t *testing.T) {
	stream := newTestStream(t, make([]byte, 2))

	for i := 0; i < 2; i++ {
		if _, err := stream.ReadUnsignedByte(); err != nil {
			t.Fatalf("ReadUnsignedByte: %v", err)
		}
	}
	if _, err := stream.ReadUnsignedByte(); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("ReadUnsignedByte past the end = %v, want an unexpected EOF", err)
	}
}

// TestDoubleClose covers PDFBOX-4242: closing twice must be allowed.
func TestDoubleClose(t *testing.T) {
	f, err := os.Open(ttfFixture + "LiberationSans-Regular.ttf")
	if err != nil {
		t.Fatalf("open the test font: %v", err)
	}
	defer f.Close()
	source, err := pdfio.NewReadBufferFromReader(f)
	if err != nil {
		t.Fatalf("NewReadBufferFromReader: %v", err)
	}

	stream, err := NewRandomAccessReadDataStream(source)
	if err != nil {
		t.Fatalf("NewRandomAccessReadDataStream: %v", err)
	}
	if err := stream.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := stream.Close(); err != nil {
		t.Errorf("second Close: %v, want nil", err)
	}
}

// TestEnsureReadFinishes covers PDFBOX-3605: before it was solved this never
// ended.
func TestEnsureReadFinishes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "apache-pdfbox.dat")
	if err := os.WriteFile(path, []byte("1234567890"), 0o600); err != nil {
		t.Fatalf("write the fixture: %v", err)
	}
	stream := newTestStreamFromFile(t, path)

	readBuffer := make([]byte, 2)
	totalAmountRead := 0
	for {
		amountRead, err := stream.ReadInto(readBuffer, 0, 2)
		if err != nil {
			t.Fatalf("ReadInto: %v", err)
		}
		if amountRead == -1 {
			break
		}
		totalAmountRead += amountRead
	}
	if totalAmountRead != 10 {
		t.Errorf("read %d bytes, want 10", totalAmountRead)
	}
}

func newTestStreamFromFile(t *testing.T, path string) *RandomAccessReadDataStream {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return newTestStream(t, data)
}

// TestReadBuffer walks several reading patterns, both within a buffer and
// across one.
func TestReadBuffer(t *testing.T) {
	path := filepath.Join(t.TempDir(), "apache-pdfbox.dat")
	const content = "012345678A012345678B012345678C012345678D"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write the fixture: %v", err)
	}
	stream := newTestStreamFromFile(t, path)
	readBuffer := make([]byte, 40)

	read := func(count int) (int, string) {
		t.Helper()
		bytesRead, err := stream.ReadInto(readBuffer, 0, count)
		if err != nil {
			t.Fatalf("ReadInto: %v", err)
		}
		if bytesRead <= 0 {
			return bytesRead, ""
		}
		return bytesRead, string(readBuffer[:bytesRead])
	}
	expect := func(count int, wantRead int, wantText string, wantPosition int64) {
		t.Helper()
		bytesRead, text := read(count)
		if bytesRead != wantRead {
			t.Errorf("read %d bytes, want %d", bytesRead, wantRead)
		}
		if text != wantText {
			t.Errorf("read %q, want %q", text, wantText)
		}
		if got := stream.CurrentPosition(); got != wantPosition {
			t.Errorf("position = %d, want %d", got, wantPosition)
		}
	}

	expect(4, 4, "0123", 4)
	expect(6, 6, "45678A", 10)
	expect(10, 10, "012345678B", 20)
	expect(10, 10, "012345678C", 30)
	expect(10, 10, "012345678D", 40)

	if value, _ := stream.Read(); value != -1 {
		t.Errorf("Read at the end = %d, want -1", value)
	}

	mustSeek(t, stream, 0)
	read(7)
	if got := stream.CurrentPosition(); got != 7 {
		t.Errorf("position = %d, want 7", got)
	}
	expect(16, 16, "78A012345678B012", 23)
	// A read longer than what is left stops at the end.
	expect(99, 17, "345678C012345678D", 40)

	if value, _ := stream.Read(); value != -1 {
		t.Errorf("Read at the end = %d, want -1", value)
	}

	mustSeek(t, stream, 0)
	read(7)
	expect(23, 23, "78A012345678B012345678C", 30)

	mustSeek(t, stream, 0)
	read(10)
	if got := stream.CurrentPosition(); got != 10 {
		t.Errorf("position = %d, want 10", got)
	}
	expect(23, 23, "012345678B012345678C012", 33)
}

func mustSeek(t *testing.T, stream *RandomAccessReadDataStream, pos int64) {
	t.Helper()
	if err := stream.SeekTo(pos); err != nil {
		t.Fatalf("SeekTo(%d): %v", pos, err)
	}
}

// TestSeekNegative pins that a negative position is refused.
func TestSeekNegative(t *testing.T) {
	stream := newTestStream(t, make([]byte, 4))
	if err := stream.SeekTo(-1); err == nil {
		t.Error("SeekTo(-1) succeeded, want an error")
	}
}

// TestSeekPastTheEnd pins that seeking beyond the data lands at the end rather
// than failing.
func TestSeekPastTheEnd(t *testing.T) {
	stream := newTestStream(t, make([]byte, 4))
	mustSeek(t, stream, 99)
	if got := stream.CurrentPosition(); got != 4 {
		t.Errorf("position = %d, want 4 — the end of the data", got)
	}
}

// TestReadFixedAndTag covers the two composite reads the table directory needs.
func TestReadFixedAndTag(t *testing.T) {
	// 0x0001 0000 is version 1.0 as a 16.16 fixed point number.
	stream := newTestStream(t, []byte{0x00, 0x01, 0x00, 0x00, 'c', 'm', 'a', 'p'})

	fixed, err := stream.Read32Fixed()
	if err != nil {
		t.Fatalf("Read32Fixed: %v", err)
	}
	if fixed != 1.0 {
		t.Errorf("Read32Fixed = %v, want 1.0", fixed)
	}

	tag, err := stream.ReadTag()
	if err != nil {
		t.Fatalf("ReadTag: %v", err)
	}
	if tag != "cmap" {
		t.Errorf("ReadTag = %q, want %q", tag, "cmap")
	}
}

// TestReadSignedValues pins the sign handling, which is where a Go port of Java
// byte arithmetic goes wrong most easily.
func TestReadSignedValues(t *testing.T) {
	stream := newTestStream(t, []byte{0xFF, 0x7F, 0xFF, 0xFE})

	if got, err := stream.ReadSignedByte(); err != nil || got != -1 {
		t.Errorf("ReadSignedByte = %d, %v, want -1", got, err)
	}
	if got, err := stream.ReadSignedByte(); err != nil || got != 127 {
		t.Errorf("ReadSignedByte = %d, %v, want 127", got, err)
	}
	if got, err := stream.ReadSignedShort(); err != nil || got != -2 {
		t.Errorf("ReadSignedShort = %d, %v, want -2", got, err)
	}
}

// TestReadUnexpectedEndOfStream pins that asking for a fixed number of bytes
// and not getting them is an error, rather than a short result.
func TestReadUnexpectedEndOfStream(t *testing.T) {
	stream := newTestStream(t, []byte{1, 2, 3})
	if _, err := stream.ReadBytes(4); err == nil {
		t.Error("ReadBytes(4) over three bytes succeeded, want an error")
	}
}
