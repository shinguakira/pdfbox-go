package encryption_test

// The `pdmodel/encryption` half of `track/java-bug-fixes`.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/encryption"
)

// TestHasSecurityHandlerAnswersItsName is JAVA-BUGS 24.
//
// Java's body is `return securityHandler == null`, so the method answers false
// where there is a handler and true where there is not. Nothing in either tree
// calls it, which is why nobody has noticed; it is a public method and its
// answer is wrong for anyone who does.
func TestHasSecurityHandlerAnswersItsName(t *testing.T) {
	enc := encryption.NewPDEncryption()
	if enc.HasSecurityHandler() {
		t.Error("a fresh PDEncryption says it has a security handler")
	}

	enc.SetSecurityHandler(encryption.NewStandardSecurityHandler())
	if !enc.HasSecurityHandler() {
		t.Error("a PDEncryption with a handler set says it has none")
	}
}
