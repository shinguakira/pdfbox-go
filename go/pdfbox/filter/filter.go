// Package filter implements the stream filters a PDF uses to compress and
// encode stream data.
//
// Port of org.apache.pdfbox.filter. Only the filters slice 1 needs are present
// so far — see migration/STATUS.md for which.
package filter

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
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
// Port of FilterFactory.getFilter, which keeps a map from name to a single
// instance of each filter; the Go filters hold no state, so the switch returns
// the same zero value every time and comparing two of them by == matches Java
// comparing two references to the one instance.
//
// Java throws IOException("Invalid filter: " + name) for a name it does not
// know. /Identity is not in Java's map either -- it is reached through
// CryptFilter -- but slice 1 put it here and the parser depends on it.
func ByName(name *cos.Name) (Filter, error) {
	switch name {
	case cos.FlateDecode, cos.Fl:
		return Flate{}, nil
	case cos.DCTDecode, cos.DCT:
		return DCT{}, nil
	case cos.CCITTFaxDecode, cos.CCF:
		return CCITTFax{}, nil
	case cos.LZWDecode, cos.LZW:
		return LZW{}, nil
	case cos.ASCIIHexDecode, cos.AHx:
		return ASCIIHex{}, nil
	case cos.ASCII85Decode, cos.A85:
		return ASCII85{}, nil
	case cos.RunLengthDecode, cos.RL:
		return RunLength{}, nil
	case cos.Crypt:
		return Crypt{}, nil
	case cos.JPXDecode:
		return JPX{}, nil
	case cos.JBIG2Decode:
		return JBIG2{}, nil
	case cos.Identity:
		return Identity{}, nil
	}
	return nil, fmt.Errorf("%w: %s", ErrUnsupportedFilter, name.Name())
}

// allFilters returns every filter the factory holds, once each.
//
// Port of the package-private FilterFactory.getAllFilters, which exists for the
// tests. Java's map holds each filter under both its name and its abbreviation
// and getAllFilters returns the values, so a Java caller sees each filter twice;
// this returns each once, because the test it serves round trips every one and
// doing that twice proves nothing.
func allFilters() []Filter {
	return []Filter{
		Flate{}, DCT{}, CCITTFax{}, LZW{}, ASCIIHex{}, ASCII85{},
		RunLength{}, Crypt{}, JPX{}, JBIG2{},
	}
}

// Decode reads encoded data through a chain of filters.
//
// Port of the static Filter.decode. It panics on an empty filter list, where
// Java throws IllegalArgumentException.
//
// Java sizes the intermediate buffer from the stream's /Length, four times over
// and clamped to a chunk size; that is a memory decision with no effect on the
// bytes, so the port lets a bytes.Buffer grow. Java also closes each
// intermediate stream in a finally, which a buffer here does not need.
func Decode(encoded io.Reader, filterList []Filter, parameters *cos.Dictionary,
	options *DecodeOptions, results *[]DecodeResult) (pdfio.RandomAccessRead, error) {
	if len(filterList) == 0 {
		panic("Empty filterList")
	}
	if len(filterList) > 1 {
		seen := make(map[Filter]bool, len(filterList))
		reduced := make([]Filter, 0, len(filterList))
		for _, f := range filterList {
			if !seen[f] {
				seen[f] = true
				reduced = append(reduced, f)
			}
		}
		if len(reduced) != len(filterList) {
			// replace origin list with the reduced one
			filterList = reduced
			slog.Warn("filter: removed duplicated filter entries")
		}
	}

	input := encoded
	var output bytes.Buffer
	// apply filters
	for i, f := range filterList {
		if i > 0 {
			input = bytes.NewReader(output.Bytes())
			output = bytes.Buffer{}
		}
		result, err := decodeWithOptions(f, &output, input, parameters, i, options)
		if results != nil {
			*results = append(*results, result)
		}
		if err != nil {
			return nil, err
		}
	}
	return pdfio.NewReadBufferBytes(output.Bytes()), nil
}

// decodeWithOptions is Java's Filter.decode overload that takes DecodeOptions,
// whose base implementation drops them; a filter that honours them overrides
// it, and Go has no override, so the ones that do are named here.
func decodeWithOptions(f Filter, w io.Writer, r io.Reader, parameters *cos.Dictionary,
	index int, options *DecodeOptions) (DecodeResult, error) {
	return f.Decode(w, r, parameters, index)
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
