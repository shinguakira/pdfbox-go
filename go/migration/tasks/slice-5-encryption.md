# Implementation Plan

Slice 5 — encrypted documents.

**Branch: `slice/5-<name>`** — from and back to `migration-base`.
Depends on `slice/1` only. Independent of slices 2, 3, 4, 6, 7 and 8.

## Rules — do not break these

- **NEVER change the Java.** No `.java`, no `pom.xml`, no test resource, for any
  reason. It is the reference; a reference that gets edited stops being one.
- **NEVER fix a bug that is in the Java.** Port it as written, comment it where
  it occurs, and record it in `migration/JAVA-BUGS.md`. **This matters most
  here:** a "fix" to a cryptographic path changes what documents open.
- **NEVER create a branch that is not in the migration plan**, and never add one
  to the plan's list.
- **NEVER change `migration/PLAN.md`.**
- **NEVER commit to `migration-base` directly.**
- **NEVER touch `apache/pdfbox`** — no PR, no pull, fetch, merge or rebase.
- **DO NOT STOP UNTIL PHASE E.** Phases A to D run end to end. Finishing a
  task is not a stopping point; neither is finishing a phase, a package, or a
  commit. Do not pause to report progress as if it were a result, do not ask
  whether to continue, and do not end a turn with a list of what is left.
  **E1 is the only stop in this file** — that is where the user reviews. Only
  the user stops the work before it.

`AGENTS.md` marks encryption a sensitive area. Port line for line.

## How each unit of work runs

Five phases, in this order, never overlapping:

**A — write the test.** Port the Java test to Go. Assertion values are copied
from the Java, never read off the Go. The implementation does not exist yet.

**B — port the implementation.** Write the Go from the Java source, line for
line. Do not look at what makes the test pass; look at what the Java does.

**C — run and fix.** `gofmt -l . && go vet ./... && go test ./...`. A failure
is a defect in the port, not in the test. Fix the Go. If the Java itself is
wrong, keep the wrong behaviour and record it in `JAVA-BUGS.md`.

**D — adversarial review.** Green tests are not evidence the port is faithful.
Read the Go against the Java looking for what the tests cannot catch, and
assume the port is wrong until each check says otherwise.

**E — user feedback.** Stop. Wait. Judge each item, and where it is a real
defect, write a strict failing test first and only then fix.

## Scope

| Java package | Files | Java tests |
| --- | ---: | ---: |
| `pdmodel/encryption` | 19 | 0 in that package |

The tests are not where the source is. They are in
`pdfbox/src/test/java/org/apache/pdfbox/encryption/`:
`TestSymmetricKeyEncryption` and `TestPublicKeyEncryption`, with fixtures in
`pdfbox/src/test/resources/org/apache/pdfbox/encryption/`.

Go covers most of the primitives: `crypto/aes`, `crypto/rc4`, `crypto/sha256`,
`crypto/md5`, `crypto/x509`. Port the key derivation, not the ciphers.

---

# Phase A — Write the tests

- [x] A1. Port `TestSymmetricKeyEncryption` — RC4 and AES, 40/128/256 bit
- [x] A2. Port `TestPublicKeyEncryption` — certificate-based
- [x] A3. Write from source for the 19 classes the two tests do not reach
  - Name which ones those are before writing, so the gap is visible

---

# Phase B — Port the implementation

- [x] B1. The security handler base and the registry
  - `SecurityHandler`, `SecurityHandlerFactory`, `ProtectionPolicy`
- [x] B2. Standard security — password
  - `StandardSecurityHandler`, `StandardProtectionPolicy`, `StandardDecryptionMaterial`,
    `AccessPermission`
- [x] B3. Public key security — certificate
  - `PublicKeySecurityHandler`, `PublicKeyProtectionPolicy`,
    `PublicKeyDecryptionMaterial`, `PublicKeyRecipient`
- [x] B4. The crypt filters and the rest of the package
- [x] B5. Wire decryption into the parser — an encrypted document must open

---

# Phase C — Run and fix

- [x] C1. `gofmt -l .` clean
- [x] C2. `go vet ./...` clean
- [x] C3. `go test ./...` green
- [x] C4. Record every Java bug found in `migration/JAVA-BUGS.md`
- [x] C5. Update `migration/STATUS.md`

---

# Phase D — Adversarial review

敵対的レビュー. Green tests prove the port passes the tests, not that it is a
faithful migration. Go in assuming it is wrong. Every check below is a question
the ported tests cannot answer.

- [ ] D1. Read every ported file against its Java side by side
  - Is any method missing? Any branch of an `if`, any `case`, any `catch`?
  - Is any loop bound, any off-by-one, any `<` that should be `<=` different?
  - Java `int` narrows on cast and `float` saturates; Go does neither. Is every
    such conversion written out?

- [ ] D2. Hunt for silently dropped behaviour
  - Anything Java does in a `finally` — is it still done on the Go error path?
  - Anything Java logs and swallows — does the Go swallow it too, or does it
    return an error the Java would not have?
  - Anything Java throws — is it an error, or a panic, and is that the right one?

- [ ] D3. Check the tests are Java-derived, not Go-derived
  - For each assertion: is that value in the Java test, or did it come from
    running the Go? A value read off the port proves nothing.
  - Does each test take the real path, with the real types? A test over a
    stand-in can pass while the path it stands for is broken.
  - Which Java test cases were dropped, and is each one recorded with a reason?

- [ ] D4. Check every deferral is real and recorded
  - Every "not ported yet" in a doc comment — is it in `migration/STATUS.md`?
  - Every deferral — is it deferred because the type is absent, or because it
    was hard? The second is not a deferral.

- [ ] D5. Check the Java bugs
  - Every bug found — is it in `migration/JAVA-BUGS.md` with where, what,
    what correct would be, where the Go carries it, and how confident?
  - Was any of them "fixed" on the way past? Revert it.

- [ ] D6. Write the review down
  - What was checked, what was found, what was fixed, what is still open

And for this branch in particular:

- [ ] D7. Check every byte-level operation
  - Java `byte` is signed and Go's is not. Every comparison, every shift, every
    array index derived from a byte is a place the port can silently differ.
  - Padding, key length truncation and the 32-byte password pad are the usual
    places this goes wrong.

- [ ] D8. Check what happens on the wrong password
  - Java's behaviour on a failed decrypt is specific. Does the Go do the same,
    or does it return a different error, or worse, garbage?

- [ ] D9. Do not accept a passing test as proof
  - A round trip that encrypts and decrypts with the same port passes even if
    both halves are wrong. Check against the checked-in encrypted fixtures,
    which Java produced.

---

# Phase E — User feedback

- [ ] E1. Stop and wait for the user's review. Do not start the next branch.

- [ ] E2. For each item of feedback, judge it before acting
  - Is it a port defect, a missing piece of scope, or a difference the Java
    itself has?
  - A Java difference is not fixed — it is recorded in `JAVA-BUGS.md` and the
    user is told why it stays.

- [ ] E3. Where it needs fixing, write a **strict** test first
  - Strict: it fails before the fix, takes the real path with the real types,
    and asserts what the Java does
  - Then fix the Go
  - Then `gofmt`, `go vet`, `go test ./...` again

- [ ] E4. Report back
  - What was changed, what was not, and why for each

---

# Blocked

- [x] `TestSymmetricKeyEncryption` writes encrypted PDFs as well as reading
      them. The writer lands in slice 7. Decide whether this branch ports only
      the reading half, or waits.
