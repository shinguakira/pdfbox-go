# track/testdata-podofo

Fetch every PDF PoDoFo keeps as test data, run the Go version against PDFBox over
them, and fix what they show the Go version doing differently.

**Branch: `track/testdata-podofo`** — made on 2026-09-17 from `migration-base` at
`c64b46eb5`, on the user's instruction. It is `track-testdata-sources.md`'s T1,
and #32 of #29 in the issues.

What the test data is and how to fetch it is in
[`../TESTDATA.md`](../TESTDATA.md); "PoDoFo against the Java" there has the
findings in full. This file is where the work stands.

## Rules

The rules of [`track-testdata-sources.md`](track-testdata-sources.md), unchanged:

- **The Java tree is read-only**, and PDFBox is the reference: a disagreement is
  the Go version's to explain, never the Java's to change.
- **Nothing fetched is committed.** `go/testdata/corpus/` and
  `go/testdata/oracle/` are ignored. `scripts/passwords/podofo.tsv` is committed:
  it holds file names and passwords read from PoDoFo's tests, and no content.
- **A disagreement in the port is fixed test-first**, with PDFBox's own output as
  the expected value. A bug found in the Java is recorded in
  [`../JAVA-BUGS.md`](../JAVA-BUGS.md), not fixed in the Java.
- **The oracle never gets worse.**
- **Pure Go**, and no new dependency without a decision.

## What is on disk

| Suite | PDFs | Where | Revision |
| --- | ---: | --- | --- |
| `podofo` | 102 | `go/testdata/corpus/podofo` | `podofo-resources` `master` at `92034ab82` |

Every PDF the repository commits, at its path there, each checked against its git
blob id: 63 at the root, `TechDocs/` 28, `ParserTests/` 7, `PQC/` 2,
`PDFUA-Reference/` 1, `Corrupted/` 1. The repository declares no licence.

## Encrypted files

32 name an encryption dictionary.

| Kind | Files | Opened with |
| --- | ---: | --- |
| RC4 40 to 128 and AESV2, AESV3R6, each with a key-length-violation twin | 14 | `userpass` and `ownerpass`, `test/unit/EncryptTest.cpp` |
| `/EncryptMetadata false` | 2 | `userpass`, the same file |
| escaped strings | 2 | `userpass`, `test/unit/StringTest.cpp`, and no password |
| `owner_user.pdf` | 1 | `user` and `owner`, `test/unit/Permissions.cpp` |
| Adobe and ISO documents under `TechDocs/` | 13 | nothing: they open with no password, and no test opens them with one |

Every candidate — no password, `userpass`, `ownerpass`, `user`, `owner` — tried
on all 32: 160 openings, and both sides accepted the same 49 and refused the same
111.

## Compared with PDFBox, 2026-09-17

```
102 files compared in 119 rows, the passwords tables opening some more than one way

  open    both 119, neither 0, behind 0, ahead 0
  pages   0 disagree
  text    both 119, neither 0, behind 0, ahead 0
  chars   118 the same length, 1 not
  digest  118 of the same length the same text, 0 not

  1 of 102 files disagree (0.98%)
```

Every file opens on both sides and every page count agrees. The one text that
differs is `TechDocs/adobe_supplement_iso32000_1.pdf`, page 7, 2,000 characters
against 2,002: **JAVA-BUGS 23**, fixed in the Go on purpose. Two CambriaMath
subsets map code 1 to `<>` in their ToUnicode CMaps; PDFBox extracts a NUL for
each glyph, the Go nothing. It is the first real document found to reach that
entry, which `JAVA-BUGS.md` now records.

## Found in the port

Nothing from the 102 PoDoFo files.

The facet comparison that followed on this branch -- twelve facets per opening
over the whole corpus, not only PoDoFo -- found nine, all fixed here with tests
whose expected values are PDFBox's. `migration/STATUS.md` has the table and
`migration/TESTDATA.md` the run.

The review of that work found one more: the renderer drew every stroke at its
72 dpi width whatever the resolution, which a comparison at 72 dpi cannot show.
`migration/STATUS.md` has it, under the review of the comparison.

## Tooling changed on this branch

- `cmd/corpus` gained the facet, write and render modes: `-facets`,
  `-facetlines`, `-comparefacets`, `-writes`, `-writelines`, `-renderpages`,
  `-comparerender` and `-workers`, in `facets.go`, `facetcompare.go`,
  `writes.go` and `render.go`.
- `migration/oracle` gained `JavaFacets.java`, `JavaWrites.java` and
  `JavaRender.java`, and `run-oracle.ps1` the switches that reach them.
- `fetch-corpus.ps1` has a `podofo` suite: a partial clone of `podofo-resources`
  and a sparse checkout of its PDFs, with `passwords/podofo.tsv` written to
  `_passwords.tsv`.

## Open tasks

**A lenient JPEG decoder, or not.** `image/jpeg` refuses two JPEGs of the corpus
that `libjpeg` decodes with a warning -- `pdfjs/bug1130815.pdf` "bad Huffman
code" and `pdfjs/issue9679.pdf` "missing 0xff00 sequence". The raw stream bytes
are identical on both sides, so the difference is the decoder alone, and the
standard library has no switch for it. Matching PDFBox means a JPEG decoder of
our own, or a fork of `image/jpeg` whose Huffman and marker errors stop the scan
rather than the image; both are large enough to need a decision first. Two files
of 40,255.

**The writes and the render comparison, at full scale.** Both are compared over
every sixteenth opening, 2,513 of the 40,255. For render that was because the port
drew a page five to six times slower than PDFBox; after the compositor fix it is
twice as slow on `sample.pdf`, 12 seconds against 6. For writes it rested on a
wrong estimate: eleven minutes the first run spent on
`PdfReaderTest/exponentialXObjectLoop.pdf`, iText's document of exponentially
nested form XObjects, were taken for the rate of the whole list. That file is as
slow in the port as in PDFBox.

**The compositing of a non-isolated group under a blend mode.**
`pdfjs/nonisolated_blend_smask.pdf` draws on both sides now, and what the port
draws is 23.3 levels a cell away from PDFBox's -- the only page of the corpus
that far out. The page's own text says what right looks like: "The two blue boxes
should match" and "text should stay visible". Nothing else in the render
comparison is beyond 16 levels, so this is one document's worth of
backdrop-and-blend arithmetic, measured and not yet run down.

**Java2D's thin strokes.** A stroke at most an eighth of a pixel wide in device
space is drawn an eighth of a pixel wide when anti-aliased, and one of a pixel or
less is drawn by a separate one-pixel pipeline when not. The port draws both at
their own width. The first happens only at 36 dpi and below, the second on every
`BINARY` page. Found while fixing the stroke width at other resolutions, which
the review of pull request #39 led to; `migration/STATUS.md` has both.
