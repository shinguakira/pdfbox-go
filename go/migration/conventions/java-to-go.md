# Java to Go porting conventions

These are the rules every ported package follows. They exist so that the port
reads as one library rather than 81 independently translated packages, and so
that a reviewer holding the Java file next to the Go file can tell at a glance
whether a difference is deliberate.

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
| `pdmodel.documentinterchange.<x>` | `pdmodel/interchange/<x>` |
| `pdmodel.interactive.documentnavigation.<x>` | `pdmodel/interactive/navigation/<x>` |
| `COSDictionary`, `PDPage` | `cos.Dictionary`, `pdmodel.Page` — the package supplies the prefix |
| `getFoo()` / `setFoo(v)` | `Foo()` / `SetFoo(v)` |
| `isFoo()` | `IsFoo()`, or a bare `Foo` field when it is plain state |
| `FOO_BAR` constant | `FooBar` |

The full package table is [`../mapping/packages.tsv`](../mapping/packages.tsv).
Add a row there before porting a package, so the inventory script can attribute
its source.

Drop the type-name prefix that the Java class carries when the Go package
already says it: `cos.COSDictionary` stutters, `cos.Dictionary` does not.

Keep the prefix only where dropping it would produce a name that reads as
something else. `COSString` becomes `cos.StringObj`, not `cos.String`, because
`cos.String(x)` reads as a conversion; likewise `COSFloat` becomes
`cos.FloatObj`. Record any such exception in the package doc comment.

## Errors

Java throws; Go returns. Every method that declares `throws IOException` gains
an `error` result.

- Fixed failure conditions become sentinel values in the package's `errors.go`,
  compared with `errors.Is`. Java callers can only match on an exception
  message, so this is strictly more usable — do not port the message strings as
  the only distinguishing feature.
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

**The Java test is ported before the Go implementation exists.** The full rules,
the reasoning, and the anti-pattern they defend against are in
[`tdd.md`](tdd.md) — read it before porting anything.

The short version:

- Port the Java test first. It will not compile. Make it compile, make it fail,
  then port the implementation until it passes, then refactor to Go idiom.
- **Copy assertion values verbatim from the Java.** Never recompute an expected
  value from your own code — that tests your misunderstanding, not PDFBox.
- One Go test file per Java test file, same order, header comment naming the
  Java source. `testPositionSkip` becomes `TestReadBufferPositionSkip`.
- JIRA regression tests keep their issue id and a comment saying what broke.
  Those encode behaviour nobody could derive from the specification.
- A Java test you do not port gets a comment saying so and why.
- Fixtures go to `t.TempDir()`; real PDFs whose bytes matter go to `testdata/`.
- Where the port deviates from Java deliberately, add a test the Java suite
  lacks so the difference is pinned rather than implied.

## Which Java to port against

Port against `trunk` / 3.0 shapes, never 2.0 patterns, and **never port a member
that 3.0 deprecated or removed**. PDFBox's own
[3.0 migration guide](https://pdfbox.apache.org/3.0/migration.html) is the list
of what changed and what went away — check it before porting any class it names.

Concretely, so far:

- Loading moved out of `PDDocument` into a `Loader` class. Port the 3.0 shape:
  a package-level `pdfbox.Open(path)`, not constructors on the document type.
- Standard 14 fonts moved from static instances to a `Standard14Fonts.FontName`
  enum. Port the enum.
- `org.apache.pdfbox.util.Charsets` was deleted. Do not port it.
- The integer 0-255 colour overloads were removed. Port only the float API.

The areas the project itself names as having changed most — reader/writer
infrastructure, font instantiation, colour signatures, the CLI, incremental
parsing — are the areas where the Java API churned most between 2.0 and 3.0.
Follow the source in this repository, not any tutorial written against 2.0.

More background in [`prior-art.md`](prior-art.md).

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
