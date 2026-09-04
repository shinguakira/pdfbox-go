package pdmodel

import (
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// delimiter separates the content streams of a page when they are read as one.
var delimiter = []byte{'\n'}

// PDPage is a page in a PDF document.
//
// Port of org.apache.pdfbox.pdmodel.PDPage.
//
// The parts that reach into subtrees this port has not covered are not here:
// annotations, thread beads, transitions, additional actions, viewports, the
// metadata stream, and the two setContents methods and getContentStreams, which
// are typed on PDStream. The ResourceCache the reading constructor takes is
// absent for the same reason. See migration/STATUS.md.
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
