package cos_test

// Port of org.apache.pdfbox.cos.TestCOSUpdateInfo.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// TestIsSetNeedToBeUpdate is testIsSetNeedToBeUpdate.
//
// The flag only takes where the object belongs to a document that has finished
// parsing: setting it before there is an origin document state does nothing,
// which is how a document being read avoids marking everything it touches as
// changed.
func TestIsSetNeedToBeUpdate(t *testing.T) {
	origin := cos.NewDocumentState()
	origin.SetParsing(false)

	for _, row := range []struct {
		name string
		info cos.UpdateInfo
	}{
		{"COSDictionary", cos.NewDictionary()},
		{"COSObject", cos.NewObject(nil)},
	} {
		t.Run(row.name, func(t *testing.T) {
			info := row.info

			// with no origin document state the flag does not take
			info.SetNeedToBeUpdated(true)
			if info.IsNeedToBeUpdated() {
				t.Error("IsNeedToBeUpdated() = true before the origin document " +
					"state is set, want false")
			}

			info.UpdateState().SetOriginDocumentState(origin)

			info.SetNeedToBeUpdated(true)
			if !info.IsNeedToBeUpdated() {
				t.Error("IsNeedToBeUpdated() = false after setting it, want true")
			}
			info.SetNeedToBeUpdated(false)
			if info.IsNeedToBeUpdated() {
				t.Error("IsNeedToBeUpdated() = true after clearing it, want false")
			}
		})
	}
}
