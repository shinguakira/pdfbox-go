package encryption

// The tolerance for a trailing partial AES block, and the two paths that
// differ on it.

import (
	"crypto/aes"
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
