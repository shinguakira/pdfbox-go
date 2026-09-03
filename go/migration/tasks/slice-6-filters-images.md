# Implementation Plan

Slice 6 — the rest of the filters, and images.

**Branch: `slice/6-<name>`** — from and back to `migration-base`.
Depends on `slice/1` only. Independent of slices 2, 3, 4, 5, 7 and 8.

## Rules — do not break these

- **NEVER change the Java.** No `.java`, no `pom.xml`, no test resource, for any
  reason. It is the reference; a reference that gets edited stops being one.
- **NEVER fix a bug that is in the Java.** Port it as written, comment it where
  it occurs, and record it in `migration/JAVA-BUGS.md`.
- **NEVER create a branch that is not in the migration plan**, and never add one
  to the plan's list.
- **NEVER change `migration/PLAN.md`.**
- **NEVER commit to `migration-base` directly.**
- **NEVER touch `apache/pdfbox`** — no PR, no pull, fetch, merge or rebase.
- **Do not stop while work in this branch's scope remains.**

## How each unit of work runs

Five phases, in this order, never overlapping. See
[`TEMPLATE.md`](TEMPLATE.md) for what each phase means.

A — write the test · B — port the implementation · C — run and fix ·
D — adversarial review · E — user feedback

## Scope

| Java package | Files | Java tests |
| --- | ---: | ---: |
| `pdfbox/filter` — what slice 1 left | ~19 of 23 | 2 |
| `pdmodel/graphics/image` | 9 | 7 |

Slice 1 ported `Filter`, `FilterFactory`, `Predictor`, `FlateFilter`,
`IdentityFilter`. Everything else in that package is here.

Go covers part of it: `compress/lzw`, `image/jpeg`. CCITT, JBIG2 and the ASCII
filters need porting. `image/jpeg` does not read CMYK JPEGs — check that before
relying on it.

---

# Phase A — Write the tests

- [ ] A1. `pdfbox/filter` — port `TestFilters` in full
  - Slice 1 ported the round-trip generator only. `testPDFBOX1977` needs LZW
    and `testRLE` needs RunLength; both are in scope here.
- [ ] A2. `pdfbox/filter` — port the second test in that package
- [ ] A3. `pdmodel/graphics/image` — port all 7 Java tests
- [ ] A4. Write from source for any filter the Java tests do not reach
  - Name which ones before writing

---

# Phase B — Port the implementation

- [ ] B1. The ASCII filters — `ASCIIHexFilter`, `ASCII85Filter`
- [ ] B2. `RunLengthDecodeFilter`
- [ ] B3. `LZWFilter` — check `compress/lzw` against the PDF variant first;
      PDF uses early change, which the stdlib may not
- [ ] B4. `DCTFilter` — JPEG. Check CMYK and YCCK handling
- [ ] B5. `CCITTFaxFilter` and its decoder — Group 3 and Group 4
- [ ] B6. `JBIG2Filter` — decide whether it is ported or left declared and
      unsupported, as Java does when the optional jar is missing
- [ ] B7. `DecodeOptions` — image subsampling, which slice 1 left out
- [ ] B8. `pdmodel/graphics/image` — `PDImageXObject`, `PDInlineImage`,
      `PDImage`, `SampledImageReader` and the rest of the 9

---

# Phase C — Run and fix

- [ ] C1. `gofmt -l .` clean
- [ ] C2. `go vet ./...` clean
- [ ] C3. `go test ./...` green
- [ ] C4. Record every Java bug found in `migration/JAVA-BUGS.md`
- [ ] C5. Update `migration/STATUS.md`

---

# Phase D — Adversarial review

敵対的レビュー. See [`TEMPLATE.md`](TEMPLATE.md) for D1–D6. Additionally:

- [ ] D7. A filter that produces *nearly* the right bytes is a failed port
  - Compare byte for byte against the Java output, not visually.

- [ ] D8. Check the damage tolerance
  - Every PDFBox filter is written to return what it decoded when the input is
    corrupt, rather than failing. Slice 1 already found one place the Go got
    this wrong. Check each new filter for the same.

- [ ] D9. Check the image types the Go stdlib does not cover
  - Where `image/jpeg` refuses a file Java reads, that is a port gap, not a
    Java bug. Record it.

---

# Phase E — User feedback

See [`TEMPLATE.md`](TEMPLATE.md) for E1–E4.

---

# Blocked

- [ ] `pdmodel/graphics/image` needs colour spaces. Slice 2 ported
      `PDColorSpace` as an interface with only `PDDeviceGray` behind it; the
      other 20 are absent. Decide whether they come here or in slice 9.
