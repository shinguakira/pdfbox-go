# track/testdata-itext

Fetch every PDF iText Core keeps as test data, in its Java and its .NET
repository, run the Go version against PDFBox over them, and fix what they show
the Go version doing differently. Since I2 below, the same for every other
repository iText publishes: 34 suites, 34,770 files.

**Branch: `track/testdata-itext`** — made on 2026-09-16 from
`track/testdata-sources` at `b023f3d52`, on the user's instruction, and merged
back into it the same day. It is that branch's T5, taken whole.

What the test data is and how to fetch it is in
[`../TESTDATA.md`](../TESTDATA.md); "iText against the Java" and "iText's other
repositories" there have the findings in full. This file is where the work
stands and what is left.

## Rules

The rules of [`track-testdata-sources.md`](track-testdata-sources.md), unchanged:

- **The Java tree is read-only**, and PDFBox is the reference: a disagreement is
  the Go version's to explain, never the Java's to change.
- **Nothing fetched is committed.** iText's documents are AGPL;
  `go/testdata/corpus/` and `go/testdata/oracle/` are ignored. The ten password
  tables in `go/migration/scripts/passwords/` are committed: they hold file
  names, passwords and the names of key and keystore files, read from iText's
  own samples and tests, and no content.
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
| iText's other 32 repositories | 20,913, and 2 keystores | one directory each under `go/testdata/corpus` | each repository's default branch, 2026-09-16, in `_revision.txt` |

Every PDF each repository commits, at its path there, each checked against its
git blob id. Counted by content, 6,678 distinct files are in both Core
repositories, 39 only in the Java one and 102 only in the .NET one: 6,819
distinct files in all. Every suite is scored whole.

**Not on disk, and why:**

- **The PDFs iText's tests write.** They exist only after iText's Maven build
  has run its tests. What the tests compare them with, the `cmp_` files, is on
  disk.
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

In the other 32 repositories, 35 files name an encryption dictionary. Eight more
tables — `i5js-sandbox.tsv`, `itext-publications-book-java.tsv`,
`-examples-java.tsv`, `-highlevel-java.tsv`, `-samples-dotnet.tsv`, `rups.tsv`,
`itextpdf.tsv`, `itextsharp.tsv` — give 28 of them 50 ways of being opened, read
from the samples and tests that write and read them; 21 open with no password
and keep their no-password line beside the owner's. Two of the tables name a
PKCS#12 keystore instead of a certificate and a key, which is the third shape a
passwords line now has. Run with all 13 candidate passwords those projects hold
against each of the 14 files a password touches, both sides accepted the same 20
opens and refused the same 162. `TESTDATA.md` has the two the material itself
keeps shut.

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

Over iText's other 32 repositories, fetched the same day under I2:

```
20913 files, opened 20935 ways
  open    both 20917, neither 17, behind 0, ahead 1
  pages   0 disagree
  text    both 20917, neither 0, behind 0, ahead 0
  chars   20917 the same length, 0 not
  digest  20917 of the same length the same text, 0 not

  1 of 20913 files disagree
```

The one is `itext-pdfhtml-dotnet`'s `background-size-near-zero-svg.pdf`, which
PDFBox did not finish inside the 20-second limit; given five minutes it reads it
and answers what the Go version answered in under a second.

And over every file on disk:

```
39860 files, opened 40136 ways, 27 of them encrypted and skipped
  open    both 40008, neither 125, behind 0, ahead 3
  pages   0 disagree
  text    both 39999, neither 9, behind 0, ahead 0
  chars   39997 the same length, 2 not
  digest  39997 of the same length the same text, 0 not

  5 of 39860 files disagree (0.01%)
```

Two of the five are JAVA-BUGS 15 and 30, which the Go version fixes on purpose,
as on `track/testdata-sources`; the other three are files PDFBox did not finish
inside its limit and agrees on when given five minutes. The 27 encrypted files
are that branch's U8.

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
- A line of three fields — a path ending, a passphrase and a PKCS#12 keystore —
  opens a file with a store the project publishes rather than with a certificate
  and a key this tooling puts into one. Both drivers hand the store to their
  loader as it is; `cmd/corpus/passwords_test.go` holds the three shapes apart.
  iText's samples ship `test.p12`, and without this the two files it decrypts —
  one in the Java examples, one in the .NET ones — could only be compared on
  both sides refusing them.
- Both tables carry a digest of the text, and `-oracle` compares it where the
  lengths agree. This is `track-testdata-sources.md`'s T6, for the whole
  document; per page it is a mode of its own, `-pages` and `-comparepages`,
  which I1 below describes.
- `JavaCorpus` writes its table in UTF-8 and `run-oracle.ps1` reads it so; rows
  named by an Arabic password came back as question marks.
- `cmd/corpus` writes a panic to the stage that was running. A panic while
  opening used to be written to the text column, with the open column left
  `ok`.

## Open tasks

- [x] **I1. Per-page digests.** Done 2026-09-16. Both drivers write a page
  table on request — `corpus -pages <out>` and `run-oracle.ps1 -Pages` — one row
  per page per way of opening a file, with that page's text digested the way the
  document's is, and `corpus -comparepages <java> <go>` joins the two and names
  every page they disagree on. It is a mode rather than a column because a page
  table over the whole corpus is a quarter of a million rows and extracting per
  page costs a pass per page; the way to use it is to score the corpus, see
  which files disagree, and run it over those. Over the two that do, it answers
  in seconds: `pdfjs/bug1175962.pdf` page 1, 117 characters against 126, and
  `pdfjs/poppler-90-0-fuzzed.pdf` page 10, 4 against 209 — which is the page
  whose content stream ends early under JAVA-BUGS 30's fix.
  `go/cmd/corpus/pages.go`, `pages_test.go`, `migration/oracle/JavaCorpus.java`,
  `scripts/run-oracle.ps1`.
- [x] **I2. iText's other 45 repositories.** Done 2026-09-16: **fetched and
  scored, all of them.** This was first recorded as "not carried", on the
  argument that 15,279 of the files are pdfHTML's `cmp_` files and most of the
  rest is expected output, so they are PDFs iText wrote and the 13,857 from that
  same writer had just agreed on every file. That is an argument about the
  producer and not a measurement of the files, and it was overruled: fetch them
  and open them.

  32 of the 45 hold PDFs — 20,913 files, 889 MB — and the other 13 hold none.
  Each is a suite of its own in `fetch-corpus.ps1`, fetched the way iText Core
  is, and every file was checked against its blob id. Both sides read all
  20,913: no file the Java opens the Go version does not, no page count
  disagrees, and all 20,917 openings that both read gave the same characters,
  digest for digest. What each group is, what its encrypted files needed, and
  the table in full is [`../TESTDATA.md`](../TESTDATA.md), "iText's other
  repositories"; the numbers are under "Compared with PDFBox" above.
