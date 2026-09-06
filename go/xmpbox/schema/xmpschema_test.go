package schema_test

// Port of org.apache.xmpbox.schema.XMPSchemaTest.

import (
	"errors"
	"testing"
	"time"

	"github.com/shinguakira/pdfbox-go/go/xmpbox"
	"github.com/shinguakira/pdfbox-go/go/xmpbox/schema"
	"github.com/shinguakira/pdfbox-go/go/xmpbox/xmptype"
)

// bareSchema is the field the Java test initialises once: a schema of a
// namespace nothing declares.
func bareSchema(t *testing.T) (*xmpbox.XMPMetadata, *schema.XMPSchema) {
	t.Helper()
	parent := xmpbox.CreateXMPMetadata()
	schem, err := schema.NewXMPSchemaOfNamespace(parent, "nsURI", "nsSchem")
	if err != nil {
		t.Fatalf("NewXMPSchemaOfNamespace: %v", err)
	}
	return parent, schem
}

func TestBagManagement(t *testing.T) {
	_, schem := bareSchema(t)
	const bagName = "BAGTEST"
	const value1 = "valueOne"
	const value2 = "valueTwo"

	text, err := schem.Metadata().TypeMapping().CreateText("", "rdf", "li", value1)
	noError(t, "CreateText", err)
	schem.AddBagField(bagName, text)
	noError(t, "AddQualifiedBagValue", schem.AddQualifiedBagValue(bagName, value2))

	values := schem.UnqualifiedBagValueList(bagName)
	if len(values) != 2 {
		t.Fatalf("UnqualifiedBagValueList = %v, want two values", values)
	}
	equal(t, "values[0]", values[0], value1)
	equal(t, "values[1]", values[1], value2)

	schem.RemoveUnqualifiedBagValue(bagName, value1)
	values2 := schem.UnqualifiedBagValueList(bagName)
	if len(values2) != 1 {
		t.Fatalf("UnqualifiedBagValueList = %v, want one value", values2)
	}
	equal(t, "values2[0]", values2[0], value2)
}

func TestArrayList(t *testing.T) {
	_, schem := bareSchema(t)
	meta := xmpbox.CreateXMPMetadata()
	tm := meta.TypeMapping()
	newSeq := tm.CreateArrayProperty("", "nsSchem", "seqType", xmptype.Seq)
	li1, err := tm.CreateText("", "rdf", "li", "valeur1")
	noError(t, "CreateText", err)
	li2, err := tm.CreateText("", "rdf", "li", "valeur2")
	noError(t, "CreateText", err)
	newSeq.Container().AddProperty(li1)
	newSeq.Container().AddProperty(li2)
	schem.AddProperty(newSeq)

	list, err := schem.UnqualifiedArrayList("seqType")
	noError(t, "UnqualifiedArrayList", err)
	if !holdsField(list, li1) || !holdsField(list, li2) {
		t.Errorf("UnqualifiedArrayList = %v, want both text properties", list)
	}
}

// holdsField reports whether the list holds the field, which is Java's
// List.contains with the identity equals AbstractField inherits.
func holdsField(list []xmptype.AbstractField, wanted xmptype.AbstractField) bool {
	for _, held := range list {
		if held == wanted {
			return true
		}
	}
	return false
}

func TestSeqManagement(t *testing.T) {
	parent, schem := bareSchema(t)
	date := time.Date(2008, time.July, 6, 5, 4, 3, 0, time.UTC)
	boolean, err := parent.TypeMapping().CreateBoolean("", "rdf", "li", true)
	noError(t, "CreateBoolean", err)
	const textVal = "seqValue"
	const seqName = "SEQNAME"

	noError(t, "AddUnqualifiedSequenceDateValue",
		schem.AddUnqualifiedSequenceDateValue(seqName, date))
	schem.AddUnqualifiedSequenceField(seqName, boolean)
	noError(t, "AddUnqualifiedSequenceValue",
		schem.AddUnqualifiedSequenceValue(seqName, textVal))

	dates := schem.UnqualifiedSequenceDateValueList(seqName)
	if len(dates) != 1 {
		t.Fatalf("UnqualifiedSequenceDateValueList = %v, want one date", dates)
	}
	if !dates[0].Equal(date) {
		t.Errorf("dates[0] = %v, want %v", dates[0], date)
	}

	values := schem.UnqualifiedSequenceValueList(seqName)
	if len(values) != 3 {
		t.Fatalf("UnqualifiedSequenceValueList = %v, want three values", values)
	}
	equal(t, "values[0]", values[0], xmpbox.ToISO8601(date))
	equal(t, "values[1]", values[1], boolean.StringValue())
	equal(t, "values[2]", values[2], textVal)

	schem.RemoveUnqualifiedSequenceDateValue(seqName, date)
	if left := schem.UnqualifiedSequenceDateValueList(seqName); len(left) != 0 {
		t.Errorf("UnqualifiedSequenceDateValueList = %v, want none left", left)
	}
	schem.RemoveUnqualifiedSequenceValue(seqName, boolean.StringValue())
	schem.RemoveUnqualifiedSequenceValue(seqName, textVal)
	if left := schem.UnqualifiedSequenceValueList(seqName); len(left) != 0 {
		t.Errorf("UnqualifiedSequenceValueList = %v, want none left", left)
	}
}

func TestRdfAbout(t *testing.T) {
	_, schem := bareSchema(t)
	equal(t, "AboutValue()", schem.AboutValue(), "")
	const about = "about"
	schem.SetAboutAsSimple(about)
	equal(t, "AboutValue()", schem.AboutValue(), about)
	schem.SetAboutAsSimple("")
	equal(t, "AboutValue()", schem.AboutValue(), "")
	// Java's third call passes null, which the port spells as the empty string.
	schem.SetAboutAsSimple("")
	equal(t, "AboutValue()", schem.AboutValue(), "")
}

func TestBadRdfAbout(t *testing.T) {
	_, schem := bareSchema(t)
	err := schem.SetAbout(xmptype.NewAttribute("", "about", ""))
	if !errors.Is(err, xmptype.ErrBadFieldValue) {
		t.Errorf("SetAbout = %v, want a bad field value", err)
	}
}

func TestSetSpecifiedSimpleTypeProperty(t *testing.T) {
	_, schem := bareSchema(t)
	const prop = "testprop"
	const val = "value"
	const val2 = "value2"

	noError(t, "SetTextPropertyValueAsSimple", schem.SetTextPropertyValueAsSimple(prop, val))
	got, err := schem.UnqualifiedTextPropertyValue(prop)
	noError(t, "UnqualifiedTextPropertyValue", err)
	equal(t, "UnqualifiedTextPropertyValue", got, val)

	noError(t, "SetTextPropertyValueAsSimple", schem.SetTextPropertyValueAsSimple(prop, val2))
	got, err = schem.UnqualifiedTextPropertyValue(prop)
	noError(t, "UnqualifiedTextPropertyValue", err)
	equal(t, "UnqualifiedTextPropertyValue", got, val2)

	// Java sets null, which removes the property; a Go string cannot be null,
	// so the removal has a method of its own.
	schem.RemoveUnqualifiedProperty(prop)
	text, err := schem.UnqualifiedTextProperty(prop)
	noError(t, "UnqualifiedTextProperty", err)
	if text != nil {
		t.Errorf("UnqualifiedTextProperty = %v, want nothing", text)
	}
}

func TestSpecifiedSimplePropertyFormer(t *testing.T) {
	_, schem := bareSchema(t)
	const prop = "testprop"
	const val = "value"
	const val2 = "value2"

	noError(t, "SetTextPropertyValueAsSimple", schem.SetTextPropertyValueAsSimple(prop, val))
	text, err := schem.Metadata().TypeMapping().CreateText("", schem.Prefix(), prop, val2)
	noError(t, "CreateText", err)
	schem.SetTextProperty(text)

	got, err := schem.UnqualifiedTextPropertyValue(prop)
	noError(t, "UnqualifiedTextPropertyValue", err)
	equal(t, "UnqualifiedTextPropertyValue", got, val2)

	stored, err := schem.UnqualifiedTextProperty(prop)
	noError(t, "UnqualifiedTextProperty", err)
	if stored != text {
		t.Errorf("UnqualifiedTextProperty = %v, want the property that was set", stored)
	}
}

func TestAsSimpleMethods(t *testing.T) {
	_, schem := bareSchema(t)
	const boolName = "bool"
	const boolVal = true
	const dateName = "date"
	dateVal := time.Date(2009, time.January, 2, 3, 4, 5, 0, time.UTC)
	const integ = "integer"
	const i = 1
	const langprop = "langprop"
	const lang = "x-default"
	const langVal = "langVal"
	const bagprop = "bagProp"
	const bagVal = "bagVal"
	const seqprop = "SeqProp"
	const seqPropVal = "seqval"
	const seqdate = "SeqDate"

	noError(t, "SetBooleanPropertyValueAsSimple",
		schem.SetBooleanPropertyValueAsSimple(boolName, boolVal))
	noError(t, "SetDatePropertyValueAsSimple",
		schem.SetDatePropertyValueAsSimple(dateName, dateVal))
	noError(t, "SetIntegerPropertyValueAsSimple",
		schem.SetIntegerPropertyValueAsSimple(integ, i))
	noError(t, "SetUnqualifiedLanguagePropertyValue",
		schem.SetUnqualifiedLanguagePropertyValue(langprop, lang, langVal))
	noError(t, "AddBagValueAsSimple", schem.AddBagValueAsSimple(bagprop, bagVal))
	noError(t, "AddUnqualifiedSequenceValue",
		schem.AddUnqualifiedSequenceValue(seqprop, seqPropVal))
	noError(t, "AddSequenceDateValueAsSimple",
		schem.AddSequenceDateValueAsSimple(seqdate, dateVal))

	boolProperty, err := schem.BooleanProperty(boolName)
	noError(t, "BooleanProperty", err)
	if got, isBool := boolProperty.Value().(bool); !isBool || got != boolVal {
		t.Errorf("BooleanProperty().Value() = %v, want %v", boolProperty.Value(), boolVal)
	}
	dateProperty, err := schem.DateProperty(dateName)
	noError(t, "DateProperty", err)
	if got, held := dateProperty.DateValue(); !held || !got.Equal(dateVal) {
		t.Errorf("DateProperty().Value() = %v, %v, want %v", got, held, dateVal)
	}
	integerProperty, err := schem.IntegerProperty(integ)
	noError(t, "IntegerProperty", err)
	equal(t, "IntegerProperty().StringValue()", integerProperty.StringValue(), "1")

	value, err := schem.UnqualifiedLanguagePropertyValue(langprop, lang)
	noError(t, "UnqualifiedLanguagePropertyValue", err)
	equal(t, "UnqualifiedLanguagePropertyValue", value, langVal)

	holdsAll(t, "UnqualifiedBagValueList", schem.UnqualifiedBagValueList(bagprop),
		[]string{bagVal})
	holdsAll(t, "UnqualifiedSequenceValueList", schem.UnqualifiedSequenceValueList(seqprop),
		[]string{seqPropVal})
	dates := schem.UnqualifiedSequenceDateValueList(seqdate)
	if len(dates) != 1 || !dates[0].Equal(dateVal) {
		t.Errorf("UnqualifiedSequenceDateValueList = %v, want %v", dates, dateVal)
	}
	languages, err := schem.UnqualifiedLanguagePropertyLanguagesValue(langprop)
	noError(t, "UnqualifiedLanguagePropertyLanguagesValue", err)
	holdsAll(t, "UnqualifiedLanguagePropertyLanguagesValue", languages, []string{lang})

	gotBool, held, err := schem.BooleanPropertyValueAsSimple(boolName)
	noError(t, "BooleanPropertyValueAsSimple", err)
	if !held || gotBool != boolVal {
		t.Errorf("BooleanPropertyValueAsSimple = %v, %v, want %v", gotBool, held, boolVal)
	}
	gotDate, held, err := schem.DatePropertyValueAsSimple(dateName)
	noError(t, "DatePropertyValueAsSimple", err)
	if !held || !gotDate.Equal(dateVal) {
		t.Errorf("DatePropertyValueAsSimple = %v, %v, want %v", gotDate, held, dateVal)
	}
	gotInt, held, err := schem.IntegerPropertyValueAsSimple(integ)
	noError(t, "IntegerPropertyValueAsSimple", err)
	if !held || gotInt != i {
		t.Errorf("IntegerPropertyValueAsSimple = %v, %v, want %v", gotInt, held, i)
	}
}

func TestProperties(t *testing.T) {
	parent, schem := bareSchema(t)
	equal(t, "Namespace()", schem.Namespace(), "nsURI")
	// In real cases, rdf ns will be declared before !
	schem.AddNamespace(xmptype.RDFNamespace, "rdf")

	const aboutVal = "aboutTest"
	schem.SetAboutAsSimple(aboutVal)
	equal(t, "AboutValue()", schem.AboutValue(), aboutVal)
	about := xmptype.NewAttribute(xmptype.RDFNamespace, "about", "YEP")
	noError(t, "SetAbout", schem.SetAbout(about))
	if schem.AboutAttribute() != about {
		t.Errorf("AboutAttribute() = %v, want the attribute that was set", schem.AboutAttribute())
	}

	const textProp = "textProp"
	const textPropVal = "TextPropTest"
	noError(t, "SetTextPropertyValue", schem.SetTextPropertyValue(textProp, textPropVal))
	got, err := schem.UnqualifiedTextPropertyValue(textProp)
	noError(t, "UnqualifiedTextPropertyValue", err)
	equal(t, "UnqualifiedTextPropertyValue", got, textPropVal)

	text, err := parent.TypeMapping().CreateText("", "nsSchem", "textType", "GRINGO")
	noError(t, "CreateText", err)
	schem.SetTextProperty(text)
	stored, err := schem.UnqualifiedTextProperty("textType")
	noError(t, "UnqualifiedTextProperty", err)
	if stored != text {
		t.Errorf("UnqualifiedTextProperty = %v, want the property that was set", stored)
	}

	dateVal := time.Date(2010, time.November, 12, 13, 14, 15, 0, time.UTC)
	const date = "nsSchem:dateProp"
	noError(t, "SetDatePropertyValue", schem.SetDatePropertyValue(date, dateVal))
	gotDate, held, err := schem.DatePropertyValue(date)
	noError(t, "DatePropertyValue", err)
	if !held || !gotDate.Equal(dateVal) {
		t.Errorf("DatePropertyValue = %v, %v, want %v", gotDate, held, dateVal)
	}
	dateType, err := parent.TypeMapping().CreateDate("", "nsSchem", "dateType", dateVal)
	noError(t, "CreateDate", err)
	schem.SetDateProperty(dateType)
	storedDate, err := schem.DateProperty("dateType")
	noError(t, "DateProperty", err)
	if storedDate != dateType {
		t.Errorf("DateProperty = %v, want the property that was set", storedDate)
	}

	const boolName = "nsSchem:booleanTestProp"
	noError(t, "SetBooleanPropertyValue", schem.SetBooleanPropertyValue(boolName, false))
	gotBool, held, err := schem.BooleanPropertyValue(boolName)
	noError(t, "BooleanPropertyValue", err)
	if !held || gotBool {
		t.Errorf("BooleanPropertyValue = %v, %v, want false", gotBool, held)
	}
	boolType, err := parent.TypeMapping().CreateBoolean("", "nsSchem", "boolType", false)
	noError(t, "CreateBoolean", err)
	schem.SetBooleanProperty(boolType)
	storedBool, err := schem.BooleanProperty("boolType")
	noError(t, "BooleanProperty", err)
	if storedBool != boolType {
		t.Errorf("BooleanProperty = %v, want the property that was set", storedBool)
	}

	const intProp = "nsSchem:IntegerTestProp"
	noError(t, "SetIntegerPropertyValue", schem.SetIntegerPropertyValue(intProp, 5))
	gotInt, held, err := schem.IntegerPropertyValue(intProp)
	noError(t, "IntegerPropertyValue", err)
	if !held || gotInt != 5 {
		t.Errorf("IntegerPropertyValue = %v, %v, want 5", gotInt, held)
	}
	intType, err := parent.TypeMapping().CreateInteger("", "nsSchem", "intType", 5)
	noError(t, "CreateInteger", err)
	schem.SetIntegerProperty(intType)
	storedInt, err := schem.IntegerProperty("intType")
	noError(t, "IntegerProperty", err)
	if storedInt != intType {
		t.Errorf("IntegerProperty = %v, want the property that was set", storedInt)
	}

	// Check bad type verification
	if _, err := schem.IntegerProperty("boolType"); !errors.Is(err, xmptype.ErrBadFieldValue) {
		t.Errorf("IntegerProperty(\"boolType\") = %v, want a bad field value", err)
	}
	if _, err := schem.DateProperty("textType"); !errors.Is(err, xmptype.ErrBadFieldValue) {
		t.Errorf("DateProperty(\"textType\") = %v, want a bad field value", err)
	}
	if _, err := schem.BooleanProperty("dateType"); !errors.Is(err, xmptype.ErrBadFieldValue) {
		t.Errorf("BooleanProperty(\"dateType\") = %v, want a bad field value", err)
	}
}

func TestAltProperties(t *testing.T) {
	_, schem := bareSchema(t)
	const altProp = "AltProp"
	const defaultLang = "x-default"
	const defaultVal = "Default Language"
	const usLang = "en-us"
	const usVal = "American Language"
	const frLang = "fr-fr"
	frVal := "Lang française"

	noError(t, "SetUnqualifiedLanguagePropertyValue",
		schem.SetUnqualifiedLanguagePropertyValue(altProp, usLang, usVal))
	noError(t, "SetUnqualifiedLanguagePropertyValue",
		schem.SetUnqualifiedLanguagePropertyValue(altProp, defaultLang, defaultVal))
	noError(t, "SetUnqualifiedLanguagePropertyValue",
		schem.SetUnqualifiedLanguagePropertyValue(altProp, frLang, frVal))

	for _, want := range []struct{ lang, value string }{
		{defaultLang, defaultVal},
		{frLang, frVal},
		{usLang, usVal},
	} {
		got, err := schem.UnqualifiedLanguagePropertyValue(altProp, want.lang)
		noError(t, "UnqualifiedLanguagePropertyValue", err)
		equal(t, "UnqualifiedLanguagePropertyValue("+want.lang+")", got, want.value)
	}

	languages, err := schem.UnqualifiedLanguagePropertyLanguagesValue(altProp)
	noError(t, "UnqualifiedLanguagePropertyLanguagesValue", err)
	// default language must be in first place
	if len(languages) == 0 || languages[0] != defaultLang {
		t.Errorf("languages = %v, want %q first", languages, defaultLang)
	}
	holdsAll(t, "languages", languages, []string{usLang, frLang})

	// Test replacement/removal
	frVal = "Langue française"
	noError(t, "SetUnqualifiedLanguagePropertyValue",
		schem.SetUnqualifiedLanguagePropertyValue(altProp, frLang, frVal))
	got, err := schem.UnqualifiedLanguagePropertyValue(altProp, frLang)
	noError(t, "UnqualifiedLanguagePropertyValue", err)
	equal(t, "UnqualifiedLanguagePropertyValue(fr-fr)", got, frVal)

	// Java sets null to remove the alternative; the port spells null as the
	// empty string.
	noError(t, "SetUnqualifiedLanguagePropertyValue",
		schem.SetUnqualifiedLanguagePropertyValue(altProp, frLang, ""))
	languages, err = schem.UnqualifiedLanguagePropertyLanguagesValue(altProp)
	noError(t, "UnqualifiedLanguagePropertyLanguagesValue", err)
	for _, language := range languages {
		if language == frLang {
			t.Errorf("languages = %v, want %q gone", languages, frLang)
		}
	}
	noError(t, "SetUnqualifiedLanguagePropertyValue",
		schem.SetUnqualifiedLanguagePropertyValue(altProp, frLang, frVal))
}

func TestMergeSchema(t *testing.T) {
	parent, _ := bareSchema(t)
	const bagName = "bagName"
	const seqName = "seqName"
	const altName = "AltProp"
	const valBagSchem1 = "BagvalSchem1"
	const valBagSchem2 = "BagvalSchem2"
	const valSeqSchem1 = "seqvalSchem1"
	const valSeqSchem2 = "seqvalSchem2"
	const valAltSchem1 = "altvalSchem1"
	const langAltSchem1 = "x-default"
	const valAltSchem2 = "altvalSchem2"
	const langAltSchem2 = "fr-fr"

	schem1, err := schema.NewXMPSchemaOfNamespace(parent, "http://www.test.org/schem/", "test")
	noError(t, "NewXMPSchemaOfNamespace", err)
	noError(t, "AddQualifiedBagValue", schem1.AddQualifiedBagValue(bagName, valBagSchem1))
	noError(t, "AddUnqualifiedSequenceValue",
		schem1.AddUnqualifiedSequenceValue(seqName, valSeqSchem1))
	noError(t, "SetUnqualifiedLanguagePropertyValue",
		schem1.SetUnqualifiedLanguagePropertyValue(altName, langAltSchem1, valAltSchem1))

	schem2, err := schema.NewXMPSchemaOfNamespace(parent, "http://www.test.org/schem/", "test")
	noError(t, "NewXMPSchemaOfNamespace", err)
	noError(t, "AddQualifiedBagValue", schem2.AddQualifiedBagValue(bagName, valBagSchem2))
	noError(t, "AddUnqualifiedSequenceValue",
		schem2.AddUnqualifiedSequenceValue(seqName, valSeqSchem2))
	noError(t, "SetUnqualifiedLanguagePropertyValue",
		schem2.SetUnqualifiedLanguagePropertyValue(altName, langAltSchem2, valAltSchem2))

	noError(t, "Merge", schem1.Merge(schem2))

	// Check if all values are present
	got, err := schem1.UnqualifiedLanguagePropertyValue(altName, langAltSchem2)
	noError(t, "UnqualifiedLanguagePropertyValue", err)
	equal(t, "UnqualifiedLanguagePropertyValue(fr-fr)", got, valAltSchem2)
	got, err = schem1.UnqualifiedLanguagePropertyValue(altName, langAltSchem1)
	noError(t, "UnqualifiedLanguagePropertyValue", err)
	equal(t, "UnqualifiedLanguagePropertyValue(x-default)", got, valAltSchem1)

	holdsAll(t, "UnqualifiedBagValueList", schem1.UnqualifiedBagValueList(bagName),
		[]string{valBagSchem1, valBagSchem2})
	holdsAll(t, "UnqualifiedSequenceValueList", schem1.UnqualifiedSequenceValueList(seqName),
		[]string{valSeqSchem1, valSeqSchem2})
}

func TestListAndContainerAccessor(t *testing.T) {
	parent, schem := bareSchema(t)
	const boolName = "bool"
	boolean, err := parent.TypeMapping().CreateBoolean("", schem.Prefix(), boolName, true)
	noError(t, "CreateBoolean", err)
	att := xmptype.NewAttribute(xmptype.RDFNamespace, "test", "vgh")
	schem.SetAttribute(att)
	schem.SetBooleanProperty(boolean)

	if !holdsField(schem.AllProperties(), boolean) {
		t.Errorf("AllProperties() = %v, want the boolean property", schem.AllProperties())
	}
	held := false
	for _, attribute := range schem.AllAttributes() {
		if attribute == att {
			held = true
		}
	}
	if !held {
		t.Errorf("AllAttributes() = %v, want the attribute that was set", schem.AllAttributes())
	}
	if schem.Property(boolName) != xmptype.AbstractField(boolean) {
		t.Errorf("Property(%q) = %v, want the boolean property", boolName,
			schem.Property(boolName))
	}
}
