package xml_test

// What a date property holding no date reads back as.
//
// PDFBOX-6029's packet is already parsed and serialized by TestEmptyDate; these
// are the accessors that read it, which Java answers null from and which the
// port has to answer "nothing" from rather than the zero time.
//
// The expected values were read by running org.apache.xmpbox against the same
// packet: getCreateDateProperty() is not null, and getValue(), getCreateDate(),
// getDatePropertyValue("CreateDate") and getStringValue() are all null. See
// migration/STATUS.md, the track's feedback round.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/xmpbox"
	"github.com/shinguakira/pdfbox-go/go/xmpbox/xmptype"
)

// emptyDatePacket is the packet PDFBOX-6029 is about: a date property with no
// value at all.
const emptyDatePacket = `<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta x:xmptk="Adobe XMP Core 4.2.1-c041 52.342996, 2008/05/07-20:48:00" xmlns:x="adobe:ns:meta/">
  <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
   <rdf:Description rdf:about="" xmlns:xmp="http://ns.adobe.com/xap/1.0/">
    <xmp:CreateDate></xmp:CreateDate>
   </rdf:Description>
  </rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>`

func TestAnEmptyDateReadsBackAsNothing(t *testing.T) {
	xmp := parse(t, emptyDatePacket)
	basic := xmp.XMPBasicSchema()
	if basic == nil {
		t.Fatal("XMPBasicSchema() = nil")
	}

	// The property itself is there -- Java's getCreateDateProperty() is not
	// null -- and it holds no date.
	property := basic.CreateDateProperty()
	if property == nil {
		t.Fatal("CreateDateProperty() = nil, want the property that was parsed")
	}
	if value := property.Value(); value != nil {
		t.Errorf("Value() = %v, want nothing, which is Java's null", value)
	}
	if _, held := property.DateValue(); held {
		t.Error("DateValue() held a date, want none")
	}
	equal(t, "StringValue()", property.StringValue(), "")

	// The schema's own accessor answers null in Java.
	if date, held := basic.CreateDate(); held {
		t.Errorf("CreateDate() = %v, true, want nothing, which is Java's null", date)
	}

	// So does the generic one.
	date, held, err := basic.DatePropertyValue("CreateDate")
	noError(t, "DatePropertyValue", err)
	if held {
		t.Errorf("DatePropertyValue = %v, true, want nothing, which is Java's null", date)
	}
}

// TestAnEmptyDateInAStructuredTypeReadsBackAsNothing is the same for a date
// field of a structured type, which goes through
// AbstractStructuredType.getDatePropertyAsCalendar.
//
// Java answers null: building a ResourceEventType with an empty stEvt:when and
// calling getWhen() prints "null".
func TestAnEmptyDateInAStructuredTypeReadsBackAsNothing(t *testing.T) {
	metadata := xmpbox.CreateXMPMetadata()
	event := xmptype.NewResourceEventType(metadata)
	when, err := metadata.TypeMapping().InstanciateSimpleProperty(event.Namespace(),
		event.Prefix(), xmptype.ResourceEventWhen, "", xmptype.Date)
	noError(t, "InstanciateSimpleProperty", err)
	event.AddProperty(when)

	if date, held := event.When(); held {
		t.Errorf("When() = %v, true, want nothing, which is Java's null", date)
	}
}
