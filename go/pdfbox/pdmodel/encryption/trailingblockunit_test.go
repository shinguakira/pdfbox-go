package encryption

// The tolerance for a trailing partial AES block, and the two paths that
// differ on it.

import (
	"bytes"
	"crypto/aes"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
)

// TestAES128RefusesATrailingPartialBlock is the other half of the same
// question, and it is why the tolerance takes a flag.
//
// Java has two AES paths and they differ exactly here. `encryptDataAESother`,
// which is AES-128 and AES-192, ends on `Cipher.doFinal`, and the
// `IllegalBlockSizeException` a short final block raises is caught as a
// `GeneralSecurityException` and rethrown as an IOException. Only
// `encryptDataAES256` reads through `CipherInputStream`, which swallows it.
//
// `aesCBC` is shared by both, so the tolerance has to be the caller's choice.
// Nine whole blocks and one byte over: AES-256 takes the nine, AES-128 refuses
// the lot.
func TestAES128RefusesATrailingPartialBlock(t *testing.T) {
	key := make([]byte, 16)
	iv := make([]byte, 16)
	data := make([]byte, 9*aes.BlockSize+1)

	if _, err := aesCBC(key, iv, data, true, false); err == nil {
		t.Error("AES-128 accepted a trailing partial block; Java's doFinal " +
			"raises IllegalBlockSizeException and encryptDataAESother " +
			"rethrows it as an IOException")
	} else if !strings.Contains(err.Error(), "not a multiple of the block size") {
		t.Errorf("AES-128 answered %v, want the block size error", err)
	}

	// The same input on the AES-256 path gets as far as the padding, which is
	// where a real document's plaintext comes from.
	if _, err := aesCBC(key, iv, data, true, true); err != nil {
		var padding *badPaddingError
		if !errors.As(err, &padding) {
			t.Errorf("AES-256 answered %v, want either plaintext or a padding "+
				"failure -- not a refusal of the input", err)
		}
	}

	// And a whole number of blocks is refused by neither.
	whole := make([]byte, 9*aes.BlockSize)
	for _, tolerate := range []tolerateShortFinalBlock{false, true} {
		if _, err := aesCBC(key, iv, whole, true, tolerate); err != nil {
			var padding *badPaddingError
			if !errors.As(err, &padding) {
				t.Errorf("a block-aligned input was refused with tolerate=%v: %v",
					tolerate, err)
			}
		}
	}
}

// TestAESPathsAnswerShortInputAsPDFBoxDoes runs both AES paths over the inputs
// either side of the initialization vector, with an all-zero key, and wants
// what PDFBox's encryptDataAESother and encryptDataAES256 answered for the same
// bytes, called through reflection on the running PDFBox.
//
// The row that failed is the stream of the vector alone. Java's doFinal on no
// data decrypts to nothing: PKCS5Padding.unpad answers 0 for an empty buffer.
// The port answered "no data to decrypt", which AES-128 turns into an error, so
// the object holding the stream could not be read at all. pdf.js's ichiji.pdf
// has two such form XObjects, 16 bytes each; PDFBox reads them as empty forms,
// and the port read them as null.
func TestAESPathsAnswerShortInputAsPDFBoxDoes(t *testing.T) {
	h := &securityHandlerBase{encryptionKey: make([]byte, 32)}
	aes128 := func(input []byte) ([]byte, error) {
		var out bytes.Buffer
		err := h.encryptDataAESother(make([]byte, 16), bytes.NewReader(input), &out, true)
		return out.Bytes(), err
	}
	aes256 := func(input []byte) ([]byte, error) {
		var out bytes.Buffer
		err := h.encryptDataAES256(bytes.NewReader(input), &out, true)
		return out.Bytes(), err
	}
	for _, c := range []struct {
		path    string
		decrypt func([]byte) ([]byte, error)
		length  int
		want    string // the output in hex, where PDFBox answered output
		wantErr string // part of the message, where PDFBox threw
	}{
		{"AES-128", aes128, 0, "", ""},
		{"AES-256", aes256, 0, "", ""},
		{"AES-128", aes128, 5, "", "AES initialization vector not fully read: only 5 bytes read instead of 16"},
		{"AES-256", aes256, 5, "", "AES initialization vector not fully read: only 5 bytes read instead of 16"},
		{"AES-128", aes128, 16, "", ""},
		{"AES-256", aes256, 16, "", ""},
		{"AES-128", aes128, 16 + 5, "", "not a multiple of the block size"},
		{"AES-256", aes256, 16 + 5, "", ""},
		{"AES-128", aes128, 32, "", "not properly padded"},
		{"AES-256", aes256, 32, "", ""},
		{"AES-128", aes128, 48, "", "not properly padded"},
		{"AES-256", aes256, 48, "67671ce1fa91ddeb0f8fbbb366b531b4", ""},
	} {
		got, err := c.decrypt(make([]byte, c.length))
		switch {
		case c.wantErr != "":
			if err == nil || !strings.Contains(err.Error(), c.wantErr) {
				t.Errorf("%s, %d bytes in: answered %x, %v; PDFBox throws %q",
					c.path, c.length, got, err, c.wantErr)
			}
		case err != nil:
			t.Errorf("%s, %d bytes in: %v; PDFBox answers %d bytes", c.path, c.length, err, len(c.want)/2)
		case hex.EncodeToString(got) != c.want:
			t.Errorf("%s, %d bytes in: answered %x, PDFBox answers %s", c.path, c.length, got, c.want)
		}
	}
}
