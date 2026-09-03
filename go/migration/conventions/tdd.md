# Test-driven porting

**The Java test is ported before the Go implementation exists.** Not alongside
it. Not after it. Before.

This is the rule the whole port runs on, and it is not negotiable per-package.

## Why this is stronger than ordinary TDD

In normal TDD you invent a test from a specification you are also inventing. The
test can agree with your misunderstanding, and nothing catches it.

Porting is different: **the test already exists, and you did not write it.** The
Java suite is evidence from an implementation that has absorbed two decades of
real-world PDF breakage — malformed xref tables, fonts that lie about their
metrics, producers that violate the spec in ways every reader must tolerate.
Those expectations are not derivable from ISO 32000. They were paid for in bug
reports.

So the expected values come from Java, not from you. You cannot write a test
that confirms your own wrong assumption, because you are not the one asserting.

## The cycle

1. **Open the Java test file** for the class about to be ported.
2. **Port the test.** It will not compile — the Go type does not exist yet.
3. **Write the minimum Go** to make it compile and fail honestly.
4. **Port the Java implementation** until the test passes.
5. **Refactor toward Go idiom** per [`java-to-go.md`](java-to-go.md). The test
   stays green throughout.

Step 5 is where the port becomes Go rather than transliterated Java, and it is
safe precisely because step 2 came first.

## The rule that matters most

**Copy assertion values verbatim from the Java.**

If the Java test says `assertEquals(6, randomAccessSource.getPosition())`, the
Go test asserts `6`. It does not assert whatever the Go code happens to return.

Never recompute an expected value from your own implementation. The moment you
do, the test stops being evidence about PDFBox and becomes a restatement of your
code — it will pass, and it will pass on code that is wrong.

**If a Java assertion looks wrong, assume it is not.** An assertion that appears
to contradict the specification usually encodes a real-world quirk. Port it as
written. If it genuinely fails against your Go, that is a finding to investigate
by running the Java — see the oracle procedure in [`../README.md`](../README.md)
— not a licence to change the number.

## The anti-pattern this exists to prevent

> Port the implementation. Then write tests that check what it does.

This produces a green suite over wrong code, and it is the default thing that
happens when tests are treated as a completion step. The tests will look
thorough. They will assert the exact behaviour of a subtly mistranslated
algorithm, and they will keep asserting it for years.

Every rule above is a defence against this one failure.

## Mechanics

- **One Go test file per Java test file.** `RandomAccessReadBufferTest.java`
  becomes `readbuffer_test.go`, with a header comment naming the Java source.
- **Same order as the Java file**, so the two can be read side by side.
- **Keep the Java test names recognisable.** `testPositionSkip` becomes
  `TestReadBufferPositionSkip`.
- **JIRA regression tests are the highest-value tests in the suite.** Keep the
  issue id in the Go test name and a comment saying what broke:
  `TestReadBufferPDFBOX5158` — *"endless loop reading a stream of a multiple of
  4096 bytes"*. These are the tests nobody could have written from the spec.
- **A Java test you do not port gets a comment saying so and why.** Network
  dependence is the usual reason. Silence looks like an oversight.
- **Fixtures:** synthetic data is written to `t.TempDir()`. Real PDFs whose exact
  bytes matter are copied into `testdata/`.
- **Where the port deliberately deviates from Java**, add a test the Java suite
  does not have, pinning the new behaviour. The deviation comment explains why;
  the test proves it holds.

## Java code with no test

Write the test first anyway, from reading the Java. Do not port untested code
untested — that is how a mistranslation reaches slice 9 undetected.

## Acceptance, beyond unit tests

From `slice/3` onward there is a second layer:
`pdfbox/src/test/resources/input/` holds **40 PDFs with checked-in expected text
extractions**, plus reading-order-sorted variants. Wire that up as a Go table
test as soon as any text comes out, and record the score.

Unit tests say the algorithm was translated correctly. The corpus says the
library actually works.

## Definition of done

A package is not ported until its Java tests are ported and green. A package
with implementation and no tests is **not** partially done — it is unverified,
and it should be recorded that way in [`../STATUS.md`](../STATUS.md).

```bash
cd go && gofmt -l . && go vet ./... && go test ./...
```
