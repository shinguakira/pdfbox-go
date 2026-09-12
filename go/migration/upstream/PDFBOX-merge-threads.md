# PDFMergerUtility clones the destination's `/Threads`

Draft of an issue and a patch for Apache PDFBox. **Not filed.** Written
2026-09-11/12 on the repository owner's instruction; see [`README.md`](README.md)
for why this directory exists at all.

This is [`JAVA-BUGS.md`](../JAVA-BUGS.md) entry **82**. That entry is the port's
record and stays as it is; this file is the outward-facing version.

| | |
| --- | --- |
| Where | `pdfbox/src/main/java/org/apache/pdfbox/multipdf/PDFMergerUtility.java`, `appendDocument`, line 546 |
| Present in | trunk, `4.0.0-SNAPSHOT`, as of the snapshot of 2026-09-07 (`3d024173c`) |
| Age | at least 2015-02-25 (`245065766`, PDFBOX-2680), which is a package move, so older |
| Fix | one identifier |
| Existing issue | **none** — see section 3 |
| Existing PR | **none** — see section 3 |
| Status | drafted, verified, **not submitted** |

---

## 1. The defect

```java
COSArray destThreads = destCatalog.getCOSObject().getCOSArray(COSName.THREADS);
COSArray srcThreads = cloner.cloneForNewDocument(destCatalog.getCOSObject().getCOSArray(
        COSName.THREADS));
if (destThreads == null)
{
    destCatalog.getCOSObject().setItem(COSName.THREADS, srcThreads);
}
else
{
    destThreads.addAll(srcThreads);
}
```

Both `getCOSArray` calls read `destCatalog`. The variable called `srcThreads` is
a clone of the **destination's** article thread array.

Every other block of `appendDocument` reads `srcCatalog` and writes
`destCatalog`; this is the only one that reads `destCatalog` twice. The
page-labels block twenty lines below reads `srcLabels` correctly, in the same
method, in the same style.

### Two consequences

**The source's threads are dropped.** Every merge loses them. The reading order
they describe is what `/Threads` is for.

**The destination's are appended to themselves, and the clone is not a
dictionary.** `cloneForNewDocument` deep-copies what it is given, and a thread
reaches a long way:

```
thread -> /F bead -> /P the page -> /Parent the page tree node
                                 -> /Resources -> the fonts
                                 -> /Contents  -> the content stream
```

All of it is copied and none of the copy is reachable, because the catalog's
page tree still names the originals. In the merged file, whose page tree is
`[22 0 R 23 0 R]`:

```
34 0 obj  /Type /Page   /Font 38 0 R  /Contents 39 0 R    <- not in the page tree
37 0 obj  /Type /Pages  /Kids [34 0 R]  /Count 1          <- an orphan page tree
39 0 obj  /Length 207                                     <- the content stream, again
42 0 obj  /Type /Font  /Subtype /Type1                    <- the font, again
```

Each merge leaves another unreachable copy, and the count doubles every time.

### What limits it

`destThreads == null` takes the other arm, where the clone of the destination's
absent `/Threads` is null, so nothing is written and nothing is copied. **Only a
destination that already carries article threads is affected.** That excludes
`pdfbox merge`, which starts from `new PDDocument()`; it includes callers of
`appendDocument` on a document they loaded.

---

## 2. Evidence

Measured by running the Java, JDK 17, against two files this repository carries.
Commands in section 10.

Merging a one-page source (2 threads) into a one-page destination (1 thread),
three times over.

**trunk as it is**

| merges | real pages | bytes | objects | `/Type /Page` | `/Type /Pages` |
| ---: | ---: | ---: | ---: | ---: | ---: |
| 1 | 2 | 3,486 | 28 | 3 | 2 |
| 2 | 3 | 7,354 | 54 | 7 | 4 |
| 3 | 4 | 15,295 | 106 | 15 | 8 |

**with the patch**

| merges | real pages | bytes | objects | `/Type /Page` | `/Type /Pages` |
| ---: | ---: | ---: | ---: | ---: | ---: |
| 1 | 2 | 3,092 | 26 | 2 | 1 |
| 2 | 3 | 4,829 | 36 | 3 | 1 |
| 3 | 4 | 6,822 | 46 | 4 | 1 |

Page objects equal real pages, one page tree, linear growth.

Thread titles, same runs:

```
trunk    after 1: DEST | DEST
         after 2: DEST | DEST | DEST | DEST
         after 3: DEST | DEST | DEST | DEST | DEST | DEST | DEST | DEST
patched  after 1: DEST | SRC-A | SRC-B
         after 2: DEST | SRC-A | SRC-B | SRC-A | SRC-B
         after 3: DEST | SRC-A | SRC-B | SRC-A | SRC-B | SRC-A | SRC-B
```

**The risk that was checked and ruled out.** Cloning the *source's* threads
could have created the mirror-image problem: orphan copies of the source's
pages. It does not. `PDFCloneUtility` caches by original object, so the page
clone the bead reaches is the same clone the page import uses. The patched table
above is the measurement — page objects equal real pages.

---

## 3. Duplicate check, 2026-09-11: none found

**JIRA**, through `issues.apache.org/jira/rest/api/2/search`:

| Query | Result |
| --- | --- |
| `project=PDFBOX AND summary~"threads"` | 34, all about Java threads (concurrency). The only one about the `/Threads` array is PDFBOX-5518, "Threads array should be indirect reference", fixed 2023-07-23 — a COSWriter issue, unrelated |
| `project=PDFBOX AND text~"article" AND text~"merge"` | 13, none about `/Threads` |
| `project=PDFBOX AND (text~"destThreads" OR text~"srcThreads" OR text~"appendDocument threads")` | 13, all closed, all unrelated |
| `project=PDFBOX AND text~"article threads"` | 0 |

**GitHub**, through `api.github.com/search/issues`:

| Query | Result |
| --- | --- |
| `repo:apache/pdfbox is:pr PDFMergerUtility` | 5, none about `/Threads` |
| `repo:apache/pdfbox is:pr threads` | 2, unrelated |
| `repo:apache/pdfbox srcThreads OR destThreads OR "article thread"` | `total_count 0` |
| all open PRs | 7: #529, #528, #526, #521, #520, #495, #446. None related |

Re-check before filing. This was a point in time.

---

## 4. Apache PDFBox conventions, from the tree and the tracker

- **JIRA first.** Create the issue, then the PR. The PR body links it.
- **PR title** `PDFBOX-NNNN: <description>` — the form used by #526, #495, #446.
- **Commit message** `PDFBOX-NNNN: <lowercase description>[, by <Name>][; closes #NNN]`.
  A committer applies it on svn and closes the GitHub PR with that trailer.
- **Tests** JUnit 5, in `PDFMergerUtilityTest`, class annotated
  `@Execution(ExecutionMode.CONCURRENT)`, output under
  `TARGETTESTDIR = "target/test-output/merge/"`.
- **Test resources** either checked in under `src/test/resources/input/merge/`,
  or downloaded by a `wget` execution in `pdfbox/pom.xml` from a JIRA
  attachment. **Neither is needed here** — the test builds its documents in
  memory.
- **Licence** ASF header on every file. A PR to `apache/*` is a contribution
  under Apache 2.0; no CLA is needed for a patch this size.
- The queue is short and moving: 7 open PRs, 5 of them opened in the five weeks
  to 2026-09-08.

---

## 5. The issue, ready to paste

**Project** PDFBox · **Type** Bug · **Component** PDModel ·
**Affects Version** 4.0.0 (check the 3.0 branch before filing)

**Summary**

```
PDFMergerUtility.appendDocument() clones the destination's /Threads instead of the source's
```

**Description**

```
appendDocument() reads the destination catalog twice where it means to read the
source once:

    COSArray destThreads = destCatalog.getCOSObject().getCOSArray(COSName.THREADS);
    COSArray srcThreads = cloner.cloneForNewDocument(destCatalog.getCOSObject().getCOSArray(
            COSName.THREADS));

The variable called srcThreads is a clone of the *destination's* article thread
array. Two things follow.

1. The source document's article threads are silently dropped by every merge.
   The reading order they describe is lost, which is what /Threads is for.

2. The destination's threads are appended to themselves. And because a thread
   reaches its page through its bead's /P entry, cloneForNewDocument() also
   deep-copies the destination's pages, its page tree node, its resources and
   its content streams. None of that copy is reachable, because the catalog's
   page tree still names the originals. Each merge leaves another unreachable
   copy, and the count doubles every time.

Merging a one-page source into a one-page destination, both with article
threads, three times over:

    merges  pages  bytes   objects  /Type /Page  /Type /Pages
         1      2   3,486       28            3             2
         2      3   7,354       54            7             4
         3      4  15,295      106           15             8

Four pages of content in a file holding fifteen page objects and eight page
trees. On a destination whose pages carry embedded fonts or images, this is the
file size doubling per merge.

With the one-word fix the same three merges give:

    merges  pages  bytes   objects  /Type /Page  /Type /Pages
         1      2   3,092       26            2             1
         2      3   4,829       36            3             1
         3      4   6,822       46            4             1

Page objects equal real pages, one page tree, linear growth.

Every other block of appendDocument() reads srcCatalog and writes destCatalog;
this is the only one that reads destCatalog twice. The page-labels block twenty
lines below reads srcLabels correctly.

What limits the damage: destThreads == null takes the other arm, where the
clone of the destination's absent /Threads is null, so nothing is written and
nothing is copied. Only a destination that already carries article threads is
affected. That excludes `pdfbox merge`, which starts from a new PDDocument; it
includes callers of appendDocument() on a document they loaded.

The line has read destCatalog since at least 2015-02-25, where it arrives with
a package move, so it is older than that.

A self-contained test is attached; it needs no test resource.
```

---

## 6. The patch

```diff
--- a/pdfbox/src/main/java/org/apache/pdfbox/multipdf/PDFMergerUtility.java
+++ b/pdfbox/src/main/java/org/apache/pdfbox/multipdf/PDFMergerUtility.java
@@ -543,7 +543,7 @@ public class PDFMergerUtility
         mergeAcroForm(cloner, destCatalog, srcCatalog);
 
         COSArray destThreads = destCatalog.getCOSObject().getCOSArray(COSName.THREADS);
-        COSArray srcThreads = cloner.cloneForNewDocument(destCatalog.getCOSObject().getCOSArray(
+        COSArray srcThreads = cloner.cloneForNewDocument(srcCatalog.getCOSObject().getCOSArray(
                 COSName.THREADS));
         if (destThreads == null)
         {
```

---

## 7. The test

Added to `PDFMergerUtilityTest`. Needs `java.util.Arrays`,
`PDDocumentInformation`, `PDThread` and `PDThreadBead` on the imports; the rest
the class already has.

```java
    /**
     * PDFBOX-NNNN: appendDocument() cloned the destination's /Threads instead of the source's, so
     * the source's article threads were dropped and the destination's were appended to themselves.
     * Because a thread reaches its page through its bead, the clone also left a copy of the
     * destination's pages in the file that no page tree names.
     *
     * @throws IOException
     */
    @Test
    void testMergeThreads() throws IOException
    {
        String filename = TARGETTESTDIR + "PDFBOX-NNNN-threads.pdf";
        try (PDDocument destination = createDocumentWithThreads("destination article");
             PDDocument source = createDocumentWithThreads("source article 1", "source article 2"))
        {
            new PDFMergerUtility().appendDocument(destination, source);
            destination.save(filename);
        }
        try (PDDocument merged = Loader.loadPDF(new File(filename)))
        {
            assertEquals(2, merged.getNumberOfPages());
            assertEquals(
                    Arrays.asList("destination article", "source article 1", "source article 2"),
                    getThreadTitles(merged));
            // cloning the wrong array also cloned the page each bead points at
            assertEquals(merged.getNumberOfPages(),
                    merged.getDocument().getObjectsByType(COSName.PAGE).size());
        }
    }

    private static PDDocument createDocumentWithThreads(String... titles) throws IOException
    {
        PDDocument document = new PDDocument();
        PDPage page = new PDPage();
        document.addPage(page);
        COSArray threads = new COSArray();
        for (String title : titles)
        {
            PDDocumentInformation info = new PDDocumentInformation();
            info.setTitle(title);
            PDThreadBead bead = new PDThreadBead();
            bead.setPage(page);
            bead.setRectangle(new PDRectangle(56, 600, 483, 100));
            PDThread thread = new PDThread();
            thread.setThreadInfo(info);
            thread.setFirstBead(bead);
            threads.add(thread);
        }
        document.getDocumentCatalog().getCOSObject().setItem(COSName.THREADS, threads);
        return document;
    }

    private static List<String> getThreadTitles(PDDocument document)
    {
        List<String> titles = new ArrayList<>();
        COSArray threads =
                document.getDocumentCatalog().getCOSObject().getCOSArray(COSName.THREADS);
        for (int i = 0; i < threads.size(); i++)
        {
            titles.add(new PDThread((COSDictionary) threads.getObject(i)).getThreadInfo()
                    .getTitle());
        }
        return titles;
    }
```

**Run against both.** On trunk as it is:

```
expected: <[destination article, source article 1, source article 2]>
 but was: <[destination article, destination article]>
```

With the patch: `1 tests successful`.

---

## 8. The pull request

**Title**

```
PDFBOX-NNNN: use the source catalog's /Threads when merging
```

**Body**

```
https://issues.apache.org/jira/browse/PDFBOX-NNNN

appendDocument() read destCatalog for both destThreads and the array it cloned
into srcThreads, so the source's article threads were never merged and the
destination's were appended to themselves. Since a thread reaches its page
through its bead, the clone also left a copy of the destination's pages in the
file that no page tree names, doubling on every merge.

One identifier changed. Every other block of the method already reads srcCatalog
and writes destCatalog.

The test builds both documents in memory, so it adds no test resource and no
pom.xml entry. It fails on trunk with

    expected: <[destination article, source article 1, source article 2]>
     but was: <[destination article, destination article]>

and passes with the change. Its second assertion pins the orphaned pages.
```

---

## 9. Procedure

1. Re-run the duplicate check in section 3. It was a point in time.
2. Create a JIRA account if there is none, and file section 5. Note the number.
3. Fork `apache/pdfbox`, branch from trunk.
4. Apply section 6 and section 7, replacing `NNNN` with the issue number.
5. `mvn -pl pdfbox test -Dtest=PDFMergerUtilityTest`
6. Commit as `PDFBOX-NNNN: use the source catalog's /Threads when merging`.
7. Open the PR with section 8.

Check whether the 3.0 branch carries the same line before naming affected
versions.

---

## 10. Reproducing it from this repository

The two documents and the driver are checked in under
`go/pdfbox/multipdf/testdata/`:

| | |
| --- | --- |
| `javabug82-dest.pdf` | one page, one article thread |
| `javabug82-src.pdf` | one page, two article threads |
| `javabug82-merged-java.pdf` | what trunk produces from them |
| `javabug82-merged-go.pdf` | what the port produces, for comparison |
| `genjavabug82.go` | writes the first two |
| `genjavabug82merged.go` | writes the fourth |
| `Merge82Drv.java` | runs the merge through the Java and prints the titles |

All four PDFs are saved uncompressed, so `/Threads` can be read out of them with
a text editor.

Building the Java needs a stand-in for `PublicKeySecurityHandler`, which imports
Bouncy Castle. Put one in a directory **outside this repository** and place it
first on the sourcepath; the repository's Java is never edited. The stand-in
only has to extend `SecurityHandler<PublicKeyProtectionPolicy>`, declare
`FILTER`, the two constructors, and throw from `prepareForDecryption` and
`prepareDocumentForEncryption`. Nothing in this case reaches it — it is only
compiled because `SecurityHandlerFactory` registers it in a static initializer.

```sh
# from the repository root; $STUB holds the stand-in, $OUT takes the classes
javac -encoding UTF-8 -proc:none -cp "$LOG4J_API_JAR" \
    -sourcepath "$STUB;pdfbox/src/main/java;fontbox/src/main/java;xmpbox/src/main/java;io/src/main/java" \
    -d "$OUT" go/pdfbox/multipdf/testdata/Merge82Drv.java

java -cp "$OUT;$LOG4J_API_JAR" Merge82Drv go/pdfbox/multipdf/testdata
```

To measure the patched behaviour, copy `PDFMergerUtility.java` into `$STUB`
under its package path, change the one identifier there, and compile again.
`$STUB` comes first on the sourcepath, so javac takes that copy over the
repository's.

Running the JUnit test needs `junit-jupiter-api`, `junit-jupiter-engine`,
`junit-platform-launcher`, `junit-platform-engine`, `junit-platform-commons`,
`opentest4j` and `apiguardian-api` on the classpath, and a six-line launcher
calling `LauncherFactory.create().execute(...)` with
`DiscoverySelectors.selectClass`.

---

## 11. Open items

- **Not filed.** Sections 5 and 8 are drafts.
- `NNNN` is a placeholder in five places: the test javadoc, the test's output
  filename, the commit message, the PR title and the PR body.
- The 3.0 branch was not checked. This repository mirrors trunk only.
- Whether to report anything else from `JAVA-BUGS.md` is a separate decision.
  That file's "How they group" section is where to start: group B, the seven
  that need a rasterizer, cannot fail upstream's build, because PDFBox has no
  rendering test.
