package pagenavigation

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

// PageLike is the page a thread bead sits on, and DocumentInformationLike is
// the information dictionary a thread carries.
//
// Java names PDPage and PDDocumentInformation, which live in pdmodel; pdmodel
// imports this package for PDPage.getThreadBeads, so Go cannot have the
// dependency run both ways. The port names what is used, and takes the two
// constructors through the variables below, which pdmodel sets from its init.
// It is the same device slice 6 used for image.DocumentLike.
type PageLike interface {
	common.COSObjectable
}

// DocumentInformationLike is the /Info dictionary of a thread.
type DocumentInformationLike interface {
	common.COSObjectable
}

// NewPageFromDictionary builds a page from its dictionary. pdmodel sets it.
var NewPageFromDictionary func(dic *cos.Dictionary) PageLike

// NewDocumentInformationFromDictionary builds an information dictionary from
// its dictionary. pdmodel sets it.
var NewDocumentInformationFromDictionary func(dic *cos.Dictionary) DocumentInformationLike

// PDThread is an article thread of a document.
//
// Port of org.apache.pdfbox.pdmodel.interactive.pagenavigation.PDThread.
type PDThread struct {
	thread *cos.Dictionary
}

var _ common.COSObjectable = (*PDThread)(nil)

// NewPDThreadOf creates a thread over the given dictionary.
func NewPDThreadOf(t *cos.Dictionary) *PDThread {
	return &PDThread{thread: t}
}

// NewPDThread creates a new empty thread.
func NewPDThread() *PDThread {
	thread := cos.NewDictionary()
	thread.SetItem(cos.Type, cos.Thread)
	return &PDThread{thread: thread}
}

// COSObject returns the dictionary.
func (t *PDThread) COSObject() cos.Base { return t.thread }

// Dictionary returns the dictionary, typed.
func (t *PDThread) Dictionary() *cos.Dictionary { return t.thread }

// ThreadInfo returns the /I information dictionary of this thread, or nil.
func (t *PDThread) ThreadInfo() DocumentInformationLike {
	info := t.thread.GetCOSDictionary(cos.I)
	if info != nil {
		return NewDocumentInformationFromDictionary(info)
	}
	return nil
}

// SetThreadInfo sets the /I information dictionary of this thread.
func (t *PDThread) SetThreadInfo(info DocumentInformationLike) {
	if info == nil {
		t.thread.SetItem(cos.I, nil)
		return
	}
	t.thread.SetItem(cos.I, info.COSObject())
}

// FirstBead returns the /F first bead of this thread, or nil.
func (t *PDThread) FirstBead() *PDThreadBead {
	bead := t.thread.GetCOSDictionary(cos.F)
	if bead != nil {
		return NewPDThreadBeadOf(bead)
	}
	return nil
}

// SetFirstBead sets the /F first bead of this thread.
func (t *PDThread) SetFirstBead(bead *PDThreadBead) {
	if bead != nil {
		bead.SetThread(t)
		t.thread.SetItem(cos.F, bead.COSObject())
		return
	}
	t.thread.SetItem(cos.F, nil)
}

// PDThreadBead is one bead of an article thread: a rectangle on a page.
//
// Port of PDThreadBead.
type PDThreadBead struct {
	bead *cos.Dictionary
}

var _ common.COSObjectable = (*PDThreadBead)(nil)

// NewPDThreadBeadOf creates a bead over the given dictionary.
func NewPDThreadBeadOf(b *cos.Dictionary) *PDThreadBead {
	return &PDThreadBead{bead: b}
}

// NewPDThreadBead creates a new bead whose next and previous are itself.
func NewPDThreadBead() *PDThreadBead {
	b := &PDThreadBead{bead: cos.NewDictionary()}
	// JAVA BUG 35: COSName.BEAD is "BEAD" and the specification says /Bead.
	// Ported as written; see migration/JAVA-BUGS.md.
	b.bead.SetItem(cos.Type, cos.BEAD)
	b.setNextBead(b)
	b.setPreviousBead(b)
	return b
}

// COSObject returns the dictionary.
func (b *PDThreadBead) COSObject() cos.Base { return b.bead }

// Dictionary returns the dictionary, typed.
func (b *PDThreadBead) Dictionary() *cos.Dictionary { return b.bead }

// Thread returns the /T thread this bead belongs to, or nil.
func (b *PDThreadBead) Thread() *PDThread {
	dic := b.bead.GetCOSDictionary(cos.T)
	if dic != nil {
		return NewPDThreadOf(dic)
	}
	return nil
}

// SetThread sets the /T thread this bead belongs to.
func (b *PDThreadBead) SetThread(thread *PDThread) {
	if thread == nil {
		b.bead.SetItem(cos.T, nil)
		return
	}
	b.bead.SetItem(cos.T, thread.COSObject())
}

// NextBead returns the /N next bead.
//
// Java wraps whatever getCOSDictionary returns, null included, so the bead it
// hands back can be one over a null dictionary; the port does the same rather
// than returning nil, because callers chain onto it.
func (b *PDThreadBead) NextBead() *PDThreadBead {
	return NewPDThreadBeadOf(b.bead.GetCOSDictionary(cos.N))
}

// setNextBead sets the /N next bead. Java declares it protected and final.
func (b *PDThreadBead) setNextBead(next *PDThreadBead) {
	if next == nil {
		b.bead.SetItem(cos.N, nil)
		return
	}
	b.bead.SetItem(cos.N, next.COSObject())
}

// PreviousBead returns the /V previous bead.
func (b *PDThreadBead) PreviousBead() *PDThreadBead {
	return NewPDThreadBeadOf(b.bead.GetCOSDictionary(cos.V))
}

// setPreviousBead sets the /V previous bead.
func (b *PDThreadBead) setPreviousBead(previous *PDThreadBead) {
	if previous == nil {
		b.bead.SetItem(cos.V, nil)
		return
	}
	b.bead.SetItem(cos.V, previous.COSObject())
}

// AppendBead links the given bead in after this one.
func (b *PDThreadBead) AppendBead(append *PDThreadBead) {
	nextBead := b.NextBead()
	nextBead.setPreviousBead(append)
	append.setNextBead(nextBead)
	b.setNextBead(append)
	append.setPreviousBead(b)
}

// Page returns the /P page this bead is on, or nil.
func (b *PDThreadBead) Page() PageLike {
	dic := b.bead.GetCOSDictionary(cos.P)
	if dic != nil {
		return NewPageFromDictionary(dic)
	}
	return nil
}

// SetPage sets the /P page this bead is on.
func (b *PDThreadBead) SetPage(page PageLike) {
	if page == nil {
		b.bead.SetItem(cos.P, nil)
		return
	}
	b.bead.SetItem(cos.P, page.COSObject())
}

// Rectangle returns the /R rectangle of this bead, or nil.
func (b *PDThreadBead) Rectangle() *common.PDRectangle {
	array := b.bead.GetCOSArray(cos.R)
	if array != nil {
		return common.NewPDRectangleOfCOSArray(array)
	}
	return nil
}

// SetRectangle sets the /R rectangle of this bead.
func (b *PDThreadBead) SetRectangle(rect *common.PDRectangle) {
	if rect == nil {
		b.bead.SetItem(cos.R, nil)
		return
	}
	b.bead.SetItem(cos.R, rect.COSObject())
}
