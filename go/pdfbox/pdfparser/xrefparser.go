package pdfparser

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// minimumSearchOffset is how far into a file an object can possibly start:
// there can't be any object at the very beginning of a pdf.
const minimumSearchOffset = 6

var (
	xrefTableMarker = []byte("xref")
	startxrefBytes  = []byte("startxref")
)

// XrefParser reads the cross-reference chain of a file: the tables, the
// streams, and the trailers that link them.
//
// Port of org.apache.pdfbox.pdfparser.XrefParser.
type XrefParser struct {
	xrefTrailerResolver *XrefTrailerResolver
	parser              *FileParser
	source              pdfio.RandomAccessRead
}

// NewXrefParser returns a cross-reference parser reading through the given file
// parser.
func NewXrefParser(fileParser *FileParser) *XrefParser {
	return &XrefParser{
		xrefTrailerResolver: NewXrefTrailerResolver(),
		parser:              fileParser,
		source:              fileParser.Source(),
	}
}

// XrefTable returns the cross-reference entries the chain resolved to.
func (x *XrefParser) XrefTable() XrefEntries {
	return FromKeyed(x.xrefTrailerResolver.XrefTable())
}

// ParseXref walks the cross-reference chain from the given startxref offset and
// returns the trailer it resolves to.
func (x *XrefParser) ParseXref(document *cos.Document, startXRefOffset int64) (*cos.Dictionary, error) {
	if _, err := x.source.Seek(startXRefOffset, 0); err != nil {
		return nil, err
	}
	parsed, err := x.parseStartXref()
	if err != nil {
		return nil, err
	}
	startXrefOffset := max(int64(0), parsed)

	// check the startxref offset
	fixedOffset, err := x.checkXRefOffset(startXrefOffset)
	if err != nil {
		return nil, err
	}
	if fixedOffset > -1 {
		startXrefOffset = fixedOffset
	}
	document.SetStartXref(startXrefOffset)

	prev := startXrefOffset
	// ---- parse whole chain of xref tables/object streams using PREV reference
	prevSet := map[int64]bool{}
	var trailer *cos.Dictionary

	for prev > 0 {
		// save expected position for loop detection
		prevSet[prev] = true

		// seek to xref table
		if _, err := x.source.Seek(prev, 0); err != nil {
			return nil, err
		}
		// skip white spaces
		if err := x.parser.SkipSpaces(); err != nil {
			return nil, err
		}
		// save current position as well due to skipped spaces
		position, err := x.source.Position()
		if err != nil {
			return nil, err
		}
		prevSet[position] = true

		// -- parse xref
		peeked, err := x.peek()
		if err != nil {
			return nil, err
		}
		if peeked == 'x' {
			// xref table and trailer
			// use existing parser to parse xref table
			okTable, err := x.parseXrefTable(prev)
			if err != nil {
				return nil, err
			}
			okTrailer := false
			if okTable {
				okTrailer, err = x.parseTrailer()
				if err != nil {
					return nil, err
				}
			}
			if !okTable || !okTrailer {
				position, _ := x.source.Position()
				return nil, fmt.Errorf("pdfparser: Expected trailer object at offset %d", position)
			}
			trailer = x.xrefTrailerResolver.CurrentTrailer()

			// check for a XRef stream, it may contain some object ids of
			// compressed objects
			if trailer.ContainsKey(cos.XRefStm) {
				streamOffset := int64(trailer.GetIntDefault(cos.XRefStm, 0))
				// check the xref stream reference
				fixedOffset, err := x.checkXRefOffset(streamOffset)
				if err != nil {
					return nil, err
				}
				if fixedOffset > -1 && fixedOffset != streamOffset {
					slog.Warn("/XRefStm offset is incorrect, corrected",
						"offset", streamOffset, "corrected", fixedOffset)
					streamOffset = fixedOffset
					trailer.SetInt(cos.XRefStm, int(streamOffset))
				}
				if streamOffset > 0 {
					if _, err := x.source.Seek(streamOffset, 0); err != nil {
						return nil, err
					}
					if err := x.parser.SkipSpaces(); err != nil {
						return nil, err
					}
					if _, err := x.parseXrefObjStream(prev, false); err != nil {
						slog.Error("Failed to parse /XRefStm", "offset", streamOffset, "err", err)
					} else {
						document.SetHasHybridXRef()
					}
				} else {
					slog.Error("Skipped XRef stream due to a corrupt offset", "offset", streamOffset)
				}
			}
			prev = trailer.GetLong(cos.Prev)
		} else {
			// parse xref stream
			prev, err = x.parseXrefObjStream(prev, true)
			if err != nil {
				return nil, err
			}
			trailer = x.xrefTrailerResolver.CurrentTrailer()
		}

		if prev > 0 {
			// check the xref table reference
			fixedOffset, err := x.checkXRefOffset(prev)
			if err != nil {
				return nil, err
			}
			if fixedOffset > -1 && fixedOffset != prev {
				prev = fixedOffset
				trailer.SetLong(cos.Prev, prev)
			}
		}
		if prevSet[prev] {
			return nil, fmt.Errorf("pdfparser: /Prev loop at offset %d", prev)
		}
	}

	// ---- build valid xrefs out of the xref chain
	x.xrefTrailerResolver.SetStartxref(startXrefOffset)
	trailer = x.xrefTrailerResolver.Trailer()
	document.SetTrailer(trailer)
	document.SetIsXRefStream(x.xrefTrailerResolver.XrefType() == XRefTypeStream)

	// check the offsets of all referenced objects
	if err := x.checkXrefOffsets(); err != nil {
		return nil, err
	}

	// copy xref table
	table := x.XrefTable()
	var highest int64
	for _, entry := range table {
		if entry.Key.Number() > highest {
			highest = entry.Key.Number()
		}
	}
	document.AddXRefTable(table.ToKeyed())

	// remember the highest XRef object number to avoid it being reused in
	// incremental saving
	document.SetHighestXRefObjectNumber(highest)
	return trailer, nil
}

// peek returns the next byte without consuming it.
func (x *XrefParser) peek() (int, error) {
	position, err := x.source.Position()
	if err != nil {
		return -1, err
	}
	b := make([]byte, 1)
	n, err := x.source.Read(b)
	if _, seekErr := x.source.Seek(position, 0); seekErr != nil {
		return -1, seekErr
	}
	if n < 1 {
		return -1, nil
	}
	if err != nil && n < 1 {
		return -1, err
	}
	return int(b[0]), nil
}

// parseTrailer reads the trailer dictionary that follows a cross-reference
// table.
func (x *XrefParser) parseTrailer() (bool, error) {
	// parse the last trailer.
	trailerOffset, err := x.source.Position()
	if err != nil {
		return false, err
	}
	// PDFBOX-1739 skip extra xref entries in RegisSTAR documents
	nextCharacter, err := x.peek()
	if err != nil {
		return false, err
	}
	for nextCharacter != 't' && isDigitByte(nextCharacter) {
		position, err := x.source.Position()
		if err != nil {
			return false, err
		}
		if position == trailerOffset {
			// warn only the first time
			slog.Warn("Expected trailer object, keep trying", "offset", trailerOffset)
		}
		if _, err := x.parser.ReadLine(); err != nil {
			return false, err
		}
		nextCharacter, err = x.peek()
		if err != nil {
			return false, err
		}
	}
	if nextCharacter != 't' {
		return false, nil
	}

	// read "trailer"
	currentOffset, err := x.source.Position()
	if err != nil {
		return false, err
	}
	nextLine, err := x.parser.ReadLine()
	if err != nil {
		return false, err
	}
	if strings.TrimSpace(nextLine) != "trailer" {
		// in some cases the EOL is missing and the trailer immediately
		// continues with "<<" or with a blank character
		// even if this does not comply with PDF reference we want to support as
		// many PDFs as possible
		// Acrobat reader can also deal with this.
		if !strings.HasPrefix(nextLine, "trailer") {
			return false, nil
		}
		// we can't just unread a portion of the read data as we don't know if
		// the EOL consist of 1 or 2 bytes
		// jump back right after "trailer"
		if _, err := x.source.Seek(currentOffset+int64(len("trailer")), 0); err != nil {
			return false, err
		}
	}

	// in some cases the EOL is missing and the trailer continues with " <<"
	// even if this does not comply with PDF reference we want to support as
	// many PDFs as possible
	// Acrobat reader can also deal with this.
	if err := x.parser.SkipSpaces(); err != nil {
		return false, err
	}
	parsedTrailer, err := x.parser.ParseCOSDictionary(true)
	if err != nil {
		return false, err
	}
	x.xrefTrailerResolver.SetTrailer(parsedTrailer)
	if err := x.parser.SkipSpaces(); err != nil {
		return false, err
	}
	return true, nil
}

// parseXrefObjStream reads a cross-reference stream and returns the offset of
// the previous one.
func (x *XrefParser) parseXrefObjStream(objByteOffset int64, isStandalone bool) (int64, error) {
	// ---- parse indirect object head
	if _, err := x.parser.ReadObjectNumber(); err != nil {
		return 0, err
	}
	if _, err := x.parser.ReadGenerationNumber(); err != nil {
		return 0, err
	}
	if err := x.parser.ReadObjectMarker(); err != nil {
		return 0, err
	}

	dict, err := x.parser.ParseCOSDictionary(false)
	if err != nil {
		return 0, err
	}
	xrefStream, err := x.parser.ParseCOSStream(dict)
	if err != nil {
		return 0, err
	}
	defer xrefStream.Close()

	// the cross reference stream of a hybrid xref table will be added to the
	// existing one and we must not override the offset and the trailer
	if isStandalone {
		x.xrefTrailerResolver.NextXrefObj(objByteOffset, XRefTypeStream)
		x.xrefTrailerResolver.SetTrailer(&xrefStream.Dictionary)
	}
	streamParser, err := NewXrefStreamParser(xrefStream)
	if err != nil {
		return 0, err
	}
	if err := streamParser.Parse(x.xrefTrailerResolver); err != nil {
		return 0, err
	}
	return dict.GetLong(cos.Prev), nil
}

// checkXRefOffset checks that the given offset really holds a cross-reference
// table or stream, returning a corrected offset or -1.
func (x *XrefParser) checkXRefOffset(startXRefOffset int64) (int64, error) {
	if _, err := x.source.Seek(startXRefOffset, 0); err != nil {
		return 0, err
	}
	if err := x.parser.SkipSpaces(); err != nil {
		return 0, err
	}
	isXref, err := x.parser.isString(xrefTableMarker)
	if err != nil {
		return 0, err
	}
	if isXref {
		return startXRefOffset, nil
	}
	if startXRefOffset > 0 {
		ok, err := x.checkXRefStreamOffset(startXRefOffset)
		if err != nil {
			return 0, err
		}
		if ok {
			return startXRefOffset, nil
		}
		return x.calculateXRefFixedOffset(startXRefOffset)
	}
	// can't find a valid offset
	return -1, nil
}

// calculateXRefFixedOffset looks for the cross-reference table by brute force.
func (x *XrefParser) calculateXRefFixedOffset(objectOffset int64) (int64, error) {
	if objectOffset < 0 {
		slog.Error("Invalid object offset when searching for a xref table/stream",
			"offset", objectOffset)
		return 0, nil
	}
	// search for the offset of the given xref table/stream among those found by
	// a brute force search.
	bf, err := x.parser.BruteForceParser()
	if err != nil {
		return 0, err
	}
	newOffset, err := bf.BFSearchForXRef(objectOffset)
	if err != nil {
		return 0, err
	}
	if newOffset > -1 {
		return newOffset, nil
	}
	slog.Error("Can't find the object xref table/stream", "offset", objectOffset)
	return 0, nil
}

// checkXRefStreamOffset reports whether a cross-reference stream really starts
// at the given offset.
func (x *XrefParser) checkXRefStreamOffset(startXRefOffset int64) (bool, error) {
	if startXRefOffset == 0 {
		return true, nil
	}
	// seek to offset-1
	if _, err := x.source.Seek(startXRefOffset-1, 0); err != nil {
		return false, err
	}
	b := make([]byte, 1)
	n, _ := x.source.Read(b)
	nextValue := -1
	if n > 0 {
		nextValue = int(b[0])
	}
	// the first character has to be a whitespace, and then a digit
	if !IsWhitespace(nextValue) {
		return false, nil
	}
	if err := x.parser.SkipSpaces(); err != nil {
		return false, err
	}
	isDigit, err := isDigitAt(x.source)
	if err != nil {
		return false, err
	}
	if !isDigit {
		return false, nil
	}

	// it's a XRef stream
	dict, err := func() (*cos.Dictionary, error) {
		if _, err := x.parser.ReadObjectNumber(); err != nil {
			return nil, err
		}
		if _, err := x.parser.ReadGenerationNumber(); err != nil {
			return nil, err
		}
		if err := x.parser.ReadObjectMarker(); err != nil {
			return nil, err
		}
		// check the dictionary to avoid false positives
		return x.parser.ParseCOSDictionary(false)
	}()
	if _, seekErr := x.source.Seek(startXRefOffset, 0); seekErr != nil {
		return false, seekErr
	}
	if err != nil {
		// there wasn't an object of a xref stream
		return false, nil
	}
	return dict.GetNameAsString(cos.Type, "") == "XRef", nil
}

// validateXrefOffsets checks that each entry of the table points at the object
// it claims to, correcting the generation where it can.
func (x *XrefParser) validateXrefOffsets(xrefOffset XrefEntries) (bool, error) {
	if xrefOffset == nil {
		return true, nil
	}
	type correction struct {
		from *cos.ObjectKey
		to   *cos.ObjectKey
	}
	var correctedKeys []correction
	validKeys := map[int64]bool{}

	for _, entry := range xrefOffset {
		// a negative offset number represents an object number itself
		// see type 2 entry in xref stream
		if entry.Offset < 0 {
			continue
		}
		foundObjectKey, found, err := x.findObjectKey(entry.Key, entry.Offset, xrefOffset)
		if err != nil {
			return false, err
		}
		if !found {
			return false, nil
		}
		if foundObjectKey.InternalHash() != entry.Key.InternalHash() {
			// Generation was fixed - need to update map later, after iteration
			correctedKeys = append(correctedKeys, correction{from: entry.Key, to: foundObjectKey})
		} else {
			validKeys[entry.Key.InternalHash()] = true
		}
	}

	correctedPointers := NewXrefEntries()
	for _, c := range correctedKeys {
		if !validKeys[c.to.InternalHash()] {
			// Only replace entries, if the original entry does not point to a
			// valid object
			offset, _ := xrefOffset.Get(c.from)
			correctedPointers.Put(c.to, offset)
		}
	}
	// remove old invalid, as some might not be replaced
	for _, c := range correctedKeys {
		xrefOffset.Delete(c.from)
	}
	xrefOffset.PutAll(correctedPointers)
	return true, nil
}

// checkXrefOffsets replaces the cross-reference table with a brute force search
// where the offsets do not hold up.
func (x *XrefParser) checkXrefOffsets() error {
	xrefOffset := FromKeyed(x.xrefTrailerResolver.XrefTable())
	valid, err := x.validateXrefOffsets(xrefOffset)
	if err != nil {
		return err
	}
	if valid {
		return nil
	}
	bf, err := x.parser.BruteForceParser()
	if err != nil {
		return err
	}
	bfCOSObjectKeyOffsets, err := bf.BFCOSObjectOffsets()
	if err != nil {
		return err
	}
	if len(bfCOSObjectKeyOffsets) == 0 {
		return nil
	}
	x.xrefTrailerResolver.ReplaceXrefTable(bfCOSObjectKeyOffsets.ToKeyed())
	return nil
}

// findObjectKey reads the object header at the given offset and returns the key
// it really carries. The second result is false where there is no valid object
// there.
func (x *XrefParser) findObjectKey(objectKey *cos.ObjectKey, offset int64,
	xrefOffset XrefEntries) (*cos.ObjectKey, bool, error) {
	// there can't be any object at the very beginning of a pdf
	if offset < minimumSearchOffset {
		return objectKey, false, nil
	}

	// Java swallows every IOException here: obviously there isn't any valid
	// object number.
	found, ok := func() (*cos.ObjectKey, bool) {
		if _, err := x.source.Seek(offset, 0); err != nil {
			return objectKey, false
		}
		if err := x.parser.SkipWhiteSpaces(); err != nil {
			return objectKey, false
		}
		position, err := x.source.Position()
		if err != nil {
			return objectKey, false
		}
		if position == offset {
			// ensure that at least one whitespace is skipped in front of the
			// object number
			if _, err := x.source.Seek(offset-1, 0); err != nil {
				return objectKey, false
			}
			position, err := x.source.Position()
			if err != nil {
				return objectKey, false
			}
			if position < offset {
				isDigit, err := isDigitAt(x.source)
				if err != nil {
					return objectKey, false
				}
				if !isDigit {
					// anything else but a digit may be some garbage of the
					// previous object -> just ignore it
					b := make([]byte, 1)
					x.source.Read(b)
				} else {
					current, err := x.source.Position()
					if err != nil {
						return objectKey, false
					}
					current--
					if _, err := x.source.Seek(current, 0); err != nil {
						return objectKey, false
					}
					for {
						isDigit, err := isDigitAt(x.source)
						if err != nil || !isDigit {
							break
						}
						current--
						if _, err := x.source.Seek(current, 0); err != nil {
							return objectKey, false
						}
					}
					newObjNr, err := x.parser.ReadObjectNumber()
					if err != nil {
						return objectKey, false
					}
					newGenNr, err := x.parser.ReadGenerationNumber()
					if err != nil {
						return objectKey, false
					}
					newObjKey, err := cos.NewObjectKey(newObjNr, newGenNr)
					if err != nil {
						return objectKey, false
					}
					existingOffset, exists := xrefOffset.Get(newObjKey)
					// the found object number belongs to another uncompressed
					// object at the same or nearby offset — something has to be
					// wrong
					if exists && existingOffset > 0 && abs64(offset-existingOffset) < 10 {
						return objectKey, false
					}
					// something seems to be wrong but it's hard to determine
					// what exactly -> simply continue
					if _, err := x.source.Seek(offset, 0); err != nil {
						return objectKey, false
					}
				}
			}
		}

		// try to read the given object/generation number
		foundObjectNumber, err := x.parser.ReadObjectNumber()
		if err != nil {
			return objectKey, false
		}
		key := objectKey
		if key.Number() != foundObjectNumber {
			slog.Warn("found wrong object number",
				"expected", key.Number(), "found", foundObjectNumber)
			corrected, err := cos.NewObjectKey(foundObjectNumber, key.Generation())
			if err != nil {
				return objectKey, false
			}
			key = corrected
		}
		genNumber, err := x.parser.ReadGenerationNumber()
		if err != nil {
			return objectKey, false
		}
		// finally try to read the object marker
		if err := x.parser.ReadObjectMarker(); err != nil {
			return objectKey, false
		}
		if genNumber == key.Generation() {
			return key, true
		}
		if genNumber > key.Generation() {
			corrected, err := cos.NewObjectKey(key.Number(), genNumber)
			if err != nil {
				return objectKey, false
			}
			return corrected, true
		}
		return objectKey, false
	}()
	return found, ok, nil
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

// parseStartXref reads the offset the startxref keyword gives, or -1.
func (x *XrefParser) parseStartXref() (int64, error) {
	startXref := int64(-1)
	isStartxref, err := x.parser.isString(startxrefBytes)
	if err != nil {
		return 0, err
	}
	if isStartxref {
		if _, err := x.parser.ReadString(); err != nil {
			return 0, err
		}
		if err := x.parser.SkipSpaces(); err != nil {
			return 0, err
		}
		// This integer is the byte offset of the first object referenced by the
		// xref or xref stream
		startXref, err = x.parser.ReadLong()
		if err != nil {
			return 0, err
		}
	}
	return startXref, nil
}

// parseXrefTable reads one cross-reference table.
func (x *XrefParser) parseXrefTable(startByteOffset int64) (bool, error) {
	peeked, err := x.peek()
	if err != nil {
		return false, err
	}
	if peeked != 'x' {
		return false, nil
	}
	xref, err := x.parser.ReadString()
	if err != nil {
		return false, err
	}
	if strings.TrimSpace(xref) != "xref" {
		return false, nil
	}

	// check for trailer after xref
	str, err := x.parser.ReadString()
	if err != nil {
		return false, err
	}
	position, err := x.source.Position()
	if err != nil {
		return false, err
	}
	if _, err := x.source.Seek(position-int64(len(str)), 0); err != nil {
		return false, err
	}

	// signal start of new XRef
	x.xrefTrailerResolver.NextXrefObj(startByteOffset, XRefTypeTable)

	if strings.HasPrefix(str, "trailer") {
		slog.Warn("skipping empty xref table")
		return false, nil
	}

	// Xref tables can have multiple sections. Each starts with a starting
	// object id and a count.
	for {
		currentLine, err := x.parser.ReadLine()
		if err != nil {
			return false, err
		}
		splitString := strings.Fields(currentLine)
		if len(splitString) != 2 {
			slog.Warn("Unexpected XRefTable Entry", "line", currentLine)
			return false, nil
		}
		// first obj id
		currObjID, err := strconv.ParseInt(splitString[0], 10, 64)
		if err != nil {
			slog.Warn("XRefTable: invalid ID for the first object", "line", currentLine)
			return false, nil
		}
		// the number of objects in the xref table
		count, err := strconv.Atoi(splitString[1])
		if err != nil {
			slog.Warn("XRefTable: invalid number of objects", "line", currentLine)
			return false, nil
		}

		if err := x.parser.SkipSpaces(); err != nil {
			return false, err
		}
		for i := 0; i < count; i++ {
			isEOF, err := x.parser.IsEOF()
			if err != nil {
				return false, err
			}
			if isEOF {
				break
			}
			nextChar, err := x.peek()
			if err != nil {
				return false, err
			}
			if nextChar == 't' || IsEndOfName(nextChar) {
				break
			}
			// Ignore table contents
			currentLine, err := x.parser.ReadLine()
			if err != nil {
				return false, err
			}
			splitString := strings.Fields(currentLine)
			if len(splitString) < 3 {
				slog.Warn("invalid xref line", "line", currentLine)
				break
			}
			if splitString[len(splitString)-1] == "n" {
				currOffset, err := strconv.ParseInt(splitString[0], 10, 64)
				if err != nil {
					return false, fmt.Errorf("pdfparser: %w", err)
				}
				// skip 0 offsets
				if currOffset > 0 {
					currGenID, err := strconv.Atoi(splitString[1])
					if err != nil {
						return false, fmt.Errorf("pdfparser: %w", err)
					}
					objKey, err := cos.NewObjectKey(currObjID, currGenID)
					if err != nil {
						return false, fmt.Errorf("pdfparser: %w", err)
					}
					x.xrefTrailerResolver.SetXRef(objKey, currOffset)
				}
			} else if splitString[2] != "f" {
				return false, fmt.Errorf("pdfparser: Corrupt XRefTable Entry - ObjID:%d", currObjID)
			}
			currObjID++
			if err := x.parser.SkipSpaces(); err != nil {
				return false, err
			}
		}
		if err := x.parser.SkipSpaces(); err != nil {
			return false, err
		}
		isDigit, err := isDigitAt(x.source)
		if err != nil {
			return false, err
		}
		if !isDigit {
			break
		}
	}
	return true, nil
}

// isDigitByte reports whether the byte is an ASCII digit.
func isDigitByte(c int) bool { return c >= '0' && c <= '9' }
