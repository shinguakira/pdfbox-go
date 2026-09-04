package compress

import (
	"slices"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/encryption"
)

// MinimumSupportedVersion is the lowest PDF version that may carry object
// streams, which a compressed save raises the document to.
//
// Port of COSWriterCompressionPool.MINIMUM_SUPPORTED_VERSION.
const MinimumSupportedVersion float32 = 1.6

// DocumentLike is what the compression pool needs of the document it
// compresses.
//
// Java takes a PDDocument, which imports pdfwriter and so, transitively, this
// package; the port names what is used, so the dependency runs one way.
type DocumentLike interface {
	// Document returns the COS document below the PD one. Port of
	// getDocument().
	Document() *cos.Document

	// Encryption returns the encryption dictionary of the document, or nil
	// where it has none. Port of getEncryption().
	Encryption() *encryption.PDEncryption
}

// CompressionPool compresses the contents of a given document.
//
// Port of org.apache.pdfbox.pdfwriter.compress.COSWriterCompressionPool.
type CompressionPool struct {
	document   DocumentLike
	parameters *Parameters

	objectPool *ObjectPool

	// topLevelObjects contains all objects that shall be directly appended to
	// the document's top level container.
	topLevelObjects []*cos.ObjectKey
	// objectStreamObjects contains all objects that may be appended to an
	// object stream.
	objectStreamObjects []*cos.ObjectKey
	// allDirectObjects is a list of all direct objects.
	allDirectObjects map[cos.Base]bool
}

// NewCompressionPool constructs an object that can be used to compress the
// contents of a given document. It provides the means to:
//
//   - Compress the COSStructure of the document, by streaming cos.Bases to
//     compressed ObjectStreams
func NewCompressionPool(document DocumentLike, parameters *Parameters) (*CompressionPool, error) {
	p := &CompressionPool{
		document:         document,
		allDirectObjects: map[cos.Base]bool{},
	}
	if parameters != nil {
		p.parameters = parameters
	} else {
		p.parameters = NewParameters()
	}
	p.objectPool = NewObjectPool(document.Document().HighestXRefObjectNumber())

	// Initialize object pool.
	trailer := document.Document().Trailer()
	var cosBaseList []cos.Base
	if root := trailer.GetCOSDictionary(cos.Root); root != nil {
		cosBaseList = append(cosBaseList, root)
	}
	if info := trailer.GetCOSDictionary(cos.Info); info != nil {
		cosBaseList = append(cosBaseList, info)
	}
	for len(cosBaseList) > 0 {
		var err error
		if cosBaseList, err = p.addStructureAll(cosBaseList); err != nil {
			return nil, err
		}
	}
	clear(p.allDirectObjects)

	slices.SortStableFunc(p.objectStreamObjects, compareKeys)
	slices.SortStableFunc(p.topLevelObjects, compareKeys)
	return p, nil
}

// compareKeys is what Collections.sort uses: COSObjectKey's compareTo.
func compareKeys(a, b *cos.ObjectKey) int { return a.Compare(b) }

// addObjectToPool adds the given cos.Base to this pool, using the given
// ObjectKey as its referencable ID. It determines an appropriate key for yet
// unregistered objects, to register them. Depending on the type of object, it
// is either appended as-is or appended to a compressed ObjectStream.
func (p *CompressionPool) addObjectToPool(key *cos.ObjectKey, base cos.Base) (cos.Base, error) {
	// Drop hollow objects.
	current := base
	if reference, ok := base.(*cos.Object); ok {
		current = reference.Object()
	}
	// to avoid to mixup indirect COSInteger objects holding the same value we have to check
	// if the given key is the same than the key which is stored for the "same" base object within the object pool
	// the same is always true for COSFloat, COSBoolean and COSName and under certain circumstances for the
	// remaining types as well
	if current == nil || (key == nil && p.objectPool.ContainsObject(current)) {
		return current, nil
	}
	if key != nil && p.objectPool.ContainsKey(key) {
		cosObject := p.objectPool.Object(key)
		// check if the key belongs to the same object
		if cosObject == current || cosObject == base {
			return current, nil
		}
	}

	// Check whether the object can not be appended to an object stream.
	// An objectStream shall only contain generation 0 objects.
	// It shall never contain the encryption dictionary.
	// It shall never contain the document's root dictionary. (relevant for document encryption)
	// It shall never contain other streams.
	_, isStream := current.(*cos.Stream)
	enc := p.document.Encryption()
	isEncryption := enc != nil && current == cos.Base(enc.COSObject())
	root := p.document.Document().Trailer().GetCOSDictionary(cos.Root)
	isRoot := root != nil && current == cos.Base(root)
	if (key != nil && key.Generation() != 0) || isStream || isEncryption || isRoot {
		actualKey, err := p.objectPool.Put(key, current)
		if err != nil {
			return nil, err
		}
		if actualKey == nil {
			return current, nil
		}
		// check if the key of the indirect object matches the key of the referenced object
		// otherwise update the key
		if _, isReference := base.(*cos.Object); !actualKey.Equals(key) && isReference {
			base.SetKey(actualKey)
		}
		p.topLevelObjects = append(p.topLevelObjects, actualKey)
		return current, nil
	}

	// Determine the object key.
	actualKey, err := p.objectPool.Put(key, current)
	if err != nil {
		return nil, err
	}
	if actualKey == nil {
		return current, nil
	}
	// check if the key of the indirect object matches the key of the referenced object
	// otherwise update the key
	if _, isReference := base.(*cos.Object); !actualKey.Equals(key) && isReference {
		base.SetKey(actualKey)
	}
	// Append it to an object stream.
	p.objectStreamObjects = append(p.objectStreamObjects, actualKey)
	return current, nil
}

// addStructureAll attempts to find yet unregistered streams and dictionaries in
// the given structures.
//
// Port of addStructure(List<COSBase>). It is an iteration rather than a
// recursion so that a deeply nested document does not overflow the stack; see
// PDFBOX-6036.
func (p *CompressionPool) addStructureAll(cosBaseList []cos.Base) ([]cos.Base, error) {
	var cosBaseListNext []cos.Base
	for _, cosBase := range cosBaseList {
		next, err := p.addStructure(cosBase)
		if err != nil {
			return nil, err
		}
		cosBaseListNext = append(cosBaseListNext, next...)
	}
	return cosBaseListNext, nil
}

// addStructure attempts to find yet unregistered streams and dictionaries in
// the given structure.
func (p *CompressionPool) addStructure(current cos.Base) ([]cos.Base, error) {
	base := current
	_, isDictionary := current.(*cos.Dictionary)
	_, isStream := current.(*cos.Stream)
	_, isArray := current.(*cos.Array)
	reference, isReference := current.(*cos.Object)

	if !current.IsDirect() && (isDictionary || isStream || isArray) {
		var err error
		if base, err = p.addObjectToPool(base.Key(), current); err != nil {
			return nil, err
		}
	} else if isReference {
		base = reference.Object()
		if base != nil {
			var err error
			if base, err = p.addObjectToPool(current.Key(), current); err != nil {
				return nil, err
			}
		}
	}
	switch value := base.(type) {
	case *cos.Array:
		return p.elements(value.ToList()), nil
	case *cos.Stream:
		// COSStream is a COSDictionary in Java, so it takes the same branch.
		return p.elements(value.Values()), nil
	case *cos.Dictionary:
		return p.elements(value.Values()), nil
	}
	return nil, nil
}

// elements collects all relevant objects from a Dictionary or Array.
func (p *CompressionPool) elements(elements []cos.Base) []cos.Base {
	var relevantElements []cos.Base
	for _, element := range elements {
		if p.filterElement(element) {
			relevantElements = append(relevantElements, element)
		}
	}
	return relevantElements
}

func (p *CompressionPool) filterElement(element cos.Base) bool {
	if cosObject, ok := element.(*cos.Object); ok {
		objectKey := cosObject.Key()
		object := cosObject.Object()
		if objectKey != nil && p.objectPool.ContainsKey(objectKey) {
			// check if the stored object matches the referenced object otherwise replace the key with a new one
			// there may differences if some imported content uses the same object numbers than the target pdf
			if p.objectPool.Object(objectKey) == object {
				return false
			}
			cosObject.SetKey(nil)
		}
		return object != nil
	}
	_, isArray := element.(*cos.Array)
	_, isDictionary := element.(*cos.Dictionary)
	_, isStream := element.(*cos.Stream)
	if isArray || ((isDictionary || isStream) && !p.allDirectObjects[element]) {
		p.allDirectObjects[element] = true
		return true
	}
	return false
}

// TopLevelObjects returns all objects that must be added to the document's top
// level container. Those objects are not valid to be added to an object stream.
func (p *CompressionPool) TopLevelObjects() []*cos.ObjectKey {
	return p.topLevelObjects
}

// ObjectStreamObjects returns all objects that can be appended to an object
// stream.
func (p *CompressionPool) ObjectStreamObjects() []*cos.ObjectKey {
	return p.objectStreamObjects
}

// Contains reports whether the given cos.Base is a registered object of this
// compression pool.
func (p *CompressionPool) Contains(object cos.Base) bool {
	return p.objectPool.ContainsObject(object)
}

// Key returns the ObjectKey that is registered for the given cos.Base in this
// compression pool.
func (p *CompressionPool) Key(object cos.Base) *cos.ObjectKey {
	return p.objectPool.Key(object)
}

// Object returns the cos.Base that is registered for the given ObjectKey in
// this compression pool.
func (p *CompressionPool) Object(key *cos.ObjectKey) cos.Base {
	return p.objectPool.Object(key)
}

// HighestXRefObjectNumber returns the highest object number that is registered
// in this compression pool.
func (p *CompressionPool) HighestXRefObjectNumber() int64 {
	return p.objectPool.HighestXRefObjectNumber()
}

// CreateObjectStreams creates ObjectStreams for all currently registered
// objects of this pool that have been marked as fit for being compressed in
// this manner. Such object streams may be added to a PDF document and shall be
// declared in a document's cross-reference stream accordingly. The objects
// contained in such a stream must not be added to the document separately.
func (p *CompressionPool) CreateObjectStreams() []*ObjectStream {
	var objectStreams []*ObjectStream
	var objectStream *ObjectStream
	for i, key := range p.objectStreamObjects {
		if objectStream == nil || i%p.parameters.ObjectStreamSize() == 0 {
			objectStream = NewObjectStream(p)
			objectStreams = append(objectStreams, objectStream)
		}
		objectStream.PrepareStreamObject(key, p.objectPool.Object(key))
	}
	return objectStreams
}
