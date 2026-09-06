package schema_test

// Port of org.apache.xmpbox.schema.BasicJobTicketSchemaTest.

import (
	"bytes"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/xmpbox"
	xmpxml "github.com/shinguakira/pdfbox-go/go/xmpbox/xml"
	"github.com/shinguakira/pdfbox-go/go/xmpbox/xmptype"
)

// jobTicketRoundTrip writes the packet out and reads it back, which is what
// each of the three cases does before it looks at the jobs.
func jobTicketRoundTrip(t *testing.T, metadata *xmpbox.XMPMetadata) []*xmptype.JobType {
	t.Helper()
	var bos bytes.Buffer
	noError(t, "Serialize", xmpxml.NewXmpSerializer().Serialize(metadata, &bos, true))
	rxmp, err := xmpxml.NewDomXmpParser().ParseBytes(bos.Bytes())
	noError(t, "ParseBytes", err)
	jt := rxmp.BasicJobTicketSchema()
	if jt == nil {
		t.Fatal("BasicJobTicketSchema() = nil, want the schema that was written")
	}
	jobs, err := jt.Jobs()
	noError(t, "Jobs", err)
	return jobs
}

func TestAddTwoJobs(t *testing.T) {
	metadata := xmpbox.CreateXMPMetadata()
	basic, err := metadata.CreateAndAddBasicJobTicketSchema()
	noError(t, "CreateAndAddBasicJobTicketSchema", err)
	noError(t, "AddJobOfPrefix", basic.AddJobOfPrefix("zeid1", "zename1", "zeurl1", "aaa"))
	noError(t, "AddJobOf", basic.AddJobOf("zeid2", "zename2", "zeurl2"))

	jobs := jobTicketRoundTrip(t, metadata)
	if len(jobs) != 2 {
		t.Fatalf("Jobs() held %d jobs, want 2", len(jobs))
	}
	equal(t, "jobs[0].ID()", jobs[0].ID(), "zeid1")
	equal(t, "jobs[0].Name()", jobs[0].Name(), "zename1")
	equal(t, "jobs[0].URL()", jobs[0].URL(), "zeurl1")
	equal(t, "jobs[1].ID()", jobs[1].ID(), "zeid2")
	equal(t, "jobs[1].Name()", jobs[1].Name(), "zename2")
	equal(t, "jobs[1].URL()", jobs[1].URL(), "zeurl2")
}

func TestAddWithDefaultPrefix(t *testing.T) {
	metadata := xmpbox.CreateXMPMetadata()
	basic, err := metadata.CreateAndAddBasicJobTicketSchema()
	noError(t, "CreateAndAddBasicJobTicketSchema", err)
	noError(t, "AddJobOf", basic.AddJobOf("zeid2", "zename2", "zeurl2"))

	jobs := jobTicketRoundTrip(t, metadata)
	if len(jobs) != 1 {
		t.Fatalf("Jobs() held %d jobs, want 1", len(jobs))
	}
	equal(t, "jobs[0].ID()", jobs[0].ID(), "zeid2")
	equal(t, "jobs[0].Name()", jobs[0].Name(), "zename2")
	equal(t, "jobs[0].URL()", jobs[0].URL(), "zeurl2")
}

func TestAddWithDefinedPrefix(t *testing.T) {
	metadata := xmpbox.CreateXMPMetadata()
	basic, err := metadata.CreateAndAddBasicJobTicketSchema()
	noError(t, "CreateAndAddBasicJobTicketSchema", err)
	noError(t, "AddJobOfPrefix", basic.AddJobOfPrefix("zeid2", "zename2", "zeurl2", "aaa"))

	jobs := jobTicketRoundTrip(t, metadata)
	if len(jobs) != 1 {
		t.Fatalf("Jobs() held %d jobs, want 1", len(jobs))
	}
	job := jobs[0]
	equal(t, "job.ID()", job.ID(), "zeid2")
	equal(t, "job.Name()", job.Name(), "zename2")
	equal(t, "job.URL()", job.URL(), "zeurl2")
	equal(t, "job.Namespace()", job.Namespace(),
		xmptype.Job.StructuredTypeInfo().Namespace)
	equal(t, "job.Prefix()", job.Prefix(), "aaa")
}
