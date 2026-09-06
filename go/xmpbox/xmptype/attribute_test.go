package xmptype

// Port of org.apache.xmpbox.type.AttributeTest.

import "testing"

// TestAtt is Java's testAtt: the three values an attribute is built from come
// back, and each setter replaces one of them.
func TestAtt(t *testing.T) {
	nsUri := "nsUri"
	localName := "localName"
	value := "value"
	att := NewAttribute(nsUri, localName, value)
	if got := att.Namespace(); got != nsUri {
		t.Errorf("Namespace() = %q, want %q", got, nsUri)
	}
	if got := att.Name(); got != localName {
		t.Errorf("Name() = %q, want %q", got, localName)
	}
	if got := att.Value(); got != value {
		t.Errorf("Value() = %q, want %q", got, value)
	}

	nsUri2 := "nsUri2"
	localName2 := "localName2"
	value2 := "value2"
	att.SetNsURI(nsUri2)
	att.SetName(localName2)
	att.SetValue(value2)
	if got := att.Namespace(); got != nsUri2 {
		t.Errorf("Namespace() = %q, want %q", got, nsUri2)
	}
	if got := att.Name(); got != localName2 {
		t.Errorf("Name() = %q, want %q", got, localName2)
	}
	if got := att.Value(); got != value2 {
		t.Errorf("Value() = %q, want %q", got, value2)
	}
}

// TestAttWithoutPrefix is Java's testAttWithoutPrefix, which builds the same
// attribute twice and asks the same two questions of each.
func TestAttWithoutPrefix(t *testing.T) {
	nsUri := "nsUri"
	localName := "localName"
	value := "value"
	att := NewAttribute(nsUri, localName, value)
	if got := att.Namespace(); got != nsUri {
		t.Errorf("Namespace() = %q, want %q", got, nsUri)
	}
	if got := att.Name(); got != localName {
		t.Errorf("Name() = %q, want %q", got, localName)
	}

	att = NewAttribute(nsUri, localName, value)
	if got := att.Namespace(); got != nsUri {
		t.Errorf("Namespace() = %q, want %q", got, nsUri)
	}
	if got := att.Name(); got != localName {
		t.Errorf("Name() = %q, want %q", got, localName)
	}
}
