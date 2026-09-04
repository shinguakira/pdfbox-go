package cff

import (
	"bytes"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// Port of org.apache.fontbox.cff.DataInputTest and
// org.apache.fontbox.cff.DataInputRandomAccessTest.
//
// The two Java classes make the same assertions over the two implementations;
// each test below runs both, so both real paths are taken.
var dataInputs = []struct {
	name string
	open func(data []byte) DataInput
}{
	{"ByteArray", func(data []byte) DataInput { return NewDataInputByteArray(data) }},
	{"RandomAccessRead", func(data []byte) DataInput {
		return NewDataInputRandomAccessRead(pdfio.NewReadBufferBytes(data))
	}},
}

func TestDataInputReadBytes(t *testing.T) {
	for _, impl := range dataInputs {
		t.Run(impl.name, func(t *testing.T) {
			data := []byte{0, 0xFF, 2, 0xFD, 4, 0xFB, 6, 0xF9, 8, 0xF7}
			dataInput := impl.open(data)
			if _, err := dataInput.ReadBytes(20); err == nil {
				t.Error("ReadBytes(20) was accepted")
			}
			if got, err := dataInput.ReadBytes(1); err != nil ||
				!bytes.Equal(got, []byte{0}) {
				t.Errorf("ReadBytes(1) = %v, %v; want [0], nil", got, err)
			}
			if got, err := dataInput.ReadBytes(3); err != nil ||
				!bytes.Equal(got, []byte{0xFF, 2, 0xFD}) {
				t.Errorf("ReadBytes(3) = %v, %v; want [255 2 253], nil", got, err)
			}
			if err := dataInput.SetPosition(6); err != nil {
				t.Fatalf("SetPosition(6): %v", err)
			}
			if got, err := dataInput.ReadBytes(3); err != nil ||
				!bytes.Equal(got, []byte{6, 0xF9, 8}) {
				t.Errorf("ReadBytes(3) = %v, %v; want [6 249 8], nil", got, err)
			}
			if _, err := dataInput.ReadBytes(-1); err == nil {
				t.Error("ReadBytes(-1) was accepted")
			}
			if _, err := dataInput.ReadBytes(5); err == nil {
				t.Error("ReadBytes(5) past the end was accepted")
			}
		})
	}
}

func TestDataInputReadByte(t *testing.T) {
	for _, impl := range dataInputs {
		t.Run(impl.name, func(t *testing.T) {
			data := []byte{0, 0xFF, 2, 0xFD, 4, 0xFB, 6, 0xF9, 8, 0xF7}
			dataInput := impl.open(data)
			assertByte(t, dataInput, 0)
			assertByte(t, dataInput, -1)
			if err := dataInput.SetPosition(6); err != nil {
				t.Fatalf("SetPosition(6): %v", err)
			}
			assertByte(t, dataInput, 6)
			assertByte(t, dataInput, -7)
			length, err := dataInput.Length()
			if err != nil {
				t.Fatalf("Length: %v", err)
			}
			if err := dataInput.SetPosition(length - 1); err != nil {
				t.Fatalf("SetPosition(%d): %v", length-1, err)
			}
			assertByte(t, dataInput, -9)
			if _, err := dataInput.ReadSignedByte(); err == nil {
				t.Error("ReadSignedByte past the end was accepted")
			}
		})
	}
}

func assertByte(t *testing.T, dataInput DataInput, want int8) {
	t.Helper()
	got, err := dataInput.ReadSignedByte()
	if err != nil {
		t.Fatalf("ReadSignedByte: %v", err)
	}
	if got != want {
		t.Errorf("ReadSignedByte = %d, want %d", got, want)
	}
}

func TestDataInputReadUnsignedByte(t *testing.T) {
	for _, impl := range dataInputs {
		t.Run(impl.name, func(t *testing.T) {
			data := []byte{0, 0xFF, 2, 0xFD, 4, 0xFB, 6, 0xF9, 8, 0xF7}
			dataInput := impl.open(data)
			assertUnsignedByte(t, dataInput, 0)
			assertUnsignedByte(t, dataInput, 255)
			if err := dataInput.SetPosition(6); err != nil {
				t.Fatalf("SetPosition(6): %v", err)
			}
			assertUnsignedByte(t, dataInput, 6)
			assertUnsignedByte(t, dataInput, 249)
			length, err := dataInput.Length()
			if err != nil {
				t.Fatalf("Length: %v", err)
			}
			if err := dataInput.SetPosition(length - 1); err != nil {
				t.Fatalf("SetPosition(%d): %v", length-1, err)
			}
			assertUnsignedByte(t, dataInput, 247)
			if _, err := dataInput.ReadUnsignedByte(); err == nil {
				t.Error("ReadUnsignedByte past the end was accepted")
			}
		})
	}
}

func assertUnsignedByte(t *testing.T, dataInput DataInput, want int) {
	t.Helper()
	got, err := dataInput.ReadUnsignedByte()
	if err != nil {
		t.Fatalf("ReadUnsignedByte: %v", err)
	}
	if got != want {
		t.Errorf("ReadUnsignedByte = %d, want %d", got, want)
	}
}

func TestDataInputBasics(t *testing.T) {
	for _, impl := range dataInputs {
		t.Run(impl.name, func(t *testing.T) {
			data := []byte{0, 0xFF, 2, 0xFD, 4, 0xFB, 6, 0xF9, 8, 0xF7}
			dataInput := impl.open(data)
			length, err := dataInput.Length()
			if err != nil {
				t.Fatalf("Length: %v", err)
			}
			if length != 10 {
				t.Errorf("Length = %d, want 10", length)
			}
			if hasRemaining, err := dataInput.HasRemaining(); err != nil || !hasRemaining {
				t.Errorf("HasRemaining = %v, %v; want true, nil", hasRemaining, err)
			}
			if err := dataInput.SetPosition(-1); err == nil {
				t.Error("SetPosition(-1) was accepted")
			}
			if err := dataInput.SetPosition(length); err == nil {
				t.Errorf("SetPosition(%d) was accepted", length)
			}
		})
	}
}

func TestDataInputPeek(t *testing.T) {
	for _, impl := range dataInputs {
		t.Run(impl.name, func(t *testing.T) {
			data := []byte{0, 0xFF, 2, 0xFD, 4, 0xFB, 6, 0xF9, 8, 0xF7}
			dataInput := impl.open(data)
			if got, err := dataInput.PeekUnsignedByte(0); err != nil || got != 0 {
				t.Errorf("PeekUnsignedByte(0) = %d, %v; want 0, nil", got, err)
			}
			if got, err := dataInput.PeekUnsignedByte(5); err != nil || got != 251 {
				t.Errorf("PeekUnsignedByte(5) = %d, %v; want 251, nil", got, err)
			}
			if _, err := dataInput.PeekUnsignedByte(-1); err == nil {
				t.Error("PeekUnsignedByte(-1) was accepted")
			}
			length, err := dataInput.Length()
			if err != nil {
				t.Fatalf("Length: %v", err)
			}
			if _, err := dataInput.PeekUnsignedByte(length); err == nil {
				t.Errorf("PeekUnsignedByte(%d) was accepted", length)
			}
		})
	}
}

func TestDataInputReadShort(t *testing.T) {
	for _, impl := range dataInputs {
		t.Run(impl.name, func(t *testing.T) {
			data := []byte{0x00, 0x0F, 0xAA, 0, 0xFE, 0xFF}
			dataInput := impl.open(data)
			for _, want := range []int16{0x000F, int16(0xAA00 - 0x10000), int16(0xFEFF - 0x10000)} {
				got, err := dataInput.ReadShort()
				if err != nil {
					t.Fatalf("ReadShort: %v", err)
				}
				if got != want {
					t.Errorf("ReadShort = %d, want %d", got, want)
				}
			}
			if _, err := dataInput.ReadShort(); err == nil {
				t.Error("ReadShort past the end was accepted")
			}
		})
	}
}

func TestDataInputReadUnsignedShort(t *testing.T) {
	for _, impl := range dataInputs {
		t.Run(impl.name, func(t *testing.T) {
			data := []byte{0x00, 0x0F, 0xAA, 0, 0xFE, 0xFF}
			dataInput := impl.open(data)
			for _, want := range []int{0x000F, 0xAA00, 0xFEFF} {
				got, err := dataInput.ReadUnsignedShort()
				if err != nil {
					t.Fatalf("ReadUnsignedShort: %v", err)
				}
				if got != want {
					t.Errorf("ReadUnsignedShort = %#x, want %#x", got, want)
				}
			}
			if _, err := dataInput.ReadUnsignedShort(); err == nil {
				t.Error("ReadUnsignedShort past the end was accepted")
			}

			dataInput2 := impl.open([]byte{0x00})
			if _, err := dataInput2.ReadUnsignedShort(); err == nil {
				t.Error("ReadUnsignedShort over one byte was accepted")
			}
		})
	}
}

func TestDataInputReadInt(t *testing.T) {
	for _, impl := range dataInputs {
		t.Run(impl.name, func(t *testing.T) {
			data := []byte{0x00, 0x0F, 0xAA, 0, 0xFE, 0xFF, 0x30, 0x50}
			dataInput := impl.open(data)
			for _, want := range []int32{0x000FAA00, int32(0xFEFF3050 - 0x100000000)} {
				got, err := dataInput.ReadInt()
				if err != nil {
					t.Fatalf("ReadInt: %v", err)
				}
				if got != want {
					t.Errorf("ReadInt = %d, want %d", got, want)
				}
			}
			if _, err := dataInput.ReadInt(); err == nil {
				t.Error("ReadInt past the end was accepted")
			}

			dataInput2 := impl.open([]byte{0x00, 0x0F, 0xAA})
			if _, err := dataInput2.ReadInt(); err == nil {
				t.Error("ReadInt over three bytes was accepted")
			}
		})
	}
}
