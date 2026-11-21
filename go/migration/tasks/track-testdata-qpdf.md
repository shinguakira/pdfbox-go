# track/testdata-qpdf

Fetch the 78 PDFs qpdf keeps outside the one directory the suite took, open the
encrypted ones the way qpdf's own tests do, and compare both sides over all of
it.

**Branch: `track/testdata-qpdf`** — made on 2026-10-08 from `migration-base` at
`0e5e70abc`, on the user's instruction. It is `track-testdata-sources.md`'s next
test-data track, and #31 of #29 in the issues.

What the test data is and how to fetch it is in
[`../TESTDATA.md`](../TESTDATA.md); "qpdf against the Java" there has the
findings in full. This file is where the work stands.

## Rules

The rules of [`track-testdata-sources.md`](track-testdata-sources.md), unchanged:

- **The Java tree is read-only**, and PDFBox is the reference: a disagreement is
  the Go version's to explain, never the Java's to change.
- **Nothing fetched is committed.** `go/testdata/corpus/` and
  `go/testdata/oracle/` are ignored. `scripts/passwords/qpdf.tsv` is committed:
  it holds file names and passwords read from qpdf's tests, and no content.
- **A disagreement in the port is fixed test-first**, with PDFBox's own output as
  the expected value. A bug found in the Java is recorded in
  [`../JAVA-BUGS.md`](../JAVA-BUGS.md), not fixed in the Java.
- **The oracle never gets worse.**
- **Pure Go**, and no new dependency without a decision.

## What is on disk

| Suite | PDFs | Where | Revision |
| --- | ---: | --- | --- |
| `qpdf` | 717 | `go/testdata/corpus/qpdf` | `qpdf` `main`, the archive of 2026-10-08 |

The suite took `qpdf/qtest/qpdf` alone, 639 of the 717. The other four
directories are fetched now, each under its own name beside the first:

| In the repository | Here | PDFs | What they are |
| --- | --- | ---: | --- |
| `qpdf/qtest/qpdf` | the suite's root | 639 | the regression inputs: xref tables and streams in every arrangement, object streams of every shape, linearised and not, damaged xrefs, every encryption revision qpdf writes |
| `examples/qtest` | `examples/` | 49 | the inputs and the expected output of the C and C++ examples |
| `compare-for-test/qtest` | `compare-for-test/` | 21 | the pairs `qpdf-test-compare` is run on |
| `libtests/qtest` | `libtests/` | 6 | the filter tests |
| `qpdf/qtest/storage` | `storage/` | 2 | two files a storage test reads |

`fetch-corpus.ps1` gained a `Subtrees` field for it — an archive suite could
take one subtree before — and `Copy-SuiteSubtrees` copies each of the four out
of the unpacked archive. The 639 stay at the suite's root, where
`oracle/list.txt` and every table taken from it already expect them, which is
why the four are not mirrored at their repository paths.

## The encrypted files

Twenty-four of the 639 refused to open on both sides because no table held their
passwords, and that refusal was all the comparison could compare. The table now
holds **29 files and 42 ways of opening them**, read from qpdf's own scripts:

| Where the passwords come from | Files |
| --- | --- |
| `encryption.test`, `@encrypted_files` — the Acrobat 5.0 and Acrobat XI files qpdf decrypts and re-encrypts, with the password in the table's second column | the 15 `enc-*`, each with the user password and, where the file has one, the owner's |
| the same table's long-password rows | `enc-long-password.pdf` and `enc-XI-long-password.pdf`, the 32-byte prefix that opens the R4 file and the whole string the R6 one needs |
| `encryption.test`, `@cenc` — the C API tests, which write one revision per row and check the committed output with the password in its last column | `c-r2`, `c-r3`, `c-r4`, `c-r5-in`, `c-r6-in` |
| `check-encryption.test` and `merge-and-split.test` | `20-pages.pdf` and `pages-copy-encryption.pdf`, which carries the same `/O` and `/U` because the merge copied them |
| `job-json-encrypt-40.json` and `-128.json`, which wrote the files they name | `job-json-encrypt-40.pdf`, `job-json-encrypt-128.pdf` |
| `specific-bugs.test`, `attachments.test`, `bookmarks.test` | `unfilterable-with-crypt.pdf`, the two attachment files, `examples/bookmarks/4.pdf` |
| `encrypted-40-bit-R3.out`, qpdf's own committed check output: "User password = 623" | `encrypted-40-bit-R3.pdf` |

**One file no password opens.** `bad-encryption-length.pdf` has an `/Encrypt`
whose `/Length` is wrong. qpdf opens it with an empty user password and says so
in its expected output; PDFBox refuses it with `InvalidPasswordException`
whatever is supplied. So no line opens it, both sides refuse it, and that is the
comparison: PDFBox is the reference, and what qpdf can do with the file is not
the question here.

## Compared with PDFBox, 2026-10-08

Five comparisons over all 717 files and the 730 openings the passwords table
makes of them. `../TESTDATA.md`, "qpdf against the Java", has the numbers and
what every difference is made of; in short:

| Comparison | Result |
| --- | --- |
| document: opens, page count, text | **0 of 717 disagree** — the same 708 open, the same 22 refuse, for the same reasons one for one |
| facets: ten of the twelve | **all 708 the same** |
| facets: `images`, `imagepixels` | 25 and 26 files differ: 57 lines are DCT, 3 are PDFBox throwing, nothing else |
| write paths: save, incremental, encrypt, split, merge, overlay, sign | **all 708 the same**, on every one of the seven |
| render, page by page | of 4,617: 115 identical to the last bit, 4,488 within a level of a cell, 6 within sixteen, 6 further, 0 where one side fails, 2 where both do |

## Found in the port

**Nothing.** Every difference above is a deviation already recorded, and the two
that are not differences in the port at all are corrections to the records:

- **`JAVA-BUGS.md` 90 said one file reached it; three do.**
  `indirect-filter.pdf`, `indirect-filter-out-0.pdf` and
  `indirect-filter-out-1.pdf` each make PDFBox raise `ClassCastException` out of
  the images facet. All three were on disk when the entry claimed
  `indirect-filter-out-0.pdf` was "the only file of 40,255": the claim came from
  a classification of 313 sampled files, not from the whole run. The entry now
  says that, and names the three.
- **"A truncated image keeps the rows that read" now has three more documents.**
  `STATUS.md` records the deviation and had measured it on `itext-java`'s
  `calRgb16bpc.pdf`. qpdf's `png-filters-no-columns.pdf`,
  `png-filters-no-columns-decoded.pdf` and `png-filters-1-column.pdf` reach the
  same guard by another route — a 16-bit image whose `/Predictor 15` carries no
  `/Columns`, so the predictor runs with `/Columns` 1 and the data runs short —
  and they are the six pages that differ most in the whole suite: PDFBox throws
  `EOFException` out of `SampledImageReader.fromAny`, loses the image and draws a
  blank page, where the port draws the rows that were there. `issue-117.pdf` is a
  fourth document for it.

## Open tasks

**Nothing of the suite.** All 717 files are fetched and scored five ways, the 24
that no password opened are down to one, and that one is explained.

The question it hands on is the same one PoDoFo's and PDFium's tracks left: a
lenient JPEG decoder, or not. The 57 DCT lines here are that difference again.
