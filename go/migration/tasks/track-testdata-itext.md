# track/testdata-itext

Fetch every PDF iText Core keeps as test data, in its Java and its .NET
repository, run the Go version against PDFBox over them, and fix what they show
the Go version doing differently.

**Branch: `track/testdata-itext`** — made on 2026-09-16 from
`track/testdata-sources` at `b023f3d52`, on the user's instruction, and merged
back into it the same day. It is that branch's T5, taken whole.

What the test data is and how to fetch it is in
[`../TESTDATA.md`](../TESTDATA.md); "iText against the Java" there has the
findings in full. This file is where the work stands and what is left.

## Rules

The rules of [`track-testdata-sources.md`](track-testdata-sources.md), unchanged:

- **The Java tree is read-only**, and PDFBox is the reference: a disagreement is
  the Go version's to explain, never the Java's to change.
- **Nothing fetched is committed.** iText's documents are AGPL;
  `go/testdata/corpus/` and `go/testdata/oracle/` are ignored. The two password
  tables in `go/migration/scripts/passwords/` are committed: they hold file
  names, passwords and key file names read from iText's tests, and no content.
- **A disagreement is fixed test-first**, with PDFBox's own output as the
  expected value. A bug found in the Java is recorded in
  [`../JAVA-BUGS.md`](../JAVA-BUGS.md), not fixed in the Java.
- **The oracle never gets worse.** After any change to the Go version,
  `corpus -oracle` must not report more disagreements than before it.
- **Pure Go**, and no new dependency without a decision.

## What is on disk

| Suite | PDFs | Where | Revision |
| --- | ---: | --- | --- |
| `itext-java` | 6,897, and 25 certificate and key files | `go/testdata/corpus/itext-java` | `develop` at `b0b6709a4`, 2026-09-14 |
| `itext-dotnet` | 6,960, and 25 certificate and key files | `go/testdata/corpus/itext-dotnet` | `develop` at `3b871a861`, 2026-09-14 |

Every PDF each repository commits, at its path there, each checked against its
git blob id. Counted by content, 6,678 distinct files are in both repositories,
39 only in the Java one and 102 only in the .NET one: 6,819 distinct files in
all. Both suites are scored whole.

**Not on disk, and why:**

- **The PDFs iText's tests write.** They exist only after iText's Maven build
  has run its tests. What the tests compare them with, the `cmp_` files, is on
  disk.
- **iText's other 45 public repositories** — pdfHTML, pdfSweep, pdfOCR, iText 5,
  the published examples — about 21,000 PDFs between them, counted in
  [`../TESTDATA-CANDIDATES.md`](../TESTDATA-CANDIDATES.md). The request was iText
  Core's test resources; these were counted so that none is left unnamed, not
  fetched.
- **Nothing is behind a link.** iText's tests read no PDF from the network; the
  one PDF address in their sources is a signature policy's, written into a
  signature and never downloaded.

## Encrypted files

474 of the 13,857 name an encryption dictionary, 234 and 240:

| Kind | Files | Opened with |
| --- | ---: | --- |
| a password PDFBox needs | 364 | every password their test class holds that PDFBox opens them with |
| encrypted for a certificate | 80 | the certificate and key their test decrypts with |
| revision 7, AES-GCM | 12 | the passwords their tests use; PDFBox refuses revision 7 whatever the password |
| open without a password | 14 | nothing for 6; the owner password as well for 8 |
| a custom security handler, `/Standart` and `/iText` | 4 | nothing; neither side has the handler |

The two tables, `scripts/passwords/itext-java.tsv` and `itext-dotnet.tsv`, give
464 files 718 ways of being opened. How they were made is in `TESTDATA.md`. Run
once more with all 47 candidate passwords on each of the 378 files a password
opens or that open with none, both sides accepted the same 622 opens and refused
the same 17,144.

## Compared with PDFBox, 2026-09-16

Over iText, after the fixes below:

```
13857 files, opened 14111 ways
  open    both 14057, neither 54, behind 0, ahead 0
  pages   0 disagree
  text    both 14053, neither 4, behind 0, ahead 0
  chars   14053 the same length, 0 not
  digest  14053 of the same length the same text, 0 not

  0 of 13857 files disagree
```

And over every file on disk, the earlier 5,090 with them:

```
18947 files, opened 19201 ways, 27 of them encrypted and skipped
  open    both 19094, neither 107, behind 0, ahead 0
  pages   0 disagree
  text    both 19085, neither 9, behind 0, ahead 0
  chars   19083 the same length, 2 not
  digest  19083 of the same length the same text, 0 not

  2 of 18947 files disagree (0.01%)
```

The two are JAVA-BUGS 15 and 30, which the Go version fixes on purpose, as on
`track/testdata-sources`. The 27 encrypted files are that branch's U8.

## Found and fixed on this branch

- **Mixed-direction text came out with its runs in logical order.**
  `text.handleDirection` asked `golang.org/x/text/unicode/bidi`, which answers
  runs in logical order and no levels, where PDFBox reorders them with
  `Bidi.reorderVisually`. `kernel/parser/BidiTextExtractionTest/in02.pdf`, in
  both repositories, is Hebrew with years in parentheses: 179 characters on
  both sides, and the Go version's lines read from their middle. It now asks
  `go/javatext/bidi`. `go/pdfbox/text/direction.go`;
  `TestHandleDirectionPutsRunsInVisualOrder`.
- **A certificate recipient wrapped with RSAES-OAEP was refused.** `cms.go`
  unwrapped PKCS#1 v1.5 only; BouncyCastle, under PDFBox, unwraps OAEP too.
  Four files, two per repository, did not open.
  `go/pdfbox/pdmodel/encryption/cms.go`; `TestOAEPRecipientUnwraps`.
- **A position's text was reversed by script, not by direction.** Found in
  pdf.js's `TaroUTR50SortedList112.pdf` when the digest went over every corpus,
  not in iText's files. `TextPosition.getVisuallyOrderedUnicode` reverses a
  position holding an R or AL code point; the port asked whether a code point
  belonged to Hebrew, Arabic or one of five other scripts, which counted those
  scripts' marks and digits and missed every other right-to-left character. `go/pdfbox/text/textposition.go`;
  `TestVisuallyOrderedUnicodeAsksTheDirectionality`. The 58 characters Unicode 14
  added remain right to left here and undefined in JDK 17.

## Found and recorded

- **JAVA-BUGS 89.** PDFBox reads a version 6 certificate-encrypted document's
  256-bit key out of a 20-byte digest and throws
  `ArrayIndexOutOfBoundsException`; the Go version panics at the same copy.
  Four files. Carried, and pinned by
  `TestVersion6CertificateEncryptionReadsPastTheDigest`.

## Tooling changed on this branch

- `fetch-corpus.ps1` fetches a suite by partial clone and sparse checkout, checks
  every file against its blob id, and writes a committed passwords table into
  the suite, fetching the certificates and keys it names.
- A passwords table may open a file more than one way, and a line may give a
  certificate and a private key instead of a password. `cmd/corpus -passwords`
  and `run-oracle.ps1 -Passwords` take more than one table.
- Both tables carry a digest of the text, and `-oracle` compares it where the
  lengths agree. This is `track-testdata-sources.md`'s T6, for the whole
  document; per page it is not done.
- `JavaCorpus` writes its table in UTF-8 and `run-oracle.ps1` reads it so; rows
  named by an Arabic password came back as question marks.
- `cmd/corpus` writes a panic to the stage that was running. A panic while
  opening used to be written to the text column, with the open column left
  `ok`.

## Open tasks

- [ ] **I1. Per-page digests.** The digest says that a document's text differs,
  not where. Both of the digest's findings on this branch were located by
  dumping each side's whole text, and for `TaroUTR50SortedList112.pdf` every
  text position as well.
  A digest per page would narrow the next one to a page.
- [ ] **I2. iText's other 45 repositories.** About 21,000 PDFs, 15,279 of them
  pdfHTML's output. Counted and not fetched; whether any of them is worth
  carrying needs a decision.
