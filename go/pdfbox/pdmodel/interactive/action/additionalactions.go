package action

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

// additionalActions is the state and the two shared methods of the five
// additional-action dictionaries, each of which is the same dictionary wrapper
// with a different set of keys.
type additionalActions struct {
	actions *cos.Dictionary
}

// COSObject returns the dictionary.
func (a *additionalActions) COSObject() cos.Base { return a.actions }

// Dictionary returns the dictionary, typed.
func (a *additionalActions) Dictionary() *cos.Dictionary { return a.actions }

// get is the `PDActionFactory.createAction(actions.getCOSDictionary(key))` the
// accessors share. PDAdditionalActions.getF omits the null check, which makes
// no difference: createAction returns null for a null dictionary.
func (a *additionalActions) get(key *cos.Name) Action {
	return CreateAction(a.actions.GetCOSDictionary(key))
}

// set is the matching setter.
func (a *additionalActions) set(key *cos.Name, action Action) {
	if action == nil {
		a.actions.SetItem(key, nil)
		return
	}
	a.actions.SetItem(key, action.COSObject())
}

// PDAdditionalActions is the additional actions of a file specification.
//
// Port of PDAdditionalActions.
type PDAdditionalActions struct{ additionalActions }

var _ common.COSObjectable = (*PDAdditionalActions)(nil)

// NewPDAdditionalActions creates an empty dictionary.
func NewPDAdditionalActions() *PDAdditionalActions {
	return &PDAdditionalActions{additionalActions{actions: cos.NewDictionary()}}
}

// NewPDAdditionalActionsOf creates one over the given dictionary.
func NewPDAdditionalActionsOf(a *cos.Dictionary) *PDAdditionalActions {
	return &PDAdditionalActions{additionalActions{actions: a}}
}

// F returns the action performed when the file is opened.
func (a *PDAdditionalActions) F() Action { return a.get(cos.F) }

// SetF sets the action performed when the file is opened.
func (a *PDAdditionalActions) SetF(action Action) { a.set(cos.F, action) }

// PDAnnotationAdditionalActions is the additional actions of an annotation.
//
// Port of PDAnnotationAdditionalActions.
type PDAnnotationAdditionalActions struct{ additionalActions }

var _ common.COSObjectable = (*PDAnnotationAdditionalActions)(nil)

// NewPDAnnotationAdditionalActions creates an empty dictionary.
func NewPDAnnotationAdditionalActions() *PDAnnotationAdditionalActions {
	return &PDAnnotationAdditionalActions{additionalActions{actions: cos.NewDictionary()}}
}

// NewPDAnnotationAdditionalActionsOf creates one over the given dictionary.
func NewPDAnnotationAdditionalActionsOf(a *cos.Dictionary) *PDAnnotationAdditionalActions {
	return &PDAnnotationAdditionalActions{additionalActions{actions: a}}
}

// E returns the action performed when the cursor enters the annotation.
func (a *PDAnnotationAdditionalActions) E() Action { return a.get(cos.E) }

// SetE sets it.
func (a *PDAnnotationAdditionalActions) SetE(e Action) { a.set(cos.E, e) }

// X returns the action performed when the cursor leaves the annotation.
func (a *PDAnnotationAdditionalActions) X() Action { return a.get(cos.X) }

// SetX sets it.
func (a *PDAnnotationAdditionalActions) SetX(x Action) { a.set(cos.X, x) }

// D returns the action performed when the mouse is pressed.
func (a *PDAnnotationAdditionalActions) D() Action { return a.get(cos.D) }

// SetD sets it.
func (a *PDAnnotationAdditionalActions) SetD(d Action) { a.set(cos.D, d) }

// U returns the action performed when the mouse is released.
func (a *PDAnnotationAdditionalActions) U() Action { return a.get(cos.U) }

// SetU sets it.
func (a *PDAnnotationAdditionalActions) SetU(u Action) { a.set(cos.U, u) }

// Fo returns the action performed when the annotation takes the focus.
func (a *PDAnnotationAdditionalActions) Fo() Action { return a.get(cos.Fo) }

// SetFo sets it.
func (a *PDAnnotationAdditionalActions) SetFo(fo Action) { a.set(cos.Fo, fo) }

// Bl returns the action performed when the annotation loses the focus.
func (a *PDAnnotationAdditionalActions) Bl() Action { return a.get(cos.Bl) }

// SetBl sets it.
func (a *PDAnnotationAdditionalActions) SetBl(bl Action) { a.set(cos.Bl, bl) }

// PO returns the action performed when the page is opened.
func (a *PDAnnotationAdditionalActions) PO() Action { return a.get(cos.PO) }

// SetPO sets it.
func (a *PDAnnotationAdditionalActions) SetPO(po Action) { a.set(cos.PO, po) }

// PC returns the action performed when the page is closed.
func (a *PDAnnotationAdditionalActions) PC() Action { return a.get(cos.PC) }

// SetPC sets it.
func (a *PDAnnotationAdditionalActions) SetPC(pc Action) { a.set(cos.PC, pc) }

// PV returns the action performed when the page becomes visible.
func (a *PDAnnotationAdditionalActions) PV() Action { return a.get(cos.PV) }

// SetPV sets it.
func (a *PDAnnotationAdditionalActions) SetPV(pv Action) { a.set(cos.PV, pv) }

// PI returns the action performed when the page is no longer visible.
func (a *PDAnnotationAdditionalActions) PI() Action { return a.get(cos.PI) }

// SetPI sets it.
func (a *PDAnnotationAdditionalActions) SetPI(pi Action) { a.set(cos.PI, pi) }

// PDDocumentCatalogAdditionalActions is the additional actions of a document
// catalogue.
//
// Port of PDDocumentCatalogAdditionalActions.
type PDDocumentCatalogAdditionalActions struct{ additionalActions }

var _ common.COSObjectable = (*PDDocumentCatalogAdditionalActions)(nil)

// NewPDDocumentCatalogAdditionalActions creates an empty dictionary.
func NewPDDocumentCatalogAdditionalActions() *PDDocumentCatalogAdditionalActions {
	return &PDDocumentCatalogAdditionalActions{additionalActions{actions: cos.NewDictionary()}}
}

// NewPDDocumentCatalogAdditionalActionsOf creates one over the given
// dictionary.
func NewPDDocumentCatalogAdditionalActionsOf(a *cos.Dictionary) *PDDocumentCatalogAdditionalActions {
	return &PDDocumentCatalogAdditionalActions{additionalActions{actions: a}}
}

// WC returns the action performed before the document is closed.
func (a *PDDocumentCatalogAdditionalActions) WC() Action { return a.get(cos.WC) }

// SetWC sets it.
func (a *PDDocumentCatalogAdditionalActions) SetWC(wc Action) { a.set(cos.WC, wc) }

// WS returns the action performed before the document is saved.
func (a *PDDocumentCatalogAdditionalActions) WS() Action { return a.get(cos.WS) }

// SetWS sets it.
func (a *PDDocumentCatalogAdditionalActions) SetWS(ws Action) { a.set(cos.WS, ws) }

// DS returns the action performed after the document is saved.
func (a *PDDocumentCatalogAdditionalActions) DS() Action { return a.get(cos.DS) }

// SetDS sets it.
func (a *PDDocumentCatalogAdditionalActions) SetDS(ds Action) { a.set(cos.DS, ds) }

// WP returns the action performed before the document is printed.
func (a *PDDocumentCatalogAdditionalActions) WP() Action { return a.get(cos.WP) }

// SetWP sets it.
func (a *PDDocumentCatalogAdditionalActions) SetWP(wp Action) { a.set(cos.WP, wp) }

// DP returns the action performed after the document is printed.
func (a *PDDocumentCatalogAdditionalActions) DP() Action { return a.get(cos.DP) }

// SetDP sets it.
func (a *PDDocumentCatalogAdditionalActions) SetDP(dp Action) { a.set(cos.DP, dp) }

// PDFormFieldAdditionalActions is the additional actions of a form field.
//
// Port of PDFormFieldAdditionalActions.
type PDFormFieldAdditionalActions struct{ additionalActions }

var _ common.COSObjectable = (*PDFormFieldAdditionalActions)(nil)

// NewPDFormFieldAdditionalActions creates an empty dictionary.
func NewPDFormFieldAdditionalActions() *PDFormFieldAdditionalActions {
	return &PDFormFieldAdditionalActions{additionalActions{actions: cos.NewDictionary()}}
}

// NewPDFormFieldAdditionalActionsOf creates one over the given dictionary.
func NewPDFormFieldAdditionalActionsOf(a *cos.Dictionary) *PDFormFieldAdditionalActions {
	return &PDFormFieldAdditionalActions{additionalActions{actions: a}}
}

// K returns the action performed when the user types into the field.
func (a *PDFormFieldAdditionalActions) K() Action { return a.get(cos.K) }

// SetK sets it.
func (a *PDFormFieldAdditionalActions) SetK(k Action) { a.set(cos.K, k) }

// F returns the action performed before the field is formatted.
func (a *PDFormFieldAdditionalActions) F() Action { return a.get(cos.F) }

// SetF sets it.
func (a *PDFormFieldAdditionalActions) SetF(f Action) { a.set(cos.F, f) }

// V returns the action performed when the field's value changes.
func (a *PDFormFieldAdditionalActions) V() Action { return a.get(cos.V) }

// SetV sets it.
func (a *PDFormFieldAdditionalActions) SetV(v Action) { a.set(cos.V, v) }

// C returns the action performed to recalculate the field.
func (a *PDFormFieldAdditionalActions) C() Action { return a.get(cos.C) }

// SetC sets it.
func (a *PDFormFieldAdditionalActions) SetC(c Action) { a.set(cos.C, c) }

// PDPageAdditionalActions is the additional actions of a page.
//
// Port of PDPageAdditionalActions.
type PDPageAdditionalActions struct{ additionalActions }

var _ common.COSObjectable = (*PDPageAdditionalActions)(nil)

// NewPDPageAdditionalActions creates an empty dictionary.
func NewPDPageAdditionalActions() *PDPageAdditionalActions {
	return &PDPageAdditionalActions{additionalActions{actions: cos.NewDictionary()}}
}

// NewPDPageAdditionalActionsOf creates one over the given dictionary.
func NewPDPageAdditionalActionsOf(a *cos.Dictionary) *PDPageAdditionalActions {
	return &PDPageAdditionalActions{additionalActions{actions: a}}
}

// O returns the action performed when the page is opened.
func (a *PDPageAdditionalActions) O() Action { return a.get(cos.O) }

// SetO sets it.
func (a *PDPageAdditionalActions) SetO(o Action) { a.set(cos.O, o) }

// C returns the action performed when the page is closed.
func (a *PDPageAdditionalActions) C() Action { return a.get(cos.C) }

// SetC sets it.
func (a *PDPageAdditionalActions) SetC(c Action) { a.set(cos.C, c) }
