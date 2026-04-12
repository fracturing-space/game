# Storage Port Contracts

Storage adapters are part of the service consistency contract, not just an
implementation detail.

Current required guarantees:

- journal adapters are safe for concurrent use
- once `AppendCommits` returns, appended records are immediately visible to
  `List`, `ListAfter`, `HeadSeq`, and `SubscribeAfter`
- `SubscribeAfter(after_seq)` emits committed records with `seq > after_seq`
  and must not miss or duplicate records across the catch-up/live boundary for
  one campaign
- projection and artifact adapters are safe for concurrent use
- projection, watermark, and artifact reads must return caller-safe values that
  do not alias mutable internal storage

These guarantees should be enforced by shared adapter contract tests and kept
green under `go test -race`.
