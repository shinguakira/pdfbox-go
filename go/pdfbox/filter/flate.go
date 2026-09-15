package filter

import (
	"bytes"
	"compress/flate"
	"compress/zlib"
	"errors"
	"io"
	"log/slog"
	"sync"

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
//
// The inflated bytes go straight on through the predictor into w, the way
// FlateFilter.decode transfers its decoder stream into the stream wrapPredictor
// hands back. This used to inflate into a buffer of its own and copy that into
// w, which held every inflated byte in memory twice, the first time in a buffer
// that grew by doubling. See migration/PERFORMANCE-PLAN.md.
func (Flate) Decode(w io.Writer, r io.Reader, parameters *cos.Dictionary, index int) (DecodeResult, error) {
	result := DecodeResult{Parameters: parameters}
	params := readPredictorParams(readOnly(decodeParamsFor(parameters, index)))

	// Skip the two zlib header bytes, as FlateFilterDecoderStream does. A
	// stream too short to have them has nothing to inflate.
	var header [2]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return result, nil
	}

	// plainReader hides a ReadByte the source may have, so compress/flate reads
	// it through its own buffer, as it always has, rather than one call a byte.
	inflater := acquireInflater(plainReader{r})
	defer releaseInflater(inflater)

	if err := decodePredictor(w, &endAtDamage{inflated: inflater}, params); err != nil {
		return result, err
	}
	return result, nil
}

// endAtDamage hands on what inflates and ends the data at the first error,
// which it logs. FlateFilterDecoderStream catches the DataFormatException, logs
// it and returns whatever inflated -- its comment reads "don't throw an
// exception, use the already read data or an empty stream" -- so a damaged PDF
// stays readable up to the damage: the predictor sees the bytes before it, and
// then the end of the data.
//
// Every error ends the data, a failing source included. That is what Decode did
// when it buffered the inflated bytes first and ran the predictor over whatever
// the buffer held, and it is kept exactly. Java lets a source's IOException out;
// that difference is older than this reader.
type endAtDamage struct {
	inflated io.Reader
	ended    bool
}

func (e *endAtDamage) Read(p []byte) (int, error) {
	if e.ended {
		return 0, io.EOF
	}
	n, err := e.inflated.Read(p)
	if err != nil {
		e.ended = true
		if err != io.EOF {
			slog.Warn("filter: premature end of flate stream", "err", err)
		}
		if n > 0 {
			return n, nil
		}
		return 0, io.EOF
	}
	return n, nil
}

// inflaters keeps decompressors between streams.
//
// flate.NewReader allocates a decompressor and a 32 KB window each time, and
// Decode needs one per stream. Reset keeps the window, the code tables and the
// input buffer, so one taken from here starts again for nothing. Java makes a
// new Inflater per stream, whose memory is native rather than heap; the pool
// changes nothing a caller can see. A deviation all the same, recorded in
// migration/STATUS.md under filter.
//
// It is a sync.Pool behind an interface only so that a test can put a pool in
// its place that never drops what it is given, and see exactly what went back.
var inflaters interface {
	Get() any
	Put(any)
} = &sync.Pool{}

func acquireInflater(r io.Reader) io.ReadCloser {
	if pooled, ok := inflaters.Get().(io.ReadCloser); ok {
		if err := pooled.(flate.Resetter).Reset(r, nil); err == nil {
			return pooled
		}
	}
	return flate.NewReader(r)
}

// releaseInflater pools a decompressor, pointed at nothing first so that it
// does not keep the last stream's source reachable while it waits.
func releaseInflater(inflater io.ReadCloser) {
	if err := inflater.(flate.Resetter).Reset(noData{}, nil); err != nil {
		return
	}
	inflaters.Put(inflater)
}

// plainReader is a reader with nothing but Read.
type plainReader struct{ io.Reader }

// noData is a reader that is already at its end.
type noData struct{}

func (noData) Read([]byte) (int, error) { return 0, io.EOF }

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

// NewFlateDecoderReader returns a reader that inflates as it is read, rather
// than into a buffer.
//
// Port of org.apache.pdfbox.filter.FlateFilterDecoderStream, which
// PDPage.getContentsForStreamParsing reads a single flate content stream
// through. It skips the two zlib header bytes and inflates raw, for the reason
// Decode above gives, and applies no predictor -- which is what Java does, and
// what makes the fast path wrong for a stream that declares one. See
// migration/JAVA-BUGS.md entry 63.
func NewFlateDecoderReader(r io.Reader) (io.ReadCloser, error) {
	// skip zlib header
	var header [2]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		// A stream too short to have a header has nothing to inflate; Java's
		// two bare in.read() calls answer -1 and carry on to inflate nothing.
		return io.NopCloser(bytes.NewReader(nil)), nil
	}
	return &flateDecoderStream{inflated: acquireInflater(plainReader{r})}, nil
}

// flateDecoderStream ends the stream where the data is damaged rather than
// failing, which is what FlateFilterDecoderStream.fetch does: it catches the
// DataFormatException, logs it, keeps whatever inflated and reports no more
// data. Its own comment reads "don't throw an exception, use the already read
// data or an empty stream" — a damaged PDF has to stay readable up to the
// damage, which is the whole reason this filter exists.
//
// Java recovers the partly inflated bytes out of its own 4096-byte buffer;
// compress/flate has already handed them to the caller by the time it reports
// the error, so there is nothing left here to recover.
//
// Only damaged data is swallowed. Java reads the source *outside* the try block
// and catches DataFormatException alone, so an IOException out of the wrapped
// stream propagates -- a failing disk must not look like the end of the page.
// compress/flate reports both through one error, so isDeflateDamage tells them
// apart.
type flateDecoderStream struct {
	inflated io.ReadCloser
	isEOF    bool

	// absorbed records that a Read took Java's decision: it saw damage, kept
	// what had inflated and reported the end of the stream. Only then may Close
	// stay quiet about the same damage. Without it, Close would have to judge
	// the error on its own, and it cannot: io.ErrUnexpectedEOF is both what a
	// truncated deflate stream looks like and what a source that stopped early
	// returns, so swallowing it unconditionally would hide a real failure of
	// the reader underneath.
	absorbed bool
}

func (f *flateDecoderStream) Read(p []byte) (int, error) {
	if f.isEOF {
		return 0, io.EOF
	}
	n, err := f.inflated.Read(p)
	if err != nil && !errors.Is(err, io.EOF) {
		if !isDeflateDamage(err) {
			// the source itself failed; Java lets this one out
			return n, err
		}
		slog.Warn("filter: premature end of flate stream", "err", err)
		f.isEOF = true
		f.absorbed = true
		f.release()
		if n > 0 {
			return n, nil
		}
		return 0, io.EOF
	}
	if errors.Is(err, io.EOF) {
		f.isEOF = true
		f.release()
	}
	return n, err
}

// release pools the decompressor as soon as the data has ended, because nothing
// needs it after that: Read answers io.EOF without it, and Close has nothing
// left to say -- a stream that ended cleanly closes with nil, and damage Read
// absorbed is exactly what Close swallows. Waiting for Close would pool almost
// nothing, since the stream engine, like Java's, never closes a content stream
// it has parsed.
func (f *flateDecoderStream) release() {
	releaseInflater(f.inflated)
	f.inflated = nil
}

// isDeflateDamage reports whether err is compress/flate complaining about the
// compressed data rather than the source underneath it.
//
// These are what Java raises as DataFormatException: a malformed stream, and a
// stream that ends before the final block. Anything else came from the reader
// being inflated and is the source's own failure.
func isDeflateDamage(err error) bool {
	var corrupt flate.CorruptInputError
	var internal flate.InternalError
	return errors.As(err, &corrupt) || errors.As(err, &internal) ||
		errors.Is(err, io.ErrUnexpectedEOF)
}

// Close releases the inflater, which is FlateFilterDecoderStream.close's
// inflater.end(). The stream it reads from is not closed: Java reaches that one
// through a RandomAccessInputStream, whose close is the inherited no-op.
//
// Damage does not come back out of here. compress/flate's reader remembers the
// error it stopped on and returns it again from Close; inflater.end() returns
// void and cannot. Reporting it would undo the whole of the Read above, which
// has already taken Java's decision to end the stream rather than fail it --
// the caller was told there was no more data and acted on it, and a close that
// then raises the same damage turns a readable page into a failed one. That is
// how qpdf/shared-images-errors.pdf failed. See
// TestFlateDecoderReaderCloseSwallowsDamage.
func (f *flateDecoderStream) Close() error {
	if f.inflated == nil {
		// pooled when the data ended, where Close answers nil; see release
		return nil
	}
	err := f.inflated.Close()
	if err != nil && f.absorbed && isDeflateDamage(err) {
		return nil
	}
	return err
}
