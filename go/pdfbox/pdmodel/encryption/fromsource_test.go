package encryption

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rc4"
	"encoding/asn1"
	"encoding/hex"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// Tests written from the source for the parts of pdmodel/encryption the two
// Java tests do not reach, which slice 5's A3 asks to be named before they are
// written.
//
// TestSymmetricKeyEncryption.testPermissions covers the reading half whole: its
// three files are R2/V1 (RC4-40), R3/V2 (RC4-128) and R6/V5 (AES-256), so
// RC4Cipher, computeHash2A, computeHash2B and SaslPrep all run.
// TestPublicKeyEncryption covers PublicKeySecurityHandler's reading half, the
// CMS reader, the PKCS#12 reader and RC2.
//
// What neither reaches, and what is tested here:
//
//   - AccessPermission's bit arithmetic, its read-only lock, and
//     getPermissionBytesForPublicKey, which only the encrypting side calls
//   - ProtectionPolicy's key length validation, and the two policies and
//     PublicKeyRecipient, which only the encrypting side builds
//   - SecurityHandlerFactory's lookups and its duplicate registration
//   - PDEncryption's setters
//   - StandardSecurityHandler's password computations in the forward
//     direction, which the reading path only ever runs backwards
//   - RC2 and RC4 as ciphers, against values from outside this port
//   - SaslPrep, against the worked examples of RFC 4013
//
// Nothing here asserts a value read off this port. The cipher values come from
// RFC 2268 and from Go's own crypto/rc4; the SASLprep values from RFC 4013; the
// password values from a file Adobe Acrobat wrote.

// TestAccessPermissionBits pins the bit each permission lives in, which
// ISO 32000-1 table 22 numbers from 1.
func TestAccessPermissionBits(t *testing.T) {
	cases := []struct {
		name string
		bit  uint
		set  func(p *AccessPermission, value bool)
		get  func(p *AccessPermission) bool
	}{
		{"print", 3, (*AccessPermission).SetCanPrint, (*AccessPermission).CanPrint},
		{"modify", 4, (*AccessPermission).SetCanModify, (*AccessPermission).CanModify},
		{"extract", 5, (*AccessPermission).SetCanExtractContent,
			(*AccessPermission).CanExtractContent},
		{"modifyAnnotations", 6, (*AccessPermission).SetCanModifyAnnotations,
			(*AccessPermission).CanModifyAnnotations},
		{"fillInForm", 9, (*AccessPermission).SetCanFillInForm,
			(*AccessPermission).CanFillInForm},
		{"extractForAccessibility", 10, (*AccessPermission).SetCanExtractForAccessibility,
			(*AccessPermission).CanExtractForAccessibility},
		{"assembleDocument", 11, (*AccessPermission).SetCanAssembleDocument,
			(*AccessPermission).CanAssembleDocument},
		{"printFaithful", 12, (*AccessPermission).SetCanPrintFaithful,
			(*AccessPermission).CanPrintFaithful},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := NewAccessPermissionOf(0)
			c.set(p, true)
			if got, want := p.PermissionBytes(), int32(1)<<(c.bit-1); got != want {
				t.Errorf("setting %s gave %#08x, want %#08x", c.name, uint32(got), uint32(want))
			}
			if !c.get(p) {
				t.Errorf("%s is not set after setting it", c.name)
			}
			c.set(p, false)
			if p.PermissionBytes() != 0 {
				t.Errorf("clearing %s left %#08x", c.name, uint32(p.PermissionBytes()))
			}
		})
	}
}

// TestAccessPermissionDefaults checks that a new permission grants everything
// but leaves bits 1 and 2 clear, which the specification reserves.
func TestAccessPermissionDefaults(t *testing.T) {
	p := NewAccessPermission()
	if !p.IsOwnerPermission() {
		t.Error("a new AccessPermission should grant everything")
	}
	want := ^int32(3)
	if got := p.PermissionBytes(); got != want {
		t.Errorf("PermissionBytes = %#08x, want %#08x", uint32(got), uint32(want))
	}
	if p.IsReadOnly() {
		t.Error("a new AccessPermission should not be read only")
	}
}

// TestAccessPermissionReadOnly checks that the lock stops every setter and
// cannot be lifted.
func TestAccessPermissionReadOnly(t *testing.T) {
	p := NewAccessPermission()
	p.SetReadOnly()
	before := p.PermissionBytes()
	p.SetCanPrint(false)
	p.SetCanModify(false)
	p.SetCanExtractContent(false)
	p.SetCanModifyAnnotations(false)
	p.SetCanFillInForm(false)
	p.SetCanExtractForAccessibility(false)
	p.SetCanAssembleDocument(false)
	p.SetCanPrintFaithful(false)
	if p.PermissionBytes() != before {
		t.Errorf("a read-only AccessPermission changed from %#08x to %#08x",
			uint32(before), uint32(p.PermissionBytes()))
	}
	if !p.IsReadOnly() {
		t.Error("the lock was lifted")
	}
}

// TestPermissionBytesForPublicKey checks the shape the public key handler
// writes: bit 1 set, bits 7, 8 and 13 to 32 clear, the rest left alone.
func TestPermissionBytesForPublicKey(t *testing.T) {
	p := NewAccessPermission()
	got := p.PermissionBytesForPublicKey()
	if got&1 == 0 {
		t.Error("bit 1 should be set")
	}
	for _, bit := range []uint{7, 8} {
		if got&(1<<(bit-1)) != 0 {
			t.Errorf("bit %d should be clear", bit)
		}
	}
	for bit := uint(13); bit <= 32; bit++ {
		if got&(int32(1)<<(bit-1)) != 0 {
			t.Errorf("bit %d should be clear", bit)
		}
	}
	// the permissions themselves survive
	if !p.CanPrint() || !p.CanModify() || !p.CanPrintFaithful() {
		t.Error("the permission bits should be left alone")
	}
}

// TestHasAnyRevision3PermissionSet checks the four permissions that force
// revision 3.
func TestHasAnyRevision3PermissionSet(t *testing.T) {
	none := NewAccessPermissionOf(0)
	if none.hasAnyRevision3PermissionSet() {
		t.Error("no permission set should not force revision 3")
	}
	setters := []func(p *AccessPermission, value bool){
		(*AccessPermission).SetCanFillInForm,
		(*AccessPermission).SetCanExtractForAccessibility,
		(*AccessPermission).SetCanAssembleDocument,
		(*AccessPermission).SetCanPrintFaithful,
	}
	for i, set := range setters {
		p := NewAccessPermissionOf(0)
		set(p, true)
		if !p.hasAnyRevision3PermissionSet() {
			t.Errorf("permission %d should force revision 3", i)
		}
	}
	// one that does not
	printOnly := NewAccessPermissionOf(0)
	printOnly.SetCanPrint(true)
	if printOnly.hasAnyRevision3PermissionSet() {
		t.Error("print alone should not force revision 3")
	}
}

// TestProtectionPolicyKeyLength checks the three lengths the specification
// allows and that anything else is refused.
func TestProtectionPolicyKeyLength(t *testing.T) {
	policy := NewStandardProtectionPolicy("owner", "user", NewAccessPermission())
	if got := policy.EncryptionKeyLength(); got != 40 {
		t.Errorf("the default key length is %d, want 40", got)
	}
	for _, length := range []int{40, 128, 256} {
		if err := policy.SetEncryptionKeyLength(length); err != nil {
			t.Errorf("SetEncryptionKeyLength(%d): %v", length, err)
		}
		if got := policy.EncryptionKeyLength(); got != length {
			t.Errorf("EncryptionKeyLength = %d, want %d", got, length)
		}
	}
	for _, length := range []int{0, 39, 64, 127, 129, 255, 257, 512} {
		if err := policy.SetEncryptionKeyLength(length); err == nil {
			t.Errorf("SetEncryptionKeyLength(%d) should be refused", length)
		}
	}
	if got := policy.EncryptionKeyLength(); got != 256 {
		t.Errorf("a refused length changed the key length to %d", got)
	}
	if policy.IsPreferAES() {
		t.Error("AES should not be preferred by default")
	}
	policy.SetPreferAES(true)
	if !policy.IsPreferAES() {
		t.Error("SetPreferAES(true) had no effect")
	}
}

// TestStandardProtectionPolicyAccessors checks what the constructor stored.
func TestStandardProtectionPolicyAccessors(t *testing.T) {
	permissions := NewAccessPermission()
	policy := NewStandardProtectionPolicy("owner", "user", permissions)
	if got := policy.OwnerPassword(); got != "owner" {
		t.Errorf("OwnerPassword = %q, want %q", got, "owner")
	}
	if got := policy.UserPassword(); got != "user" {
		t.Errorf("UserPassword = %q, want %q", got, "user")
	}
	if policy.Permissions() != permissions {
		t.Error("Permissions returned a different object")
	}
	policy.SetOwnerPassword("o2")
	policy.SetUserPassword("u2")
	other := NewAccessPermission()
	policy.SetPermissions(other)
	if policy.OwnerPassword() != "o2" || policy.UserPassword() != "u2" ||
		policy.Permissions() != other {
		t.Error("a setter had no effect")
	}
}

// TestPublicKeyProtectionPolicyRecipients checks the recipient list.
func TestPublicKeyProtectionPolicyRecipients(t *testing.T) {
	policy := NewPublicKeyProtectionPolicy()
	if got := policy.NumberOfRecipients(); got != 0 {
		t.Errorf("a new policy has %d recipients, want 0", got)
	}
	first := &PublicKeyRecipient{}
	first.SetPermission(NewAccessPermission())
	second := &PublicKeyRecipient{}
	policy.AddRecipient(first)
	policy.AddRecipient(second)
	if got := policy.NumberOfRecipients(); got != 2 {
		t.Errorf("NumberOfRecipients = %d, want 2", got)
	}
	if got := policy.Recipients(); len(got) != 2 || got[0] != first || got[1] != second {
		t.Error("the recipients came back in the wrong order")
	}
	if !policy.RemoveRecipient(first) {
		t.Error("RemoveRecipient should report that it removed one")
	}
	if policy.RemoveRecipient(first) {
		t.Error("RemoveRecipient should report that there was nothing to remove")
	}
	if got := policy.NumberOfRecipients(); got != 1 {
		t.Errorf("NumberOfRecipients = %d, want 1", got)
	}
	if first.Permission() == nil {
		t.Error("the recipient lost its permission")
	}
}

// TestSecurityHandlerFactory checks the two lookups and the duplicate guard.
func TestSecurityHandlerFactory(t *testing.T) {
	factory := SecurityHandlerFactoryInstance

	standard := factory.NewSecurityHandlerForFilter(StandardSecurityHandlerFilter)
	if _, ok := standard.(*StandardSecurityHandler); !ok {
		t.Errorf("the handler for %q is %T", StandardSecurityHandlerFilter, standard)
	}
	publicKey := factory.NewSecurityHandlerForFilter(PublicKeySecurityHandlerFilter)
	if _, ok := publicKey.(*PublicKeySecurityHandler); !ok {
		t.Errorf("the handler for %q is %T", PublicKeySecurityHandlerFilter, publicKey)
	}
	if got := factory.NewSecurityHandlerForFilter("NoSuchFilter"); got != nil {
		t.Errorf("an unknown filter gave %T, want nil", got)
	}

	fromPolicy := factory.NewSecurityHandlerForPolicy(
		NewStandardProtectionPolicy("o", "u", NewAccessPermission()))
	if _, ok := fromPolicy.(*StandardSecurityHandler); !ok {
		t.Errorf("the handler for a standard policy is %T", fromPolicy)
	}
	if got := fromPolicy.KeyLength(); got != 40 {
		t.Errorf("the handler took key length %d from the policy, want 40", got)
	}

	if err := factory.RegisterHandler(StandardSecurityHandlerFilter, nil, "x", nil); err == nil {
		t.Error("registering a filter name twice should be refused")
	}
}

// TestPDEncryptionRoundTrip checks that what the setters write the getters read
// back, through the dictionary.
func TestPDEncryptionRoundTrip(t *testing.T) {
	e := NewPDEncryption()
	e.SetFilter("Standard")
	e.SetSubFilter("adbe.pkcs7.s4")
	e.SetVersion(4)
	e.SetLength(128)
	e.SetRevision(4)
	e.SetPermissions(-3904)
	e.SetOwnerKey(bytes.Repeat([]byte{1}, 32))
	e.SetUserKey(bytes.Repeat([]byte{2}, 32))
	e.SetPerms(bytes.Repeat([]byte{3}, 16))

	// read them back through a second wrapper over the same dictionary, so that
	// nothing is answered out of a field
	read := NewPDEncryptionOf(e.COSObject())
	if got := read.Filter(); got != "Standard" {
		t.Errorf("Filter = %q", got)
	}
	if got := read.SubFilter(); got != "adbe.pkcs7.s4" {
		t.Errorf("SubFilter = %q", got)
	}
	if got := read.Version(); got != 4 {
		t.Errorf("Version = %d", got)
	}
	if got := read.Length(); got != 128 {
		t.Errorf("Length = %d", got)
	}
	if got := read.Revision(); got != 4 {
		t.Errorf("Revision = %d", got)
	}
	if got := read.Permissions(); got != -3904 {
		t.Errorf("Permissions = %d", got)
	}
	if got := read.OwnerKey(); !bytes.Equal(got, bytes.Repeat([]byte{1}, 32)) {
		t.Errorf("OwnerKey = % x", got)
	}
	if got := read.UserKey(); !bytes.Equal(got, bytes.Repeat([]byte{2}, 32)) {
		t.Errorf("UserKey = % x", got)
	}
	if got := read.Perms(); !bytes.Equal(got, bytes.Repeat([]byte{3}, 16)) {
		t.Errorf("Perms = % x", got)
	}
	if got := read.StreamFilterName(); !cos.Identity.Equals(got) {
		t.Errorf("StreamFilterName = %v, want Identity by default", got)
	}
	read.SetStreamFilterName(cos.StdCF)
	read.SetStringFilterName(cos.StdCF)
	if got := read.StreamFilterName(); !cos.StdCF.Equals(got) {
		t.Errorf("StreamFilterName = %v", got)
	}
	if got := read.StringFilterName(); !cos.StdCF.Equals(got) {
		t.Errorf("StringFilterName = %v", got)
	}

	// the crypt filter dictionary
	cryptFilter := NewPDCryptFilterDictionary()
	cryptFilter.SetCryptFilterMethod(cos.AESV2)
	cryptFilter.SetLength(128)
	read.SetStdCryptFilterDictionary(cryptFilter)
	back := read.StdCryptFilterDictionary()
	if back == nil {
		t.Fatal("StdCryptFilterDictionary came back nil")
	}
	if got := back.CryptFilterMethod(); !cos.AESV2.Equals(got) {
		t.Errorf("CryptFilterMethod = %v", got)
	}
	if got := back.Length(); got != 128 {
		t.Errorf("crypt filter Length = %d", got)
	}
	if !back.IsEncryptMetaData() {
		t.Error("IsEncryptMetaData should default to true")
	}
	back.SetEncryptMetaData(false)
	if back.IsEncryptMetaData() {
		t.Error("SetEncryptMetaData(false) had no effect")
	}

	read.RemoveV45filters()
	if read.COSObject().GetItem(cos.CF) != nil {
		t.Error("RemoveV45filters left /CF behind")
	}
}

// TestRC2Vectors runs the test vectors of RFC 2268 section 5.
func TestRC2Vectors(t *testing.T) {
	cases := []struct {
		key              string
		effectiveKeyBits int
		plain            string
		cipher           string
	}{
		{"0000000000000000", 63, "0000000000000000", "ebb773f993278eff"},
		{"ffffffffffffffff", 64, "ffffffffffffffff", "278b27e42e2f0d49"},
		{"3000000000000000", 64, "1000000000000001", "30649edf9be7d2c2"},
		{"88", 64, "0000000000000000", "61a8a244adacccf0"},
		{"88bca90e90875a", 64, "0000000000000000", "6ccf4308974c267f"},
		{"88bca90e90875a7f0f79c384627bafb2", 64, "0000000000000000", "1a807d272bbe5db1"},
		{"88bca90e90875a7f0f79c384627bafb2", 128, "0000000000000000", "2269552ab0f85ca6"},
		{"88bca90e90875a7f0f79c384627bafb216f80a6f85920584c42fceb0be255daf1e", 129,
			"0000000000000000", "5b78d3a43dfff1f1"},
	}
	for _, c := range cases {
		t.Run(c.key+"/"+itoa(c.effectiveKeyBits), func(t *testing.T) {
			key := mustDecodeHex(t, c.key)
			plain := mustDecodeHex(t, c.plain)
			want := mustDecodeHex(t, c.cipher)

			block, err := newRC2Cipher(key, c.effectiveKeyBits)
			if err != nil {
				t.Fatalf("newRC2Cipher: %v", err)
			}
			got := make([]byte, len(plain))
			block.Encrypt(got, plain)
			if !bytes.Equal(got, want) {
				t.Errorf("Encrypt = %x, want %x", got, want)
			}
			back := make([]byte, len(want))
			block.Decrypt(back, want)
			if !bytes.Equal(back, plain) {
				t.Errorf("Decrypt = %x, want %x", back, plain)
			}
		})
	}
}

// TestRC4CipherAgainstStdlib checks PDFBox's own RC4 against Go's, which is an
// implementation this port did not write.
func TestRC4CipherAgainstStdlib(t *testing.T) {
	keys := [][]byte{
		{0x01},
		[]byte("Key"),
		[]byte("Secret"),
		bytes.Repeat([]byte{0xff}, 32),
	}
	plains := [][]byte{
		[]byte(""),
		[]byte("Plaintext"),
		[]byte("Attack at dawn"),
		bytes.Repeat([]byte("The quick brown fox. "), 100),
	}
	for _, key := range keys {
		for _, plain := range plains {
			want := make([]byte, len(plain))
			reference, err := rc4.NewCipher(key)
			if err != nil {
				t.Fatalf("rc4.NewCipher: %v", err)
			}
			reference.XORKeyStream(want, plain)

			ported := newRC4Cipher()
			if err := ported.setKey(key); err != nil {
				t.Fatalf("setKey: %v", err)
			}
			var got bytes.Buffer
			if err := ported.writeBytes(plain, &got); err != nil {
				t.Fatalf("writeBytes: %v", err)
			}
			if !bytes.Equal(got.Bytes(), want) {
				t.Errorf("key %x, %d bytes: RC4Cipher gave %x, crypto/rc4 gave %x",
					key, len(plain), got.Bytes(), want)
			}

			// and through the streaming half, which the R2 and R3 paths use
			ported = newRC4Cipher()
			if err := ported.setKey(key); err != nil {
				t.Fatalf("setKey: %v", err)
			}
			var streamed bytes.Buffer
			if err := ported.writeStream(bytes.NewReader(plain), &streamed); err != nil {
				t.Fatalf("writeStream: %v", err)
			}
			if !bytes.Equal(streamed.Bytes(), want) {
				t.Errorf("key %x, %d bytes: writeStream gave %x, want %x",
					key, len(plain), streamed.Bytes(), want)
			}
		}
	}
	// the key length the class refuses
	if err := newRC4Cipher().setKey(nil); err == nil {
		t.Error("an empty key should be refused")
	}
	if err := newRC4Cipher().setKey(make([]byte, 33)); err == nil {
		t.Error("a 33 byte key should be refused")
	}
}

// TestSaslPrepExamples runs the worked examples of RFC 4013 section 3, which is
// the profile SaslPrep implements.
func TestSaslPrepExamples(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"soft hyphen mapped to nothing", "I­X", "IX", false},
		{"no transformation", "user", "user", false},
		{"case preserved", "USER", "USER", false},
		{"output is NFKC", "ª", "a", false},
		{"output is NFKC, roman numeral", "Ⅸ", "IX", false},
		{"prohibited character", "", "", true},
		{"bidirectional check", "ا1", "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := SaslPrepQuery(c.input)
			if c.wantErr {
				if err == nil {
					t.Errorf("SaslPrepQuery(%q) = %q, want an error", c.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("SaslPrepQuery(%q): %v", c.input, err)
			}
			if got != c.want {
				t.Errorf("SaslPrepQuery(%q) = %q, want %q", c.input, got, c.want)
			}
		})
	}
}

// TestSaslPrepStoredRejectsUnassigned checks the one way the stored profile
// differs from the query one.
func TestSaslPrepStoredRejectsUnassigned(t *testing.T) {
	// U+0378 is unassigned and has been since Unicode 1.1.
	const unassigned = "͸"
	if _, err := SaslPrepQuery(unassigned); err != nil {
		t.Errorf("a query may contain an unassigned code point: %v", err)
	}
	if _, err := SaslPrepStored(unassigned); err == nil {
		t.Error("a stored string may not contain an unassigned code point")
	}
}

// TestTruncateOrPad checks the 32-byte pad of algorithm 2.
func TestTruncateOrPad(t *testing.T) {
	empty := truncateOrPad(nil)
	if !bytes.Equal(empty, encryptPadding) {
		t.Errorf("the empty password padded to % x", empty)
	}
	short := truncateOrPad([]byte("ab"))
	if len(short) != 32 || short[0] != 'a' || short[1] != 'b' ||
		!bytes.Equal(short[2:], encryptPadding[:30]) {
		t.Errorf("a two byte password padded to % x", short)
	}
	long := truncateOrPad(bytes.Repeat([]byte{9}, 40))
	if len(long) != 32 || !bytes.Equal(long, bytes.Repeat([]byte{9}, 32)) {
		t.Errorf("a forty byte password truncated to % x", long)
	}
}

// TestTruncate127 checks the cut revisions 5 and 6 make.
func TestTruncate127(t *testing.T) {
	if got := truncate127(bytes.Repeat([]byte{1}, 127)); len(got) != 127 {
		t.Errorf("127 bytes came back as %d", len(got))
	}
	if got := truncate127(bytes.Repeat([]byte{1}, 128)); len(got) != 127 {
		t.Errorf("128 bytes came back as %d", len(got))
	}
	if got := truncate127([]byte{1, 2}); len(got) != 2 {
		t.Errorf("two bytes came back as %d", len(got))
	}
}

func mustDecodeHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("decoding %q: %v", s, err)
	}
	return b
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var digits []byte
	for i > 0 {
		digits = append([]byte{byte('0' + i%10)}, digits...)
		i /= 10
	}
	return string(digits)
}

// TestRegisterHandlerReplacesADuplicatePolicy pins JAVA-BUGS 26. Java's
// registerHandler javadoc says an exception is thrown "if another handler was
// previously registered for the same filter name or for the same policy name",
// and the method only looks in nameToHandler, so a second registration under a
// new name takes the policy over without a word. The port carries it.
//
// The registration mutates the factory, so this builds its own rather than
// reaching for the singleton the tests above use.
func TestRegisterHandlerReplacesADuplicatePolicy(t *testing.T) {
	factory := newSecurityHandlerFactory()
	standardPolicy := NewStandardProtectionPolicy("o", "u", NewAccessPermission())

	err := factory.RegisterHandler("Other.Filter",
		func() SecurityHandler { return NewPublicKeySecurityHandler() },
		standardPolicy.policyKey(),
		func(ProtectionPolicy) SecurityHandler { return NewPublicKeySecurityHandler() })
	if err != nil {
		t.Fatalf("a duplicate policy under a new filter name should be accepted: %v", err)
	}
	forPolicy := factory.NewSecurityHandlerForPolicy(standardPolicy)
	if _, ok := forPolicy.(*PublicKeySecurityHandler); !ok {
		t.Errorf("the standard policy maps to %T, want the registration that replaced it",
			forPolicy)
	}
	// the filter name it was registered under first still finds the first handler
	if got := factory.NewSecurityHandlerForFilter(StandardSecurityHandlerFilter); got == nil {
		t.Error("the standard filter name lost its handler")
	}
}

// TestCMSContentParameters pins the two shapes a CMS content encryption
// algorithm carries its initialisation vector in, which are not the same shape.
//
// RFC 3370 section 5.2 and RFC 3565 section 4.1 give DES-EDE3-CBC and AES-CBC a
// bare OCTET STRING. RFC 3370 section 5.3 gives RC2-CBC a SEQUENCE of the
// version and the IV, and RC2-CBC is what PDFBox itself writes --
// PublicKeySecurityHandler.createDERForRecipient asks the JCE for
// PKCSObjectIdentifiers.RC2_CBC -- so a /Recipients entry produced by the Java
// takes the RC2 path.
//
// The ciphers themselves are checked elsewhere: RC2 against RFC 2268's vectors
// above, AES by Go's own crypto/aes. What this pins is the parameter decoding.
func TestCMSContentParameters(t *testing.T) {
	// 24 bytes is what a PDF /Recipients envelope holds: the 20 byte seed and
	// the 4 permission bytes.
	plain := []byte("0123456789abcdefghijklmn")

	t.Run("rc2", func(t *testing.T) {
		key := mustDecodeHex(t, "000102030405060708090a0b0c0d0e0f")
		iv := mustDecodeHex(t, "0011223344556677")
		// version 58 is RFC 2268 section 6's encoding of 128 effective key bits
		parameters, err := asn1.Marshal(struct {
			Version int
			IV      []byte
		}{58, iv})
		if err != nil {
			t.Fatal(err)
		}
		block, err := newRC2Cipher(key, 128)
		if err != nil {
			t.Fatal(err)
		}
		checkCMSContentRoundTrip(t, oidRC2CBC, parameters, key, block, iv, plain)
	})

	t.Run("aes128", func(t *testing.T) {
		key := mustDecodeHex(t, "000102030405060708090a0b0c0d0e0f")
		iv := mustDecodeHex(t, "000102030405060708090a0b0c0d0e0f")
		parameters, err := asn1.Marshal(iv)
		if err != nil {
			t.Fatal(err)
		}
		block, err := aes.NewCipher(key)
		if err != nil {
			t.Fatal(err)
		}
		checkCMSContentRoundTrip(t, oidAES128CBC, parameters, key, block, iv, plain)
	})
}

// checkCMSContentRoundTrip encrypts the given plaintext the way a CMS envelope
// carries it -- CBC over a PKCS#7 padded body -- and checks that
// decryptCMSContent reads the algorithm parameters and gives it back.
func checkCMSContentRoundTrip(t *testing.T, oid asn1.ObjectIdentifier, parameters, key []byte,
	block cipher.Block, iv, plain []byte) {
	t.Helper()
	padding := block.BlockSize() - len(plain)%block.BlockSize()
	padded := append(append([]byte{}, plain...), bytes.Repeat([]byte{byte(padding)}, padding)...)
	encrypted := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(encrypted, padded)

	algorithm := algorithmIdentifier{
		Algorithm:  oid,
		Parameters: asn1.RawValue{FullBytes: parameters},
	}
	got, err := decryptCMSContent(algorithm, key, encrypted)
	if err != nil {
		t.Fatalf("decryptCMSContent: %v", err)
	}
	if !bytes.Equal(got, plain) {
		t.Errorf("decryptCMSContent gave %q, want %q", got, plain)
	}
}
