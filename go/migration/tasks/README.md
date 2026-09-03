# Branch task lists

One file per branch. Each carries the rules, the branch's scope, and the five
phases the work runs through.

These are **not** migration-wide documents. The migration-wide ones are
[`../PLAN.md`](../PLAN.md), [`../BRANCHING.md`](../BRANCHING.md),
[`../STATUS.md`](../STATUS.md) and [`../JAVA-BUGS.md`](../JAVA-BUGS.md).

[`TEMPLATE.md`](TEMPLATE.md) is the template. Copy it for a new branch. Do not
change it to suit one branch — change the copy.

| Branch | File | State |
| --- | --- | --- |
| `slice/0-*` | — | done, predates these files |
| `slice/1-open-document` | — | done, predates these files |
| `slice/2-content-streams` | — | done, predates these files |
| `slice/3-*` | [`slice-3-text-simple-fonts.md`](slice-3-text-simple-fonts.md) | next |
| `slice/4-*` | [`slice-4-text-cid-cff.md`](slice-4-text-cid-cff.md) | after 3 |
| `slice/5-*` | [`slice-5-encryption.md`](slice-5-encryption.md) | open — needs only slice 1 |
| `slice/6-*` | [`slice-6-filters-images.md`](slice-6-filters-images.md) | open — needs only slice 1 |
| `slice/7-*` | [`slice-7-write-merge.md`](slice-7-write-merge.md) | open — needs only slice 1 |
| `slice/8-*` | [`slice-8-forms-annotations.md`](slice-8-forms-annotations.md) | open — needs only slice 1 |
| `slice/9-*` | [`slice-9-rendering.md`](slice-9-rendering.md) | needs 3 and 6, and a decision |
| `track/xmpbox` | [`track-xmpbox.md`](track-xmpbox.md) | open — depends on nothing |
| `track/scratchfile` | [`track-scratchfile.md`](track-scratchfile.md) | open — needs only slice 0 |

The branch names above are the pattern `slice/N-name` from
[`../BRANCHING.md`](../BRANCHING.md). Only `slice/1-open-document` and
`slice/2-content-streams` have literal names so far, because those branches
exist. **The rest are not decided — do not invent one.**

## The five phases

**A — write the test.** Port the Java test to Go. Assertion values are copied
from the Java, never read off the Go. The implementation does not exist yet.

**B — port the implementation.** Write the Go from the Java source, line for
line. Do not look at what makes the test pass; look at what the Java does.

**C — run and fix.** `gofmt`, `go vet`, `go test`. A failure is a defect in the
port, not in the test.

**D — adversarial review.** 敵対的レビュー. Green tests prove the port passes
the tests, not that the migration is faithful.

**E — user feedback.** Stop. Wait. Judge each item before acting, and where it
needs fixing, write the failing test first.
