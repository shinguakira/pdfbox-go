# Implementation Plan

Track — `scratchfile`. The spill-to-disk path in `pdfio`.

**Branch: `track/scratchfile`** — from and back to `migration-base`.
Depends on `slice/0`, which has landed. Start it whenever memory pressure
matters.

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
D — adversarial review · E — user feedback, strict test first

## Scope

The five files `migration/STATUS.md` records as deferred out of `slice/0`:

| Java source | Status in STATUS.md |
| --- | --- |
| `ScratchFile.java` | not started |
| `ScratchFileBuffer.java` | not started |
| `MemoryUsageSetting.java` | not started — only meaningful once `ScratchFile` exists |
| `RandomAccessReadMemoryMappedFile.java` | not started — needs a decision on `golang.org/x/exp/mmap` vs `syscall` |
| `NonSeekableRandomAccessReadInputStream.java` | not started |

`NonSeekableRandomAccessReadInputStream` is also what
`PDPage.getContentsForStreamParsing` needs for its flate fast path — slice 2
records that as the reason it takes the general path instead.

---

# Phase A — Write the tests

- [ ] A1. Port `ScratchFileBufferTest`, which `slice/0` could not
- [ ] A2. Port `NonSeekableRandomAccessReadInputStreamTest`
- [ ] A3. Port `RandomAccessReadMemoryMappedFileTest`
- [ ] A4. Write from source for `ScratchFile` and `MemoryUsageSetting` if Java
      has no test for them — check before assuming

---

# Phase B — Port the implementation

- [ ] B0. **Take the memory-mapping decision first.** `STATUS.md` names it:
      `golang.org/x/exp/mmap` or `syscall`. Adding a dependency is a decision,
      not an implementation detail.
- [ ] B1. `MemoryUsageSetting`
- [ ] B2. `ScratchFile` and `ScratchFileBuffer`
- [ ] B3. `NonSeekableRandomAccessReadInputStream`
- [ ] B4. `RandomAccessReadMemoryMappedFile`
- [ ] B5. Wire the flate fast path back into
      `PDPage.ContentsForStreamParsing`, and update the slice 2 note in
      `STATUS.md`

---

# Phase C — Run and fix

- [ ] C1. `gofmt -l .` clean
- [ ] C2. `go vet ./...` clean
- [ ] C3. `go test ./...` green
- [ ] C4. Record every Java bug found in `migration/JAVA-BUGS.md`
- [ ] C5. Update `migration/STATUS.md` — the phase 0 rows and the slice 2 note

---

# Phase D — Adversarial review

敵対的レビュー. See [`TEMPLATE.md`](TEMPLATE.md) for D1–D6. Additionally:

- [ ] D7. Check the temporary files are cleaned up
  - Java uses `File.deleteOnExit` and explicit close. Go has neither
    automatically. A scratch file left behind on a crash is a real defect.

- [ ] D8. Check concurrency honestly
  - `slice/0` already deviated once here: `CreateView` hands each caller its
    own cursor where Java caches one per thread id. Whatever this track does,
    document the contract on every exported type.

- [ ] D9. Re-read the `slice/0` note in `STATUS.md`
  - `pdfio` was not ported test-first. It is the one package where a later
    re-read against the Java is worth doing on its own, and this track is
    already in that code.

---

# Phase E — User feedback

See [`TEMPLATE.md`](TEMPLATE.md) for E1–E4. E3 is the strict-test rule.

---

# Blocked

- [ ] B0. The memory-mapping decision blocks B4 only. Everything else can
      proceed without it.
