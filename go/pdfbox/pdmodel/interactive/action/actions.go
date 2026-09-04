package action

import (
	"strings"
	"unicode/utf8"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common/filespecification"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/documentnavigation/destination"
)

// openInNewWindow is the getOpenInNewWindow that PDActionLaunch,
// PDActionRemoteGoTo and PDActionEmbeddedGoTo each declare identically.
func openInNewWindow(dict *cos.Dictionary) OpenMode {
	if b, ok := dict.GetDictionaryObject(cos.NewWindow).(*cos.Boolean); ok {
		if b.Value() {
			return OpenModeNewWindow
		}
		return OpenModeSameWindow
	}
	return OpenModeUserPreference
}

// setOpenInNewWindow is the matching setter the same three declare.
func setOpenInNewWindow(dict *cos.Dictionary, value OpenMode) {
	switch value {
	case OpenModeUserPreference:
		dict.RemoveItem(cos.NewWindow)
	case OpenModeSameWindow:
		dict.SetBoolean(cos.NewWindow, false)
	case OpenModeNewWindow:
		dict.SetBoolean(cos.NewWindow, true)
	}
}

// PDActionGoTo goes to a destination in this document.
type PDActionGoTo struct{ PDAction }

var _ Action = (*PDActionGoTo)(nil)

// NewPDActionGoTo creates a new /GoTo action.
func NewPDActionGoTo() *PDActionGoTo {
	a := &PDActionGoTo{}
	a.InitAction()
	a.SetSubType(SubTypeGoTo)
	return a
}

// NewPDActionGoToOf creates one over the given dictionary.
func NewPDActionGoToOf(dict *cos.Dictionary) *PDActionGoTo {
	a := &PDActionGoTo{}
	a.InitActionOf(dict)
	return a
}

// Destination returns the /D destination.
func (a *PDActionGoTo) Destination() (destination.PDDestination, error) {
	return destination.Create(a.Action.GetDictionaryObject(cos.D))
}

// SetDestination sets the /D destination.
//
// Java throws IllegalArgumentException where a page destination names anything
// but a page dictionary, which is unchecked, so the port panics.
func (a *PDActionGoTo) SetDestination(d destination.PDDestination) {
	if pageDest, ok := d.(interface{ COSObject() cos.Base }); ok {
		if destArray, isArray := pageDest.COSObject().(*cos.Array); isArray && !destArray.IsEmpty() {
			if _, isDictionary := asDictionary(destArray.GetObject(0)); !isDictionary {
				panic("Destination of a GoTo action must be a page dictionary object")
			}
		}
	}
	setItemOrNil(a.Action, cos.D, d)
}

// asDictionary is Java's `instanceof COSDictionary`, which a COSStream also
// satisfies.
func asDictionary(base cos.Base) (*cos.Dictionary, bool) {
	switch value := base.(type) {
	case *cos.Stream:
		return &value.Dictionary, true
	case *cos.Dictionary:
		return value, true
	}
	return nil, false
}

// setItemOrNil stores a COSObjectable, or clears the entry where it is nil.
// Java's setItem(COSName, COSObjectable) does the same with a null.
func setItemOrNil(dict *cos.Dictionary, key *cos.Name, value common.COSObjectable) {
	if value == nil {
		dict.SetItem(key, nil)
		return
	}
	dict.SetItem(key, value.COSObject())
}

// PDActionRemoteGoTo goes to a destination in another document.
type PDActionRemoteGoTo struct{ PDAction }

var _ Action = (*PDActionRemoteGoTo)(nil)

// NewPDActionRemoteGoTo creates a new /GoToR action.
func NewPDActionRemoteGoTo() *PDActionRemoteGoTo {
	a := &PDActionRemoteGoTo{}
	a.InitAction()
	a.SetSubType(SubTypeGoToR)
	return a
}

// NewPDActionRemoteGoToOf creates one over the given dictionary.
func NewPDActionRemoteGoToOf(dict *cos.Dictionary) *PDActionRemoteGoTo {
	a := &PDActionRemoteGoTo{}
	a.InitActionOf(dict)
	return a
}

// File returns the /F file this action opens.
func (a *PDActionRemoteGoTo) File() (filespecification.PDFileSpecification, error) {
	return filespecification.CreateFS(a.Action.GetDictionaryObject(cos.F))
}

// SetFile sets the /F file this action opens.
func (a *PDActionRemoteGoTo) SetFile(fs filespecification.PDFileSpecification) {
	setItemOrNil(a.Action, cos.F, fs)
}

// D returns the /D destination, which is an array or a string.
func (a *PDActionRemoteGoTo) D() cos.Base { return a.Action.GetDictionaryObject(cos.D) }

// SetD sets the /D destination.
func (a *PDActionRemoteGoTo) SetD(d cos.Base) { a.Action.SetItem(cos.D, d) }

// OpenInNewWindow returns where the document is opened.
func (a *PDActionRemoteGoTo) OpenInNewWindow() OpenMode { return openInNewWindow(a.Action) }

// SetOpenInNewWindow sets where the document is opened.
func (a *PDActionRemoteGoTo) SetOpenInNewWindow(value OpenMode) {
	setOpenInNewWindow(a.Action, value)
}

// PDActionEmbeddedGoTo goes to a destination in an embedded document.
type PDActionEmbeddedGoTo struct{ PDAction }

var _ Action = (*PDActionEmbeddedGoTo)(nil)

// NewPDActionEmbeddedGoTo creates a new /GoToE action.
func NewPDActionEmbeddedGoTo() *PDActionEmbeddedGoTo {
	a := &PDActionEmbeddedGoTo{}
	a.InitAction()
	a.SetSubType(SubTypeGoToE)
	return a
}

// NewPDActionEmbeddedGoToOf creates one over the given dictionary.
func NewPDActionEmbeddedGoToOf(dict *cos.Dictionary) *PDActionEmbeddedGoTo {
	a := &PDActionEmbeddedGoTo{}
	a.InitActionOf(dict)
	return a
}

// Destination returns the /D destination.
func (a *PDActionEmbeddedGoTo) Destination() (destination.PDDestination, error) {
	return destination.Create(a.Action.GetDictionaryObject(cos.D))
}

// SetDestination sets the /D destination.
//
// Java throws IllegalArgumentException where a page destination names anything
// but an integer, which is unchecked, so the port panics.
func (a *PDActionEmbeddedGoTo) SetDestination(d destination.PDDestination) {
	if d != nil {
		if destArray, isArray := d.COSObject().(*cos.Array); isArray && !destArray.IsEmpty() {
			if _, isInteger := destArray.GetObject(0).(*cos.Integer); !isInteger {
				panic("Destination of a GoToE action must be an integer")
			}
		}
	}
	setItemOrNil(a.Action, cos.D, d)
}

// File returns the /F file this action opens.
func (a *PDActionEmbeddedGoTo) File() (filespecification.PDFileSpecification, error) {
	return filespecification.CreateFS(a.Action.GetDictionaryObject(cos.F))
}

// SetFile sets the /F file this action opens.
func (a *PDActionEmbeddedGoTo) SetFile(fs filespecification.PDFileSpecification) {
	setItemOrNil(a.Action, cos.F, fs)
}

// OpenInNewWindow returns where the document is opened.
func (a *PDActionEmbeddedGoTo) OpenInNewWindow() OpenMode { return openInNewWindow(a.Action) }

// SetOpenInNewWindow sets where the document is opened.
func (a *PDActionEmbeddedGoTo) SetOpenInNewWindow(value OpenMode) {
	setOpenInNewWindow(a.Action, value)
}

// TargetDirectory returns the /T target directory, or nil.
func (a *PDActionEmbeddedGoTo) TargetDirectory() *PDTargetDirectory {
	if targetDict := a.Action.GetCOSDictionary(cos.T); targetDict != nil {
		return NewPDTargetDirectoryOf(targetDict)
	}
	return nil
}

// SetTargetDirectory sets the /T target directory.
func (a *PDActionEmbeddedGoTo) SetTargetDirectory(targetDirectory *PDTargetDirectory) {
	if targetDirectory == nil {
		a.Action.SetItem(cos.T, nil)
		return
	}
	a.Action.SetItem(cos.T, targetDirectory.COSObject())
}

// PDActionLaunch launches an application.
type PDActionLaunch struct{ PDAction }

var _ Action = (*PDActionLaunch)(nil)

// NewPDActionLaunch creates a new /Launch action.
func NewPDActionLaunch() *PDActionLaunch {
	a := &PDActionLaunch{}
	a.InitAction()
	a.SetSubType(SubTypeLaunch)
	return a
}

// NewPDActionLaunchOf creates one over the given dictionary.
func NewPDActionLaunchOf(dict *cos.Dictionary) *PDActionLaunch {
	a := &PDActionLaunch{}
	a.InitActionOf(dict)
	return a
}

// File returns the /F file this action launches.
func (a *PDActionLaunch) File() (filespecification.PDFileSpecification, error) {
	return filespecification.CreateFS(a.Action.GetDictionaryObject(cos.F))
}

// SetFile sets the /F file this action launches.
func (a *PDActionLaunch) SetFile(fs filespecification.PDFileSpecification) {
	setItemOrNil(a.Action, cos.F, fs)
}

// WinLaunchParams returns the /Win parameters, or nil.
func (a *PDActionLaunch) WinLaunchParams() *PDWindowsLaunchParams {
	if win := a.Action.GetCOSDictionary(cos.Win); win != nil {
		return NewPDWindowsLaunchParamsOf(win)
	}
	return nil
}

// SetWinLaunchParams sets the /Win parameters.
func (a *PDActionLaunch) SetWinLaunchParams(win *PDWindowsLaunchParams) {
	if win == nil {
		a.Action.SetItem(cos.Win, nil)
		return
	}
	a.Action.SetItem(cos.Win, win.COSObject())
}

// F returns the /F entry as a string.
func (a *PDActionLaunch) F() string { return a.Action.GetString(cos.F, "") }

// SetF sets the /F entry as a string.
func (a *PDActionLaunch) SetF(f string) { a.Action.SetString(cos.F, f) }

// D returns the /D entry.
func (a *PDActionLaunch) D() string { return a.Action.GetString(cos.D, "") }

// SetD sets the /D entry.
func (a *PDActionLaunch) SetD(d string) { a.Action.SetString(cos.D, d) }

// O returns the /O entry.
func (a *PDActionLaunch) O() string { return a.Action.GetString(cos.O, "") }

// SetO sets the /O entry.
func (a *PDActionLaunch) SetO(o string) { a.Action.SetString(cos.O, o) }

// P returns the /P entry.
func (a *PDActionLaunch) P() string { return a.Action.GetString(cos.P, "") }

// SetP sets the /P entry.
func (a *PDActionLaunch) SetP(p string) { a.Action.SetString(cos.P, p) }

// OpenInNewWindow returns where the document is opened.
func (a *PDActionLaunch) OpenInNewWindow() OpenMode { return openInNewWindow(a.Action) }

// SetOpenInNewWindow sets where the document is opened.
func (a *PDActionLaunch) SetOpenInNewWindow(value OpenMode) {
	setOpenInNewWindow(a.Action, value)
}

// PDActionThread jumps to an article thread.
type PDActionThread struct{ PDAction }

var _ Action = (*PDActionThread)(nil)

// NewPDActionThread creates a new /Thread action.
func NewPDActionThread() *PDActionThread {
	a := &PDActionThread{}
	a.InitAction()
	a.SetSubType(SubTypeThread)
	return a
}

// NewPDActionThreadOf creates one over the given dictionary.
func NewPDActionThreadOf(dict *cos.Dictionary) *PDActionThread {
	a := &PDActionThread{}
	a.InitActionOf(dict)
	return a
}

// D returns the /D entry, which is a dictionary, an integer or a string.
func (a *PDActionThread) D() cos.Base { return a.Action.GetDictionaryObject(cos.D) }

// SetD sets the /D entry.
func (a *PDActionThread) SetD(d cos.Base) { a.Action.SetItem(cos.D, d) }

// File returns the /F file the thread is in.
func (a *PDActionThread) File() (filespecification.PDFileSpecification, error) {
	return filespecification.CreateFS(a.Action.GetDictionaryObject(cos.F))
}

// SetFile sets the /F file the thread is in.
func (a *PDActionThread) SetFile(fs filespecification.PDFileSpecification) {
	setItemOrNil(a.Action, cos.F, fs)
}

// B returns the /B entry, which is a dictionary or an integer.
func (a *PDActionThread) B() cos.Base { return a.Action.GetDictionaryObject(cos.B) }

// SetB sets the /B entry.
func (a *PDActionThread) SetB(b cos.Base) { a.Action.SetItem(cos.B, b) }

// PDActionURI opens a URI.
type PDActionURI struct{ PDAction }

var _ Action = (*PDActionURI)(nil)

// NewPDActionURI creates a new /URI action.
func NewPDActionURI() *PDActionURI {
	a := &PDActionURI{}
	a.InitAction()
	a.SetSubType(SubTypeURI)
	return a
}

// NewPDActionURIOf creates one over the given dictionary.
func NewPDActionURIOf(dict *cos.Dictionary) *PDActionURI {
	a := &PDActionURI{}
	a.InitActionOf(dict)
	return a
}

// URI returns the /URI entry, or "" where there is none.
//
// A string with a UTF-16 byte order mark is read the way every other PDF text
// string is; anything else is read as UTF-8, which is not what the
// specification says a PDF string holds but is what Java does here.
func (a *PDActionURI) URI() string {
	base, ok := a.Action.GetDictionaryObject(cos.URI).(*cos.StringObj)
	if !ok {
		return ""
	}
	bytes := base.Bytes()
	if len(bytes) >= 2 {
		// UTF-16 (BE)
		if bytes[0] == 0xFE && bytes[1] == 0xFF {
			return a.Action.GetString(cos.URI, "")
		}
		// UTF-16 (LE)
		if bytes[0] == 0xFF && bytes[1] == 0xFE {
			return a.Action.GetString(cos.URI, "")
		}
	}
	// Java's new String(bytes, UTF_8) replaces invalid sequences rather than
	// failing; a Go conversion keeps them, so the port replaces them too.
	return toValidUTF8(bytes)
}

// toValidUTF8 is Java's new String(byte[], StandardCharsets.UTF_8): every byte
// sequence that is not valid UTF-8 becomes U+FFFD.
func toValidUTF8(b []byte) string {
	if utf8.Valid(b) {
		return string(b)
	}
	return strings.ToValidUTF8(string(b), "�")
}

// SetURI sets the /URI entry.
func (a *PDActionURI) SetURI(uri string) { a.Action.SetString(cos.URI, uri) }

// ShouldTrackMousePosition reports the /IsMap entry.
func (a *PDActionURI) ShouldTrackMousePosition() bool {
	return a.Action.GetBoolean(cos.GetPDFName("IsMap"), false)
}

// SetTrackMousePosition sets the /IsMap entry.
func (a *PDActionURI) SetTrackMousePosition(value bool) {
	a.Action.SetBoolean(cos.GetPDFName("IsMap"), value)
}

// PDActionSound plays a sound.
type PDActionSound struct{ PDAction }

var _ Action = (*PDActionSound)(nil)

// NewPDActionSound creates a new /Sound action.
func NewPDActionSound() *PDActionSound {
	a := &PDActionSound{}
	a.InitAction()
	a.SetSubType(SubTypeSound)
	return a
}

// NewPDActionSoundOf creates one over the given dictionary.
func NewPDActionSoundOf(dict *cos.Dictionary) *PDActionSound {
	a := &PDActionSound{}
	a.InitActionOf(dict)
	return a
}

// SetSound sets the /Sound stream.
func (a *PDActionSound) SetSound(sound *cos.Stream) {
	if sound == nil {
		a.Action.SetItem(cos.Sound, nil)
		return
	}
	a.Action.SetItem(cos.Sound, sound)
}

// Sound returns the /Sound stream, or nil.
func (a *PDActionSound) Sound() *cos.Stream {
	stream, _ := a.Action.GetDictionaryObject(cos.Sound).(*cos.Stream)
	return stream
}

// SetVolume sets the /Volume, which must be between -1 and 1.
//
// Java throws IllegalArgumentException outside that range, which is unchecked,
// so the port panics.
func (a *PDActionSound) SetVolume(volume float32) {
	if volume < -1 || volume > 1 {
		panic("volume outside of the range −1.0 to 1.0")
	}
	a.Action.SetFloat(cos.Volume, volume)
}

// Volume returns the /Volume, which defaults to 1 and is clamped to 1 where the
// file holds something outside the range.
func (a *PDActionSound) Volume() float32 {
	volume := a.Action.GetFloat(cos.Volume, 1)
	if volume < -1 || volume > 1 {
		return 1
	}
	return volume
}

// SetSynchronous sets the /Synchronous entry.
func (a *PDActionSound) SetSynchronous(synchronous bool) {
	a.Action.SetBoolean(cos.Synchronous, synchronous)
}

// Synchronous reports the /Synchronous entry.
func (a *PDActionSound) Synchronous() bool {
	return a.Action.GetBoolean(cos.Synchronous, false)
}

// SetRepeat sets the /Repeat entry.
func (a *PDActionSound) SetRepeat(repeat bool) { a.Action.SetBoolean(cos.Repeat, repeat) }

// Repeat reports the /Repeat entry.
func (a *PDActionSound) Repeat() bool { return a.Action.GetBoolean(cos.Repeat, false) }

// SetMix sets the /Mix entry.
func (a *PDActionSound) SetMix(mix bool) { a.Action.SetBoolean(cos.Mix, mix) }

// Mix reports the /Mix entry.
func (a *PDActionSound) Mix() bool { return a.Action.GetBoolean(cos.Mix, false) }

// PDActionMovie plays a movie.
//
// Java declares no members beyond the two constructors.
type PDActionMovie struct{ PDAction }

var _ Action = (*PDActionMovie)(nil)

// NewPDActionMovie creates a new /Movie action.
func NewPDActionMovie() *PDActionMovie {
	a := &PDActionMovie{}
	a.InitAction()
	a.SetSubType(SubTypeMovie)
	return a
}

// NewPDActionMovieOf creates one over the given dictionary.
func NewPDActionMovieOf(dict *cos.Dictionary) *PDActionMovie {
	a := &PDActionMovie{}
	a.InitActionOf(dict)
	return a
}

// PDActionHide hides or shows annotations.
type PDActionHide struct{ PDAction }

var _ Action = (*PDActionHide)(nil)

// NewPDActionHide creates a new /Hide action.
func NewPDActionHide() *PDActionHide {
	a := &PDActionHide{}
	a.InitAction()
	a.SetSubType(SubTypeHide)
	return a
}

// NewPDActionHideOf creates one over the given dictionary.
func NewPDActionHideOf(dict *cos.Dictionary) *PDActionHide {
	a := &PDActionHide{}
	a.InitActionOf(dict)
	return a
}

// T returns the /T entry, which is a dictionary, a string or an array.
func (a *PDActionHide) T() cos.Base { return a.Action.GetDictionaryObject(cos.T) }

// SetT sets the /T entry.
func (a *PDActionHide) SetT(t cos.Base) { a.Action.SetItem(cos.T, t) }

// H reports the /H entry, which defaults to true.
func (a *PDActionHide) H() bool { return a.Action.GetBoolean(cos.H, true) }

// SetH sets the /H entry.
func (a *PDActionHide) SetH(h bool) { a.Action.SetItem(cos.H, cos.GetBoolean(h)) }

// PDActionResetForm resets a form's fields.
type PDActionResetForm struct{ PDAction }

var _ Action = (*PDActionResetForm)(nil)

// NewPDActionResetForm creates a new /ResetForm action.
func NewPDActionResetForm() *PDActionResetForm {
	a := &PDActionResetForm{}
	a.InitAction()
	a.SetSubType(SubTypeResetForm)
	return a
}

// NewPDActionResetFormOf creates one over the given dictionary.
func NewPDActionResetFormOf(dict *cos.Dictionary) *PDActionResetForm {
	a := &PDActionResetForm{}
	a.InitActionOf(dict)
	return a
}

// Fields returns the /Fields array, or nil.
func (a *PDActionResetForm) Fields() *cos.Array { return a.Action.GetCOSArray(cos.Fields) }

// SetFields sets the /Fields array.
func (a *PDActionResetForm) SetFields(array *cos.Array) { a.Action.SetItem(cos.Fields, array) }

// Flags returns the /Flags entry, which defaults to 0.
func (a *PDActionResetForm) Flags() int { return a.Action.GetIntDefault(cos.Flags, 0) }

// SetFlags sets the /Flags entry.
func (a *PDActionResetForm) SetFlags(flags int) { a.Action.SetInt(cos.Flags, flags) }

// PDActionImportData imports form data from a file.
type PDActionImportData struct{ PDAction }

var _ Action = (*PDActionImportData)(nil)

// NewPDActionImportData creates a new /ImportData action.
func NewPDActionImportData() *PDActionImportData {
	a := &PDActionImportData{}
	a.InitAction()
	a.SetSubType(SubTypeImportData)
	return a
}

// NewPDActionImportDataOf creates one over the given dictionary.
func NewPDActionImportDataOf(dict *cos.Dictionary) *PDActionImportData {
	a := &PDActionImportData{}
	a.InitActionOf(dict)
	return a
}

// File returns the /F file the data comes from.
func (a *PDActionImportData) File() (filespecification.PDFileSpecification, error) {
	return filespecification.CreateFS(a.Action.GetDictionaryObject(cos.F))
}

// SetFile sets the /F file the data comes from.
func (a *PDActionImportData) SetFile(fs filespecification.PDFileSpecification) {
	setItemOrNil(a.Action, cos.F, fs)
}

// PDActionSubmitForm submits a form.
type PDActionSubmitForm struct{ PDAction }

var _ Action = (*PDActionSubmitForm)(nil)

// NewPDActionSubmitForm creates a new /SubmitForm action.
func NewPDActionSubmitForm() *PDActionSubmitForm {
	a := &PDActionSubmitForm{}
	a.InitAction()
	a.SetSubType(SubTypeSubmitForm)
	return a
}

// NewPDActionSubmitFormOf creates one over the given dictionary.
func NewPDActionSubmitFormOf(dict *cos.Dictionary) *PDActionSubmitForm {
	a := &PDActionSubmitForm{}
	a.InitActionOf(dict)
	return a
}

// File returns the /F file the form is submitted to.
func (a *PDActionSubmitForm) File() (filespecification.PDFileSpecification, error) {
	return filespecification.CreateFS(a.Action.GetDictionaryObject(cos.F))
}

// SetFile sets the /F file the form is submitted to.
func (a *PDActionSubmitForm) SetFile(fs filespecification.PDFileSpecification) {
	setItemOrNil(a.Action, cos.F, fs)
}

// Fields returns the /Fields array, or nil.
func (a *PDActionSubmitForm) Fields() *cos.Array { return a.Action.GetCOSArray(cos.Fields) }

// SetFields sets the /Fields array.
func (a *PDActionSubmitForm) SetFields(array *cos.Array) { a.Action.SetItem(cos.Fields, array) }

// Flags returns the /Flags entry, which defaults to 0.
func (a *PDActionSubmitForm) Flags() int { return a.Action.GetIntDefault(cos.Flags, 0) }

// SetFlags sets the /Flags entry.
func (a *PDActionSubmitForm) SetFlags(flags int) { a.Action.SetInt(cos.Flags, flags) }

// PDActionJavaScript runs JavaScript.
type PDActionJavaScript struct{ PDAction }

var _ Action = (*PDActionJavaScript)(nil)

// NewPDActionJavaScript creates a new /JavaScript action.
func NewPDActionJavaScript() *PDActionJavaScript {
	a := &PDActionJavaScript{}
	a.InitAction()
	a.SetSubType(SubTypeJavaScript)
	return a
}

// NewPDActionJavaScriptOfSource creates one running the given source.
func NewPDActionJavaScriptOfSource(js string) *PDActionJavaScript {
	a := NewPDActionJavaScript()
	a.SetActionSource(js)
	return a
}

// NewPDActionJavaScriptOf creates one over the given dictionary.
func NewPDActionJavaScriptOf(dict *cos.Dictionary) *PDActionJavaScript {
	a := &PDActionJavaScript{}
	a.InitActionOf(dict)
	return a
}

// SetActionSource sets the /JS source.
//
// Java names it setAction, which would collide with the embedded field here.
func (a *PDActionJavaScript) SetActionSource(sAction string) {
	a.Action.SetString(cos.JS, sAction)
}

// ActionSource returns the /JS source, which may be a string or a stream, or ""
// where it is neither.
//
// Java names it getAction.
func (a *PDActionJavaScript) ActionSource() (string, error) {
	switch value := a.Action.GetDictionaryObject(cos.JS).(type) {
	case *cos.StringObj:
		return value.Value(), nil
	case *cos.Stream:
		return value.TextString()
	}
	return "", nil
}

// PDActionNamed performs one of the viewer's named actions.
type PDActionNamed struct{ PDAction }

var _ Action = (*PDActionNamed)(nil)

// NewPDActionNamed creates a new /Named action.
func NewPDActionNamed() *PDActionNamed {
	a := &PDActionNamed{}
	a.InitAction()
	a.SetSubType(SubTypeNamed)
	return a
}

// NewPDActionNamedOf creates one over the given dictionary.
func NewPDActionNamedOf(dict *cos.Dictionary) *PDActionNamed {
	a := &PDActionNamed{}
	a.InitActionOf(dict)
	return a
}

// N returns the /N name of the action.
func (a *PDActionNamed) N() string {
	return a.Action.GetNameAsString(cos.GetPDFName("N"), "")
}

// SetN sets the /N name of the action.
func (a *PDActionNamed) SetN(name string) {
	a.Action.SetName(cos.GetPDFName("N"), name)
}

// PDTargetDirectory names a document in the target hierarchy of an embedded
// go-to action.
//
// Port of PDTargetDirectory.
type PDTargetDirectory struct {
	dict *cos.Dictionary
}

var _ common.COSObjectable = (*PDTargetDirectory)(nil)

// NewPDTargetDirectory creates an empty target directory.
func NewPDTargetDirectory() *PDTargetDirectory {
	return &PDTargetDirectory{dict: cos.NewDictionary()}
}

// NewPDTargetDirectoryOf creates one over the given dictionary.
func NewPDTargetDirectoryOf(dictionary *cos.Dictionary) *PDTargetDirectory {
	return &PDTargetDirectory{dict: dictionary}
}

// COSObject returns the dictionary.
func (d *PDTargetDirectory) COSObject() cos.Base { return d.dict }

// Relationship returns the /R entry.
func (d *PDTargetDirectory) Relationship() *cos.Name { return d.dict.GetCOSName(cos.R) }

// SetRelationship sets the /R entry, which must be /P or /C.
//
// Java throws IllegalArgumentException otherwise, which is unchecked, so the
// port panics.
func (d *PDTargetDirectory) SetRelationship(relationship *cos.Name) {
	if relationship != cos.P && relationship != cos.C {
		panic("The only valid are P or C, not " + relationship.Name())
	}
	d.dict.SetItem(cos.R, relationship)
}

// Filename returns the /N entry.
func (d *PDTargetDirectory) Filename() string { return d.dict.GetString(cos.N, "") }

// SetFilename sets the /N entry.
func (d *PDTargetDirectory) SetFilename(filename string) { d.dict.SetString(cos.N, filename) }

// TargetDirectory returns the /T entry, or nil.
func (d *PDTargetDirectory) TargetDirectory() *PDTargetDirectory {
	if targetDict := d.dict.GetCOSDictionary(cos.T); targetDict != nil {
		return NewPDTargetDirectoryOf(targetDict)
	}
	return nil
}

// SetTargetDirectory sets the /T entry.
func (d *PDTargetDirectory) SetTargetDirectory(targetDirectory *PDTargetDirectory) {
	if targetDirectory == nil {
		d.dict.SetItem(cos.T, nil)
		return
	}
	d.dict.SetItem(cos.T, targetDirectory.COSObject())
}

// PageNumber returns the /P entry as a page number, or -1.
func (d *PDTargetDirectory) PageNumber() int { return d.dict.GetIntDefault(cos.P, -1) }

// SetPageNumber sets the /P entry as a page number; a negative value clears it.
func (d *PDTargetDirectory) SetPageNumber(pageNumber int) {
	if pageNumber < 0 {
		d.dict.RemoveItem(cos.P)
	} else {
		d.dict.SetInt(cos.P, pageNumber)
	}
}

// NamedDestination returns the /P entry as a named destination, or nil.
func (d *PDTargetDirectory) NamedDestination() *destination.PDNamedDestination {
	if base, ok := d.dict.GetDictionaryObject(cos.P).(*cos.StringObj); ok {
		return destination.NewPDNamedDestinationOfString(base)
	}
	return nil
}

// SetNamedDestination sets the /P entry as a named destination.
func (d *PDTargetDirectory) SetNamedDestination(dest *destination.PDNamedDestination) {
	if dest == nil {
		d.dict.RemoveItem(cos.P)
	} else {
		d.dict.SetItem(cos.P, dest.COSObject())
	}
}

// AnnotationIndex returns the /A entry as an index, or -1.
func (d *PDTargetDirectory) AnnotationIndex() int { return d.dict.GetIntDefault(cos.A, -1) }

// SetAnnotationIndex sets the /A entry as an index; a negative value clears it.
func (d *PDTargetDirectory) SetAnnotationIndex(index int) {
	if index < 0 {
		d.dict.RemoveItem(cos.A)
	} else {
		d.dict.SetInt(cos.A, index)
	}
}

// AnnotationName returns the /A entry as a name.
func (d *PDTargetDirectory) AnnotationName() string { return d.dict.GetString(cos.A, "") }

// SetAnnotationName sets the /A entry as a name.
func (d *PDTargetDirectory) SetAnnotationName(name string) { d.dict.SetString(cos.A, name) }

// PDURIDictionary is the /URI entry of a document catalogue.
//
// Port of PDURIDictionary.
type PDURIDictionary struct {
	uriDictionary *cos.Dictionary
}

var _ common.COSObjectable = (*PDURIDictionary)(nil)

// NewPDURIDictionary creates an empty URI dictionary.
func NewPDURIDictionary() *PDURIDictionary {
	return &PDURIDictionary{uriDictionary: cos.NewDictionary()}
}

// NewPDURIDictionaryOf creates one over the given dictionary.
func NewPDURIDictionaryOf(dictionary *cos.Dictionary) *PDURIDictionary {
	return &PDURIDictionary{uriDictionary: dictionary}
}

// COSObject returns the dictionary.
func (d *PDURIDictionary) COSObject() cos.Base { return d.uriDictionary }

// Base returns the /Base entry.
func (d *PDURIDictionary) Base() string {
	return d.uriDictionary.GetString(cos.GetPDFName("Base"), "")
}

// SetBase sets the /Base entry.
func (d *PDURIDictionary) SetBase(base string) {
	d.uriDictionary.SetString(cos.GetPDFName("Base"), base)
}

// The two operations a Windows launch can perform.
const (
	// OperationOpen opens the file.
	OperationOpen = "open"
	// OperationPrint prints the file.
	OperationPrint = "print"
)

// PDWindowsLaunchParams is the /Win parameters of a launch action.
//
// Port of PDWindowsLaunchParams.
type PDWindowsLaunchParams struct {
	// Params is the protected `params` field.
	Params *cos.Dictionary
}

var _ common.COSObjectable = (*PDWindowsLaunchParams)(nil)

// NewPDWindowsLaunchParams creates empty parameters.
func NewPDWindowsLaunchParams() *PDWindowsLaunchParams {
	return &PDWindowsLaunchParams{Params: cos.NewDictionary()}
}

// NewPDWindowsLaunchParamsOf creates parameters over the given dictionary.
func NewPDWindowsLaunchParamsOf(p *cos.Dictionary) *PDWindowsLaunchParams {
	return &PDWindowsLaunchParams{Params: p}
}

// COSObject returns the dictionary.
func (p *PDWindowsLaunchParams) COSObject() cos.Base { return p.Params }

// Filename returns the /F entry.
func (p *PDWindowsLaunchParams) Filename() string { return p.Params.GetString(cos.F, "") }

// SetFilename sets the /F entry.
func (p *PDWindowsLaunchParams) SetFilename(file string) { p.Params.SetString(cos.F, file) }

// Directory returns the /D entry.
func (p *PDWindowsLaunchParams) Directory() string { return p.Params.GetString(cos.D, "") }

// SetDirectory sets the /D entry.
func (p *PDWindowsLaunchParams) SetDirectory(dir string) { p.Params.SetString(cos.D, dir) }

// Operation returns the /O entry, which defaults to "open".
func (p *PDWindowsLaunchParams) Operation() string {
	return p.Params.GetString(cos.O, OperationOpen)
}

// SetOperation sets the operation.
//
// JAVA BUG 36: it writes /D, which is the directory, and Operation reads /O. So
// setting the operation overwrites the directory and does not change what
// Operation returns. Ported as written; see migration/JAVA-BUGS.md.
func (p *PDWindowsLaunchParams) SetOperation(op string) { p.Params.SetString(cos.D, op) }

// ExecuteParam returns the /P entry.
func (p *PDWindowsLaunchParams) ExecuteParam() string { return p.Params.GetString(cos.P, "") }

// SetExecuteParam sets the /P entry.
func (p *PDWindowsLaunchParams) SetExecuteParam(param string) { p.Params.SetString(cos.P, param) }
