// Package action holds the actions a PDF triggers: following a link, running
// JavaScript, submitting a form.
//
// Port of org.apache.pdfbox.pdmodel.interactive.action.
package action

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

// TypeAction is the /Type every action dictionary carries.
//
// Port of PDAction.TYPE.
const TypeAction = "Action"

// The /S subtypes, which Java declares one per class as SUB_TYPE. Go has one
// package namespace, so they sit together and each carries the class it is on.
const (
	// SubTypeGoTo is PDActionGoTo.SUB_TYPE.
	SubTypeGoTo = "GoTo"
	// SubTypeGoToR is PDActionRemoteGoTo.SUB_TYPE.
	SubTypeGoToR = "GoToR"
	// SubTypeGoToE is PDActionEmbeddedGoTo.SUB_TYPE.
	SubTypeGoToE = "GoToE"
	// SubTypeLaunch is PDActionLaunch.SUB_TYPE.
	SubTypeLaunch = "Launch"
	// SubTypeThread is PDActionThread.SUB_TYPE.
	SubTypeThread = "Thread"
	// SubTypeURI is PDActionURI.SUB_TYPE.
	SubTypeURI = "URI"
	// SubTypeSound is PDActionSound.SUB_TYPE.
	SubTypeSound = "Sound"
	// SubTypeMovie is PDActionMovie.SUB_TYPE.
	SubTypeMovie = "Movie"
	// SubTypeHide is PDActionHide.SUB_TYPE.
	SubTypeHide = "Hide"
	// SubTypeNamed is PDActionNamed.SUB_TYPE.
	SubTypeNamed = "Named"
	// SubTypeSubmitForm is PDActionSubmitForm.SUB_TYPE.
	SubTypeSubmitForm = "SubmitForm"
	// SubTypeResetForm is PDActionResetForm.SUB_TYPE.
	SubTypeResetForm = "ResetForm"
	// SubTypeImportData is PDActionImportData.SUB_TYPE.
	SubTypeImportData = "ImportData"
	// SubTypeJavaScript is PDActionJavaScript.SUB_TYPE.
	SubTypeJavaScript = "JavaScript"
)

// Action is what an action is, whatever its subtype.
//
// Java's PDAction is an abstract class; the port splits it into this interface
// for the contract and PDAction below for the state.
type Action interface {
	common.PDDestinationOrAction

	// ActionDictionary returns the action dictionary, which getCOSObject
	// narrows to in Java.
	ActionDictionary() *cos.Dictionary

	// Type returns the /Type, which is always "Action".
	Type() string

	// SubType returns the /S subtype.
	SubType() string
}

// PDAction carries the state and the concrete methods every action shares.
//
// Port of the non-abstract half of PDAction.
type PDAction struct {
	// Action is the protected `action` field the subclasses write through.
	Action *cos.Dictionary
}

var _ Action = (*PDAction)(nil)

// InitAction is the protected PDAction() constructor.
func (a *PDAction) InitAction() {
	a.Action = cos.NewDictionary()
	a.SetType(TypeAction)
}

// InitActionOf is the protected PDAction(COSDictionary) constructor.
func (a *PDAction) InitActionOf(dict *cos.Dictionary) {
	a.Action = dict
}

// COSObject returns the action dictionary.
func (a *PDAction) COSObject() cos.Base { return a.Action }

// ActionDictionary returns the action dictionary, typed.
func (a *PDAction) ActionDictionary() *cos.Dictionary { return a.Action }

// Type returns the /Type, which is always "Action".
func (a *PDAction) Type() string {
	return a.Action.GetNameAsString(cos.Type, "")
}

// SetType sets the /Type. Java declares it protected and final.
func (a *PDAction) SetType(actionType string) {
	a.Action.SetName(cos.Type, actionType)
}

// SubType returns the /S subtype.
func (a *PDAction) SubType() string {
	return a.Action.GetNameAsString(cos.S, "")
}

// SetSubType sets the /S subtype. Java declares it protected and final.
func (a *PDAction) SetSubType(s string) {
	a.Action.SetName(cos.S, s)
}

// Next returns the actions performed after this one, or nil where there are
// none.
func (a *PDAction) Next() *common.COSArrayList[Action] {
	next := a.Action.GetDictionaryObject(cos.Next)
	switch value := next.(type) {
	case *cos.Stream:
		// COSStream is a COSDictionary in Java, so it takes the same branch.
		pdAction := CreateAction(&value.Dictionary)
		return common.NewCOSArrayListOfItem(pdAction, next, a.Action, cos.Next)
	case *cos.Dictionary:
		pdAction := CreateAction(value)
		return common.NewCOSArrayListOfItem(pdAction, next, a.Action, cos.Next)
	case *cos.Array:
		actions := make([]Action, 0, value.Size())
		for i := 0; i < value.Size(); i++ {
			// Java casts each element to COSDictionary without a check and
			// throws ClassCastException where it is not one; the port asserts
			// the same way.
			actions = append(actions, CreateAction(mustDictionary(value.GetObject(i))))
		}
		return common.NewCOSArrayListOf(actions, value)
	}
	return nil
}

// mustDictionary is Java's unchecked `(COSDictionary) base` cast.
func mustDictionary(base cos.Base) *cos.Dictionary {
	if stream, ok := base.(*cos.Stream); ok {
		return &stream.Dictionary
	}
	return base.(*cos.Dictionary)
}

// SetNext sets the actions performed after this one.
func (a *PDAction) SetNext(next []Action) {
	array := cos.NewArray()
	for _, action := range next {
		array.Add(action.COSObject())
	}
	a.Action.SetItem(cos.Next, array)
}

// CreateAction returns the action the given dictionary holds, or nil where the
// dictionary is nil or its subtype is not one of the fourteen.
//
// Port of the static PDActionFactory.createAction.
func CreateAction(action *cos.Dictionary) Action {
	if action == nil {
		return nil
	}
	switch action.GetNameAsString(cos.S, "") {
	case SubTypeJavaScript:
		return NewPDActionJavaScriptOf(action)
	case SubTypeGoTo:
		return NewPDActionGoToOf(action)
	case SubTypeLaunch:
		return NewPDActionLaunchOf(action)
	case SubTypeGoToR:
		return NewPDActionRemoteGoToOf(action)
	case SubTypeURI:
		return NewPDActionURIOf(action)
	case SubTypeNamed:
		return NewPDActionNamedOf(action)
	case SubTypeSound:
		return NewPDActionSoundOf(action)
	case SubTypeMovie:
		return NewPDActionMovieOf(action)
	case SubTypeImportData:
		return NewPDActionImportDataOf(action)
	case SubTypeResetForm:
		return NewPDActionResetFormOf(action)
	case SubTypeHide:
		return NewPDActionHideOf(action)
	case SubTypeSubmitForm:
		return NewPDActionSubmitFormOf(action)
	case SubTypeThread:
		return NewPDActionThreadOf(action)
	case SubTypeGoToE:
		return NewPDActionEmbeddedGoToOf(action)
	}
	return nil
}

// OpenMode is where a remote or embedded destination is opened.
//
// Port of the enum OpenMode.
type OpenMode int

const (
	// OpenModeUserPreference leaves it to the viewer.
	OpenModeUserPreference OpenMode = iota
	// OpenModeSameWindow opens in the same window.
	OpenModeSameWindow
	// OpenModeNewWindow opens in a new window.
	OpenModeNewWindow
)
