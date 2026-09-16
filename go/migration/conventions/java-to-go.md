# Java to Go porting conventions

These are the rules every ported package follows. They exist so that the port
reads as one library rather than as many unrelated translations as there are
packages, and so that a reviewer holding the Java file next to the Go file can
tell at a glance whether a difference is deliberate.

The governing principle: **idiomatic Go at the boundary, faithful algorithm
inside.** Public API shapes get translated into Go idiom, because callers have
to live with them and because it lets the port compose with the standard
library. Parsing logic, arithmetic, table lookups and format quirks get ported
line for line, because that is where PDFBox has absorbed two decades of
real-world PDF breakage that no rewrite would rediscover.

## Naming

| Java | Go |
| --- | --- |
| `org.apache.pdfbox.io` | `pdfio` — renamed, `io` would shadow the stdlib |
| `org.apache.pdfbox.<x>` | `pdfbox/<x>` |
| `org.apache.fontbox.<x>` | `fontbox/<x>` |
| `org.apache.xmpbox.type` | `xmpbox/xmptype` — renamed, `type` is a keyword |
| `COSDictionary` | `cos.Dictionary` — the package supplies the prefix |
| `getFoo()` / `setFoo(v)` | `Foo()` / `SetFoo(v)` |
| `isFoo()` | `IsFoo()`, or a bare `Foo` field when it is plain state |
| `FOO_BAR` constant | `FooBar` |

The full package table is [`../mapping/packages.tsv`](../mapping/packages.tsv).
Add a row there before porting a package, so the inventory script can attribute
its source.

Drop the type-name prefix that the Java class carries when the Go package
already says it: `cos.COSDictionary` stutters, `cos.Dictionary` does not. That
is what `cos` does; `pdmodel` keeps Java's `PD`, so the type is
`pdmodel.PDPage`.

Keep the prefix only where dropping it would produce a name that reads as
something else. `COSString` becomes `cos.StringObj`, not `cos.String`, because
`cos.String(x)` reads as a conversion. Record any such exception in the package
doc comment, as `go/pdfbox/cos/doc.go` does for that one.

## Errors

Java throws; Go returns. Every method that declares `throws IOException` gains
an `error` result.

- Fixed failure conditions become exported sentinel values compared with
  `errors.Is`, declared beside the code that returns them, or collected into the
  package's `errors.go` once there are several. Java callers can only match on an
  exception message, so this is strictly more usable — do not port the message
  strings as the only distinguishing feature.
- Wrap with `%w` when adding context: `fmt.Errorf("parsing xref at %d: %w", off, err)`.
- `IllegalArgumentException` from a constructor becomes an `error` from the
  `NewXxx` function, not a panic. Panic only where the Java code would have
  thrown from a static initialiser over a compiled-in constant — that is a bug
  in the library, not in the PDF.
- End of input is `io.EOF`, never a `-1` sentinel return. Java's `read()`
  returning `-1` becomes `(byte, error)` with `io.EOF`.

## Class translation

**Concrete class** becomes a struct with a `NewXxx` constructor. Keep fields
unexported and expose accessors only where Java has them; a Java field that is
public is a Go exported field.

**Interface** becomes a Go interface. Java's `default` methods have no Go
equivalent: promote them to package-level helper functions taking the interface
as first parameter. `RandomAccessRead.peek()` became `pdfio.Peek(r)` for exactly
this reason.

**Abstract class** splits in two: an interface for the abstract methods, and a
struct holding the shared state and concrete methods, which implementations
embed. Do not try to reproduce the single-rooted hierarchy.

**`extends` on a concrete class** becomes struct embedding. `RandomAccessReadWriteBuffer
extends RandomAccessReadBuffer` became `ReadWriteBuffer{ ReadBuffer }`. Embedding
promotes methods but gives no virtual dispatch: if the Java subclass overrides a
method that the superclass calls internally, embedding will silently call the
superclass version. Where that pattern appears, pass the behaviour in as an
interface field instead — and note it in the type's doc comment.

**Static utility class** becomes package-level functions. `IOUtils.closeQuietly`
became `pdfio.CloseQuietly`.

**Enum** becomes a defined integer or string type with a `const` block and a
`String()` method. Enums carrying behaviour become a struct with package-level
instances.

**Inner and anonymous classes** become named unexported types, or closures when
they capture one thing and are used once.

## Concurrency

Java's `synchronized`, `ConcurrentHashMap` and thread-local caching do not
translate mechanically.

- `synchronized` on a method becomes a `sync.Mutex` field guarding the state it
  actually protects, not the whole struct by reflex.
- **Thread-local caches keyed by `Thread.currentThread().getId()` have no port.**
  Go has no stable goroutine identity, and reaching for one is a mistake. The
  pattern exists in Java to give each thread an independent cursor over a shared
  source; in Go, hand each caller its own cursor value and share the immutable
  data behind it. `ReadBuffer.CreateView` and `BufferedFile.CreateView` both do
  this, and the result is safe for concurrent use where the Java original is not.
- Document the concurrency contract of every exported type. If it is not safe
  for concurrent use, say so.

## Standard library

Prefer the Go stdlib over porting a Java utility that only exists because the
JDK lacked something:

| PDFBox / JDK | Go |
| --- | --- |
| `IOUtils.toByteArray(in)` | `io.ReadAll` |
| `IOUtils.copy(in, out)` | `io.Copy` |
| `IOUtils.populateBuffer(in, b)` | `io.ReadFull` |
| `InputStream` / `OutputStream` | `io.Reader` / `io.Writer` |
| `ByteArrayOutputStream` | `bytes.Buffer` |
| `Inflater` / `Deflater` | `compress/zlib`, `compress/flate` |
| `javax.crypto` (AES, RC4) | `crypto/aes`, `crypto/cipher`, `crypto/rc4` |
| `MessageDigest` | `crypto/md5`, `crypto/sha256` |
| `java.awt.geom.AffineTransform` | own type in `pdfbox/util` — no stdlib equivalent |
| `java.awt.image.BufferedImage` | `image.Image` / `image.RGBA` |
| Log4j `LOG.debug(...)` | `log/slog` at the matching level |

`java.awt` is the deep one. Rendering, printing and the debugger lean on AWT and
Java2D throughout, and Go has no equivalent; see PLAN.md slice 9 for how that is
scoped.

### Methods whose contract is not what the Go one looks like

The table above is for methods with a Go equivalent. These are the ones where a
Go function has the same *name* or the same *shape* and different semantics.
Each of them has already cost this port at least one defect. **Port the contract
once, in a helper, rather than at the call site** — every one of these was found
more than once because it was patched where it bit instead of written down.

**`String.split(regex)` with the default limit.** Three rules, and no function
in `strings` has all three:

| Input | `String.split(",")` | `strings.Split` | `strings.FieldsFunc` |
| --- | --- | --- | --- |
| `",1"` | `["", "1"]` | `["", "1"]` | `["1"]` ✗ |
| `"1,,2"` | `["1", "", "2"]` | `["1", "", "2"]` | `["1", "2"]` ✗ |
| `"1,2,"` | `["1", "2"]` | `["1", "2", ""]` ✗ | `["1", "2"]` |
| `",,"` | `[]` | `["", "", ""]` ✗ | `[]` |
| `""` | `[""]` | `[""]` | `[]` ✗ |

So: leading and interior empties are **kept**, every trailing empty is
**dropped**, and where the separator never occurs the whole input comes back
untrimmed. `pdmodel/fdf`'s `splitJavaFunc` writes all three out; `util.SplitOnSpace`
is the same contract for `\s`. `track/test-backfill` hit this in several packages
at once, and one of them made the port silently accept a coordinate list Java
rejects; [`../STATUS.md`](../STATUS.md) has each with the test that pins it.

**`HashMap.put` keeps the key object it already has**, and updates only the
value. Two keys that are `equals` but carry different extra state — as
`COSObjectKey` does with its stream index — do not replace one another. Removing
first is how Java changes one; `cos.Document` has `RemoveXRefOffset` for it.

**`(long) someFloat` saturates.** Out of range it gives `Long.MAX_VALUE` or
`Long.MIN_VALUE`; the Go conversion is implementation-defined and gives
-9223372036854775808 either way on amd64. See `util.int64OfFloat`. `(int)` from
a wider integer narrows rather than saturating, which is the opposite rule —
both are in [Numeric types](#numeric-types).

**Xerces holds a `NamedNodeMap` sorted by qualified name**, not in document
order, because it searches the map with a binary search. Anything that walks
`getAttributes()` sees the sorted order, and PDFBox writes XML out that way.
Both DOMs in this port do the same; `w3c/dom` did not until a test asserted the
bytes.

## Numeric types

Java has no unsigned types, so PDFBox masks constantly: `b & 0xff`, `x & 0xffff`,
`>>>`. In Go the byte is already unsigned.

- Drop `& 0xff` when reading a `byte` into an int — it is a no-op that reads as
  though something is being masked.
- Java `int` is 32-bit: use `int32` where the width is load-bearing (format
  fields, overflow-sensitive arithmetic), and `int` where it is just a count.
- Java `long` becomes `int64`. File offsets and object numbers are `int64`.
- `>>>` becomes `>>` on an unsigned type. Do not port it as a signed shift.
- Java `char` is a UTF-16 code unit, not a rune. A `char[]` walking a string is
  usually `[]uint16` or `[]byte`, not `[]rune` — check which before assuming.

## Tests

**The Java test is ported before the Go implementation exists**, and assertion
values are copied verbatim from the Java rather than recomputed from the Go. The
rules, the reasoning, the anti-pattern they defend against and how a ported test
file is written are in [`tdd.md`](tdd.md) — read it before porting anything.

One of them belongs here too, because it is a translation rule: where the port
deviates from Java deliberately, add a test the Java suite lacks, so the
difference is pinned rather than implied. See "Recording deviations" below.

## Which Java to port against

Port against `trunk` / 3.0 shapes, never 2.0 patterns, and **never port a member
that 3.0 deprecated or removed**. PDFBox's own
[3.0 migration guide](https://pdfbox.apache.org/3.0/migration.html) is the list
of what changed and what went away — check it before porting any class it names.

Concretely, so far:

- Loading moved out of `PDDocument` into a `Loader` class. Port the 3.0 shape:
  package-level functions, not constructors on the document type. The port's are
  `pdfbox.LoadPDF` and its variants, in `go/pdfbox/loader.go`.
- Standard 14 fonts moved from static instances to a `Standard14Fonts.FontName`
  enum. Port the enum.
- `org.apache.pdfbox.util.Charsets` was deleted. Do not port it.
- The integer 0-255 colour overloads were removed. Port only the float API.

Follow the source in this repository, not any tutorial written against 2.0. The
guide, the areas the project itself names as unsettled, and what the whole of it
implies for the port are in [`prior-art.md`](prior-art.md).

## Never change the Java

**The Java tree is strictly read-only.** No `.java` file, `pom.xml` or test
resource is ever edited — not to ease a port, not to fix a bug found while
reading it, not to silence a warning. The Java is the reference the Go is
checked against, and a reference that gets edited stops being one.

If the port appears to need a Java change, the port is wrong.

## Do not fix Java bugs

**Port the behaviour as written, including behaviour that is plainly wrong.**
This is a migration, not a bug hunt. A bug faithfully carried over stays
findable by diffing against the Java; a bug silently corrected during the port
does not. Real PDFs and real callers depend on quirks — an arithmetic slip that
truncates a value, a comparison that ignores a field — and something downstream
may already match them.

When something looks like a Java bug:

1. Port it exactly.
2. Comment at that point that it looks wrong, and say what the correct
   behaviour would be.
3. Add an entry to [`../JAVA-BUGS.md`](../JAVA-BUGS.md) — not fixing a bug is
   not the same as forgetting it, and the moment you are reading the Java
   closely enough to notice is the only moment it is cheap to write down. That
   file states what an entry has to say.
4. Move on. Do not open the question in the code.

The only thing to fix is a bug **introduced by the port itself** — something the
Java cannot do. A Go initialisation-order mistake, a nil dereference where Java
had a primitive, an infinite loop from a zero value Java constructors cannot
produce: those are port defects, not ported behaviour, and they get fixed.

The same applies to ported tests. A Java test helper that does not check what it
claims to check gets ported as it is; the ported tests then verify exactly what
the Java tests verify, no more. Strengthening it silently would mean the Go
suite and the Java suite no longer test the same thing.

**One branch was released from this rule, and it is closed.** After the port was
finished, `track/java-bug-fixes` went through `JAVA-BUGS.md` and corrected in the
Go the entries worth correcting; each of those carries a **Fixed in the Go**
paragraph there and a comment at the site. A divergence carrying such a comment
is not a defect to revert. Nothing above is relaxed for any other branch: a newly
found Java bug is still ported, commented and recorded.

## Recording deviations

Where the port deliberately differs from Java, say so in a comment at the point
of difference, naming the Java behaviour and the reason. Examples in `pdfio`:

- `ReadBuffer.Read` stops instead of adding a `-1` to the running count, which
  the Java loop does.
- `fileSource.page` does not reuse the evicted page buffer the way the Java LRU
  does, so a cursor holding an evicted page keeps reading valid bytes.
- `BufferedFile.IsEOF` compares offset to length instead of `peek() == -1`.

These comments are the migration's audit trail. Anyone diffing a Go file
against the Java it came from needs to know which differences were decisions.
