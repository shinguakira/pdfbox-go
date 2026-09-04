package filter

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// The two codes an LZW stream reserves.
//
// Port of LZWFilter.CLEAR_TABLE and LZWFilter.EOD.
const (
	lzwClearTable = 256
	lzwEOD        = 257
)

// LZW is the LZWDecode filter.
//
// Port of org.apache.pdfbox.filter.LZWFilter.
type LZW struct{}

var _ Filter = LZW{}

// Decode expands the codes and undoes any predictor.
func (LZW) Decode(w io.Writer, r io.Reader, parameters *cos.Dictionary,
	index int) (DecodeResult, error) {
	result := DecodeResult{Parameters: parameters}
	decodeParams := decodeParamsFor(parameters, index)
	earlyChange := true
	if decodeParams != nil {
		earlyChange = decodeParams.GetIntDefault(cos.EarlyChange, 1) != 0
	}

	// Java wraps the destination in a predictor stream and pushes into it; the
	// port decodes into a buffer and pulls the predictor over it, the way the
	// flate filter does.
	var raw bytes.Buffer
	if err := doLZWDecode(&raw, r, earlyChange); err != nil {
		return result, err
	}
	if err := decodePredictor(w, bytes.NewReader(raw.Bytes()),
		readPredictorParams(readOnly(decodeParams))); err != nil {
		return result, err
	}
	return result, nil
}

// errInvalidLZWCode stands for Java's EOFException("Invalid LZW code: n"),
// which doLZWDecode raises and then catches itself.
var errInvalidLZWCode = errors.New("filter: invalid LZW code")

func doLZWDecode(decoded io.Writer, encoded io.Reader, earlyChange bool) error {
	codeTable := createCodeTable() // includes CLEAR/EOD handling as needed
	chunk := 9
	in := newBitReader(encoded)
	out := bufio.NewWriter(decoded)

	var prev []byte // no previous string yet

	// Java runs the loop inside a try that catches EOFException, logs it and
	// falls through to the flush -- both the end of a truncated stream and the
	// invalid code below land there, so whatever decoded is kept.
	err := func() error {
		for {
			nextCommand, err := in.readBits(chunk)
			if err != nil {
				return err
			}
			if nextCommand == lzwEOD {
				return nil
			}

			if nextCommand == lzwClearTable {
				chunk = 9
				codeTable = createCodeTable()
				prev = nil
				continue
			}

			var curr []byte

			switch {
			case nextCommand < int64(len(codeTable)):
				// Normal case: code exists
				curr = codeTable[nextCommand]
				if _, err := out.Write(curr); err != nil {
					return err
				}

				if prev != nil {
					// Add prev + first(curr)
					entry := make([]byte, len(prev)+1)
					copy(entry, prev)
					entry[len(prev)] = curr[0]
					codeTable = append(codeTable, entry)
				}

			case nextCommand == int64(len(codeTable)) && prev != nil:
				// KwKwK case: code equals next available index
				curr = make([]byte, len(prev)+1)
				copy(curr, prev)
				curr[len(prev)] = prev[0]
				if _, err := out.Write(curr); err != nil {
					return err
				}
				codeTable = append(codeTable, curr)

			default:
				// Corrupt stream (code out of range, or KwKwK without prev)
				return fmt.Errorf("%w: %d", errInvalidLZWCode, nextCommand)
			}

			prev = curr // move forward
			chunk = calculateLZWChunk(len(codeTable), earlyChange)
		}
	}()
	if err != nil {
		slog.Warn("filter: premature EOF in LZW stream, EOD code missing", "err", err)
	}

	return out.Flush()
}

// Encode compresses the data into LZW codes.
func (LZW) Encode(w io.Writer, rawData io.Reader, parameters *cos.Dictionary) error {
	codeTable := createCodeTable()
	chunk := 9

	var inputPattern []byte
	out := newBitWriter(w)
	in := bufio.NewReader(rawData)

	if err := out.writeBits(lzwClearTable, chunk); err != nil {
		return err
	}
	foundCode := -1
	for {
		r, err := in.ReadByte()
		if err != nil {
			break
		}
		by := r
		if inputPattern == nil {
			inputPattern = []byte{by}
			foundCode = int(by) & 0xff
			continue
		}
		inputPattern = append(inputPattern, by)
		newFoundCode := findPatternCode(codeTable, inputPattern)
		if newFoundCode == -1 {
			// use previous
			chunk = calculateLZWChunk(len(codeTable)-1, true)
			if err := out.writeBits(int64(foundCode), chunk); err != nil {
				return err
			}
			// create new table entry
			codeTable = append(codeTable, inputPattern)

			if len(codeTable) == 4096 {
				// code table is full
				if err := out.writeBits(lzwClearTable, chunk); err != nil {
					return err
				}
				codeTable = createCodeTable()
			}

			inputPattern = []byte{by}
			foundCode = int(by) & 0xff
		} else {
			foundCode = newFoundCode
		}
	}
	if foundCode != -1 {
		chunk = calculateLZWChunk(len(codeTable)-1, true)
		if err := out.writeBits(int64(foundCode), chunk); err != nil {
			return err
		}
	}

	// PDFBOX-1977: the decoder wouldn't know that the encoder would output
	// an EOD as code, so he would have increased his own code table and
	// possibly adjusted the chunk. Therefore, the encoder must behave as
	// if the code table had just grown and thus it must be checked it is
	// needed to adjust the chunk, based on an increased table size parameter
	chunk = calculateLZWChunk(len(codeTable), true)

	if err := out.writeBits(lzwEOD, chunk); err != nil {
		return err
	}

	// pad with 0
	if err := out.writeBits(0, 7); err != nil {
		return err
	}

	// must do or file will be empty :-(
	return out.flush()
}

// findPatternCode returns the code of a pattern already in the table, or -1.
//
// Port of LZWFilter.findPatternCode. Its first branch returns the byte itself
// for a pattern of one, which is a signed byte in Java and so negative from
// 0x80 up; nothing reaches it, because encode only asks about patterns of two
// or more. See migration/JAVA-BUGS.md.
func findPatternCode(codeTable [][]byte, pattern []byte) int {
	// for the first 256 entries, index matches value
	if len(pattern) == 1 {
		return int(int8(pattern[0]))
	}

	// no need to test the first 256 + 2 entries against longer patterns
	for i := 257; i < len(codeTable); i++ {
		if bytes.Equal(codeTable[i], pattern) {
			return i
		}
	}

	return -1
}

// createCodeTable returns a fresh code table: the 256 single bytes, then the
// two reserved slots.
//
// Port of LZWFilter.createCodeTable, which copies a shared initial table.
func createCodeTable() [][]byte {
	codeTable := make([][]byte, 0, 4096)
	codeTable = append(codeTable, initialCodeTable...)
	return codeTable
}

// initialCodeTable is LZWFilter.INITIAL_CODE_TABLE. Entries 256 and 257 are
// null in Java, which is what the two nil slices are here: EOD and CLEAR_TABLE
// are handled before the table is indexed.
var initialCodeTable = createInitialCodeTable()

func createInitialCodeTable() [][]byte {
	codeTable := make([][]byte, 0, 258)
	for i := 0; i < 256; i++ {
		codeTable = append(codeTable, []byte{byte(i & 0xFF)})
	}
	codeTable = append(codeTable, nil) // 256 EOD
	codeTable = append(codeTable, nil) // 257 CLEAR_TABLE
	return codeTable
}

// calculateLZWChunk returns how many bits the next code takes.
//
// Port of LZWFilter.calculateChunk.
func calculateLZWChunk(tabSize int, earlyChange bool) int {
	i := tabSize
	if earlyChange {
		i++
	}
	if i >= 2048 {
		return 12
	}
	if i >= 1024 {
		return 11
	}
	if i >= 512 {
		return 10
	}
	return 9
}
