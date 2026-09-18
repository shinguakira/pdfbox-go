package util

import "testing"

// TestWhatIsLeftIsNotAZoneName checks the dates whose tail parseTZoffset reads
// as a zone name, and which name no zone.
//
// Java trims what is left after the date and asks TimeZone.getTimeZone for it,
// which answers GMT for a name it does not know, and parseTZoffset takes GMT for
// no zone at all; toCalendar then refuses the date, because it was not read to
// its end. Trimmed, a line feed or a tab is the empty name, and Go's
// time.LoadLocation answers UTC for "" and the local zone for "Local", where
// TimeZone knows neither. Found by the corpus comparison of document
// information: pdfjs/issue3903.pdf writes its /CreationDate as
// " Mon Mar 20 13:03:38 2000\n ", which PDFBox reads as no date and the Go
// version read as 20 March 2000.
//
// The expected values are PDFBox's, printed by DateConverter.toCalendar as the
// milliseconds and the zone's raw offset.
func TestWhatIsLeftIsNotAZoneName(t *testing.T) {
	for _, c := range []struct {
		text   string
		millis int64 // bad where PDFBox answers null
		offset int
	}{
		{" Mon Mar 20 13:03:38 2000\n ", bad, 0},
		{"Mon Mar 20 13:03:38 2000", 953557418000, 0},
		{"Mon Mar 20 13:03:38 2000 ", 953557418000, 0},
		{"Mon Mar 20 13:03:38 2000\n", bad, 0},
		{"Mon Mar 20 13:03:38 2000\t", bad, 0},
		{"Mon Mar 20 13:03:38 2000 Local", bad, 0},
		{"Mon Mar 20 13:03:38 2000 PST", 953586218000, -8 * hrs},
		{"Mon Mar 20 13:03:38 2000 UTC", 953557418000, 0},
		{"Mon Mar 20 13:03:38 2000 Etc/GMT", 953557418000, 0},
		{"Mon Mar 20 13:03:38 2000 America/New_York", 953575418000, -5 * hrs},
		{"Mon Mar 20 13:03:38 2000 Nowhere", bad, 0},
		{"D:20000320130338\n", bad, 0},
		{"D:20000320130338 ", 953557418000, 0},
		{"D:20000320130338Local", bad, 0},
		{"20000320130338 Local", bad, 0},
	} {
		got, ok := ToCalendar(c.text)
		if c.millis == bad {
			if ok {
				t.Errorf("ToCalendar(%q) = %d, want no date, which is what PDFBox answers", c.text, got.UnixMilli())
			}
			continue
		}
		if !ok {
			t.Errorf("ToCalendar(%q) is no date, want %d", c.text, c.millis)
			continue
		}
		_, offset := got.Zone()
		if got.UnixMilli() != c.millis || offset*1000 != c.offset {
			t.Errorf("ToCalendar(%q) = %d at offset %d, want %d at offset %d",
				c.text, got.UnixMilli(), offset*1000, c.millis, c.offset)
		}
	}
}
