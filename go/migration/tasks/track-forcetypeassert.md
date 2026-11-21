# Implementation Plan

Track — the type assertions the port does not check. Adding the one linter that
finds them, and handling what it finds.

**Branch: `track/forcetypeassert`** — from `update/sync-original-update`, and
back to `migration-base` behind it. It starts there rather than from
`migration-base` because that branch has already converted five of these sites,
in `pdmodel/common/function/type4`, and starting fresh would mean doing them
twice and resolving the same five lines in a merge.

## Why

A Java cast that fails throws `ClassCastException`. In Go the equivalent is
`x.(T)` in a single-value context, which panics with
`*runtime.TypeAssertionError`. The two are the same failure until something
catches it:

- `PDFunctionType4.eval` catches `ClassCastException` since PDFBOX-6320 and
  hands its callers an `IOException`. The port recovers a panic that wraps
  `type4.ErrWrongType`. A `*runtime.TypeAssertionError` wraps nothing, so the
  recover re-panicked and a malformed function took the page rather than the
  shading. That was five casts in `bitshift`, `if` and `ifelse`, found in the
  review of pull request #44 and fixed on `update/sync-original-update`.
- Four other `recover` boundaries in the library stand for a Java `catch` in the
  same way: `contentstream/operator/state` (`Concatenate`, PDFBOX-6255),
  `pdmodel/font/pdfont.go`, `pdmodel/fixup/processor` and
  `fontbox/type1/type1parser.go`.

So the class matters, and nothing in the repository's gate was looking for it:
`gofmt`, `go vet` and `go test` do not check it, and
`TestEveryTypeAssertionIsChecked` on `update/sync-original-update` checks one
package by reading its own source.

## What the linter found at the start of this branch

`forcetypeassert` v0.2.0 over `./...`: **288 findings, 106 of them in `_test.go`
files and 182 in the library.** The configuration leaves the tests alone --- a
bare assertion in a test is a test failure at the line that made it, with no
caller to mislead --- so the 182 are the work.

| Package | | Package | |
| --- | ---: | --- | ---: |
| `pdfbox/multipdf` | 45 | `pdfbox/pdmodel/graphics/image` | 3 |
| `pdfbox/pdmodel/font` | 20 | `pdfbox/pdfwriter` | 3 |
| `pdfbox/pdmodel/graphics/color` | 12 | `pdfbox/filter` | 3 |
| `pdfbox/pdmodel/common` | 11 | `pdfbox/cos` | 3 |
| `pdfbox/contentstream/operator/state` | 11 | `pdfio` | 2 |
| `pdfbox/pdmodel/interactive/form` | 10 | `pdfbox/rendering` | 2 |
| `pdfbox/pdmodel` | 8 | `pdfbox/pdmodel/interactive/annotation` | 2 |
| `pdfbox/pdmodel/encryption` | 7 | `pdfbox/pdmodel/documentinterchange/taggedpdf` | 2 |
| `fontbox/cff` | 7 | `pdfbox/pdmodel/common/function` | 2 |
| `pdfbox/util` | 6 | `cmd/corpus` | 2 |
| `pdfbox/rendering/raster` | 5 | eleven packages with one each | 11 |
| `pdfbox/pdmodel/fdf` | 5 | | |

The linter reports two messages, and both are this class: *type assertion must
be checked* for `v := x.(T)`, and *right hand must be only type assertion* for
an assertion with a call on the end of it, `x.(*Object).Object()`, which has to
be split before it can be checked.

## What a fix is

**Not** a behaviour change. A failed cast stays a failure; what changes is that
it says so in a way the recover boundaries can read, and that the message names
what was found instead:

```go
object, isObject := base.(*Object)
if !isObject {
    panic(fmt.Sprintf("cos: %T is not an object", base))
}
```

Three rules, in order:

1. **Inside a `recover` boundary that stands for a Java `catch`** --- the five
   packages named above --- the panic wraps the boundary's own sentinel, as
   `type4.ErrWrongType` does, so the recover can tell it from a defect of the
   port.
2. **Everywhere else** the panic is the port's existing convention for an
   unchecked Java throw: it panics, with a message naming the type it did not
   get. Java's `ClassCastException` is unchecked and uncaught on these paths,
   and a Go error return here would be a deviation with no caller to justify
   it.
3. **No `//nolint`.** An exception that cannot be read from the code is an
   exception the next reader has to take on trust.

## Rules — do not break these

- **NEVER change the Java**, and nothing on this branch needs to: the defect is
  Go-only. Java cannot have it, because `catch (ClassCastException)` takes every
  failed cast.
- **Behaviour stays.** A document that panicked before this branch panics after
  it, unless it reaches one of the five boundaries, where it becomes the error
  the Java's `catch` already produced. Any other change is a bug in this branch.
- **One package, one commit**, after the lint commit, so that any one can be
  reverted alone.
- **The suite runs per commit**: `go build ./...`, `gofmt -l .`, `go vet ./...`,
  `go test -count=1 ./...`, and the linter over the package the commit touched.

## Phases

- **A** — add the linter and the gate: `go/.golangci.yml`, the line in
  `AGENTS.md`, this file, the row in [`README.md`](README.md). The count above
  is the baseline.
- **B** — the packages, biggest first, one commit each.
- **C** — the linter clean over `./...`, the suite clean, and
  `TestEveryTypeAssertionIsChecked` either generalised to every boundary package
  or dropped in favour of the linter.
- **D** — adversarial review: for each boundary package, a test that the failure
  arrives as the boundary's error rather than as a panic.
- **E** — user feedback.

## What this branch does not do

It does not put the linter in CI: the repository has one workflow,
`codeql-analysis.yml`, and no Go build or test job, so the Go gate is run by
hand as `AGENTS.md` describes. Adding CI is a separate decision.
