# User-Created PMCS Server Schema and API Design

**Date:** 2026-07-28  
**Status:** Approved design; pending written-spec review  
**Scope:** Postgres schema, synchronization protocol, and HTTP API contract  
**Target repository:** `miltechserver`

## 1. Authority and purpose

This document is the server-side design authority for remotely storing,
synchronizing, and distributing user-created PMCS checklists.

It builds on:

- `docs/client/2026-07-27-user-pmcs-server-sync-schema-design.md`;
- the implemented Flutter/Drift user-PMCS schema;
- the implemented client authoring and publication lifecycle;
- the existing Gin, Firebase Authentication, Postgres, and Jet server
  conventions; and
- the design decisions approved during the 2026-07-28 server review.

The client-originated document remains the source for the implemented local
content model and the original product decisions. This document supersedes it
where the server review changed or completed the contract, including:

- permanent checklist and subscription tombstones;
- removal of initial moderation support;
- monotonic community releases with no rollback;
- embedded complete aggregates in account-delta pages;
- strong ETag preconditions;
- concrete endpoints and response semantics;
- database-side account deletion integration;
- initial resource ceilings; and
- migration, verification, and deployment requirements.

This document does not contain migration SQL, Go implementation tasks, or
Flutter implementation tasks. Those require a separately approved
implementation plan.

## 2. Product boundary

The server supports:

- authenticated, private, cross-device synchronization of owned checklists;
- synchronization of incomplete mutable drafts;
- synchronization of immutable published revisions;
- explicit release of selected immutable revisions to a public community
  library;
- anonymous browsing of active community sources;
- authenticated linked, read-only subscriptions;
- explicit subscriber acceptance of a newer release; and
- server retrieval of an active subscriber's pinned release.

The server does not store custom PMCS execution or progress. The schema and
API contain no:

- active or completed custom inspections;
- inspection snapshots;
- equipment selections;
- procedure-step completions;
- faults;
- inspection notes;
- DA Form 2404 exports; or
- custom inspection history.

An installed community checklist is a linked subscription. It is not an
editable copy or fork.

## 3. Approved product decisions

1. Private drafts and publications synchronize between a user's devices.
2. Public community sharing is supported.
3. Community sharing is not targeted to individual users or shops.
4. Community installations are linked and read-only.
5. A subscriber remains pinned to an immutable installed revision until
   explicitly accepting an update.
6. Stale mutations are rejected through strong ETag preconditions.
7. Complete changed aggregates are embedded in account-delta responses.
8. Owned delta entries contain only the current draft and current publication.
9. Superseded revisions remain server-addressable but are not repeatedly
   included in owned account deltas.
10. Deletion wins permanently over stale offline uploads.
11. Checklist UUID and subscription tombstones are retained while the account
    exists.
12. Local Publish and Release to community are separate actions.
13. Community release is explicit and revision-specific.
14. Community release numbers are strictly increasing; rollback is not
    supported.
15. A public source can be active or retired.
16. Initial implementation has no moderation roles, suspension state, or
    moderation endpoints.
17. Community discovery provides recent keyset-paginated browsing and an
    optional exact normalized-model filter.
18. Community creator attribution uses the owner's current public display
    name, never UID or email.
19. Subscription update availability is read through a separate lightweight
    endpoint; releases do not fan out writes to subscribers.
20. Account deletion handles Postgres cleanup only. Firebase identity deletion
    remains part of the existing client/account workflow.
21. An authenticated Firebase user must already have a matching Postgres
    `users` row before using PMCS sync.
22. Existing account-free device-local checklists remain local-only.
23. A future Copy to my account operation creates a new checklist and a fully
    new UUID tree.
24. The server uses normalized relational content tables, not JSONB revision
    documents.

## 4. Architecture

The feature has four server boundaries:

```text
Account synchronization authority
└── user_pmcs_sync_state
    ├── owned checklist changes
    └── subscription changes

Owned checklist aggregate
├── user_pmcs_checklists
└── user_pmcs_revisions
    ├── user_pmcs_revision_models
    └── user_pmcs_sections
        ├── user_pmcs_section_models
        └── user_pmcs_items
            ├── user_pmcs_notices
            └── user_pmcs_procedure_steps

Community distribution
├── user_pmcs_community_sources
└── user_pmcs_community_releases

Linked installation
└── user_pmcs_subscriptions
```

The stable checklist row is the owned-aggregate concurrency authority. The
subscription row is the linked-installation concurrency authority. One
transactionally locked per-user sync-state row is the account-delta ordering
authority.

Authored fields belong to revisions. Public discovery state belongs to the
community source and release tables. Execution progress remains local.

## 5. Synchronization version model

### 5.1 Per-resource `sync_version`

Each checklist and subscription has a positive, monotonically increasing
`BIGINT` `sync_version`.

The current `sync_version` is represented to HTTP clients as an opaque strong
ETag. Clients must return the received validator in `If-Match` for mutations.
Clients must never construct ETags from version numbers themselves.

The version increments once per committed logical mutation, including:

- draft create or replacement;
- draft discard;
- publication;
- community release;
- community retirement;
- checklist deletion;
- subscription creation or resubscription;
- subscription update acceptance; and
- unsubscribe.

### 5.2 Per-account `account_change_version`

`user_pmcs_sync_state.current_version` is a nonnegative, monotonically
increasing `BIGINT` scoped to one user.

Every checklist or subscription mutation for that user:

1. locks the user's sync-state row;
2. increments `current_version`;
3. copies the new value into the changed root's
   `account_change_version`; and
4. commits both changes in one transaction.

The value is not a raw Postgres sequence. Sequence allocation is not coupled
to transaction commit order and therefore cannot serve as a lossless ordered
cursor for this protocol.

Mutations for different users do not share a sync-state row and are not
serialized with one another.

Multiple changes to the same root between client pulls collapse into that
root's latest complete state. Gaps in account versions are valid.

### 5.3 First-mutation sync-state creation

The authenticated user must already exist in `users`.

On the user's first PMCS mutation, the transaction:

1. inserts `user_pmcs_sync_state` with version zero using
   `ON CONFLICT DO NOTHING`;
2. selects the row `FOR UPDATE`; and
3. increments it as part of the mutation.

Concurrent first mutations therefore converge on one locked row.

## 6. Postgres schema

All timestamps are `TIMESTAMPTZ`. All authored text uses `TEXT`. All client
identity columns use `UUID` without server defaults.

### 6.1 `user_pmcs_sync_state`

Columns:

- `user_uid TEXT PRIMARY KEY`;
- `current_version BIGINT NOT NULL DEFAULT 0`;
- `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`.

Constraints:

- `user_uid` references `users(uid)` with `ON UPDATE CASCADE` and
  `ON DELETE CASCADE`;
- `current_version >= 0`.

### 6.2 `user_pmcs_checklists`

This is the stable owned root and permanent deletion authority.

Columns:

- `id UUID PRIMARY KEY`;
- `owner_uid TEXT`;
- `sync_version BIGINT NOT NULL`;
- `account_change_version BIGINT NOT NULL`;
- `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`;
- `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`;
- `deleted_at TIMESTAMPTZ`.

Constraints and rules:

- active rows require non-null `owner_uid`;
- null ownership is permitted only for deleted account-retention rows;
- `sync_version > 0`;
- `account_change_version > 0`;
- ownership is always derived from authentication;
- `owner_uid` references `users(uid)` with `ON UPDATE CASCADE` and
  `ON DELETE RESTRICT`;
- account cleanup must complete before deleting `users`;
- a checklist UUID is never physically removed while its owning account
  exists;
- a tombstoned checklist UUID can never be recreated; and
- an account-deletion cleanup may remove a private-only tombstone because the
  account and its stale authenticated clients cease to exist.

### 6.3 `user_pmcs_revisions`

Columns:

- `id UUID PRIMARY KEY`;
- `checklist_id UUID NOT NULL`;
- `state TEXT NOT NULL`;
- `revision_number INTEGER`;
- `name TEXT NOT NULL DEFAULT ''`;
- `description TEXT NOT NULL DEFAULT ''`;
- `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`;
- `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`;
- `published_at TIMESTAMPTZ`.

Constraints and rules:

- `state` is `draft`, `published`, or `superseded`;
- a draft has null `revision_number` and null `published_at`;
- published and superseded rows have positive `revision_number` and non-null
  `published_at`;
- at most one draft exists per checklist;
- at most one published row exists per checklist;
- non-null revision numbers are unique within a checklist;
- `(checklist_id, id)` is unique for composite ownership references;
- the checklist foreign key uses `ON UPDATE CASCADE` and `ON DELETE CASCADE`;
- after publication, tree content and revision metadata are immutable through
  ordinary repositories; the only permitted revision-row change is the
  lifecycle transition from `published` to `superseded`; and
- publication accepts the client-generated revision UUID and exact submitted
  next revision number.

The publication transaction locks the checklist, verifies the ETag, confirms
the submitted revision number is exactly one greater than the maximum existing
number, validates the complete submitted tree, supersedes the old publication,
and promotes the submitted draft.

The server never silently renumbers an offline publication.

### 6.4 `user_pmcs_revision_models`

Columns:

- `revision_id UUID NOT NULL`;
- `display_text TEXT NOT NULL`;
- `normalized_text TEXT NOT NULL`.

The primary key is `(revision_id, normalized_text)`.

The server derives `normalized_text` from `display_text`; clients do not supply
normalization authority.

### 6.5 `user_pmcs_sections`

Columns:

- `id UUID PRIMARY KEY`;
- `revision_id UUID NOT NULL`;
- `position INTEGER NOT NULL`;
- `title TEXT NOT NULL DEFAULT ''`.

Constraints:

- `position > 0`;
- `(revision_id, position)` is unique;
- the revision foreign key uses `ON DELETE CASCADE`.

Blank draft titles are permitted. Publication rejects them.

### 6.6 `user_pmcs_section_models`

Columns:

- `section_id UUID NOT NULL`;
- `display_text TEXT NOT NULL`;
- `normalized_text TEXT NOT NULL`.

The primary key is `(section_id, normalized_text)`.

Zero rows means the section applies universally.

### 6.7 `user_pmcs_items`

Columns:

- `id UUID PRIMARY KEY`;
- `section_id UUID NOT NULL`;
- `position INTEGER NOT NULL`;
- `interval TEXT NOT NULL DEFAULT ''`;
- `item_to_be_checked_or_serviced TEXT NOT NULL DEFAULT ''`;
- `performed_by TEXT NOT NULL DEFAULT ''`.

Constraints:

- `position > 0`;
- `(section_id, position)` is unique;
- the section foreign key uses `ON DELETE CASCADE`.

Blank authored fields are permitted in drafts and rejected where required for
publication.

### 6.8 `user_pmcs_notices`

Columns:

- `id UUID PRIMARY KEY`;
- `item_id UUID NOT NULL`;
- `position INTEGER NOT NULL`;
- `type TEXT`;
- `notice_text TEXT NOT NULL DEFAULT ''`.

Constraints:

- `position > 0`;
- `(item_id, position)` is unique;
- non-null `type` is `warning`, `caution`, or `note`;
- the item foreign key uses `ON DELETE CASCADE`.

Null type and blank text support incomplete drafts. Publication requires a
supported type and nonblank text.

### 6.9 `user_pmcs_procedure_steps`

Columns:

- `id UUID PRIMARY KEY`;
- `item_id UUID NOT NULL`;
- `position INTEGER NOT NULL`;
- `step_text TEXT NOT NULL DEFAULT ''`;
- `fault_found_if TEXT NOT NULL DEFAULT ''`.

Constraints:

- `position > 0`;
- `(item_id, position)` is unique;
- the item foreign key uses `ON DELETE CASCADE`.

Blank step text is allowed in drafts. Publication requires at least one
nonblank procedure step per item.

### 6.10 Content UUID and replacement rules

Content-node UUIDs are globally unique. A UUID already belonging to a
different revision, parent, checklist, or owner cannot be grafted into a
submitted tree.

A complete draft save:

1. validates all submitted UUIDs and parent relationships;
2. removes the existing mutable draft children;
3. inserts the complete submitted draft tree in batches; and
4. updates the draft and checklist root atomically.

Published and superseded trees are never passed through this replacement path.

### 6.11 `user_pmcs_community_sources`

This is the public lifecycle root for a checklist that has ever been released.

Columns:

- `checklist_id UUID PRIMARY KEY`;
- `status TEXT NOT NULL`;
- `current_release_revision_id UUID`;
- `latest_release_revision_number INTEGER NOT NULL`;
- `first_released_at TIMESTAMPTZ NOT NULL`;
- `updated_at TIMESTAMPTZ NOT NULL`;
- `retired_at TIMESTAMPTZ`.

Constraints and rules:

- `status` is `active` or `retired`;
- `latest_release_revision_number > 0`;
- active requires a current release and null `retired_at`;
- retired requires null current release and non-null `retired_at`;
- the source references its checklist with `ON DELETE RESTRICT`;
- the current release, when present, must belong to the same checklist;
- `latest_release_revision_number` is preserved while retired;
- voluntarily retired sources may reactivate only through a strictly higher
  release; and
- a tombstoned or account-deleted checklist can never reactivate.

There is no suspended state and no moderation metadata.

### 6.12 `user_pmcs_community_releases`

Columns:

- `revision_id UUID PRIMARY KEY`;
- `checklist_id UUID NOT NULL`;
- `released_at TIMESTAMPTZ NOT NULL DEFAULT now()`.

Constraints and rules:

- `(checklist_id, revision_id)` is unique;
- the composite foreign key verifies that the revision belongs to the
  checklist;
- only published or superseded revisions may be released;
- the complete immutable revision is revalidated before first release;
- repeating the same current release is idempotent;
- a different release must have a higher revision number than
  `latest_release_revision_number`;
- rollback is rejected; and
- the current release and active subscriptions pin their referenced release.

An unpinned historical release may be removed because the source separately
retains the highest ever released revision number.

### 6.13 `user_pmcs_subscriptions`

Columns:

- `subscriber_uid TEXT NOT NULL`;
- `checklist_id UUID NOT NULL`;
- `installed_revision_id UUID`;
- `sync_version BIGINT NOT NULL`;
- `account_change_version BIGINT NOT NULL`;
- `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`;
- `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`;
- `deleted_at TIMESTAMPTZ`.

The primary key is `(subscriber_uid, checklist_id)`.

Constraints and rules:

- active rows require non-null `installed_revision_id`;
- tombstones require null `installed_revision_id`;
- `sync_version > 0`;
- `account_change_version > 0`;
- subscriber identity is derived from authentication;
- the subscriber references `users(uid)` with `ON DELETE CASCADE`;
- the source is identified by `checklist_id`;
- an installed revision must be a community release belonging to that source;
- an owner cannot subscribe to their own source;
- a new subscription installs the source's current release;
- unsubscribe clears `installed_revision_id` and permanently retains the
  lightweight row while the subscriber account exists;
- resubscribe explicitly clears the tombstone and installs the then-current
  release; and
- update acceptance moves only to the source's current, higher-numbered
  release.

Clearing `installed_revision_id` on unsubscribe ensures permanent tombstones
do not permanently pin release content.

## 7. Text and Unicode contract

### 7.1 Grapheme limits

The client advertises:

- 200 graphemes for short fields; and
- 4,000 graphemes for long fields.

Short fields:

- checklist name;
- model display and normalized values;
- section title;
- item interval; and
- performed-by value.

Long fields:

- checklist description;
- item text;
- notice text;
- procedure-step text; and
- fault-found-if text.

Go must count Unicode extended grapheme clusters compatibly with Flutter's
`characters` package. Postgres `char_length` is not sufficient because it
counts code points rather than user-perceived graphemes.

### 7.2 Postgres byte ceilings

Postgres additionally enforces:

- 8 KiB for each short field; and
- 64 KiB for each long field.

These checks are resource-protection ceilings, not user-facing character
limits. A pathological combining sequence can pass the grapheme limit and fail
the byte ceiling intentionally.

### 7.3 Model normalization

The canonical model normalization algorithm is:

1. trim leading and trailing whitespace;
2. collapse each whitespace run to one ASCII space; and
3. lowercase the result according to the shared Flutter/Go fixtures.

Mutation requests send `display_text`. The Go service derives and stores
`normalized_text`. Responses return both values.

Shared fixtures must include:

- ASCII and non-ASCII whitespace;
- composed and decomposed graphemes;
- mixed case;
- repeated whitespace;
- leading and trailing whitespace; and
- duplicate values that normalize identically.

## 8. Initial resource ceilings

The initial limits are configurable deployment settings with these defaults
per revision:

| Resource | Maximum |
|---|---:|
| Checklist-level models | 100 |
| Sections | 100 |
| Section models per section | 100 |
| Section models total | 1,000 |
| Items total | 2,000 |
| Notices per item | 100 |
| Notices total | 4,000 |
| Procedure steps per item | 250 |
| Procedure steps total | 10,000 |
| Mutation JSON body | 8 MiB uncompressed |

Mutation endpoints accept uncompressed `application/json` only in the initial
release. Unknown JSON fields, trailing JSON values, and invalid content types
are rejected.

These ceilings are server abuse-protection limits. They may be raised after
representative load and memory testing without changing response schemas.

## 9. Write transaction model

Writes use `READ COMMITTED` plus explicit row locks.

### 9.1 Validation before locks

The service performs pure validation before opening the transaction:

- body size;
- JSON shape;
- UUID syntax and duplicate UUIDs within the submission;
- node counts;
- grapheme limits;
- byte limits;
- normalized model derivation;
- sibling-position contiguity; and
- publication content completeness where applicable.

Authorization, current database state, parent ownership, and concurrency are
rechecked inside the transaction.

### 9.2 Lock order

Transactions acquire only required locks and use a consistent order:

1. relevant `user_pmcs_sync_state` rows in `user_uid` order;
2. checklist roots in UUID order;
3. community source rows in checklist UUID order;
4. subscription rows in subscriber/checklist order; and
5. dependent revisions and content rows.

Ordinary mutations affect one account sync-state row. Account deletion can
lock multiple roots and orders them deterministically.

### 9.3 Mutation commit

While holding locks, the transaction:

1. derives and verifies authorization;
2. evaluates `If-Match` or `If-None-Match`;
3. rechecks UUID ownership and foreign relationships;
4. validates the requested state transition;
5. applies batched writes;
6. increments the root `sync_version`;
7. increments and copies `account_change_version`;
8. assigns server timestamps; and
9. commits.

No external network call, response encoding, or authored-text logging occurs
inside the transaction.

### 9.4 Transaction retries

The repository may retry a complete write transaction a small bounded number
of times for Postgres:

- deadlock detected (`40P01`); and
- serialization failure (`40001`).

Retries use short jitter and the original validated request. All other SQL
errors fail immediately. Exhausted retry errors return a generic server
failure and are observable through metrics.

## 10. Optimistic concurrency and idempotency

### 10.1 Strong ETags

Checklist and subscription reads return opaque strong ETags. Every mutation
requires:

- `If-Match` for an existing resource; or
- `If-None-Match: *` for first creation of a client-ID resource.

Missing required preconditions return `428 Precondition Required`. Failed
preconditions return `412 Precondition Failed`.

Preconditions are evaluated while holding the root lock.

### 10.2 Idempotent resource operations

Creation, publication, release, subscription, update acceptance, retirement,
and deletion are modeled as `PUT` or `DELETE` against client-known resource
identities.

When a precondition no longer matches, the server may return success without
another mutation only when it can prove:

- the authenticated principal is authorized for the existing result;
- the exact requested target state is already present; and
- no requested state would overwrite a later incompatible state.

The response returns the current canonical resource representation and
validator.

Examples:

- recreating an existing checklist with an identical client representation;
- retrying publication of the same revision when it is already either the
  current publication or an immutable superseded publication;
- retrying release of the source's same current revision;
- retrying installation of the same active subscription; and
- retrying deletion of an already-tombstoned owned resource.

An existing different representation, a different owner, a newer incompatible
state, or a tombstoned UUID submitted as a new resource fails without revealing
private content.

No generic idempotency-key table is introduced.

## 11. Account delta protocol

### 11.1 Endpoint

```http
GET /api/v1/auth/user-pmcs/sync?after={version}&limit={n}
```

`after` is a nonnegative account cursor. The default entry limit is 10 and the
maximum is 25.

### 11.2 Consistent snapshot

The endpoint uses one read-only `REPEATABLE READ` transaction:

1. read the user's current account version;
2. union checklist roots owned by the user and subscription roots belonging to
   the user;
3. filter roots with `account_change_version > after`;
4. order by `account_change_version`;
5. load each selected root's complete chosen representation from the same
   snapshot; and
6. commit before JSON encoding.

A concurrent mutation committed after the snapshot receives a later account
version and appears on a subsequent pull.

### 11.3 Embedded change shapes

An active checklist change embeds:

- checklist metadata;
- current draft and its complete tree, when present;
- current publication and its complete tree, when present; and
- current community-source summary, when one exists.

It does not embed superseded revisions.

A checklist tombstone embeds:

- checklist UUID;
- final versions;
- server timestamps; and
- `deleted_at`.

It contains no authored tree.

An active subscription change embeds:

- subscription metadata;
- source summary; and
- the complete immutable installed revision.

An unsubscribe tombstone embeds subscription identity, versions, timestamps,
and `deleted_at`, but no installed content.

### 11.4 Response and page boundary

```json
{
  "from_cursor": 41,
  "through_cursor": 46,
  "account_version": 49,
  "has_more": true,
  "changes": []
}
```

Rules:

- entries are ordered by `account_change_version`;
- one aggregate is never split between pages;
- the server stops before 20 MiB of uncompressed canonical JSON;
- one otherwise-valid aggregate may occupy a page by itself;
- `through_cursor` is the highest returned change version, or `after` when the
  page is empty;
- `account_version` is the sync-state value observed by the snapshot; and
- `has_more` is true when additional changed roots from that snapshot remain
  after `through_cursor`.

The response supports compression and returns `Vary: Accept-Encoding`.

### 11.5 Client cursor advancement

The client must:

1. receive the entire page;
2. apply every change in one durable local transaction;
3. store `through_cursor` in that same transaction; and
4. request the next page only after local commit succeeds.

It must not advance the cursor after partially applying a page.

## 12. Community discovery

### 12.1 Browse endpoint

```http
GET /api/v1/user-pmcs/community?after={cursor}&limit={n}&model={normalized}
```

The endpoint:

- requires no authentication;
- returns active sources only;
- returns one entry per source representing its current release;
- sorts by current source `updated_at DESC`, then checklist UUID;
- uses an opaque keyset cursor;
- accepts an optional exact canonical normalized-model filter; and
- does not provide text search, ratings, or popularity ranking.

The server normalizes the supplied model filter with the canonical algorithm.

Public summaries contain:

- source checklist UUID;
- current revision UUID and revision number;
- checklist name and description;
- checklist models;
- current public creator display name;
- release timestamp; and
- source update timestamp.

They never contain owner UID or email.

### 12.2 Current release endpoint

```http
GET /api/v1/user-pmcs/community/{checklist_id}
```

This returns the complete current immutable release for an active source.
Retired, deleted, and never-released sources are unavailable publicly.

Creator attribution is resolved from the current `users.username`. Retained
subscriber content whose owner account was deleted uses a generic deleted-user
label.

## 13. Subscription update discovery

```http
GET /api/v1/auth/user-pmcs/subscriptions/updates
```

The endpoint compares each active subscription's installed revision with its
source's current active release.

It returns only lightweight metadata:

- source checklist UUID;
- source status;
- installed revision UUID and number;
- current release UUID and number, when active; and
- whether a newer release is available.

It does not mutate subscription rows, increment account versions, download
release content, or fan out writes when creators release revisions.

Retired sources are reported as retired with no update target. The installed
revision remains usable and redownloadable by that active subscriber.

## 14. HTTP endpoint surface

### 14.1 Owned synchronization and authoring

| Method | Route | Purpose |
|---|---|---|
| `GET` | `/api/v1/auth/user-pmcs/sync` | Embedded account delta |
| `GET` | `/api/v1/auth/user-pmcs/checklists/{checklist_id}` | Current owned aggregate and conflict recovery |
| `PUT` | `/api/v1/auth/user-pmcs/checklists/{checklist_id}` | Create an offline-generated root and initial draft |
| `PUT` | `/api/v1/auth/user-pmcs/checklists/{checklist_id}/drafts/{revision_id}` | Create or replace the complete current draft |
| `DELETE` | `/api/v1/auth/user-pmcs/checklists/{checklist_id}/drafts/{revision_id}` | Discard a draft when a publication exists |
| `PUT` | `/api/v1/auth/user-pmcs/checklists/{checklist_id}/publications/{revision_id}` | Atomically submit and publish the complete revision |
| `GET` | `/api/v1/auth/user-pmcs/checklists/{checklist_id}/revisions/{revision_id}` | Retrieve an owned immutable historical revision |
| `DELETE` | `/api/v1/auth/user-pmcs/checklists/{checklist_id}` | Permanently tombstone a checklist |

Publication accepts the complete revision tree. An offline edit-and-publish
does not require a separate draft upload before the publication request.

### 14.2 First synchronization of offline publication history

A newly account-backed checklist can contain multiple locally published
revisions before its first successful server connection. The client
reconstructs that immutable history in ascending revision-number order rather
than sending unbounded history in one request:

1. create the checklist root and revision 1 tree as its initial draft with
   `If-None-Match: *`;
2. publish revision 1 with the returned checklist ETag;
3. for each later local publication, upload its tree and original
   client-generated revision UUID as the current draft, then publish the exact
   next revision number;
4. after reconstructing the current publication, upload the current local
   draft when one exists; and
5. refresh the owned aggregate and resume at the next missing revision after
   any interrupted request sequence.

The server preserves client-generated revision UUIDs and rejects gaps,
renumbering, or divergent existing content. If the server already contains an
identical prefix, proven idempotent operations return the current canonical
state and the client continues from the next missing revision. A mismatch
returns `412` or `409` as appropriate and requires reconciliation.

Each tree is still bounded by the per-mutation 8 MiB uncompressed request
ceiling. This protocol synchronizes every authored revision without requiring
an oversized first-upload payload.

### 14.3 Owner community operations

| Method | Route | Purpose |
|---|---|---|
| `PUT` | `/api/v1/auth/user-pmcs/checklists/{checklist_id}/community-releases/{revision_id}` | Release a valid strictly higher revision |
| `DELETE` | `/api/v1/auth/user-pmcs/checklists/{checklist_id}/community-source` | Retire the source |

### 14.4 Public community endpoints

| Method | Route | Purpose |
|---|---|---|
| `GET` | `/api/v1/user-pmcs/community` | Recent active sources with optional model filter |
| `GET` | `/api/v1/user-pmcs/community/{checklist_id}` | Complete current public release |

### 14.5 Authenticated subscription endpoints

| Method | Route | Purpose |
|---|---|---|
| `PUT` | `/api/v1/auth/user-pmcs/subscriptions/{checklist_id}` | Install or explicitly resubscribe |
| `DELETE` | `/api/v1/auth/user-pmcs/subscriptions/{checklist_id}` | Unsubscribe |
| `GET` | `/api/v1/auth/user-pmcs/subscriptions/updates` | Discover newer releases |
| `PUT` | `/api/v1/auth/user-pmcs/subscriptions/{checklist_id}/installed-releases/{revision_id}` | Accept the current higher release |
| `GET` | `/api/v1/auth/user-pmcs/subscriptions/{checklist_id}/installed-releases/{revision_id}` | Redownload the pinned release |

The pinned-release GET remains authorized after source retirement. It succeeds
only for the active subscription's installed revision.

## 15. Response and error contract

### 15.1 Canonical values

- UUIDs are lowercase canonical strings.
- Timestamps are RFC 3339 UTC values controlled by the server.
- ETags are opaque quoted validators.
- Clients do not submit owner or subscriber UID.
- Mutation responses return the committed canonical representation.
- Immutable revision responses use cache validators appropriate to their
  visibility.

### 15.2 Success response

JSON success responses use the application's existing standard envelope:

```json
{
  "status": 200,
  "message": "",
  "data": {}
}
```

The first successful resource creation returns `201 Created`. Other successful
mutations, including a proven idempotent retry, return `200 OK` with the
current canonical representation and ETag. Deletes return `200 OK` with the
resulting tombstone or retired representation; these APIs do not use a
bodyless `204` response.

Owned aggregate, immutable revision, current community release, and pinned
release GETs accept `If-None-Match`. A matching current validator returns
`304 Not Modified` with no response body. Account delta pagination remains
cursor-driven and does not use `If-None-Match` as a substitute for `after`.

### 15.3 Error response

Errors use stable machine-readable codes and safe messages. They do not expose
SQL, constraint names, stack traces, or private resource content.

Errors use a stable extension of the application response envelope:

```json
{
  "status": 412,
  "message": "resource changed",
  "data": null,
  "error": {
    "code": "stale_precondition",
    "details": {}
  }
}
```

The `details` object is omitted when empty. When safe and applicable, it can
include:

- resource UUID;
- current ETag when safe and applicable;
- validation entity UUID;
- validation field path; and
- configured limit.

The minimum stable code set is:

| Code | Status | Meaning |
|---|---:|---|
| `invalid_request` | 400 | Request syntax, UUID, cursor, JSON, or content type is invalid |
| `invalid_precondition` | 400 | A conditional header is malformed or unsupported |
| `authentication_required` | 401 | Firebase authentication is missing or invalid |
| `forbidden` | 403 | The operation is forbidden and existence can safely be revealed |
| `resource_not_found` | 404 | The resource is absent or intentionally hidden |
| `account_not_initialized` | 409 | The authenticated Firebase UID has no Postgres `users` row |
| `invalid_transition` | 409 | The current resource cannot make the requested domain transition |
| `stale_precondition` | 412 | A syntactically valid conditional does not match current state |
| `content_too_large` | 413 | The uncompressed request or aggregate exceeds its ceiling |
| `validation_failed` | 422 | A structurally valid PMCS tree violates publication or field rules |
| `precondition_required` | 428 | The required conditional header is absent |
| `rate_limited` | 429 | The request exceeded a configured limiter |
| `internal_error` | 500 | An unexpected server failure occurred |

### 15.4 Status semantics

| Status | Meaning |
|---|---|
| `400 Bad Request` | Malformed JSON, invalid UUID or cursor, unknown field, trailing JSON, or invalid content type |
| `401 Unauthorized` | Missing or invalid Firebase authentication |
| `403 Forbidden` | Authenticated operation is forbidden and revealing resource existence is safe |
| `404 Not Found` | Resource unavailable or intentionally hidden |
| `409 Conflict` | Current ETag is valid but the requested domain transition is invalid |
| `412 Precondition Failed` | `If-Match` or `If-None-Match` failed |
| `413 Content Too Large` | Request exceeds the aggregate byte ceiling |
| `422 Unprocessable Content` | Structurally valid request fails PMCS content validation |
| `428 Precondition Required` | Required conditional header is missing |
| `429 Too Many Requests` | Configured rate limit exceeded |
| `500 Internal Server Error` | Generic unexpected server failure |

Cross-owner lookups return `404` rather than proving another user's private
resource exists.

## 16. Publication validation

Draft synchronization allows incomplete authored content. Publication and
first community release revalidate the complete revision.

Publication requires:

- nonblank checklist name;
- at least one valid checklist model;
- at least one section;
- contiguous one-based section positions;
- nonblank section title;
- at least one item per section;
- contiguous one-based item positions;
- nonblank item interval;
- nonblank item-to-be-checked-or-serviced text;
- contiguous one-based notice and step positions;
- supported notice type and nonblank notice text;
- at least one procedure step per item;
- nonblank procedure-step text;
- valid model display values;
- unique canonical model values within each parent;
- all grapheme, byte, node, and body ceilings;
- unique UUIDs and valid parent ownership;
- exact next publication number; and
- immutable prior publications.

The server never trusts prior client validation.

## 17. Deletion and retention

### 17.1 Private-only checklist

Deletion:

- permanently tombstones the checklist UUID;
- advances both version fields;
- removes its draft and revisions;
- removes all content children; and
- retains only the lightweight root while the owner account exists.

Every later create or mutation using the UUID fails.

### 17.2 Released checklist

Deletion:

- tombstones the checklist;
- retires the community source;
- clears the current release;
- removes the draft and every unpinned revision;
- retains active-subscriber-pinned releases and trees; and
- prevents all future release or reactivation.

### 17.3 Unsubscribe

Unsubscribe:

- clears `installed_revision_id`;
- sets `deleted_at`;
- advances both subscription versions; and
- permanently retains the lightweight composite-key tombstone while the
  subscriber account exists.

A later stale install cannot silently recreate the subscription. Resubscribe
is an explicit conditional mutation of the tombstoned row.

### 17.4 Account deletion

The existing authenticated database-account deletion path must derive UID from
the verified token. A request-body UID is not deletion authority.

One Postgres transaction:

1. locks the user's sync state;
2. locks owned checklist roots in UUID order;
3. locks related community sources in UUID order;
4. locks the user's subscription roots in checklist order;
5. deletes private checklist content and roots;
6. tombstones formerly public checklist roots needed by pinned releases;
7. retires their sources and clears current releases;
8. removes unpinned releases and content;
9. nulls ownership on retained released content;
10. deletes the user's subscriptions and sync state; and
11. deletes the `users` row.

No Firebase Admin deletion is added to this feature.

## 18. Access control

### 18.1 Owner

Only the authenticated owner may:

- read private checklist state;
- save or discard a draft;
- publish a revision;
- retrieve superseded revisions;
- release a revision;
- retire a source; or
- delete the checklist.

### 18.2 Public user

Any user may browse active community summaries and retrieve the active source's
current release.

### 18.3 Subscriber

An authenticated active subscriber may:

- retrieve the exact installed release;
- check for a newer release;
- explicitly accept the current higher release; and
- unsubscribe.

Retired sources remain readable only through active subscriptions pinned to a
retained release.

### 18.4 Database authorization

The application owns the database connection. Postgres RLS is not introduced.
Authorization is enforced in service and repository queries and proven with
integration tests.

## 19. Logging, metrics, and rate limiting

Logs may contain:

- operation name;
- authenticated UID;
- checklist, revision, or source UUID;
- status and stable error code;
- duration;
- retry count;
- node counts; and
- request and response byte counts.

Logs must not contain authored checklist text, descriptions, notices, steps,
fault-found-if text, email, tokens, or request bodies.

Metrics include:

- operation request count and latency;
- database transaction latency;
- response encoding latency;
- ETag conflict count;
- validation rejection count by code;
- transaction retry and exhaustion count;
- delta page entry and byte sizes;
- community query latency;
- subscription update count; and
- rate-limit rejection count.

Public browse/detail and authenticated sync/mutation routes use separately
configurable per-IP and per-user rate limits.

## 20. Index design

Indexes are limited to constraints, foreign-key support, and approved endpoint
queries.

### 20.1 Checklists

- primary key on `id`;
- `(owner_uid, account_change_version)` for owner delta.

No speculative owner-list index is added without a corresponding endpoint and
query-plan evidence.

### 20.2 Revisions

- partial unique `checklist_id` where state is `draft`;
- partial unique `checklist_id` where state is `published`;
- unique `(checklist_id, revision_number)` where revision number is non-null;
- unique `(checklist_id, id)` for composite references.

A redundant checklist/state index is not added unless measured plans require
it.

### 20.3 Ordered content

- revision models primary key `(revision_id, normalized_text)`;
- revision models reverse index `(normalized_text, revision_id)`;
- sections unique `(revision_id, position)`;
- section models primary key `(section_id, normalized_text)`;
- items unique `(section_id, position)`;
- notices unique `(item_id, position)`;
- procedure steps unique `(item_id, position)`.

These indexes enforce sibling uniqueness and cover parent-ordered reads.

### 20.4 Community

- sources primary key `checklist_id`;
- sources partial `(updated_at DESC, checklist_id)` where status is `active`;
- releases primary key `revision_id`;
- releases unique `(checklist_id, revision_id)`;
- releases `(checklist_id, released_at DESC)`.

The reverse revision-model index supports exact normalized-model discovery.

### 20.5 Subscriptions

- primary key `(subscriber_uid, checklist_id)`;
- `(subscriber_uid, account_change_version)` for delta;
- partial active subscriber/checklist/installed-revision index for update
  discovery;
- partial `installed_revision_id` where `deleted_at IS NULL` for retention;
- checklist-leading coverage where required by the source and composite release
  foreign keys.

Every foreign-key column must be covered by an index whose leading columns
match the relevant join or delete/restriction access path. Primary and unique
indexes are reused instead of duplicated.

## 21. Migration and Jet generation

The current target branch contains migrations through `008`, so this design
currently maps to:

- `migrations/009_create_user_pmcs_sync.sql`; and
- `migrations/009_rollback_user_pmcs_sync.sql`.

The number must be reverified immediately before implementation.

The forward migration is additive and creates all tables, constraints, and
indexes required by this design.

The rollback drops child tables before parents. It is valid for disposable
environments and pre-launch verification only.

After applying the canonical forward migration to the generation database,
Jet models under `.gen` must be regenerated. Generated files are never
hand-edited.

## 22. Required verification

### 22.1 Migration verification

- forward migration succeeds;
- every constraint, cascade, and restriction behaves as designed;
- rollback succeeds before feature data exists;
- forward migration reapplies after rollback;
- generated Jet models match the migrated schema.

### 22.2 Validation and service tests

- Flutter/Go Unicode normalization fixtures match;
- grapheme and byte boundaries are exact;
- node and body ceilings are enforced;
- drafts accept incomplete content;
- publication rejects incomplete content;
- UUID grafting across parents, revisions, checklists, and owners fails;
- unknown JSON fields and trailing documents fail.

### 22.3 Authorization and HTTP tests

- account initialization is required;
- owner identity comes only from authentication;
- cross-owner resources are hidden;
- public endpoints expose no private revision or identity data;
- missing, current, stale, and malformed ETags map correctly;
- equivalent resource retries are idempotent;
- incompatible retries fail;
- every stable error code and status is covered.

### 22.4 Concurrency tests

Using a real Postgres database:

- simultaneous draft saves cannot both overwrite;
- simultaneous publications cannot both succeed;
- publication numbers cannot skip or duplicate;
- releases cannot roll back;
- per-user account changes commit in version order;
- different users are not serialized by one global lock;
- bounded deadlock retries behave correctly.

### 22.5 Delta tests

- checklist and subscription branches merge into one ordered page;
- current complete aggregates match the page snapshot;
- no aggregate is split;
- entry and byte boundaries are respected;
- a concurrent later mutation appears on a subsequent pull;
- a failed local apply does not require cursor advancement;
- permanent tombstones defeat stale recreation and mutation.

### 22.6 Community and retention tests

- drafts and unreleased revisions never appear publicly;
- retired and deleted sources never appear publicly;
- exact normalized-model filtering is correct;
- keyset pagination has no duplicates or skips;
- new releases do not mutate existing subscriptions;
- update discovery identifies only higher current releases;
- unsubscribe releases its content pin;
- pinned releases survive retirement and owner account deletion;
- public creator attribution follows current username;
- deleted-account retained content is anonymized.

### 22.7 Performance verification

Representative data at the approved node ceilings must be tested with:

- maximum-size draft replacement;
- maximum-size publication;
- compressed and uncompressed delta encoding;
- concurrent independent users;
- model-filtered public browsing; and
- subscription update checks.

Capture:

- request validation time;
- database time;
- lock wait time;
- JSON encoding time;
- peak memory;
- compressed and uncompressed bytes; and
- `EXPLAIN (ANALYZE, BUFFERS)` for every indexed access path.

## 23. Deployment and rollback

Deployment order:

1. apply the additive schema migration;
2. regenerate and deploy Jet-backed server code;
3. deploy private synchronization endpoints;
4. deploy community and subscription endpoints;
5. run production read/write probes with test accounts;
6. stabilize and publish the API contract to the mobile team;
7. implement mobile remote synchronization;
8. preserve existing account-free checklists as local-only.

After production feature writes begin, normal rollback means disabling or
reverting the API binary while retaining the new tables. Dropping the schema
would destroy user-authored content and is not an acceptable routine rollback.

## 24. Explicitly deferred work

This design does not add:

- moderation roles or endpoints;
- suspended community sources;
- direct user sharing;
- shop-scoped sharing;
- collaborative editing;
- automatic tree merges;
- editable forks or lineage;
- ratings, reviews, comments, reports, or popularity ranking;
- community name or description search;
- images or attachments;
- automatic subscription updates;
- release rollback;
- custom inspection or progress synchronization;
- automatic conversion of existing local checklists;
- Firebase identity deletion;
- Postgres RLS;
- an append-only account change ledger;
- JSONB revision documents; or
- an implementation-plan commitment before written-spec approval.
