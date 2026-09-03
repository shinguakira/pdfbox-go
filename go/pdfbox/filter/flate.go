package filter

import (
	"bytes"
	"compress/flate"
	"compress/zlib"
	"io"
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// Flate is the FlateDecode filter, zlib-compressed stream data.
//
// Port of org.apache.pdfbox.filter.FlateFilter together with
// FlateFilterDecoderStream.
type Flate struct{}

var _ Filter = Flate{}

// Decode inflates the data and undoes any predictor.
//
// PDFBox does not use a zlib reader here. FlateFilterDecoderStream reads and
// discards the two header bytes itself and then inflates with nowrap, which
// bypasses both the zlib header and the trailing Adler-32 checksum. That is
// deliberate: real PDFs are frequently truncated or carry a wrong checksum, and
// a checksum-verifying reader would throw away data that decoded correctly. The
// port does the same with compress/flate, which is raw deflate.
//
// Whatever decoded before an error is still written out, for the same reason.
func (Flate) Decode(w io.Writer, r io.Reader, parameters *cos.Dictionary, index int) (DecodeResult, error) {
	result := DecodeResult{Parameters: parameters}
	params := readPredictorParams(readOnly(decodeParamsFor(parameters, index)))

	br := &byteCountingReader{r: r}

	// Skip the two zlib header bytes, as FlateFilterDecoderStream does. A
	// stream too short to have them has nothing to inflate.
	var header [2]byte
	if _, err := io.ReadFull(br, header[:]); err != nil {
		return result, nil
	}

	inflated := flate.NewReader(br)
	defer inflated.Close()

	// The predictor is applied to the inflated bytes, so the two are chained.
	// Buffering between them keeps a decode error from losing the bytes that
	// did inflate.
	var raw bytes.Buffer
	_, inflateErr := io.Copy(&raw, inflated)

	if inflateErr != nil {
		// FlateFilterDecoderStream catches the DataFormatException, logs it and
		// returns whatever inflated — its comment reads "don't throw an
		// exception, use the already read data or an empty stream". A damaged
		// PDF has to stay readable up to the damage, so the error is reported
		// and not propagated.
		slog.Warn("filter: premature end of flate stream", "err", inflateErr)
	}

	if err := decodePredictor(w, bytes.NewReader(raw.Bytes()), params); err != nil {
		return result, err
	}
	return result, nil
}

// CompressionLevel is the deflate level used when encoding.
//
// Port of Filter.getCompressionLevel, which Java reads from the
// org.apache.pdfbox.filter.deflatelevel system property and clamps to -1..9,
// defaulting to Deflater.DEFAULT_COMPRESSION. Go has no system properties, so
// it is a package variable with the same default and the same range.
var CompressionLevel = zlib.DefaultCompression

// Encode deflates the data.
func (Flate) Encode(w io.Writer, r io.Reader, parameters *cos.Dictionary) error {
	level := CompressionLevel
	if level < -1 || level > 9 {
		// Java clamps out-of-range property values rather than failing.
		level = zlib.DefaultCompression
	}
	zw, err := zlib.NewWriterLevel(w, level)
	if err != nil {
		return err
	}
	if _, err := io.Copy(zw, r); err != nil {
		zw.Close()
		return err
	}
	return zw.Close()
}

// readOnly narrows a dictionary for the predictor parameter reader, mapping a
// nil dictionary to a nil interface rather than a non-nil interface holding a
// nil pointer.
func readOnly(d *cos.Dictionary) cos.ReadOnlyDictionary {
	if d == nil {
		return nil
	}
	return d
}

// byteCountingReader reports whether anything was read, so that a stream
// shorter than the two-byte header can be told from an empty one.
type byteCountingReader struct {
	r io.Reader
	n int64
}

func (b *byteCountingReader) Read(p []byte) (int, error) {
	n, err := b.r.Read(p)
	b.n += int64(n)
	return n, err
}
