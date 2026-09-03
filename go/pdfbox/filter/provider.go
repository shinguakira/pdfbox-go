package filter

import (
	"io"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// Provider resolves PDF filter names to codecs for cos.Stream.
//
// Port of the role FilterFactory.INSTANCE plays in Java. It exists as a type
// rather than a package-level singleton so that a Stream is handed its codecs
// explicitly; see the StreamCodec doc in cos for why the dependency runs this
// way round.
type Provider struct{}

var _ cos.CodecProvider = Provider{}

// CodecForName returns the codec for a PDF filter name.
func (Provider) CodecForName(name *cos.Name) (cos.StreamCodec, error) {
	f, err := ByName(name)
	if err != nil {
		return nil, err
	}
	return codec{f}, nil
}

// codec adapts a Filter to cos.StreamCodec, which differs only in dropping the
// DecodeResult that cos has no use for.
type codec struct {
	filter Filter
}

func (c codec) Decode(w io.Writer, r io.Reader, parameters *cos.Dictionary, index int) error {
	_, err := c.filter.Decode(w, r, parameters, index)
	return err
}

func (c codec) Encode(w io.Writer, r io.Reader, parameters *cos.Dictionary) error {
	return c.filter.Encode(w, r, parameters)
}
