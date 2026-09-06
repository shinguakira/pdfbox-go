package util

import (
	"time"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// The date accessors of a dictionary.
//
// Java declares these on COSDictionary, beside its other typed accessors. The
// port cannot: they need DateConverter, and this package imports cos for
// Matrix, so cos importing it back would close a cycle. They read as functions
// over the dictionary instead, and cos/dictionary.go says where they went.

// DictionaryDate returns the date under key, and reports false where the entry
// is missing or is not a date, which is the null Java answers.
//
// Port of COSDictionary.getDate(COSName).
func DictionaryDate(d *cos.Dictionary, key *cos.Name) (time.Time, bool) {
	if base, isString := d.GetDictionaryObject(key).(*cos.StringObj); isString {
		return ToCalendar(base.Value())
	}
	return time.Time{}, false
}

// DictionaryDateDefault returns the date under key, or defaultValue.
//
// Port of COSDictionary.getDate(COSName, Calendar).
func DictionaryDateDefault(d *cos.Dictionary, key *cos.Name, defaultValue time.Time) time.Time {
	if retval, ok := DictionaryDate(d, key); ok {
		return retval
	}
	return defaultValue
}

// SetDictionaryDate writes the given date under key.
//
// Port of COSDictionary.setDate(COSName, Calendar).
func SetDictionaryDate(d *cos.Dictionary, key *cos.Name, date time.Time) {
	d.SetString(key, ToString(date))
}

// EmbeddedDate returns the date under key of the dictionary under embedded, and
// reports false where either is missing.
//
// Port of COSDictionary.getEmbeddedDate(COSName, COSName).
func EmbeddedDate(d *cos.Dictionary, embedded, key *cos.Name) (time.Time, bool) {
	if eDic := d.GetCOSDictionary(embedded); eDic != nil {
		return DictionaryDate(eDic, key)
	}
	return time.Time{}, false
}

// SetEmbeddedDate writes the given date under key of the dictionary under
// embedded, adding that dictionary where it is missing.
//
// Port of COSDictionary.setEmbeddedDate(COSName, COSName, Calendar).
func SetEmbeddedDate(d *cos.Dictionary, embedded, key *cos.Name, date time.Time) {
	dic := d.GetCOSDictionary(embedded)
	if dic == nil {
		dic = cos.NewDictionary()
		d.SetItem(embedded, dic)
	}
	SetDictionaryDate(dic, key, date)
}
