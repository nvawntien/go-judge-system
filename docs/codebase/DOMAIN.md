# Domain model

## Identity

**User** (`services/auth/internal/domain/entity/user.go`) is identified by UUID string and owns username/email, bcrypt password hash, RBAC role, active flag, rating, profile attributes, and optional MinIO avatar URL/object key. A new user defaults to `user`, inactive and rating 0. `Activate`, `UpdatePassword`, `UpdateProfile`, `UploadAvatar`, and `AssignRole` are the domain-level transitions. Username and email are unique in persistence.

## Problem catalogue

**Problem** has title/slug, markdown-like description content, difficulty (`easy`, `medium`, `hard`), tags, examples, constraints, hints, limits, author, hidden flag and soft-delete timestamp (`services/problem/internal/domain/entity/problem.go`). It is created hidden; `Publish` exposes it and `Hidden` withdraws it. A hidden/deleted problem is not generally public. **Tag** has name/unique slug, description and active state. **TestCase** is one versioned ZIP bundle metadata record per problem: object key, test count and version. The test bundle contents are not domain-persisted in PostgreSQL.

## Submission and attempts

**Submission** stores immutable-at-creation contestant source/language/problem/user identity plus mutable current attempt, status and aggregate execution information (`services/submission/internal/domain/entity/submission.go`). Supported executable submission languages are CPP, Go, Python and Java; C and JavaScript parse but `IsExecutable` rejects them for official submission creation.

```mermaid
stateDiagram-v2
  [*] --> PENDING
  PENDING --> JUDGING
  PENDING --> ACCEPTED
  PENDING --> WRONG_ANSWER
  PENDING --> TIME_LIMIT_EXCEEDED
  PENDING --> MEMORY_LIMIT_EXCEEDED
  PENDING --> OUTPUT_LIMIT_EXCEEDED
  PENDING --> RUNTIME_ERROR
  PENDING --> COMPILATION_ERROR
  PENDING --> SYSTEM_ERROR
  JUDGING --> ACCEPTED
  JUDGING --> WRONG_ANSWER
  JUDGING --> TIME_LIMIT_EXCEEDED
  JUDGING --> MEMORY_LIMIT_EXCEEDED
  JUDGING --> OUTPUT_LIMIT_EXCEEDED
  JUDGING --> RUNTIME_ERROR
  JUDGING --> COMPILATION_ERROR
  JUDGING --> SYSTEM_ERROR
  ACCEPTED --> PENDING: rejudge
  WRONG_ANSWER --> PENDING: rejudge
```

The code defines terminal statuses as Accepted, Wrong Answer, time/memory/output limit, runtime, compilation and system errors. The diagram describes transitions supported by current methods/use cases; it is not a database-enforced state machine.

**SubmissionAttempt** is the immutable audit/provenance record for each submit or admin rejudge: unique `attempt_id`, trigger, triggering user, eventual status and testcase version/count/checksum. **SubmissionResult** is a per-testcase record tied to a submission and attempt. Results are replaced for the applied attempt, allowing detail APIs to show the current attempt’s output without retaining official inputs/expected output from worker messages.

## Queue and execution contracts

`pkg/judge.JobMessage` carries submission/problem/user/language/source/attempt and enqueue time. `pkg/judge.ResultMessage` carries aggregate status, compile/error information, metrics, testcase provenance and per-test summary. The attempt ID is explicitly documented as the idempotency key.

**Execution** (`workers/judge/internal/application/port/outbound`) consists of an execution request, language commands, resource limits, and testcase results. The worker maps executor statuses into public submission verdicts. For official jobs it removes input and expected output before publishing (`sanitizeOfficialResult`), preserving testcase confidentiality.
