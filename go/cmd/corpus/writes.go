package main

// What the port writes, read back: the write paths it implements, run on each
// document and summarised the way migration/oracle/JavaWrites.java summarises
// PDFBox's.
//
// Each write facet loads the document afresh, does one thing to it, writes it,
// loads what it wrote, and summarises that as its page count and a digest of its
// text. The bytes of two writers are not expected to agree -- object numbering,
// spacing and random encryption keys differ -- so they are not compared; what
// each writer's output reads back as is. A document opened with a password or a
// key is not written: the write facets are about the writer, and encryption on
// writing has a facet of its own.

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/multipdf"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/encryption"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/digitalsignature"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/form"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/text"
)

// writeNames are the write columns, in the order JavaWrites.NAMES has them.
var writeNames = []string{"save", "incremental", "encrypt", "split", "merge", "overlay", "sign"}

var writeGroup = group{names: writeNames, child: "-onewrites"}

// maxParts is how many split parts are summarised; beyond it they are counted.
const maxParts = 200

func summary(doc *pdmodel.PDDocument) (string, error) {
	t, err := text.NewPDFTextStripper().GetText(doc)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("pages=%d text=%s", doc.NumberOfPages(), digest(t)), nil
}

func saved(doc *pdmodel.PDDocument) ([]byte, error) {
	var out bytes.Buffer
	if err := doc.Save(&out); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// readBack loads what a write produced and writes its summary to the facet.
func readBack(written []byte, password string, f *facet, suffix func(*pdmodel.PDDocument) string) error {
	var back *pdmodel.PDDocument
	var err error
	if password == "" {
		back, err = pdfbox.LoadPDFBytes(written)
	} else {
		back, err = pdfbox.LoadPDFBytesWithPassword(written, password)
	}
	if err != nil {
		return err
	}
	defer back.Close()
	line, err := summary(back)
	if err != nil {
		return err
	}
	if suffix != nil {
		line += suffix(back)
	}
	f.line(line)
	return nil
}

func writeSave(j job, f *facet) error {
	doc, err := openDocument(j)
	if err != nil {
		return err
	}
	defer doc.Close()
	written, err := saved(doc)
	if err != nil {
		return err
	}
	return readBack(written, "", f, nil)
}

func writeIncremental(j job, f *facet) error {
	doc, err := openDocument(j)
	if err != nil {
		return err
	}
	defer doc.Close()
	info := doc.DocumentInformation()
	info.SetTitle("corpus facet")
	info.Dictionary().SetNeedToBeUpdated(true)
	var out bytes.Buffer
	if err := doc.SaveIncremental(&out); err != nil {
		return err
	}
	return readBack(out.Bytes(), "", f, func(back *pdmodel.PDDocument) string {
		return " title=" + fstr(back.DocumentInformation().Title())
	})
}

func writeEncrypt(j job, f *facet) error {
	doc, err := openDocument(j)
	if err != nil {
		return err
	}
	defer doc.Close()
	policy := encryption.NewStandardProtectionPolicy("corpus-owner", "corpus-user", encryption.NewAccessPermission())
	if err := policy.SetEncryptionKeyLength(256); err != nil {
		return err
	}
	if err := doc.Protect(policy); err != nil {
		return err
	}
	written, err := saved(doc)
	if err != nil {
		return err
	}
	return readBack(written, "corpus-user", f, func(back *pdmodel.PDDocument) string {
		return fmt.Sprintf(" encrypted=%t", back.IsEncrypted())
	})
}

func writeSplit(j job, f *facet) error {
	doc, err := openDocument(j)
	if err != nil {
		return err
	}
	defer doc.Close()
	parts, err := multipdf.NewSplitter().Split(doc)
	defer func() {
		for _, part := range parts {
			part.Close()
		}
	}()
	if err != nil {
		return err
	}
	f.line(fmt.Sprintf("parts=%d", len(parts)))
	for i, part := range parts {
		if i >= maxParts {
			break
		}
		line, err := summary(part)
		if err != nil {
			return err
		}
		f.line(fmt.Sprintf("part %d %s", i, line))
	}
	return nil
}

func writeMerge(j job, f *facet) error {
	first, err := openDocument(j)
	if err != nil {
		return err
	}
	defer first.Close()
	second, err := openDocument(j)
	if err != nil {
		return err
	}
	defer second.Close()
	destination := pdmodel.NewPDDocument()
	defer destination.Close()
	merger := multipdf.NewPDFMergerUtility()
	if err := merger.AppendDocument(destination, first); err != nil {
		return err
	}
	if err := merger.AppendDocument(destination, second); err != nil {
		return err
	}
	written, err := saved(destination)
	if err != nil {
		return err
	}
	return readBack(written, "", f, nil)
}

func writeOverlay(j job, f *facet) error {
	input, err := openDocument(j)
	if err != nil {
		return err
	}
	defer input.Close()
	over, err := openDocument(j)
	if err != nil {
		return err
	}
	defer over.Close()
	overlay := multipdf.NewOverlay()
	overlay.SetInputPDF(input)
	overlay.SetDefaultOverlayPDF(over)
	result, err := overlay.Overlay(map[int]string{})
	if err != nil {
		return err
	}
	written, err := saved(result)
	if err != nil {
		return err
	}
	return readBack(written, "", f, nil)
}

func writeSign(j job, f *facet) error {
	doc, err := openDocument(j)
	if err != nil {
		return err
	}
	defer doc.Close()
	signature := digitalsignature.NewPDSignature()
	signature.SetFilter(digitalsignature.FilterAdobePPKLite)
	signature.SetSubFilter(digitalsignature.SubFilterAdbePkcs7Detached)
	signature.SetName("corpus facet")
	signature.SetSignDate(time.UnixMilli(1767225600000).UTC())
	if err := form.AddSignature(doc, signature, nil); err != nil {
		return err
	}
	var out bytes.Buffer
	support, err := form.SaveIncrementalForExternalSigning(doc, &out)
	if err != nil {
		return err
	}
	content, err := support.Content()
	if err != nil {
		return err
	}
	handed, err := io.ReadAll(content)
	if err != nil {
		return err
	}
	if err := support.SetSignature([]byte{0x30, 0x03, 0x02, 0x01, 0x00}); err != nil {
		return err
	}
	written := out.Bytes()
	back, err := pdfbox.LoadPDFBytes(written)
	if err != nil {
		return err
	}
	defer back.Close()
	signatures := form.SignatureDictionariesOfDocument(back)
	if len(signatures) == 0 {
		return errors.New("no signature read back")
	}
	last := signatures[len(signatures)-1]
	ranges := last.ByteRange()
	covers := len(ranges) == 4 && ranges[0] == 0 && ranges[2]+ranges[3] == len(written)
	signed, err := last.SignedContentOfBytes(written)
	if err != nil {
		return err
	}
	line, err := summary(back)
	if err != nil {
		return err
	}
	f.line(fmt.Sprintf("%s signatures=%d ranges=%d covers=%t handed=%t", line, len(signatures), len(ranges),
		covers, bytes.Equal(handed, signed)))
	return nil
}

var writeComputations = []func(job, *facet) error{
	writeSave, writeIncremental, writeEncrypt, writeSplit, writeMerge, writeOverlay, writeSign,
}

// writesNow runs one way of opening one file's write facets in this process, in
// the shape facetsNow answers.
func writesNow(j job, keep bool) []string {
	document, err := func() (d *pdmodel.PDDocument, err error) {
		defer func() {
			if p := recover(); p != nil {
				err = fmt.Errorf("panic: %v", p)
			}
		}()
		return openDocument(j)
	}()
	if err != nil {
		if keep {
			return []string{"open\t" + fstr(short(err))}
		}
		row := []string{short(err)}
		for range writeNames {
			row = append(row, "-")
		}
		return []string{strings.Join(row, "\t")}
	}
	encrypted := document.IsEncrypted()
	document.Close()
	if j.open >= 0 || encrypted {
		if keep {
			return []string{"open\tencrypted, not written"}
		}
		row := []string{"ok"}
		for range writeNames {
			row = append(row, "-")
		}
		return []string{strings.Join(row, "\t")}
	}

	row := []string{"ok"}
	var lines []string
	for i, compute := range writeComputations {
		f := newFacet(keep)
		err := func() (err error) {
			defer func() {
				if p := recover(); p != nil {
					err = fmt.Errorf("panic: %v", p)
				}
			}()
			return compute(j, f)
		}()
		if err != nil {
			row = append(row, "error")
			if keep {
				f.kept = append(f.kept, "error "+fstr(short(err)))
			}
		} else {
			row = append(row, f.cell())
		}
		for _, line := range f.kept {
			lines = append(lines, writeNames[i]+"\t"+line)
		}
	}
	if keep {
		return lines
	}
	return []string{strings.Join(row, "\t")}
}
