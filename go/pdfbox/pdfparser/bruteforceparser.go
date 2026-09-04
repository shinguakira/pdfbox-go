package pdfparser

import (
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// The markers the brute force search looks for.
var (
	bfXrefTable     = []byte("xref")
	bfXrefStream    = []byte("/XRef")
	bfEOFMarker     = []byte("%%EOF")
	bfObjMarker     = []byte("obj")
	bfTrailerMarker = []byte("trailer")
	bfObjStream     = []byte("/ObjStm")
)

// BruteForceParser finds the objects of a file by scanning it, for a file whose
// cross-reference table is missing or wrong.
//
// Port of org.apache.pdfbox.pdfparser.BruteForceParser.
type BruteForceParser struct {
	bfSearchCOSObjectKeyOffsets XrefEntries
	bfSearchTriggered           bool

	parser   *FileParser
	document *cos.Document
	source   pdfio.RandomAccessRead
}

// NewBruteForceParser returns a recovery parser over the given file.
func NewBruteForceParser(cosDocument *cos.Document, fileParser *FileParser) *BruteForceParser {
	return &BruteForceParser{
		bfSearchCOSObjectKeyOffsets: NewXrefEntries(),
		document:                    cosDocument,
		parser:                      fileParser,
		source:                      fileParser.Source(),
	}
}

// BFSearchTriggered reports whether the scan has already run.
func (b *BruteForceParser) BFSearchTriggered() bool { return b.bfSearchTriggered }

// BFCOSObjectOffsets returns where the scan found each object, running the scan
// the first time it is asked for.
func (b *BruteForceParser) BFCOSObjectOffsets() (XrefEntries, error) {
	if !b.bfSearchTriggered {
		b.bfSearchTriggered = true
		if err := b.bfSearchForObjects(); err != nil {
			return nil, err
		}
	}
	return b.bfSearchCOSObjectKeyOffsets, nil
}

// bfSearchForObjects scans the file for "N G obj" headers.
func (b *BruteForceParser) bfSearchForObjects() error {
	lastEOFMarker, err := b.bfSearchForLastEOFMarker()
	if err != nil {
		return err
	}
	originOffset, err := b.source.Position()
	if err != nil {
		return err
	}
	currentOffset := int64(minimumSearchOffset)
	lastObjectId := int64(minInt64)
	lastGenID := minInt32
	lastObjOffset := int64(minInt64)
	endobjString := []byte("ndo")
	endobjRemainingString := []byte("bj")
	endOfObjFound := false

	for {
		if _, err := b.source.Seek(currentOffset, 0); err != nil {
			return err
		}
		nextChar, err := b.readByteAt()
		if err != nil {
			return err
		}
		currentOffset++

		if IsWhitespace(nextChar) {
			isObj, err := b.parser.isString(bfObjMarker)
			if err != nil {
				return err
			}
			if isObj {
				tempOffset := currentOffset - 2
				if _, err := b.source.Seek(tempOffset, 0); err != nil {
					return err
				}
				genID, err := b.peek()
				if err != nil {
					return err
				}
				// is the next char a digit?
				if IsDigit(genID) {
					genID -= 48
					tempOffset--
					if _, err := b.source.Seek(tempOffset, 0); err != nil {
						return err
					}
					isWhitespace, err := b.isWhitespaceAt()
					if err != nil {
						return err
					}
					if isWhitespace {
						for tempOffset > minimumSearchOffset {
							ws, err := b.isWhitespaceAt()
							if err != nil {
								return err
							}
							if !ws {
								break
							}
							tempOffset--
							if _, err := b.source.Seek(tempOffset, 0); err != nil {
								return err
							}
						}
						objectIDFound := false
						for tempOffset > minimumSearchOffset {
							digit, err := isDigitAt(b.source)
							if err != nil {
								return err
							}
							if !digit {
								break
							}
							tempOffset--
							if _, err := b.source.Seek(tempOffset, 0); err != nil {
								return err
							}
							objectIDFound = true
						}
						if objectIDFound {
							if _, err := b.readByteAt(); err != nil {
								return err
							}
							objectId, err := b.parser.ReadObjectNumber()
							if err != nil {
								return err
							}
							if lastObjOffset > 0 {
								// add the former object ID only if there was a
								// subsequent object ID
								key, err := cos.NewObjectKey(lastObjectId, lastGenID)
								if err == nil {
									b.bfSearchCOSObjectKeyOffsets.Put(key, lastObjOffset)
								}
							}
							lastObjectId = objectId
							lastGenID = genID
							lastObjOffset = tempOffset + 1
							currentOffset += int64(len(bfObjMarker)) - 1
							endOfObjFound = false
						}
					}
				}
			}
		} else if nextChar == 'e' {
			// check for "endo" as abbreviation for "endobj", as the pdf may be
			// cut off in the middle of the keyword, see PDFBOX-3936.
			// We could possibly implement a more intelligent algorithm if
			// necessary
			isEndo, err := b.parser.isString(endobjString)
			if err != nil {
				return err
			}
			if isEndo {
				currentOffset += int64(len(endobjString))
				if _, err := b.source.Seek(currentOffset, 0); err != nil {
					return err
				}
				isEOF, err := b.parser.IsEOF()
				if err != nil {
					return err
				}
				if isEOF {
					endOfObjFound = true
				} else {
					isRest, err := b.parser.isString(endobjRemainingString)
					if err != nil {
						return err
					}
					if isRest {
						currentOffset += int64(len(endobjRemainingString))
						endOfObjFound = true
					}
				}
			}
		}

		isEOF, err := b.parser.IsEOF()
		if err != nil {
			return err
		}
		if currentOffset >= lastEOFMarker || isEOF {
			break
		}
	}

	if (lastEOFMarker < maxInt64 || endOfObjFound) && lastObjOffset > 0 {
		// if the pdf wasn't cut off in the middle or if the last object ends
		// with a "endobj" marker the last object id has to be added here so
		// that it can't get lost as there isn't any subsequent object id
		key, err := cos.NewObjectKey(lastObjectId, lastGenID)
		if err == nil {
			b.bfSearchCOSObjectKeyOffsets.Put(key, lastObjOffset)
		}
	}
	// reestablish origin position
	_, err = b.source.Seek(originOffset, 0)
	return err
}

// The sentinels Java starts its running values at.
const (
	minInt64 = -1 << 63
	maxInt64 = 1<<63 - 1
	minInt32 = -1 << 31
)

// BFSearchForXRef looks for the cross-reference table nearest the given offset.
func (b *BruteForceParser) BFSearchForXRef(xrefOffset int64) (int64, error) {
	newOffset := int64(-1)
	// initialize bfSearchXRefTablesOffsets -> not null
	bfSearchXRefTablesOffsets, err := b.bfSearchForXRefTables()
	if err != nil {
		return 0, err
	}
	// initialize bfSearchXRefStreamsOffsets -> not null
	bfSearchXRefStreamsOffsets, err := b.bfSearchForXRefStreams()
	if err != nil {
		return 0, err
	}

	// TODO to be optimized, this won't work in every case
	newOffsetTable := searchNearestValue(bfSearchXRefTablesOffsets, xrefOffset)
	// TODO to be optimized, this won't work in every case
	newOffsetStream := searchNearestValue(bfSearchXRefStreamsOffsets, xrefOffset)

	// choose the nearest value
	switch {
	case newOffsetTable > -1 && newOffsetStream > -1:
		differenceTable := xrefOffset - newOffsetTable
		differenceStream := xrefOffset - newOffsetStream
		if abs64(differenceTable) > abs64(differenceStream) {
			newOffset = newOffsetStream
		} else {
			newOffset = newOffsetTable
		}
	case newOffsetTable > -1:
		newOffset = newOffsetTable
	case newOffsetStream > -1:
		newOffset = newOffsetStream
	}
	// Java removes the chosen offset from its list so that a second call
	// returns the next nearest; the lists are rebuilt on every call here, so
	// there is nothing to remove from.
	return newOffset, nil
}

// searchNearestValue returns the value of the list nearest the given offset, or
// -1 where the list is empty.
func searchNearestValue(values []int64, offset int64) int64 {
	newValue := int64(-1)
	var currentDifference int64
	haveDifference := false
	currentOffsetIndex := -1

	// find the nearest value
	for i, value := range values {
		newDifference := offset - value
		// find the nearest offset
		if !haveDifference || abs64(currentDifference) > abs64(newDifference) {
			currentDifference = newDifference
			haveDifference = true
			currentOffsetIndex = i
		}
	}
	if currentOffsetIndex > -1 {
		newValue = values[currentOffsetIndex]
	}
	return newValue
}

// BFSearchForObjStreams finds every object stream by scanning, and records the
// objects each of them holds.
func (b *BruteForceParser) BFSearchForObjStreams(xrefTable XrefEntries) error {
	// save origin offset
	originOffset, err := b.source.Position()
	if err != nil {
		return err
	}
	bfSearchForObjStreamOffsets, err := b.bfSearchForObjStreamOffsets()
	if err != nil {
		return err
	}
	bfCOSObjectOffsets, err := b.BFCOSObjectOffsets()
	if err != nil {
		return err
	}

	// collect all stream offsets, warning about the ones that are incomplete
	var objStreamOffsets []int64
	for offset, key := range bfSearchForObjStreamOffsets {
		recorded, ok := bfCOSObjectOffsets.Get(key)
		if !ok {
			// log warning about skipped stream
			slog.Warn("Skipped incomplete object stream", "object", key, "offset", offset)
			continue
		}
		if recorded == offset {
			objStreamOffsets = append(objStreamOffsets, offset)
		}
	}

	// add all found compressed objects to the brute force search result
	for _, offset := range objStreamOffsets {
		if _, err := b.source.Seek(offset, 0); err != nil {
			return err
		}
		stmObjNumber, err := b.parser.ReadObjectNumber()
		if err != nil {
			return err
		}
		if _, err := b.parser.ReadGenerationNumber(); err != nil {
			return err
		}
		if err := b.parser.ReadExpectedString(string(bfObjMarker), true); err != nil {
			return err
		}

		// Java swallows an IOException here and skips the corrupt stream.
		objectNumbers, err := func() (map[int64]int, error) {
			dict, err := b.parser.ParseCOSDictionary(false)
			if err != nil {
				return nil, err
			}
			stream, err := b.parser.ParseCOSStream(dict)
			if err != nil {
				return nil, err
			}
			defer stream.Close()
			objStreamParser, err := NewObjectStreamParser(stream, b.document)
			if err != nil {
				return nil, err
			}
			return objStreamParser.ReadObjectNumbers()
		}()
		if err != nil {
			continue
		}

		for objNumber := range objectNumbers {
			objKey, err := cos.NewObjectKey(objNumber, 0)
			if err != nil {
				continue
			}
			existingOffset, exists := bfCOSObjectOffsets.Get(objKey)
			if exists && existingOffset < 0 {
				// translate stream object key to its offset
				objStmKey, err := cos.NewObjectKey(abs64(existingOffset), 0)
				if err != nil {
					continue
				}
				existingOffset, exists = bfCOSObjectOffsets.Get(objStmKey)
			}
			if !exists || offset > existingOffset {
				bfCOSObjectOffsets.Put(objKey, -stmObjNumber)
				xrefTable.Put(objKey, -stmObjNumber)
			}
		}
	}
	// restore origin offset
	_, err = b.source.Seek(originOffset, 0)
	return err
}

// bfSearchForTrailer looks for a trailer dictionary naming a usable catalogue
// and document information dictionary.
func (b *BruteForceParser) bfSearchForTrailer(trailer *cos.Dictionary) (bool, error) {
	originOffset, err := b.source.Position()
	if err != nil {
		return false, err
	}
	if _, err := b.source.Seek(minimumSearchOffset, 0); err != nil {
		return false, err
	}
	// search for trailer marker
	trailerOffset, err := b.findString(bfTrailerMarker)
	if err != nil {
		return false, err
	}
	for trailerOffset != -1 {
		// Java swallows an IOException here and tries the next marker.
		done, err := func() (bool, error) {
			rootFound := false
			infoFound := false
			if err := b.parser.SkipSpaces(); err != nil {
				return false, err
			}
			trailerDict, err := b.parser.ParseCOSDictionary(true)
			if err != nil {
				return false, err
			}
			rootObj := trailerDict.GetCOSObject(cos.Root)
			if rootObj != nil {
				// check if the dictionary can be dereferenced and is the one we
				// are looking for
				if rootDict, ok := rootObj.Object().(*cos.Dictionary); ok && isCatalog(rootDict) {
					rootFound = true
				}
			}
			infoObj := trailerDict.GetCOSObject(cos.Info)
			if infoObj != nil {
				// check if the dictionary can be dereferenced and is the one we
				// are looking for
				if infoDict, ok := infoObj.Object().(*cos.Dictionary); ok && isInfo(infoDict) {
					infoFound = true
				}
			}
			if !rootFound || !infoFound {
				return false, nil
			}
			trailer.SetItem(cos.Root, rootObj)
			trailer.SetItem(cos.Info, infoObj)
			encObj := trailerDict.GetCOSObject(cos.Encrypt)
			// check if the dictionary can be dereferenced
			// TODO check if the dictionary is an encryption dictionary?
			if encObj != nil {
				if _, ok := encObj.Object().(*cos.Dictionary); ok {
					trailer.SetItem(cos.Encrypt, encObj)
				}
			}
			if idObj, ok := trailerDict.GetItem(cos.ID).(*cos.Array); ok {
				trailer.SetItem(cos.ID, idObj)
			}
			return true, nil
		}()
		if err == nil && done {
			return true, nil
		}
		trailerOffset, err = b.findString(bfTrailerMarker)
		if err != nil {
			return false, err
		}
	}
	if _, err := b.source.Seek(originOffset, 0); err != nil {
		return false, err
	}
	return false, nil
}

// searchForTrailerItems picks the catalogue and information dictionary out of
// the objects the scan found.
func (b *BruteForceParser) searchForTrailerItems(trailer *cos.Dictionary) (bool, error) {
	var rootObject *cos.Object
	var infoObject *cos.Object

	offsets, err := b.BFCOSObjectOffsets()
	if err != nil {
		return false, err
	}
	for _, entry := range offsets {
		cosObject := b.document.ObjectFromPool(entry.Key)
		dictionary, ok := cosObject.Object().(*cos.Dictionary)
		if !ok {
			continue
		}
		switch {
		case isCatalog(dictionary):
			// document catalog
			rootObject = b.compareCOSObjects(cosObject, entry.Offset, rootObject)
		case isInfo(dictionary):
			// info dictionary
			infoObject = b.compareCOSObjects(cosObject, entry.Offset, infoObject)
		}
		// encryption dictionary, if existing, is lost
		// We can't run "Algorithm 2" from PDF specification because of missing ID
	}
	if rootObject != nil {
		trailer.SetItem(cos.Root, rootObject)
	}
	if infoObject != nil {
		trailer.SetItem(cos.Info, infoObject)
	}
	return rootObject != nil, nil
}

// compareCOSObjects picks the newer of two candidates for the same trailer
// entry.
func (b *BruteForceParser) compareCOSObjects(newObject *cos.Object, newOffset int64,
	currentObject *cos.Object) *cos.Object {
	if currentObject == nil || currentObject.Key() == nil {
		return newObject
	}
	currentKey := currentObject.Key()
	newKey := newObject.Key()
	// check if the current object is an updated version of the previous found
	// object
	if currentKey.Number() == newKey.Number() {
		if currentKey.Generation() < newKey.Generation() {
			return newObject
		}
		return currentObject
	}
	// most likely the object with the bigger offset is the newer one
	currentOffset, ok := b.document.XRefOffset(currentKey)
	if ok && newOffset > currentOffset {
		return newObject
	}
	return currentObject
}

// bfSearchForLastEOFMarker finds the last %%EOF that is not followed by more
// content.
func (b *BruteForceParser) bfSearchForLastEOFMarker() (int64, error) {
	lastEOFMarker := int64(-1)
	originOffset, err := b.source.Position()
	if err != nil {
		return 0, err
	}
	if _, err := b.source.Seek(minimumSearchOffset, 0); err != nil {
		return 0, err
	}
	tempMarker, err := b.findString(bfEOFMarker)
	if err != nil {
		return 0, err
	}
	for tempMarker != -1 {
		// check if the following data is some valid pdf content
		// which most likely indicates that the pdf is linearized,
		// updated or just cut off somewhere in the middle
		err := func() error {
			if err := b.parser.SkipSpaces(); err != nil {
				return err
			}
			isXref, err := b.parser.isString(bfXrefTable)
			if err != nil {
				return err
			}
			if isXref {
				return nil
			}
			if _, err := b.parser.ReadObjectNumber(); err != nil {
				return err
			}
			_, err = b.parser.ReadGenerationNumber()
			return err
		}()
		if err != nil {
			// save the EOF marker as the following data is most likely some
			// garbage
			lastEOFMarker = tempMarker
		}
		tempMarker, err = b.findString(bfEOFMarker)
		if err != nil {
			return 0, err
		}
	}
	if _, err := b.source.Seek(originOffset, 0); err != nil {
		return 0, err
	}
	// no EOF marker found
	if lastEOFMarker == -1 {
		lastEOFMarker = maxInt64
	}
	return lastEOFMarker, nil
}

// bfSearchForObjStreamOffsets finds where each object stream begins.
func (b *BruteForceParser) bfSearchForObjStreamOffsets() (map[int64]*cos.ObjectKey, error) {
	bfSearchObjStreamsOffsets := map[int64]*cos.ObjectKey{}
	if _, err := b.source.Seek(minimumSearchOffset, 0); err != nil {
		return nil, err
	}
	objString := []byte(" obj")
	// search for object stream marker
	positionObjStream, err := b.findString(bfObjStream)
	if err != nil {
		return nil, err
	}
	for positionObjStream != -1 {
		// search backwards for the beginning of the object
		newOffset := int64(-1)
		objFound := false
		for i := int64(1); i < 40 && !objFound; i++ {
			currentOffset := positionObjStream - i*10
			if currentOffset <= 0 {
				continue
			}
			if _, err := b.source.Seek(currentOffset, 0); err != nil {
				return nil, err
			}
			for j := 0; j < 10; j++ {
				isObj, err := b.parser.isString(objString)
				if err != nil {
					return nil, err
				}
				if !isObj {
					currentOffset++
					if _, err := b.readByteAt(); err != nil {
						return nil, err
					}
					continue
				}
				tempOffset := currentOffset - 1
				if _, err := b.source.Seek(tempOffset, 0); err != nil {
					return nil, err
				}
				// is the next char a digit?
				digit, err := isDigitAt(b.source)
				if err != nil {
					return nil, err
				}
				if digit {
					tempOffset--
					if _, err := b.source.Seek(tempOffset, 0); err != nil {
						return nil, err
					}
					isSpace, err := b.isSpaceAt()
					if err != nil {
						return nil, err
					}
					if isSpace {
						length := 0
						tempOffset--
						if _, err := b.source.Seek(tempOffset, 0); err != nil {
							return nil, err
						}
						for tempOffset > minimumSearchOffset {
							digit, err := isDigitAt(b.source)
							if err != nil {
								return nil, err
							}
							if !digit {
								break
							}
							tempOffset--
							if _, err := b.source.Seek(tempOffset, 0); err != nil {
								return nil, err
							}
							length++
						}
						if length > 0 {
							if _, err := b.readByteAt(); err != nil {
								return nil, err
							}
							newOffset, err = b.source.Position()
							if err != nil {
								return nil, err
							}
							objNumber, err := b.parser.ReadObjectNumber()
							if err != nil {
								return nil, err
							}
							genNumber, err := b.parser.ReadGenerationNumber()
							if err != nil {
								return nil, err
							}
							streamObjectKey, err := cos.NewObjectKey(objNumber, genNumber)
							if err == nil {
								bfSearchObjStreamsOffsets[newOffset] = streamObjectKey
							}
						}
					}
				}
				objFound = true
				break
			}
		}
		if _, err := b.source.Seek(positionObjStream+int64(len(bfObjStream)), 0); err != nil {
			return nil, err
		}
		positionObjStream, err = b.findString(bfObjStream)
		if err != nil {
			return nil, err
		}
	}
	return bfSearchObjStreamsOffsets, nil
}

// bfSearchForXRefTables finds where each cross-reference table begins.
func (b *BruteForceParser) bfSearchForXRefTables() ([]int64, error) {
	var bfSearchXRefTablesOffsets []int64
	// a pdf may contain more than one xref entry
	if _, err := b.source.Seek(minimumSearchOffset, 0); err != nil {
		return nil, err
	}
	// search for xref tables
	newOffset, err := b.findString(bfXrefTable)
	if err != nil {
		return nil, err
	}
	for newOffset != -1 {
		if _, err := b.source.Seek(newOffset-1, 0); err != nil {
			return nil, err
		}
		// ensure that we don't read "startxref" instead of "xref"
		isWhitespace, err := b.isWhitespaceAt()
		if err != nil {
			return nil, err
		}
		if isWhitespace {
			bfSearchXRefTablesOffsets = append(bfSearchXRefTablesOffsets, newOffset)
		}
		if _, err := b.source.Seek(newOffset+4, 0); err != nil {
			return nil, err
		}
		newOffset, err = b.findString(bfXrefTable)
		if err != nil {
			return nil, err
		}
	}
	return bfSearchXRefTablesOffsets, nil
}

// bfSearchForXRefStreams finds where each cross-reference stream begins.
func (b *BruteForceParser) bfSearchForXRefStreams() ([]int64, error) {
	var bfSearchXRefStreamsOffsets []int64
	// a pdf may contain more than one /XRef entry
	if _, err := b.source.Seek(minimumSearchOffset, 0); err != nil {
		return nil, err
	}
	// search for XRef streams
	objString := []byte(" obj")
	xrefOffset, err := b.findString(bfXrefStream)
	if err != nil {
		return nil, err
	}
	for xrefOffset != -1 {
		// search backwards for the beginning of the stream
		newOffset := int64(-1)
		objFound := false
		for i := int64(1); i < 40 && !objFound; i++ {
			currentOffset := xrefOffset - i*10
			if currentOffset <= 0 {
				continue
			}
			if _, err := b.source.Seek(currentOffset, 0); err != nil {
				return nil, err
			}
			for j := 0; j < 10; j++ {
				isObj, err := b.parser.isString(objString)
				if err != nil {
					return nil, err
				}
				if !isObj {
					currentOffset++
					if _, err := b.readByteAt(); err != nil {
						return nil, err
					}
					continue
				}
				tempOffset := currentOffset - 1
				if _, err := b.source.Seek(tempOffset, 0); err != nil {
					return nil, err
				}
				// is the next char a digit?
				digit, err := isDigitAt(b.source)
				if err != nil {
					return nil, err
				}
				if digit {
					tempOffset--
					if _, err := b.source.Seek(tempOffset, 0); err != nil {
						return nil, err
					}
					isSpace, err := b.isSpaceAt()
					if err != nil {
						return nil, err
					}
					if isSpace {
						length := 0
						tempOffset--
						if _, err := b.source.Seek(tempOffset, 0); err != nil {
							return nil, err
						}
						for tempOffset > minimumSearchOffset {
							digit, err := isDigitAt(b.source)
							if err != nil {
								return nil, err
							}
							if !digit {
								break
							}
							tempOffset--
							if _, err := b.source.Seek(tempOffset, 0); err != nil {
								return nil, err
							}
							length++
						}
						if length > 0 {
							if _, err := b.readByteAt(); err != nil {
								return nil, err
							}
							newOffset, err = b.source.Position()
							if err != nil {
								return nil, err
							}
						}
					}
				}
				objFound = true
				break
			}
		}
		if newOffset > -1 {
			bfSearchXRefStreamsOffsets = append(bfSearchXRefStreamsOffsets, newOffset)
		}
		if _, err := b.source.Seek(xrefOffset+5, 0); err != nil {
			return nil, err
		}
		xrefOffset, err = b.findString(bfXrefStream)
		if err != nil {
			return nil, err
		}
	}
	return bfSearchXRefStreamsOffsets, nil
}

// isInfo reports whether the dictionary looks like a document information
// dictionary.
func isInfo(dictionary *cos.Dictionary) bool {
	if dictionary.ContainsKey(cos.Parent) || dictionary.ContainsKey(cos.A) ||
		dictionary.ContainsKey(cos.Dest) {
		return false
	}
	return dictionary.ContainsKey(cos.ModDate) || dictionary.ContainsKey(cos.Title) ||
		dictionary.ContainsKey(cos.Author) || dictionary.ContainsKey(cos.Subject) ||
		dictionary.ContainsKey(cos.Keywords) || dictionary.ContainsKey(cos.Creator) ||
		dictionary.ContainsKey(cos.Producer) || dictionary.ContainsKey(cos.CreationDate)
}

// isCatalog reports whether the dictionary looks like a document catalogue.
func isCatalog(dictionary *cos.Dictionary) bool {
	return cos.Catalog.Equals(dictionary.GetCOSName(cos.Type)) ||
		dictionary.ContainsKey(cos.FDF)
}

// findString scans forward for the given bytes and returns where they start, or
// -1.
func (b *BruteForceParser) findString(str []byte) (int64, error) {
	position := int64(-1)
	stringLength := len(str)
	counter := 0
	readChar, err := b.readByteAt()
	if err != nil {
		return 0, err
	}
	for readChar != -1 {
		if readChar == int(str[counter]) {
			if counter == 0 {
				pos, err := b.source.Position()
				if err != nil {
					return 0, err
				}
				position = pos - 1
			}
			counter++
			if counter == stringLength {
				return position, nil
			}
		} else if counter > 0 {
			counter = 0
			position = -1
			continue
		}
		readChar, err = b.readByteAt()
		if err != nil {
			return 0, err
		}
	}
	return position, nil
}

// RebuildTrailer rebuilds the trailer from the objects the scan found.
func (b *BruteForceParser) RebuildTrailer(xrefTable XrefEntries) (*cos.Dictionary, error) {
	// use a new trailer resolver
	trailerResolver := NewXrefTrailerResolver()
	// use the found objects to rebuild the trailer resolver
	trailerResolver.NextXrefObj(0, XRefTypeTable)
	offsets, err := b.BFCOSObjectOffsets()
	if err != nil {
		return nil, err
	}
	for _, entry := range offsets {
		trailerResolver.SetXRef(entry.Key, entry.Offset)
	}
	trailerResolver.SetStartxref(0)

	// transfer xref-table to document
	resolved := FromKeyed(trailerResolver.XrefTable())
	b.document.ClearXRefTable()
	b.document.AddXRefTable(resolved.ToKeyed())

	// remember the highest XRef object number to avoid it being reused in
	// incremental saving
	var maxValue int64
	for _, entry := range resolved {
		if entry.Key.Number() > maxValue {
			maxValue = entry.Key.Number()
		}
	}
	b.document.SetHighestXRefObjectNumber(maxValue)

	trailer := trailerResolver.Trailer()
	b.document.SetTrailer(trailer)
	xrefTable.PutAll(resolved)

	searchForObjStreamsDone := false
	foundTrailer, err := b.bfSearchForTrailer(trailer)
	if err != nil {
		return nil, err
	}
	if !foundTrailer {
		foundItems, err := b.searchForTrailerItems(trailer)
		if err != nil {
			return nil, err
		}
		if !foundItems {
			// root entry wasn't found, maybe it is part of an object stream
			// brute force search for all object streams.
			if err := b.BFSearchForObjStreams(xrefTable); err != nil {
				return nil, err
			}
			searchForObjStreamsDone = true
			// search again for the root entry
			if _, err := b.searchForTrailerItems(trailer); err != nil {
				return nil, err
			}
		}
	}

	// prepare decryption if necessary
	if err := b.parser.prepareDecryption(); err != nil {
		return nil, err
	}

	if !searchForObjStreamsDone {
		// brute force search for all object streams.
		if err := b.BFSearchForObjStreams(xrefTable); err != nil {
			return nil, err
		}
	}
	return trailer, nil
}

// readByteAt reads one byte, returning -1 at the end of the file.
func (b *BruteForceParser) readByteAt() (int, error) {
	buf := make([]byte, 1)
	n, _ := b.source.Read(buf)
	if n < 1 {
		return -1, nil
	}
	return int(buf[0]), nil
}

// peek returns the next byte without consuming it.
func (b *BruteForceParser) peek() (int, error) {
	position, err := b.source.Position()
	if err != nil {
		return -1, err
	}
	c, err := b.readByteAt()
	if err != nil {
		return -1, err
	}
	if _, err := b.source.Seek(position, 0); err != nil {
		return -1, err
	}
	return c, nil
}

// isWhitespaceAt reports whether the byte under the cursor is whitespace.
func (b *BruteForceParser) isWhitespaceAt() (bool, error) {
	c, err := b.peek()
	if err != nil {
		return false, err
	}
	return IsWhitespace(c), nil
}

// isSpaceAt reports whether the byte under the cursor is a space.
func (b *BruteForceParser) isSpaceAt() (bool, error) {
	c, err := b.peek()
	if err != nil {
		return false, err
	}
	return c == ' ', nil
}
