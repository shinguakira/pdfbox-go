package filter

import (
	"fmt"
	"io"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// Crypt decrypts data encrypted by a security handler, reproducing the data as
// it was before encryption.
//
// Port of org.apache.pdfbox.filter.CryptFilter.
type Crypt struct{}

var _ Filter = Crypt{}

// Decode passes the data through where the crypt filter is the identity one.
func (Crypt) Decode(w io.Writer, r io.Reader, parameters *cos.Dictionary,
	index int) (DecodeResult, error) {
	encryptionName := parameters.GetCOSName(cos.NameKey)
	if encryptionName == nil || encryptionName == cos.Identity {
		// currently the only supported implementation is the Identity crypt filter
		if _, err := (Identity{}).Decode(w, r, parameters, index); err != nil {
			return DecodeResult{Parameters: parameters}, err
		}
		return DecodeResult{Parameters: parameters}, nil
	}
	return DecodeResult{Parameters: parameters},
		fmt.Errorf("Unsupported crypt filter %s", encryptionName.Name())
}

// Encode passes the data through where the crypt filter is the identity one.
func (Crypt) Encode(w io.Writer, r io.Reader, parameters *cos.Dictionary) error {
	encryptionName := parameters.GetCOSName(cos.NameKey)
	if encryptionName == nil || encryptionName == cos.Identity {
		// currently the only supported implementation is the Identity crypt filter
		return (Identity{}).Encode(w, r, parameters)
	}
	return fmt.Errorf("Unsupported crypt filter %s", encryptionName.Name())
}
