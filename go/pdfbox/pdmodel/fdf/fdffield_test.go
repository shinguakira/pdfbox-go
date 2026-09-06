package fdf_test

// Port of org.apache.pdfbox.pdmodel.fdf.FDFFieldTest.

import (
	"reflect"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/fdf"
)

// TestCOSStringValue is testCOSStringValue.
func TestCOSStringValue(t *testing.T) {
	const testString = "Test value"
	testCOSString := cos.NewStringObj(testString)

	field := fdf.NewFDFField()
	if err := field.SetValue(testCOSString); err != nil {
		t.Fatalf("SetValue: %v", err)
	}

	cosValue, err := field.COSValue()
	if err != nil {
		t.Fatalf("COSValue: %v", err)
	}
	if cosValue != cos.Base(testCOSString) {
		t.Errorf("COSValue() = %v, want the string that was set", cosValue)
	}

	value, err := field.Value()
	if err != nil {
		t.Fatalf("Value: %v", err)
	}
	if value != testString {
		t.Errorf("Value() = %v, want %q", value, testString)
	}
}

// TestTextAsCOSStreamValue is testTextAsCOSStreamValue: a value held as a
// stream reads back as the text in it.
func TestTextAsCOSStreamValue(t *testing.T) {
	const testString = "Test value"

	stream := cos.NewStream(nil)
	writer, err := stream.CreateWriter()
	if err != nil {
		t.Fatalf("CreateWriter: %v", err)
	}
	if _, err := writer.Write([]byte(testString)); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	field := fdf.NewFDFField()
	if err := field.SetValue(stream); err != nil {
		t.Fatalf("SetValue: %v", err)
	}

	value, err := field.Value()
	if err != nil {
		t.Fatalf("Value: %v", err)
	}
	if value != testString {
		t.Errorf("Value() = %v, want %q", value, testString)
	}
}

// TestCOSNameValue is testCOSNameValue.
func TestCOSNameValue(t *testing.T) {
	const testString = "Yes"
	testCOSName := cos.GetPDFName(testString)

	field := fdf.NewFDFField()
	if err := field.SetValue(testCOSName); err != nil {
		t.Fatalf("SetValue: %v", err)
	}

	cosValue, err := field.COSValue()
	if err != nil {
		t.Fatalf("COSValue: %v", err)
	}
	if cosValue != cos.Base(testCOSName) {
		t.Errorf("COSValue() = %v, want the name that was set", cosValue)
	}

	value, err := field.Value()
	if err != nil {
		t.Fatalf("Value: %v", err)
	}
	if value != testString {
		t.Errorf("Value() = %v, want %q", value, testString)
	}
}

// TestCOSArrayValue is testCOSArrayValue: an array of strings reads back as the
// list of them.
func TestCOSArrayValue(t *testing.T) {
	testList := []string{"A", "B"}
	testCOSArray := cos.ArrayOfStrings(testList)

	field := fdf.NewFDFField()
	if err := field.SetValue(testCOSArray); err != nil {
		t.Fatalf("SetValue: %v", err)
	}

	cosValue, err := field.COSValue()
	if err != nil {
		t.Fatalf("COSValue: %v", err)
	}
	if cosValue != cos.Base(testCOSArray) {
		t.Errorf("COSValue() = %v, want the array that was set", cosValue)
	}

	value, err := field.Value()
	if err != nil {
		t.Fatalf("Value: %v", err)
	}
	if !reflect.DeepEqual(value, testList) {
		t.Errorf("Value() = %v, want %v", value, testList)
	}
}
