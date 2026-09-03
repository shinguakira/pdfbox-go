// Package filter implements the stream filters a PDF uses to compress and
// encode stream data.
//
// Port of org.apache.pdfbox.filter. Only the filters slice 1 needs are present
// so far — see migration/STATUS.md for which.
package filter

import (
	"errors"
	"fmt"
	"io"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// ErrUnsupportedFilter is returned for a filter name with no implementation.
var ErrUnsupportedFilter = errors.New("filter: unsupported filter")

// DecodeResult carries what a decoder learned about the data as it ran.
//
// Port of org.apache.pdfbox.filter.DecodeResult. Java also carries a JPX colour
// space and a soft mask image, which belong to the JPX filter and need pdmodel
// and image decoding; those fields arrive with that filter.
type DecodeResult struct {
	// Parameters is the stream dictionary the decoder ran against, which a
	// filter may amend as it decodes.
	Parameters *cos.Dictionary
}

// Filter decodes and encodes PDF stream data.
//
// Port of the abstract class org.apache.pdfbox.filter.Filter. Java has an
// abstract decode plus an overload taking DecodeOptions, which carries image
// subsampling settings; the port takes only the base form until an image filter
// needs the rest.
type Filter interface {
	// Decode reads encoded data from r and writes the decoded data to w.
	// parameters is the stream dictionary, and index identifies which entry of
	// its filter array this call is for.
	Decode(w io.Writer, r io.Reader, parameters *cos.Dictionary, index int) (DecodeResult, error)

	// Encode reads raw data from r and writes the encoded data to w.
	Encode(w io.Writer, r io.Reader, parameters *cos.Dictionary) error
}

// decodeParamsFor returns the decode parameter dictionary for one entry of a
// stream's filter array.
//
// Port of Filter.getDecodeParams. The /DecodeParms entry is either a single
// dictionary, when there is one filter, or an array parallel to /Filter. The
// abbreviated keys /F and /DP appear in inline images.
func decodeParamsFor(dictionary *cos.Dictionary, index int) *cos.Dictionary {
	if dictionary == nil {
		return nil
	}

	filterEntry := dictionary.GetDictionaryObject2(cos.F, cos.Filter)
	paramsEntry := dictionary.GetDictionaryObject2(cos.DP, cos.DecodeParms)

	if _, isName := filterEntry.(*cos.Name); isName {
		if params, ok := paramsEntry.(*cos.Dictionary); ok {
			// PDFBOX-3932: the specification says a single filter with
			// parameters puts them in a dictionary, but Adobe reads that as
			// "one filter name object", which is what this branch matches.
			return params
		}
	}

	if array, ok := paramsEntry.(*cos.Array); ok {
		if index < array.Size() {
			if params, ok := array.GetObject(index).(*cos.Dictionary); ok {
				return params
			}
		}
		return nil
	}

	if params, ok := paramsEntry.(*cos.Dictionary); ok {
		return params
	}
	return nil
}

// ByName returns the filter for a PDF filter name.
//
// Port of FilterFactory.getFilter. Names not yet implemented return
// ErrUnsupportedFilter rather than a filter that silently does nothing.
func ByName(name *cos.Name) (Filter, error) {
	switch name {
	case cos.FlateDecode, cos.Fl:
		return Flate{}, nil
	case cos.Identity:
		return Identity{}, nil
	}
	return nil, fmt.Errorf("%w: %s", ErrUnsupportedFilter, name.Name())
}

// Identity passes data through unchanged.
//
// Port of org.apache.pdfbox.filter.IdentityFilter.
type Identity struct{}

var _ Filter = Identity{}

// Decode copies the data unchanged.
func (Identity) Decode(w io.Writer, r io.Reader, parameters *cos.Dictionary, index int) (DecodeResult, error) {
	_, err := io.Copy(w, r)
	return DecodeResult{Parameters: parameters}, err
}

// Encode copies the data unchanged.
func (Identity) Encode(w io.Writer, r io.Reader, parameters *cos.Dictionary) error {
	_, err := io.Copy(w, r)
	return err
}
