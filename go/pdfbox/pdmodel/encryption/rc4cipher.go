package encryption

import (
	"errors"
	"io"
)

// rc4Cipher is an implementation of the RC4 stream cipher.
//
// Port of the package-private org.apache.pdfbox.pdmodel.encryption.RC4Cipher.
// Go has crypto/rc4, but the Java carries its own and this is a migration: the
// salt walk below is the Java's, byte for byte.
type rc4Cipher struct {
	salt [256]int
	b    int
	c    int
}

// newRC4Cipher returns a cipher with no key set.
func newRC4Cipher() *rc4Cipher { return &rc4Cipher{} }

// setKey sets the key to be used when encrypting or decrypting.
func (r *rc4Cipher) setKey(key []byte) error {
	r.b = 0
	r.c = 0

	if len(key) < 1 || len(key) > 32 {
		return errors.New("number of bytes must be between 1 and 32")
	}
	for i := 0; i < len(r.salt); i++ {
		r.salt[i] = i
	}

	keyIndex := 0
	saltIndex := 0
	for i := 0; i < len(r.salt); i++ {
		saltIndex = (fixByte(key[keyIndex]) + r.salt[i] + saltIndex) % 256
		swapSalt(&r.salt, i, saltIndex)
		keyIndex = (keyIndex + 1) % len(key)
	}
	return nil
}

// fixByte is Java's `aByte < 0 ? 256 + aByte : aByte` over a signed byte, which
// is the unsigned value of the byte.
func fixByte(aByte byte) int { return int(aByte) }

// swapSalt swaps two elements in an array.
func swapSalt(data *[256]int, firstIndex, secondIndex int) {
	tmp := data[firstIndex]
	data[firstIndex] = data[secondIndex]
	data[secondIndex] = tmp
}

// encrypt encrypts a single byte of data.
func (r *rc4Cipher) encrypt(aByte byte) byte {
	r.b = (r.b + 1) % 256
	r.c = (r.salt[r.b] + r.c) % 256
	swapSalt(&r.salt, r.b, r.c)
	saltIndex := (r.salt[r.b] + r.salt[r.c]) % 256
	return aByte ^ byte(r.salt[saltIndex])
}

// writeBytes encrypts the given byte array to the output, which is also how it
// is decrypted.
func (r *rc4Cipher) writeBytes(data []byte, output io.Writer) error {
	return r.writeLen(data, len(data), output, make([]byte, len(data)))
}

// writeStream encrypts everything the reader yields to the output.
func (r *rc4Cipher) writeStream(data io.Reader, output io.Writer) error {
	buffer := make([]byte, 1024)
	for {
		amountRead, err := data.Read(buffer)
		if amountRead > 0 {
			if writeErr := r.writeLen(buffer, amountRead, output, buffer); writeErr != nil {
				return writeErr
			}
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if amountRead == 0 {
			// Java's InputStream.read returns -1 at the end and never 0 for a
			// non-empty buffer, so this cannot loop there; a Go reader may.
			return nil
		}
	}
}

func (r *rc4Cipher) writeLen(data []byte, length int, output io.Writer, buffer []byte) error {
	for i := 0; i < length; i++ {
		buffer[i] = r.encrypt(data[i])
	}
	_, err := output.Write(buffer[:length])
	return err
}
