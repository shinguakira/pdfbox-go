package pdfparser

import (
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"strconv"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/encryption"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// The markers the file-level parse looks for.
const (
	pdfHeader         = "%PDF-"
	fdfHeader         = "%FDF-"
	pdfDefaultVersion = "1.4"
	fdfDefaultVersion = "1.0"

	streamString = "stream"
)

var (
	startxrefMarker = []byte("startxref")
	eofMarker       = []byte("%%EOF")
)

// defaultTrailByteCount is how far back from the end of the file the startxref
// offset is looked for.
const defaultTrailByteCount = 2048

// FileParser reads the objects of a PDF file: the header, the trailer, the
// cross-reference table, and every object reached through it.
//
// Port of the file half of org.apache.pdfbox.pdfparser.COSParser. The object
// half is ObjectParser and the stream half is StreamParser, both of which this
// embeds through StreamParser.
//
// Encryption is not ported: `pdmodel/encryption` is a package this port has not
// reached, so an encrypted document is reported rather than decrypted. See
// migration/STATUS.md.
type FileParser struct {
	*StreamParser

	// xrefTable is what the cross-reference table said, kept alongside the
	// document's own copy because the brute force parser adds to it.
	xrefTable XrefEntries

	// decompressedObjects holds the objects read out of each object stream, so
	// that a stream is expanded once however many of its objects are asked for.
	decompressedObjects map[int64]map[int64]cos.Base

	initialParseDone  bool
	trailerWasRebuild bool
	bruteForceParser  *BruteForceParser

	// The decryption material and what prepareDecryption makes of it.
	password         string
	keyStoreInput    io.Reader
	keyAlias         string
	encryption       *encryption.PDEncryption
	securityHandler  encryption.SecurityHandler
	accessPermission *encryption.AccessPermission

	readTrailBytes int
}

var _ cos.Parser = (*FileParser)(nil)

// NewFileParser returns a parser over the given file, together with the
// document it fills.
func NewFileParser(source pdfio.RandomAccessRead, cache pdfio.StreamCache, codecs cos.CodecProvider) (*FileParser, error) {
	return NewFileParserWithPassword(source, "", nil, "", cache, codecs)
}

// NewFileParserWithPassword returns a parser over the given file, which it
// decrypts with the given password, or with the given PKCS#12 keystore and
// alias where the document uses public key security.
//
// Port of the COSParser constructor that takes all four.
func NewFileParserWithPassword(source pdfio.RandomAccessRead, password string,
	keyStore io.Reader, keyAlias string, cache pdfio.StreamCache,
	codecs cos.CodecProvider) (*FileParser, error) {
	length, err := source.Length()
	if err != nil {
		return nil, err
	}
	p := &FileParser{
		StreamParser: &StreamParser{
			fileLen:   length,
			isLenient: true,
			strmBuf:   make([]byte, strmBufLen),
		},
		xrefTable:           NewXrefEntries(),
		decompressedObjects: map[int64]map[int64]cos.Base{},
		readTrailBytes:      defaultTrailByteCount,
		password:            password,
		keyStoreInput:       keyStore,
		keyAlias:            keyAlias,
	}
	// The document needs the parser as its cos.Parser and the parser needs the
	// document as its object pool, so the document is built here rather than by
	// NewStreamParser -- it has to hold this parser, not the stream half.
	document := cos.NewDocumentWithCache(cache, codecs, p)
	p.ObjectParser = NewObjectParser(source, document)
	return p, nil
}

// SetEOFLookupRange sets how far back from the end of the file the startxref
// offset is looked for.
func (p *FileParser) SetEOFLookupRange(byteCount int) {
	if byteCount > 15 {
		p.readTrailBytes = byteCount
	}
}

// XrefTable returns what the cross-reference table said.
func (p *FileParser) XrefTable() XrefEntries { return p.xrefTable }

// TrailerWasRebuild reports whether the trailer had to be rebuilt by brute
// force.
func (p *FileParser) TrailerWasRebuild() bool { return p.trailerWasRebuild }

// RetrieveTrailer reads the trailer, rebuilding it by brute force where the
// cross-reference table is missing or broken and the parser is lenient.
func (p *FileParser) RetrieveTrailer() (*cos.Dictionary, error) {
	var trailer *cos.Dictionary
	rebuildTrailer := false

	// parse startxref
	// TODO FDF files don't have a startxref value, so that rebuildTrailer is
	// triggered
	startXRefOffset, err := p.startxrefOffset()
	if err == nil {
		if startXRefOffset > -1 {
			xrefParser := NewXrefParser(p)
			trailer, err = xrefParser.ParseXref(p.Document(), startXRefOffset)
			if err == nil {
				p.xrefTable.PutAll(xrefParser.XrefTable())
			}
		} else {
			rebuildTrailer = p.IsLenient()
		}
	}
	if err != nil {
		if !p.IsLenient() {
			return nil, err
		}
		rebuildTrailer = true
	}

	// check if the trailer contains a Root object
	if trailer != nil && trailer.GetItem(cos.Root) == nil {
		rebuildTrailer = p.IsLenient()
	}

	if rebuildTrailer {
		// reset cross reference table
		p.xrefTable.Clear()
		bf, err := p.BruteForceParser()
		if err != nil {
			return nil, err
		}
		trailer, err = bf.RebuildTrailer(p.xrefTable)
		if err != nil {
			return nil, err
		}
		p.trailerWasRebuild = true
		return trailer, nil
	}

	// prepare decryption if necessary
	if err := p.prepareDecryption(); err != nil {
		return nil, err
	}
	// don't use the getter as it creates an instance of BruteForceParser
	if p.bruteForceParser != nil && p.bruteForceParser.BFSearchTriggered() {
		if err := p.bruteForceParser.BFSearchForObjStreams(p.xrefTable); err != nil {
			return nil, err
		}
	}
	return trailer, nil
}

// startxrefOffset returns where the last startxref keyword is, or -1 where the
// file has none and the parser is lenient.
func (p *FileParser) startxrefOffset() (int64, error) {
	fileLen, err := p.Source().Length()
	if err != nil {
		return 0, err
	}

	// read trailing bytes into buffer
	trailByteCount := p.readTrailBytes
	if fileLen < int64(trailByteCount) {
		trailByteCount = int(fileLen)
	}
	buf := make([]byte, trailByteCount)
	skipBytes := fileLen - int64(trailByteCount)
	readErr := func() error {
		if _, err := p.Source().Seek(skipBytes, 0); err != nil {
			return err
		}
		off := 0
		for off < trailByteCount {
			readBytes, err := p.Source().Read(buf[off:trailByteCount])
			if err != nil && readBytes < 1 {
				// in order to not get stuck in a loop we check readBytes (this
				// should never happen)
				return fmt.Errorf("pdfparser: No more bytes to read for trailing buffer, but expected: %d",
					trailByteCount-off)
			}
			if readBytes < 1 {
				return fmt.Errorf("pdfparser: No more bytes to read for trailing buffer, but expected: %d",
					trailByteCount-off)
			}
			off += readBytes
		}
		return nil
	}()
	if _, err := p.Source().Seek(0, 0); err != nil {
		return 0, err
	}
	if readErr != nil {
		return 0, readErr
	}

	// find last '%%EOF'
	bufOff := lastIndexOfPattern(eofMarker, buf, len(buf))
	if bufOff < 0 {
		if p.IsLenient() {
			// in lenient mode the '%%EOF' isn't needed
			bufOff = len(buf)
		} else {
			return 0, fmt.Errorf("pdfparser: Missing end of file marker '%s'", eofMarker)
		}
	}
	// find last startxref preceding EOF marker
	bufOff = lastIndexOfPattern(startxrefMarker, buf, bufOff)
	if bufOff < 0 {
		return 0, fmt.Errorf("pdfparser: Missing 'startxref' marker.")
	}
	return skipBytes + int64(bufOff), nil
}

// lastIndexOfPattern returns where the pattern last starts in buf before
// endOff, or -1.
func lastIndexOfPattern(pattern, buf []byte, endOff int) int {
	lastPatternChOff := len(pattern) - 1
	bufOff := endOff
	patOff := lastPatternChOff
	lookupCh := pattern[patOff]
	for {
		bufOff--
		if bufOff < 0 {
			break
		}
		if bufOff < len(buf) && buf[bufOff] == lookupCh {
			patOff--
			if patOff < 0 {
				// whole pattern matched
				return bufOff
			}
			// matched current char, advance to preceding one
			lookupCh = pattern[patOff]
		} else if patOff < lastPatternChOff {
			// no char match but already matched some chars; reset
			patOff = lastPatternChOff
			lookupCh = pattern[patOff]
		}
	}
	return -1
}

// DereferenceObject resolves an indirect reference, leaving the cursor where it
// found it.
func (p *FileParser) DereferenceObject(obj *cos.Object) (cos.Base, error) {
	currentPos, err := p.Source().Position()
	if err != nil {
		return nil, err
	}
	key := obj.Key()
	parsedObj, err := p.parseObjectDynamically(key, false)
	if err != nil {
		return nil, err
	}
	if parsedObj != nil {
		parsedObj.SetDirect(false)
		parsedObj.SetKey(key)
	}
	if currentPos > 0 {
		if _, err := p.Source().Seek(currentPos, 0); err != nil {
			return nil, err
		}
	}
	return parsedObj, nil
}

// parseObjectDynamically reads the object the key names, from wherever the
// cross-reference table says it is.
func (p *FileParser) parseObjectDynamically(objKey *cos.ObjectKey, requireExistingNotCompressedObj bool) (cos.Base, error) {
	pdfObject := p.Document().ObjectFromPool(objKey)
	if !pdfObject.IsObjectNull() {
		return pdfObject.Object(), nil
	}

	offsetOrObjstmObNr, found, err := p.objectOffset(objKey, requireExistingNotCompressedObj)
	if err != nil {
		return nil, err
	}

	var referencedObject cos.Base
	if found {
		if offsetOrObjstmObNr > 0 {
			referencedObject, err = p.parseFileObject(offsetOrObjstmObNr, objKey)
		} else {
			// xref value is object nr of object stream containing object to be
			// parsed since our object was not found it means object stream was
			// not parsed so far
			referencedObject, err = p.ParseObjectStreamObject(-offsetOrObjstmObNr, objKey)
		}
		if err != nil {
			return nil, err
		}
	}

	if referencedObject == nil || referencedObject == cos.Base(cos.NullObject) {
		// not defined object -> NULL object (Spec. 1.7, chap. 3.2.9)
		// or some other issue with dereferencing
		// remove parser to avoid endless recursion
		pdfObject.SetToNull()
	}
	return referencedObject, nil
}

// objectOffset returns where the object is, negative where it is inside an
// object stream. The second result is false where nothing knows.
func (p *FileParser) objectOffset(objKey *cos.ObjectKey, requireExistingNotCompressedObj bool) (int64, bool, error) {
	// read offset or object stream object number from xref table
	offsetOrObjstmObNr, found := p.Document().XRefOffset(objKey)

	// maybe something is wrong with the xref table -> perform brute force
	// search for all objects
	if !found && p.IsLenient() {
		bf, err := p.BruteForceParser()
		if err != nil {
			return 0, false, err
		}
		offsets, err := bf.BFCOSObjectOffsets()
		if err != nil {
			return 0, false, err
		}
		offsetOrObjstmObNr, found = offsets.Get(objKey)
		if found {
			p.Document().PutXRefOffset(objKey, offsetOrObjstmObNr)
		}
	}

	// test to circumvent loops with broken documents
	if requireExistingNotCompressedObj && (!found || offsetOrObjstmObNr <= 0) {
		return 0, false, fmt.Errorf("pdfparser: Object must be defined and must not be compressed object: %d:%d",
			objKey.Number(), objKey.Generation())
	}
	return offsetOrObjstmObNr, found, nil
}

// parseFileObject reads the object written out at the given offset.
func (p *FileParser) parseFileObject(objOffset int64, objKey *cos.ObjectKey) (cos.Base, error) {
	// jump to the object start
	if _, err := p.Source().Seek(objOffset, 0); err != nil {
		return nil, err
	}

	// an indirect object starts with the object number/generation number
	readObjNr, err := p.ReadObjectNumber()
	if err != nil {
		return nil, err
	}
	readObjGen, err := p.ReadGenerationNumber()
	if err != nil {
		return nil, err
	}
	if err := p.ReadObjectMarker(); err != nil {
		return nil, err
	}

	// consistency check
	if readObjNr != objKey.Number() || readObjGen != objKey.Generation() {
		return nil, fmt.Errorf("pdfparser: XREF for %d:%d points to wrong object: %d:%d at offset %d",
			objKey.Number(), objKey.Generation(), readObjNr, readObjGen, objOffset)
	}

	if err := p.SkipSpaces(); err != nil {
		return nil, err
	}
	parsedObject, err := p.ParseDirObject()
	if err != nil {
		return nil, err
	}
	if parsedObject != nil {
		parsedObject.SetDirect(false)
		parsedObject.SetKey(objKey)
	}

	endObjectKey, err := p.ReadString()
	if err != nil {
		return nil, err
	}
	if endObjectKey == streamString {
		if err := p.rewind(int64(len(endObjectKey))); err != nil {
			return nil, err
		}
		dictionary, ok := parsedObject.(*cos.Dictionary)
		if !ok {
			// this is not legal
			// the combination of a dict and the stream/endstream
			// forms a complete stream object
			return nil, fmt.Errorf("pdfparser: Stream not preceded by dictionary (offset: %d).", objOffset)
		}
		stream, err := p.ParseCOSStream(dictionary)
		if err != nil {
			return nil, err
		}

		if p.securityHandler != nil {
			if err := p.securityHandler.DecryptStream(stream, objKey.Number(),
				int64(objKey.Generation())); err != nil {
				return nil, err
			}
		}
		parsedObject = stream

		if err := p.SkipSpaces(); err != nil {
			return nil, err
		}
		endObjectKey, err = p.ReadLine()
		if err != nil {
			return nil, err
		}
		// we have case with a second 'endstream' before endobj
		if !strings.HasPrefix(endObjectKey, endobjString) &&
			strings.HasPrefix(endObjectKey, endstreamString) {
			endObjectKey = strings.TrimSpace(endObjectKey[9:])
			if endObjectKey == "" {
				// no other characters in extra endstream line
				// read next line
				endObjectKey, err = p.ReadLine()
				if err != nil {
					return nil, err
				}
			}
		}
	} else if p.securityHandler != nil {
		decrypted, err := p.securityHandler.Decrypt(parsedObject, objKey.Number(),
			int64(objKey.Generation()))
		if err != nil {
			return nil, err
		}
		parsedObject = decrypted
		parsedObject.SetKey(objKey)
	}

	if !strings.HasPrefix(endObjectKey, endobjString) {
		if p.IsLenient() {
			slog.Warn("Object does not end with 'endobj'",
				"object", fmt.Sprintf("%d:%d", readObjNr, readObjGen),
				"offset", objOffset, "found", endObjectKey)
		} else {
			return nil, fmt.Errorf("pdfparser: Object (%d:%d) at offset %d does not end with 'endobj' but with '%s'",
				readObjNr, readObjGen, objOffset, endObjectKey)
		}
	}
	return parsedObject, nil
}

// ParseObjectStreamObject reads one object out of an object stream, expanding
// the stream the first time any of its objects is asked for.
func (p *FileParser) ParseObjectStreamObject(objstmObjNr int64, key *cos.ObjectKey) (cos.Base, error) {
	streamObjects, ok := p.decompressedObjects[objstmObjNr]
	if !ok {
		streamObjects = map[int64]cos.Base{}
		p.decompressedObjects[objstmObjNr] = streamObjects
	}

	// did we already read the compressed object stream?
	if objectStreamObject, ok := streamObjects[key.InternalHash()]; ok {
		delete(streamObjects, key.InternalHash())
		return objectStreamObject, nil
	}

	objKey, err := p.ObjectKey(objstmObjNr, 0)
	if err != nil {
		return nil, err
	}
	objstmBaseObj := p.Document().ObjectFromPool(objKey).Object()
	objstm, ok := objstmBaseObj.(*cos.Stream)
	if !ok {
		return nil, nil
	}

	parser, err := NewObjectStreamParser(objstm, p.Document())
	if err == nil {
		var allStreamObjects map[int64]cos.Base
		allStreamObjects, err = parser.ParseAllObjects()
		if err == nil {
			objectStreamObject := allStreamObjects[key.InternalHash()]
			delete(allStreamObjects, key.InternalHash())
			for k, v := range allStreamObjects {
				if _, exists := streamObjects[k]; !exists {
					streamObjects[k] = v
				}
			}
			return objectStreamObject, nil
		}
	}
	if !p.IsLenient() {
		return nil, err
	}
	slog.Error("object stream could not be parsed", "object", objstmObjNr, "err", err)
	return nil, nil
}

// BruteForceParser returns the recovery parser, building it the first time it
// is needed.
func (p *FileParser) BruteForceParser() (*BruteForceParser, error) {
	if p.bruteForceParser == nil {
		p.bruteForceParser = NewBruteForceParser(p.Document(), p)
	}
	return p.bruteForceParser, nil
}

// CheckPages checks that the page tree of the document is usable, repairing the
// counts where the trailer had to be rebuilt.
func (p *FileParser) CheckPages(root *cos.Dictionary) error {
	if p.trailerWasRebuild {
		// check if all page objects are dereferenced
		if pages := root.GetCOSDictionary(cos.Pages); pages != nil {
			p.checkPagesDictionary(pages, map[*cos.Object]bool{})
		}
	}
	if root.GetCOSDictionary(cos.Pages) == nil {
		return fmt.Errorf("pdfparser: Page tree root must be a dictionary")
	}
	return nil
}

// checkPagesDictionary drops the kids that did not dereference and fixes the
// count, returning how many pages the node really has.
func (p *FileParser) checkPagesDictionary(pagesDict *cos.Dictionary, set map[*cos.Object]bool) int {
	// check for kids
	kidsArray := pagesDict.GetCOSArray(cos.Kids)
	numberOfPages := 0
	if kidsArray != nil {
		kidsList := kidsArray.ToList()
		for _, kid := range kidsList {
			kidObject, ok := kid.(*cos.Object)
			if !ok || set[kidObject] {
				kidsArray.Remove(kid)
				continue
			}
			kidBaseobject := kidObject.Object()
			// object wasn't dereferenced -> remove it
			if kidBaseobject == nil || kidBaseobject == cos.Base(cos.NullObject) {
				slog.Warn("Removed null object from pages dictionary")
				kidsArray.Remove(kid)
				continue
			}
			kidDictionary, ok := kidBaseobject.(*cos.Dictionary)
			if !ok {
				continue
			}
			switch kidDictionary.GetCOSName(cos.Type) {
			case cos.Pages:
				// process nested pages dictionaries
				set[kidObject] = true
				numberOfPages += p.checkPagesDictionary(kidDictionary, set)
			case cos.Page:
				// count pages
				numberOfPages++
			}
		}
	}
	// fix counter
	pagesDict.SetInt(cos.Count, numberOfPages)
	return numberOfPages
}

// ParsePDFHeader reads the %PDF- header and records the version it names.
func (p *FileParser) ParsePDFHeader() (bool, error) {
	return p.parseHeader(pdfHeader, pdfDefaultVersion)
}

// ParseFDFHeader reads the %FDF- header.
func (p *FileParser) ParseFDFHeader() (bool, error) {
	return p.parseHeader(fdfHeader, fdfDefaultVersion)
}

// headerWithVersion matches a header whose marker is followed by a version.
//
// Java builds the expression from the marker it was given and calls
// String.matches, which anchors both ends.
var headerWithVersion = regexp.MustCompile(`^%(?:PDF|FDF)-\d.\d$`)

func (p *FileParser) parseHeader(headerMarker, defaultVersion string) (bool, error) {
	// read first line
	header, err := p.ReadLine()
	if err != nil {
		return false, err
	}
	// some pdf-documents are broken and the pdf-version is in one of the
	// following lines
	if !strings.Contains(header, headerMarker) {
		header, err = p.ReadLine()
		if err != nil {
			return false, err
		}
		for !strings.Contains(header, headerMarker) {
			// if a line starts with a digit, it has to be the first one with
			// data in it
			if header != "" && header[0] >= '0' && header[0] <= '9' {
				break
			}
			header, err = p.ReadLine()
			if err != nil {
				return false, err
			}
		}
	}

	// nothing found
	if !strings.Contains(header, headerMarker) {
		if _, err := p.Source().Seek(0, 0); err != nil {
			return false, err
		}
		return false, nil
	}

	// sometimes there is some garbage in the header before the header
	// actually starts, so lets try to find the header first.
	headerStart := strings.Index(header, headerMarker)
	// greater than zero because if it is zero then there is no point of
	// trimming
	if headerStart > 0 {
		// trim off any leading characters
		header = header[headerStart:]
	}

	// This is used if there is garbage after the header on the same line
	if strings.HasPrefix(header, headerMarker) && !headerWithVersion.MatchString(header) {
		if len(header) < len(headerMarker)+3 {
			// No version number at all, set to 1.4 as default
			header = headerMarker + defaultVersion
		} else {
			headerGarbage := header[len(headerMarker)+3:] + "\n"
			header = header[:len(headerMarker)+3]
			if err := p.rewind(int64(len(headerGarbage))); err != nil {
				return false, err
			}
		}
	}

	headerVersion := float32(-1)
	if headerParts := strings.Split(header, "-"); len(headerParts) == 2 {
		if v, err := strconv.ParseFloat(headerParts[1], 32); err == nil {
			headerVersion = float32(v)
		}
	}

	if headerVersion < 0 {
		if !p.IsLenient() {
			return false, fmt.Errorf("pdfparser: Error getting header version: %s", header)
		}
		headerVersion = 1.7
	}
	p.Document().SetVersion(headerVersion)

	// rewind
	if _, err := p.Source().Seek(0, 0); err != nil {
		return false, err
	}
	return true, nil
}

// prepareDecryption prepares the decryption of the document, reporting an
// InvalidPasswordError where the password is wrong.
func (p *FileParser) prepareDecryption() error {
	if p.encryption != nil {
		return nil
	}
	encryptionDictionary := p.Document().EncryptionDictionary()
	if encryptionDictionary == nil {
		return nil
	}

	p.encryption = encryption.NewPDEncryptionOf(encryptionDictionary)
	var decryptionMaterial encryption.DecryptionMaterial
	if p.keyStoreInput != nil {
		ks, err := encryption.LoadPKCS12(p.keyStoreInput, p.password)
		if err != nil {
			return err
		}
		decryptionMaterial = encryption.NewPublicKeyDecryptionMaterial(ks, p.keyAlias, p.password)
	} else {
		decryptionMaterial = encryption.NewStandardDecryptionMaterial(p.password)
	}

	securityHandler, err := p.encryption.SecurityHandler()
	if err != nil {
		return err
	}
	p.securityHandler = securityHandler
	if err := securityHandler.PrepareForDecryption(p.encryption,
		p.Document().DocumentID(), decryptionMaterial); err != nil {
		return err
	}
	p.accessPermission = securityHandler.CurrentAccessPermission()
	return nil
}

// SecurityHandler returns the security handler of the document. The document
// must be parsed before this is called.
func (p *FileParser) SecurityHandler() encryption.SecurityHandler { return p.securityHandler }

// Encryption returns the encryption dictionary of the document, or nil.
func (p *FileParser) Encryption() *encryption.PDEncryption { return p.encryption }

// AccessPermission returns the permissions the password granted, or nil where
// the document is not encrypted.
func (p *FileParser) AccessPermission() *encryption.AccessPermission {
	return p.accessPermission
}
