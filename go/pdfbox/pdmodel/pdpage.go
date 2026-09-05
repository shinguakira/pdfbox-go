package pdmodel

import (
	"bytes"
	"io"
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/action"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/measurement"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/pagenavigation"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// delimiter separates the content streams of a page when they are read as one.
var delimiter = []byte{'\n'}

// PDPage is a page in a PDF document.
//
// Port of org.apache.pdfbox.pdmodel.PDPage.
//
// removePageResourceFromCache is not here: it purges the colour space, ext
// gstate, pattern, properties, shading and XObject halves of the resource
// cache, and the ported ResourceCache holds only fonts. See
// migration/STATUS.md.
type PDPage struct {
	page      *cos.Dictionary
	resources *PDResources
	mediaBox  *common.PDRectangle

	// resourceCache is what the page reads its resources through. Java takes a
	// PDDocument here and asks it for the cache; the port takes the cache
	// itself, which is all the document was for.
	resourceCache ResourceCache
}

var _ common.COSObjectable = (*PDPage)(nil)

// NewPDPage returns a new page for embedding, with a size of U.S. Letter
// (8.5 x 11 inches).
func NewPDPage() *PDPage {
	return NewPDPageOfSize(common.Letter)
}

// NewPDPageOfSize returns a new page for embedding, with the given media box.
func NewPDPageOfSize(mediaBox *common.PDRectangle) *PDPage {
	page := cos.NewDictionary()
	page.SetItem(cos.Type, cos.Page)
	page.SetItem(cos.MediaBox, mediaBox.COSObject())
	return &PDPage{page: page}
}

// NewPDPageOf returns the page held by the given page dictionary, for reading.
func NewPDPageOf(pageDictionary *cos.Dictionary) *PDPage {
	return &PDPage{page: pageDictionary}
}

// NewPDPageOfCache returns the page the given dictionary holds, reading its
// resources through the given cache.
func NewPDPageOfCache(pageDictionary *cos.Dictionary, cache ResourceCache) *PDPage {
	return &PDPage{page: pageDictionary, resourceCache: cache}
}

// COSObject returns the dictionary behind this page.
func (p *PDPage) COSObject() cos.Base { return p.page }

// Dictionary returns the dictionary behind this page.
func (p *PDPage) Dictionary() *cos.Dictionary { return p.page }

// ContentsForRandomAccess returns the content stream or streams of this page as
// one random access read, never nil. Several content streams are concatenated
// and separated with a newline; a page with none gives an empty read.
//
// Java narrows the signature here and declares no exception, but the interface
// method it implements does; the port keeps the error result so that PDPage
// still satisfies contentstream.PDContentStream. It is always nil.
func (p *PDPage) ContentsForRandomAccess() (pdfio.RandomAccessRead, error) {
	if contentStream := p.getCOSStream(cos.Contents); contentStream != nil {
		view, err := contentStream.CreateView()
		if err != nil {
			slog.Warn("skipped malformed content stream", "err", err)
			return pdfio.NewReadBufferBytes(delimiter), nil
		}
		return view, nil
	}
	if array := p.page.GetCOSArray(cos.Contents); array != nil {
		var reads []pdfio.RandomAccessRead
		for _, base := range array.ToList() {
			if object, ok := base.(*cos.Object); ok {
				base = object.Object()
			}
			stream, ok := base.(*cos.Stream)
			if !ok {
				continue
			}
			view, err := stream.CreateView()
			if err != nil {
				slog.Warn("malformed substream of content stream skipped", "err", err)
				continue
			}
			reads = append(reads, view, pdfio.NewReadBufferBytes(delimiter))
		}
		if len(reads) > 0 {
			sequence, err := pdfio.NewSequenceRead(reads)
			if err != nil {
				slog.Warn("skipped malformed content stream", "err", err)
				return pdfio.NewReadBufferBytes(delimiter), nil
			}
			return sequence, nil
		}
	}
	return pdfio.NewReadBufferBytes(nil), nil
}

// ContentsForStreamParsing returns the content of this page for a parser that
// only reads forwards.
//
// Java has a fast path here that decodes a single flate stream as it is read,
// rather than into a buffer. It needs the decoder stream and the non-seekable
// wrapper, neither of which is ported yet, so this is the general path for now
// — the same one Java falls back to. See migration/STATUS.md.
func (p *PDPage) ContentsForStreamParsing() (pdfio.RandomAccessRead, error) {
	return p.ContentsForRandomAccess()
}

// getCOSStream returns the value of key as a stream, resolving an indirect
// reference, or nil if it is not one.
func (p *PDPage) getCOSStream(key *cos.Name) *cos.Stream {
	stream, _ := p.page.GetDictionaryObject(key).(*cos.Stream)
	return stream
}

// HasContents reports whether this page has one or more content streams.
func (p *PDPage) HasContents() bool {
	contents := p.page.GetDictionaryObject(cos.Contents)
	switch value := contents.(type) {
	case *cos.Stream:
		return value.Size() > 0
	case *cos.Array:
		return !value.IsEmpty()
	}
	return false
}

// Resources returns the dictionary containing any resources required by the
// page, or nil. Note that it is an error for resources to not be present.
func (p *PDPage) Resources() *PDResources {
	if p.resources == nil {
		base := GetInheritableAttribute(p.page, cos.Resources)
		if dict, ok := base.(*cos.Dictionary); ok {
			p.resources = NewPDResourcesOfCache(dict, p.resourceCache)
		}
	}
	return p.resources
}

// SetResources sets the resources for this page.
func (p *PDPage) SetResources(resources *PDResources) {
	p.resources = resources
	if resources != nil {
		p.page.SetItem(cos.Resources, resources.COSObject())
	} else {
		p.page.RemoveItem(cos.Resources)
	}
}

// StructParents returns the key of this page in the structural parent tree, or
// -1 if there isn't any.
func (p *PDPage) StructParents() int {
	return p.page.GetInt(cos.StructParents)
}

// SetStructParents sets the key for this page in the structural parent tree.
func (p *PDPage) SetStructParents(structParents int) {
	p.page.SetInt(cos.StructParents, structParents)
}

// BBox returns the bounding box of the contents, which for a page is its crop
// box.
func (p *PDPage) BBox() *common.PDRectangle { return p.CropBox() }

// Matrix returns the matrix which transforms from the page's space to user
// space.
func (p *PDPage) Matrix() *util.Matrix {
	// todo: take into account user-space unit redefinition as scale?
	return util.NewMatrix()
}

// MediaBox returns the rectangle, expressed in default user space units,
// defining the boundaries of the physical medium on which the page is intended
// to be displayed or printed.
func (p *PDPage) MediaBox() *common.PDRectangle {
	if p.mediaBox == nil {
		base := GetInheritableAttribute(p.page, cos.MediaBox)
		if array, ok := base.(*cos.Array); ok {
			p.mediaBox = common.NewPDRectangleOfCOSArray(array)
		} else {
			slog.Debug("can't find MediaBox, will use U.S. Letter")
			p.mediaBox = common.Letter
		}
	}
	return p.mediaBox
}

// SetMediaBox sets the media box for this page.
func (p *PDPage) SetMediaBox(mediaBox *common.PDRectangle) {
	p.mediaBox = mediaBox
	if mediaBox == nil {
		p.page.RemoveItem(cos.MediaBox)
	} else {
		p.page.SetItem(cos.MediaBox, mediaBox.COSObject())
	}
}

// CropBox returns the rectangle, expressed in default user space units,
// defining the visible region of default user space. When the page is displayed
// or printed, its contents are to be clipped (cropped) to this rectangle.
func (p *PDPage) CropBox() *common.PDRectangle {
	base := GetInheritableAttribute(p.page, cos.CropBox)
	if array, ok := base.(*cos.Array); ok {
		return p.clipToMediaBox(common.NewPDRectangleOfCOSArray(array))
	}
	return p.MediaBox()
}

// SetCropBox sets the crop box for this page.
func (p *PDPage) SetCropBox(cropBox *common.PDRectangle) {
	if cropBox == nil {
		p.page.RemoveItem(cos.CropBox)
	} else {
		p.page.SetItem(cos.CropBox, cropBox.COSArray())
	}
}

// BleedBox returns the rectangle, expressed in default user space units,
// defining the region to which the contents of the page should be clipped when
// output in a production environment. The default is the crop box.
func (p *PDPage) BleedBox() *common.PDRectangle {
	if bleedBox := p.page.GetCOSArray(cos.BleedBox); bleedBox != nil {
		return p.clipToMediaBox(common.NewPDRectangleOfCOSArray(bleedBox))
	}
	return p.CropBox()
}

// SetBleedBox sets the bleed box for this page.
func (p *PDPage) SetBleedBox(bleedBox *common.PDRectangle) {
	if bleedBox == nil {
		p.page.RemoveItem(cos.BleedBox)
	} else {
		p.page.SetItem(cos.BleedBox, bleedBox.COSObject())
	}
}

// TrimBox returns the rectangle, expressed in default user space units,
// defining the intended dimensions of the finished page after trimming. The
// default is the crop box.
func (p *PDPage) TrimBox() *common.PDRectangle {
	if trimBox := p.page.GetCOSArray(cos.TrimBox); trimBox != nil {
		return p.clipToMediaBox(common.NewPDRectangleOfCOSArray(trimBox))
	}
	return p.CropBox()
}

// SetTrimBox sets the trim box for this page.
func (p *PDPage) SetTrimBox(trimBox *common.PDRectangle) {
	if trimBox == nil {
		p.page.RemoveItem(cos.TrimBox)
	} else {
		p.page.SetItem(cos.TrimBox, trimBox.COSObject())
	}
}

// ArtBox returns the rectangle, expressed in default user space units, defining
// the extent of the page's meaningful content (including potential white space)
// as intended by the page's creator. The default is the crop box.
func (p *PDPage) ArtBox() *common.PDRectangle {
	if artBox := p.page.GetCOSArray(cos.ArtBox); artBox != nil {
		return p.clipToMediaBox(common.NewPDRectangleOfCOSArray(artBox))
	}
	return p.CropBox()
}

// SetArtBox sets the art box for this page.
func (p *PDPage) SetArtBox(artBox *common.PDRectangle) {
	if artBox == nil {
		p.page.RemoveItem(cos.ArtBox)
	} else {
		p.page.SetItem(cos.ArtBox, artBox.COSObject())
	}
}

// clipToMediaBox clips the given box to the bounds of the media box.
func (p *PDPage) clipToMediaBox(box *common.PDRectangle) *common.PDRectangle {
	mediaBox := p.MediaBox()
	result := common.NewPDRectangle()
	result.SetLowerLeftX(max(mediaBox.LowerLeftX(), box.LowerLeftX()))
	result.SetLowerLeftY(max(mediaBox.LowerLeftY(), box.LowerLeftY()))
	result.SetUpperRightX(min(mediaBox.UpperRightX(), box.UpperRightX()))
	result.SetUpperRightY(min(mediaBox.UpperRightY(), box.UpperRightY()))
	return result
}

// Rotation returns the rotation angle in degrees by which the page should be
// rotated clockwise when displayed or printed, in normalized form (0, 90, 180
// or 270), or 0 if invalid or not set at this level. Valid values in a PDF must
// be a multiple of 90.
func (p *PDPage) Rotation() int {
	obj := GetInheritableAttribute(p.page, cos.Rotate)
	if number, ok := obj.(cos.Number); ok {
		rotationAngle := number.IntValue()
		if rotationAngle%90 == 0 {
			return (rotationAngle%360 + 360) % 360
		}
	}
	return 0
}

// SetRotation sets the rotation for this page in degrees.
func (p *PDPage) SetRotation(rotation int) {
	p.page.SetInt(cos.Rotate, rotation)
}

// Equals reports whether both pages stand for the same dictionary.
func (p *PDPage) Equals(other *PDPage) bool {
	if p == other {
		return true
	}
	if other == nil {
		return false
	}
	return p.page == other.page
}

// ContentStreams returns the content streams of this page.
//
// Port of getContentStreams, whose Iterator becomes a slice.
func (p *PDPage) ContentStreams() []*common.PDStream {
	base := p.page.GetDictionaryObject(cos.Contents)
	switch value := base.(type) {
	case *cos.Stream:
		return []*common.PDStream{common.NewPDStream(value)}
	case *cos.Array:
		streams := make([]*common.PDStream, 0, value.Size())
		for i := 0; i < value.Size(); i++ {
			stream, ok := value.GetObject(i).(*cos.Stream)
			if !ok {
				// Java casts without a check and throws ClassCastException.
				panic("pdmodel: a content stream array holds something that is not a stream")
			}
			streams = append(streams, common.NewPDStream(stream))
		}
		return streams
	}
	return nil
}

// Contents returns the content stream or streams of this page as a single
// reader.
//
// Port of getContents().
func (p *PDPage) Contents() (io.Reader, error) {
	contentsForRandomAccess, err := p.ContentsForRandomAccess()
	if err != nil {
		return nil, err
	}
	if contentsForRandomAccess != nil {
		return pdfio.NewReader(contentsForRandomAccess), nil
	}
	return bytes.NewReader(nil), nil
}

// SetContents sets the contents of this page.
//
// Port of setContents(PDStream).
func (p *PDPage) SetContents(contents *common.PDStream) {
	p.page.SetItem(cos.Contents, contents.COSObject())
}

// SetContentsOfList sets the contents of this page to the given streams.
//
// Port of setContents(List<PDStream>).
func (p *PDPage) SetContentsOfList(contents []*common.PDStream) {
	array := cos.NewArray()
	for _, stream := range contents {
		array.Add(stream.COSObject())
	}
	p.page.SetItem(cos.Contents, array)
}

// ThreadBeads returns the article threads on this page, which is empty where
// there are none.
//
// The list is backed by the beads array, so adding to it or deleting from it
// changes the document too.
func (p *PDPage) ThreadBeads() *common.COSArrayList[*pagenavigation.PDThreadBead] {
	beads := p.page.GetCOSArray(cos.B)
	if beads == nil {
		beads = cos.NewArray()
	}
	pdObjects := make([]*pagenavigation.PDThreadBead, 0, beads.Size())
	for i := 0; i < beads.Size(); i++ {
		base := beads.GetObject(i)
		var bead *pagenavigation.PDThreadBead
		// in some cases the bead is null
		if dic, isDictionary := base.(*cos.Dictionary); isDictionary {
			bead = pagenavigation.NewPDThreadBeadOf(dic)
		}
		pdObjects = append(pdObjects, bead)
	}
	return common.NewCOSArrayListOf(pdObjects, beads)
}

// SetThreadBeads sets the article threads on this page, and removes them for a
// nil list.
func (p *PDPage) SetThreadBeads(beads []*pagenavigation.PDThreadBead) {
	if beads == nil {
		p.page.RemoveItem(cos.B)
		return
	}
	p.page.SetItem(cos.B, common.NewCOSArrayOfObjectables(beads))
}

// Metadata returns the metadata of this page, or nil where it has none.
func (p *PDPage) Metadata() *common.PDMetadata {
	metadata := p.page.GetCOSStream(cos.Metadata)
	if metadata != nil {
		return common.NewPDMetadataOfStream(metadata)
	}
	return nil
}

// SetMetadata sets the metadata of this page, which may be nil.
func (p *PDPage) SetMetadata(meta *common.PDMetadata) {
	p.page.SetItem(cos.Metadata, common.COSObjectOrNil(meta))
}

// Actions returns the additional actions of this page.
//
// Java adds the dictionary to the page when it is absent, so reading the
// actions of a page that has none writes an empty /AA to it. The port carries
// that. See migration/JAVA-BUGS.md.
func (p *PDPage) Actions() *action.PDPageAdditionalActions {
	addAct := p.page.GetCOSDictionary(cos.AA)
	if addAct == nil {
		addAct = cos.NewDictionary()
		p.page.SetItem(cos.AA, addAct)
	}
	return action.NewPDPageAdditionalActionsOf(addAct)
}

// SetActions sets the additional actions of this page.
func (p *PDPage) SetActions(actions *action.PDPageAdditionalActions) {
	p.page.SetItem(cos.AA, common.COSObjectOrNil(actions))
}

// Transition returns the transition of this page, or nil where it has none.
func (p *PDPage) Transition() *pagenavigation.PDTransition {
	transition := p.page.GetCOSDictionary(cos.Trans)
	if transition != nil {
		return pagenavigation.NewPDTransitionOf(transition)
	}
	return nil
}

// SetTransition sets the transition of this page.
func (p *PDPage) SetTransition(transition *pagenavigation.PDTransition) {
	p.page.SetItem(cos.Trans, common.COSObjectOrNil(transition))
}

// SetTransitionOfDuration sets the transition of this page along with the
// longest time, in seconds, the page is shown during a presentation before the
// viewer moves on by itself.
//
// Port of setTransition(PDTransition, float).
func (p *PDPage) SetTransitionOfDuration(transition *pagenavigation.PDTransition, duration float32) {
	p.page.SetItem(cos.Trans, common.COSObjectOrNil(transition))
	p.page.SetItem(cos.Dur, cos.NewFloat(duration))
}

// Annotations returns the annotations of this page, never nil.
//
// The list is backed by the annotations array, so adding to it or deleting from
// it changes the document too.
func (p *PDPage) Annotations() *common.COSArrayList[annotation.PDAnnotation] {
	return p.AnnotationsOfFilter(func(annotation.PDAnnotation) bool { return true })
}

// AnnotationsOfFilter returns the annotations of this page the given filter
// accepts, never nil.
//
// Port of getAnnotations(AnnotationFilter).
func (p *PDPage) AnnotationsOfFilter(
	annotationFilter annotation.AnnotationFilter) *common.COSArrayList[annotation.PDAnnotation] {
	annots := p.page.GetCOSArray(cos.Annots)
	if annots == nil {
		return common.NewCOSArrayListOfDictionary[annotation.PDAnnotation](p.page, cos.Annots)
	}

	actuals := []annotation.PDAnnotation{}
	for i := 0; i < annots.Size(); i++ {
		item := annots.GetObject(i)
		if item == nil {
			continue
		}
		createdAnnotation, err := annotation.CreateAnnotation(item)
		if err != nil {
			slog.Error("pdmodel: annotation not read", slog.Any("err", err))
			continue
		}
		if annotationFilter(createdAnnotation) {
			actuals = append(actuals, createdAnnotation)
		}
	}
	return common.NewCOSArrayListOf(actuals, annots)
}

// SetAnnotations sets the annotations of this page.
//
// This is optional, but take care that any annotation newly created links back
// to this page, by calling SetPage on it. Not doing it can cause trouble when
// PDFs get signed; see https://stackoverflow.com/questions/74836898/.
func (p *PDPage) SetAnnotations(annotations []annotation.PDAnnotation) {
	p.page.SetItem(cos.Annots, common.NewCOSArrayOfObjectables(annotations))
}

// ResourceCache returns the cache this page reads its resources through, or nil
// where it has none.
func (p *PDPage) ResourceCache() ResourceCache { return p.resourceCache }

// Viewports returns the viewports of this page, or nil where it has no /VP.
func (p *PDPage) Viewports() []*measurement.PDViewportDictionary {
	array := p.page.GetCOSArray(cos.VP)
	if array == nil {
		return nil
	}
	viewports := make([]*measurement.PDViewportDictionary, 0, array.Size())
	for i := 0; i < array.Size(); i++ {
		base2 := array.GetObject(i)
		if dic, isDictionary := base2.(*cos.Dictionary); isDictionary {
			viewports = append(viewports, measurement.NewPDViewportDictionaryOf(dic))
		} else {
			slog.Warn("pdmodel: array element is skipped, must be a (viewport) dictionary",
				slog.Any("element", base2))
		}
	}
	return viewports
}

// SetViewports sets the viewports of this page, and removes them for a nil
// list.
func (p *PDPage) SetViewports(viewports []*measurement.PDViewportDictionary) {
	if viewports == nil {
		p.page.RemoveItem(cos.VP)
		return
	}
	p.page.SetItem(cos.VP, common.NewCOSArrayOfObjectables(viewports))
}

// UserUnit returns the size of a default user space unit in multiples of 1/72
// inch, which is 1 where it has not been set. PDF 1.6 and higher support it.
func (p *PDPage) UserUnit() float32 {
	userUnit := p.page.GetFloat(cos.UserUnit, 1.0)
	if userUnit > 0 {
		return userUnit
	}
	return 1.0
}

// SetUserUnit sets the size of a default user space unit in multiples of 1/72
// inch. PDF 1.6 and higher support it.
//
// Java throws IllegalArgumentException where the unit is not positive, which is
// unchecked, so the port panics.
func (p *PDPage) SetUserUnit(userUnit float32) {
	if userUnit <= 0 {
		panic("User unit must be positive")
	}
	p.page.SetFloat(cos.UserUnit, userUnit)
}
