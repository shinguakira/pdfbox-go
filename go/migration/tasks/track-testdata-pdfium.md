# track/testdata-pdfium

Fetch every PDF PDFium keeps as test data, run the Go version against PDFBox over
them, and fix what they show the Go version doing differently.

**Branch: `track/testdata-pdfium`** — made on 2026-09-29 from `migration-base` at
`90e1a7b42`, on the user's instruction. It is `track-testdata-sources.md`'s next
test-data track, and #33 of #29 in the issues.

What the test data is and how to fetch it is in
[`../TESTDATA.md`](../TESTDATA.md); "PDFium against the Java" there has the
findings in full. This file is where the work stands.

## Rules

The rules of [`track-testdata-sources.md`](track-testdata-sources.md), unchanged:

- **The Java tree is read-only**, and PDFBox is the reference: a disagreement is
  the Go version's to explain, never the Java's to change.
- **Nothing fetched is committed.** `go/testdata/corpus/` and
  `go/testdata/oracle/` are ignored. `scripts/passwords/pdfium.tsv` is committed:
  it holds file names and passwords read from PDFium's tests, and no content.
- **A disagreement in the port is fixed test-first**, with PDFBox's own output as
  the expected value. A bug found in the Java is recorded in
  [`../JAVA-BUGS.md`](../JAVA-BUGS.md), not fixed in the Java.
- **The oracle never gets worse.**
- **Pure Go**, and no new dependency without a decision.

## What is on disk

| Suite | PDFs | Where | Revision |
| --- | ---: | --- | --- |
| `pdfium` | 341 | `go/testdata/corpus/pdfium` | `pdfium` `main` at `8e99133990b55ea7023ebbdd6a2fa391bec9e9f4` |

Every PDF the repository commits, at its path there, each checked against its git
blob id: 296 in `testing/resources`, `pixel/` 24, `pixel/xfa_specific/` 9,
`xfa/` 6, `javascript/xfa_specific/` 6. 3.5 MB. BSD-3-Clause.

The issue says 301; it is 341 now, counted from the tree at that commit.

## The `.in` templates, which are not fetched

PDFium writes most of its test inputs as templates rather than PDFs:
`testing/resources` holds **562** `.in` files against those 341 PDFs.
`testing/tools/fixup_pdf_template.py` expands one into a PDF, replacing nine
directives — `{{header}}`, `{{object x y}}`, `{{streamlen}}`, `{{xref}}`,
`{{trailer}}`, `{{trailersize}}`, `{{startxref}}`, `{{startxrefobj x y}}` and
`{{include path}}` — with the byte offsets a PDF needs.

| | count |
| --- | ---: |
| `.in` templates | 562 |
| of those, committed as a `.pdf` beside the template | 237 |
| of those, **only** a template | 325 |
| committed `.pdf` with no template | 104 |

**None of the 325 should be expanded here**, and the directory they live in says
why: 197 are in `pixel/`, 22 in `pixel/xfa_specific/`, 4 in its `use_ahem` and
`use_symbolneu` subdirectories, 49 in `javascript/` and 53 in
`javascript/xfa_specific/`. Every one is a pixel test — compared against a
committed `.png`, which #29 puts out of scope — or a JavaScript or XFA test,
which PDFBox has no engine for. The templates that carry parser and document
structure, the ones this comparison could ask something about, are exactly the
237 whose expansion PDFium commits, and those are on disk already. Expanding the
rest would need the Python tooling this repository does not run, and would answer
only "does it open".

## Encrypted files

Ten of the 341 name an encryption dictionary.

| File | Opened with | From |
| --- | --- | --- |
| `encrypted.pdf` | `1234` (user), `5678` (owner) | `cpdf_security_handler_embeddertest.cpp` |
| `bug_644.pdf` | `a` (owner), `b` (user), AESV3 revision 5 | the same |
| `encrypted_hello_world_r{2,3,5,6}.pdf` | `âge` (owner), `hôtel` (user) | the same, each tested as UTF-8 and as Latin-1 |
| `encrypted_hello_world_r{2,3}_bad_okey.pdf` | `a`, which PDFium expects to fail | the same, crbug.com/42270437 |
| `bug_1124998.pdf`, `bug_424613308.pdf` | nothing: they open with no password on both sides | the security-handler test above does not name them |

[`../scripts/passwords/pdfium.tsv`](../scripts/passwords/pdfium.tsv) holds them.
A password there is a string, not a byte sequence: PDFium tests each of `âge` and
`hôtel` twice, once as UTF-8 bytes and once as Latin-1, and both PDFBox and the
port encode a password the way the revision asks — ISO-8859-1 for revisions 2 to
4, UTF-8 for 5 and 6 — so one line covers both of PDFium's cases.

## Compared with PDFBox, 2026-09-29

Five comparisons, each over all 341 files and the 347 openings the passwords
table makes of them. The numbers and what each difference is made of are in
`../TESTDATA.md`; in short:

| Comparison | Result |
| --- | --- |
| document: opens, page count, text | **0 of 341 disagree** — the same 322 open, the same 25 refuse, every page count and every text digest the same |
| facets: ten of the twelve | **all 322 the same** — positions, info, XMP, XMP schemas, outline, labels, boxes, structure, annotations, fields |
| facets: `images` and `imagepixels`, the other two | 6 and 7 differ, every one a JPEG |
| write paths: save, incremental, encrypt, split, merge, overlay, sign | **all 322 the same**, on every one of the seven |
| render, page by page | of 381: 173 identical to the last bit, 193 within a level of a cell, 5 further than that, 1 where one side fails, 9 where both do |

## Found in the port

**One defect, fixed here.** `bug_481363.pdf` is damaged in a way that leaves its
`/CS1` colour space unresolvable. PDFBox logs that and draws the rest of the page
— `PDColorSpace.create` throws `MissingResourceException` and
`PDFStreamEngine.operatorException` swallows it — and the port ended the page
with the error instead. The cause was the port's own: Java has one
`MissingResourceException`, and the port had grown **two** sentinels for it, one
in `pdmodel` and one in `pdmodel/graphics/color`, because the colour spaces
cannot import `pdmodel`. `OperatorException` knew only the first. The sentinel
now lives in `pdmodel/common`, which both already import, and the two names are
that one error.

`TestAMissingColourSpaceDoesNotEndThePage` renders a page whose colour space is
missing and requires the fill to still reach the backend;
`TestAMissingColourSpaceIsAMissingResource` pins the sentinel itself, so that a
reader of either package sees why there is only one. The page then renders
identically to PDFBox's.

## Explained, not defects

- **JPEG decoded pixels, 6 files.** Both sides decode, the lengths agree, and the
  bytes differ: 1 to 3 levels on `jpeg_reduced_size.pdf`,
  `jpeg_reduced_size_with_smask.pdf` and `jpeg_unaligned_no_reduce.pdf`, and up
  to 38 on the three 4:2:0 images of `bug_650.pdf`. Two decoders, both within the
  JPEG standard: Go's `image/jpeg` against the libjpeg-derived reader Java's
  ImageIO uses, different inverse DCT, and nearest-neighbour chroma upsampling
  against libjpeg's fancy upsampling where the image is subsampled.
- **`pixel/bug_603518.pdf`.** Its JPEG declares 640 by 63,760 pixels in 6,382
  bytes. Java's reader returns the whole 122,419,200-byte buffer; `image/jpeg`
  stops at "bad Huffman code" and the port has no image. This is the open
  question `track-testdata-podofo.md` left — a lenient JPEG decoder, or not — and
  its third instance.
- **`bug_42270471.pdf`**, every pixel of a 50 by 50 image out by up to 80 levels
  on a channel: `PDICCBased` always takes the `/Alternate` colour space here,
  which `STATUS.md` records, so an ICC profile that says a warm grey comes out as
  `DeviceGray`.
- **`bug_1549.pdf`** and **`pattern_stroke.pdf`**, 50 and 13 levels: both draw in
  `DeviceCMYK` — `1 0 0 0` and `0 1 0.91 0` — and `PDDeviceCMYK` converts
  naively where Java converts through an ICC profile. Also recorded.
- **`pixel/bug_42271010.pdf`**, 34 levels: its tiling pattern has
  `/XStep 200.001`, so the port's raster is 201 pixels where Java truncates to
  200. `JAVA-BUGS.md` 85, fixed in the Go on purpose.
- **`pixel/bug_440028542.pdf`**, the one page where a side fails: PDFBox dies on
  it and the port does not. Its type 6 shading names a type 4 function that
  returns 2 values where the `/Range` asks for 3, and PDFBox throws
  `IllegalStateException` from `PDFunctionType4.eval` while building the pixel
  table, out through `AlphaPaintPipe.startSequence`, killing the page. The port's
  own type 4 function raises the same error when it is evaluated — measured — but
  its mesh paints no pixels for this shading, whose `/Decode` gives the parameter
  a range of plus and minus `FLT_MAX`, so the function is never asked. The port
  draws the page white.

## Open tasks

**Nothing is left of the suite.** Every file is fetched, every file is scored
four ways, and every difference above is either fixed or named.

The two questions it raises for elsewhere:

- **A lenient JPEG decoder, or not.** Now three files: PDFium's
  `pixel/bug_603518.pdf` and PoDoFo's two. `track-testdata-podofo.md` has the
  decision as stated; this branch does not take it.
- **PDFBox aborting a page on a malformed type 4 function** is filed as
  `JAVA-BUGS.md` **91**: an unchecked `IllegalStateException` out of a paint
  context, where PDFBox's own convention for malformed content is to log and
  carry on. Certain in what the Java does — the stack trace in the entry is
  PDFBox's own — and uncertain only in scope: the port does not reach that code
  path on this document, so its carry of the defect is untested. Testing it needs
  a document whose mesh the port paints and whose function is short.

  Noticed while filing it: the reproduction-group index at the top of
  `JAVA-BUGS.md` does not carry entries **89 and 90**, and did not when they were
  written. 91 is placed in group B and the count corrected; 89 and 90 are left
  for whoever judges where they belong.
