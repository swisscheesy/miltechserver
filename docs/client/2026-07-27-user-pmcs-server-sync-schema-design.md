# User-Created PMCS Server Sync Schema Design

**Date:** 2026-07-27  
**Status:** Approved design; pending written-spec review  
**Scope:** Postgres schema and database-facing synchronization contract  
**Target repository:** `miltechserver`

## 1. Authority and purpose

This document defines the approved database design for remotely storing and
synchronizing user-created PMCS checklists.

It is based on the implemented Flutter and Drift architecture in:

- `lib/_data/database/tables.dart`;
- `lib/_data/repository/user_pmcs_authoring_store.dart`;
- `lib/_data/models/user_pmcs/user_pmcs_models.dart`;
- `lib/_data/models/user_pmcs/user_pmcs_validator.dart`; and
- `lib/_data/models/user_pmcs/user_pmcs_model_normalizer.dart`.

It supersedes the database structure proposed in
`docs/features/user_created_pmcs_server_tables.md`. That earlier sketch remains
useful historical context, but it models a checklist as one mutable content
tree. The implemented feature instead has a stable checklist identity, a
separate mutable draft, immutable published and superseded revisions, and new
content-node UUIDs for every cloned revision.

This design contains no migration SQL, Go implementation, API endpoint design,
or Flutter implementation plan.

## 2. Product boundary

The server will support both:

- private, authenticated, cross-device synchronization of owned checklists;
  and
- explicit release of selected immutable revisions to a public community
  library.

Individual PMCS execution remains local-only. The server does not store:

- active or completed custom inspections;
- inspection snapshots;
- equipment selections;
- procedure-step completions;
- faults;
- inspection notes;
- DA Form 2404 exports; or
- custom inspection history.

An installed community checklist is a linked, read-only subscription to its
creator's source. It is not an editable fork.

## 3. Approved product decisions

1. Private drafts and published revisions synchronize between a user's
   devices.
2. Public community sharing is supported.
3. Community sharing is not targeted to specific users or shops.
4. Community installations are linked subscriptions.
5. Subscribers remain on an immutable installed revision until they explicitly
   accept an update.
6. Stale writes are rejected through optimistic concurrency.
7. Deletion wins over a later upload from a stale offline device.
8. Deleting or unpublishing a public source retires it from discovery while
   preserving revisions required by active subscribers.
9. Local Publish and Release to community are separate actions.
10. Every community release is explicit and revision-specific.
11. A community release is immediately discoverable.
12. Administrators can suspend and reactivate community sources.
13. Account deletion removes private content, retires public sources,
    anonymizes retained content, and preserves subscriber-pinned revisions.
14. Existing account-free device-local checklists remain local-only.
15. A future Copy to my account operation must create a completely new UUID
    tree rather than claiming a legacy local checklist.
16. The server uses normalized relational content tables rather than JSONB
    revision documents.
17. Community browsing is public, but installing a linked subscription
    requires authentication.

## 4. Architecture

The schema has four boundaries:

```text
Owned checklist aggregate
├── revisions
│   ├── revision models
│   └── sections
│       ├── section models
│       └── items
│           ├── notices
│           └── procedure steps
├── community source
│   └── immutable community releases
└── user subscriptions
```

The stable checklist row is the checklist concurrency authority. One
transactionally locked per-user sync-state row is the account delta authority.
Authored fields belong to revisions. Public state belongs to community tables,
not to drafts or private revision rows.

## 5. Per-user synchronization state

Raw Postgres sequences are not safe as lossless client cursors for this design.
Sequence values are allocated outside transaction commit ordering, so a client
could advance past a higher committed value while a lower value is still
uncommitted.

The schema instead includes `user_pmcs_sync_state`.

Columns:

- `user_uid`: `TEXT` primary key and foreign key to `users(uid)`;
- `current_version`: nonnegative `BIGINT`; and
- `updated_at`: server-controlled `TIMESTAMPTZ`.

The user foreign key uses `ON UPDATE CASCADE` and `ON DELETE CASCADE`. The row
is created transactionally on the user's first user-PMCS mutation.

Every checklist or subscription mutation for a user locks this row, increments
`current_version` transactionally, and copies the resulting value into the
changed checklist or subscription as `account_change_version`.

Checklist and subscription rows therefore each contain:

- a per-row `sync_version`; and
- an `account_change_version`.

These values have different purposes.

### 5.1 `sync_version`

`sync_version` is a positive, monotonically increasing `BIGINT` scoped to one
checklist or subscription.

Clients submit the version they last read. A mutating transaction locks the
row, compares the expected version, rejects stale writes, and increments the
version after a successful change.

### 5.2 `account_change_version`

`account_change_version` is copied from the transactionally incremented
per-user sync-state row.

It provides a lossless account-wide delta cursor. Mutations for one account are
serialized by the sync-state row lock, so a later version cannot commit ahead
of an earlier version. A rolled-back transaction does not advance the stored
counter.

Multiple changes to one aggregate between client pulls may collapse into its
latest row because checklist synchronization transfers the current complete
aggregate state.

Timestamps must not be used as lossless synchronization cursors.

## 6. Table design

### 6.1 `user_pmcs_checklists`

This is the stable owned aggregate and serialization point. It contains no
authored checklist content.

Columns:

- `id`: client-generated `UUID`, primary key, no server default;
- `owner_uid`: nullable `TEXT` foreign key to `users(uid)`;
- `sync_version`: positive `BIGINT`;
- `account_change_version`: positive `BIGINT`;
- `created_at`: server-controlled `TIMESTAMPTZ`;
- `updated_at`: server-controlled `TIMESTAMPTZ`; and
- `deleted_at`: nullable server-controlled `TIMESTAMPTZ`.

Rules:

- `owner_uid` is required for an active checklist.
- A null owner is permitted only for a deleted aggregate retained because an
  active subscription still depends on a released revision.
- Ownership is derived from authentication, never from a request body.
- The user foreign key uses `ON UPDATE CASCADE` and `ON DELETE RESTRICT`.
  Account deletion must run the explicit cleanup and anonymization transaction
  before deleting the user row.
- A deletion sets `deleted_at`, increments `sync_version`, and assigns a new
  `account_change_version`.
- Ordinary owner queries exclude deleted rows.
- Synchronization queries include tombstones until their retention period
  expires.

### 6.2 `user_pmcs_revisions`

This table stores mutable drafts and immutable publications.

Columns:

- `id`: client-generated `UUID`, primary key, no server default;
- `checklist_id`: `UUID` foreign key to `user_pmcs_checklists`;
- `state`: `TEXT` constrained to `draft`, `published`, or `superseded`;
- `revision_number`: nullable positive `INTEGER`;
- `name`: `TEXT`;
- `description`: `TEXT`;
- `created_at`: server-controlled `TIMESTAMPTZ`;
- `updated_at`: server-controlled `TIMESTAMPTZ`; and
- `published_at`: nullable server-controlled `TIMESTAMPTZ`.

Rules:

- A checklist has at most one draft.
- A checklist has at most one current published revision.
- `draft` requires null `revision_number` and null `published_at`.
- `published` and `superseded` require a positive `revision_number` and
  non-null `published_at`.
- Revision numbers are unique within a checklist.
- The table exposes a unique `checklist_id + id` key for composite ownership
  references.
- Published and superseded rows and their children are immutable through all
  ordinary repository write paths.
- The checklist foreign key uses `ON UPDATE CASCADE` and `ON DELETE CASCADE`.
- Deleting a checklist may cascade into revisions only when no retained
  community release restricts that deletion.

Offline publication already occurs in the implemented app. Therefore the
server must not silently replace a client's published revision number with a
different number. When synchronizing a publication, the server:

1. locks the checklist;
2. verifies the expected `sync_version`;
3. verifies that the submitted number is exactly the next number for that
   checklist;
4. validates the complete revision;
5. accepts the submitted number unchanged; and
6. returns canonical server timestamps.

A stale competing offline publication fails before any renumbering or partial
write occurs.

### 6.3 `user_pmcs_revision_models`

Columns:

- `revision_id`: foreign key to `user_pmcs_revisions`;
- `display_text`: `TEXT`; and
- `normalized_text`: `TEXT`.

The primary key is `revision_id + normalized_text`.

Both text values must be nonblank. The Go service validates normalized values
against the shared model-normalization contract.

### 6.4 `user_pmcs_sections`

Columns:

- `id`: client-generated `UUID`, primary key;
- `revision_id`: foreign key to `user_pmcs_revisions`;
- `position`: positive, one-based `INTEGER`; and
- `title`: `TEXT`.

`revision_id + position` is unique.

Blank titles are allowed in drafts and rejected during publication.

### 6.5 `user_pmcs_section_models`

Columns:

- `section_id`: foreign key to `user_pmcs_sections`;
- `display_text`: `TEXT`; and
- `normalized_text`: `TEXT`.

The primary key is `section_id + normalized_text`.

Zero rows means that the section applies universally. Stored model values must
be nonblank and valid under the shared normalization contract.

### 6.6 `user_pmcs_items`

Columns:

- `id`: client-generated `UUID`, primary key;
- `section_id`: foreign key to `user_pmcs_sections`;
- `position`: positive, one-based `INTEGER`;
- `interval`: `TEXT`;
- `item_to_be_checked_or_serviced`: `TEXT`; and
- `performed_by`: `TEXT` with an empty-string default.

`section_id + position` is unique.

Blank required authored fields are allowed in drafts and rejected during
publication.

### 6.7 `user_pmcs_notices`

Columns:

- `id`: client-generated `UUID`, primary key;
- `item_id`: foreign key to `user_pmcs_items`;
- `position`: positive, one-based `INTEGER`;
- `type`: nullable `TEXT`, constrained when present to `warning`, `caution`,
  or `note`; and
- `notice_text`: `TEXT`.

`item_id + position` is unique.

Nullable type and blank notice text support incomplete draft synchronization.
Publication requires a supported type and nonblank text.

### 6.8 `user_pmcs_procedure_steps`

Columns:

- `id`: client-generated `UUID`, primary key;
- `item_id`: foreign key to `user_pmcs_items`;
- `position`: positive, one-based `INTEGER`;
- `step_text`: `TEXT`; and
- `fault_found_if`: `TEXT` with an empty-string default.

`item_id + position` is unique.

Blank step text is allowed in drafts. Publication requires nonblank step text
and at least one step per item.

### 6.9 Content-tree deletion behavior

Revision-owned model, section, item, notice, and procedure-step foreign keys
use `ON UPDATE CASCADE` and `ON DELETE CASCADE`.

Every foreign-key column not already covered as the leading column of a
primary, unique, or ordered index receives an explicit index. Postgres does not
automatically index referencing foreign keys.

## 7. Community distribution

### 7.1 `user_pmcs_community_releases`

This table records revisions explicitly released to the community. A release
row is immutable after insertion, but may later be deleted by an authorized
retention cleanup when no source or active subscription references it.

Columns:

- `revision_id`: primary key and foreign key to `user_pmcs_revisions`;
- `checklist_id`: foreign key to `user_pmcs_checklists`; and
- `released_at`: server-controlled `TIMESTAMPTZ`.

Rules:

- `checklist_id + revision_id` uniquely references a revision belonging to the
  same checklist.
- The composite revision foreign key uses `ON UPDATE CASCADE` and
  `ON DELETE RESTRICT`.
- Only `published` or `superseded` revisions may be released.
- A draft can never be released.
- A revision can be released at most once.
- The service revalidates the complete revision before release.

### 7.2 `user_pmcs_community_sources`

This table represents the discoverable public source separately from private
authoring state.

Columns:

- `checklist_id`: primary key and foreign key to `user_pmcs_checklists`;
- `status`: `TEXT` constrained to `active`, `retired`, or `suspended`;
- `current_release_revision_id`: nullable `UUID`;
- `first_released_at`: server-controlled `TIMESTAMPTZ`;
- `updated_at`: server-controlled `TIMESTAMPTZ`;
- `retired_at`: nullable `TIMESTAMPTZ`;
- `suspended_at`: nullable `TIMESTAMPTZ`;
- `moderated_by`: nullable `TEXT` foreign key to `users(uid)`; and
- `moderation_reason`: nullable `TEXT`.

`checklist_id + current_release_revision_id`, when non-null, references a
community release belonging to the same checklist.

The checklist and current-release foreign keys use `ON UPDATE CASCADE` and
`ON DELETE RESTRICT`.

State rules:

- `active` requires a current release and null retirement and suspension
  metadata.
- `suspended` retains a current release, requires suspension time and reason,
  and is excluded from public and download queries.
- `retired` has no current release, requires `retired_at`, and is excluded
  from discovery and new installations.
- Administrator reactivation clears suspension metadata.
- Owner unpublishing may later be reversed through a new explicit release while
  the underlying checklist remains owned and undeleted.
- A source retired because its checklist or account was deleted cannot be
  reactivated.

Deleting the administrator user sets `moderated_by` to null without deleting
the moderation outcome.

### 7.3 `user_pmcs_subscriptions`

This table stores one logical installation per user and community source.

Columns:

- `subscriber_uid`: `TEXT` foreign key to `users(uid)`;
- `checklist_id`: `UUID` foreign key to
  `user_pmcs_community_sources(checklist_id)`;
- `installed_revision_id`: `UUID`;
- `sync_version`: positive `BIGINT`;
- `account_change_version`: positive `BIGINT`;
- `created_at`: server-controlled `TIMESTAMPTZ`;
- `updated_at`: server-controlled `TIMESTAMPTZ`; and
- `deleted_at`: nullable server-controlled `TIMESTAMPTZ`.

The primary key is `subscriber_uid + checklist_id`.

`checklist_id + installed_revision_id` references a community release from the
same checklist.

Rules:

- Subscriber identity is derived from authentication.
- Creating or reactivating a subscription requires authentication.
- The subscriber foreign key uses `ON UPDATE CASCADE` and `ON DELETE CASCADE`.
- The community-source and installed-release foreign keys use
  `ON UPDATE CASCADE` and `ON DELETE RESTRICT`.
- An owner cannot subscribe to their own source; owned content is already
  available through private synchronization.
- A new subscription installs the source's current release.
- Accepting an update changes `installed_revision_id` through optimistic
  concurrency.
- Update availability is derived by comparing the installed revision with the
  source's current release.
- Subscribers cannot mutate source content.
- Unsubscribe sets `deleted_at`, increments `sync_version`, and assigns a new
  `account_change_version`.
- Resubscribe is an explicit operation that clears the tombstone and installs
  the then-current release.
- Deleting a subscriber account cascades its subscription rows.

## 8. Access model

### 8.1 Owner

An owner may read all private and released revisions belonging to the
checklist. Only the owner may create, save, publish, release, unpublish, or
delete it.

### 8.2 Public user

An unauthenticated or unaffiliated user may browse only active community
sources and their current releases. Installing a linked subscription requires
authentication.

### 8.3 Subscriber

A subscriber may retrieve:

- the immutable release installed by that active subscription; and
- the active source's current release when checking or accepting an update.

Retired sources remain readable only through existing active subscriptions.

### 8.4 Suspended source

Suspension hides the source and blocks discovery, installation, server
redownload, and updates. It cannot recall data already downloaded to a device.

### 8.5 Administrator

An authenticated administrator may suspend or reactivate a community source.
Internal moderation reasons are not part of the public response contract.

The server uses an application-owned database connection. Postgres row-level
security is not part of this design; authorization remains in the
service/repository layer and must be proven with integration tests.

## 9. Write transactions

The checklist row is the serialization point for all checklist aggregate
mutations. Transactions acquire locks in this order:

1. authenticated user's `user_pmcs_sync_state` row;
2. checklist or subscription aggregate row; and
3. dependent rows required by the operation.

Using one lock order across repositories prevents avoidable deadlocks.

### 9.1 New checklist upload

A remotely synchronized checklist is created only for an authenticated user.
The service:

1. derives the owner from authentication;
2. verifies the submitted checklist UUID does not already exist;
3. validates the complete submitted aggregate;
4. rejects a UUID owned or tombstoned by any other aggregate;
5. creates the checklist and revision tree atomically; and
6. returns canonical server versions, account change version, and timestamps.

### 9.2 Draft save

A draft save submits the complete draft tree and expected checklist
`sync_version`.

The transaction:

1. locks and authorizes the checklist;
2. rejects a deleted checklist;
3. compares the expected version;
4. verifies the target is the checklist's current draft;
5. validates UUID uniqueness, parentage, positions, normalization, and text
   limits;
6. replaces the complete draft child tree with batched operations;
7. updates server timestamps;
8. increments `sync_version`;
9. assigns a new `account_change_version`; and
10. commits atomically.

Published and superseded content is never replaced through this path.

### 9.3 Publication synchronization

Publication validates the complete draft tree and submitted next revision
number while holding the checklist lock.

On success:

1. the previous current publication becomes `superseded`;
2. the submitted draft becomes `published`;
3. the submitted next revision number is accepted unchanged;
4. canonical server publication and update timestamps are assigned;
5. `sync_version` is incremented; and
6. a new `account_change_version` is assigned.

Failure leaves the previous publication and draft unchanged.

### 9.4 Community release

Release to community:

1. locks and authorizes the checklist;
2. rejects deleted or unowned checklists;
3. verifies the revision belongs to the checklist;
4. verifies it is published or superseded;
5. revalidates the immutable content tree;
6. inserts the release record if it does not already exist;
7. creates or reactivates the community source;
8. points the source to the released revision;
9. increments checklist `sync_version`;
10. assigns a new checklist `account_change_version`; and
11. commits atomically.

Releasing a newer revision does not change any existing subscription.

### 9.5 Retirement

Unpublishing:

- changes the source to `retired`;
- clears its current release;
- blocks discovery and new installs;
- retains subscription-pinned releases; and
- leaves the owned private checklist active.

Checklist deletion additionally tombstones the owned checklist and blocks
future reactivation.

## 10. Validation boundary

### 10.1 Database-enforced rules

Postgres enforces:

- primary and foreign-key relationships;
- UUID uniqueness;
- valid revision, source, and notice-type values;
- positive positions;
- unique sibling positions;
- unique normalized model values within a parent;
- one draft per checklist;
- one current publication per checklist;
- unique revision numbers per checklist;
- revision and release ownership consistency;
- subscription and installed-release consistency;
- state-dependent nullability that can be represented through row checks; and
- reasonable byte ceilings selected as abuse-protection limits.

### 10.2 Service-enforced publication rules

Drafts intentionally permit incomplete authored content, so publication
revalidates:

- nonblank checklist name;
- at least one valid checklist model;
- at least one section;
- nonblank title and at least one item in every section;
- nonblank interval and item text;
- at least one procedure step per item;
- nonblank procedure-step text;
- supported notice type and nonblank notice text;
- valid display and normalized model pairs;
- unique and contiguous sibling positions;
- exact user-facing grapheme limits; and
- complete-tree parentage and UUID ownership.

The server never trusts prior mobile validation.

### 10.3 Text and Unicode contract

The implemented app uses:

- 200 graphemes for short fields; and
- 4,000 graphemes for long fields.

Postgres `char_length` counts code points rather than user-perceived graphemes.
Authored columns therefore use `TEXT`, not `VARCHAR(n)`.

The Go service must use Unicode grapheme segmentation compatible with the
Flutter `characters` package. Postgres additionally enforces these
abuse-protection ceilings with `octet_length` checks:

- 8 KiB for short fields; and
- 64 KiB for long fields.

Short fields are checklist names, model display and normalized values, section
titles, item intervals, and performed-by values. Long fields are descriptions,
item text, notice text, procedure-step text, fault-found-if text, and
moderation reasons.

These byte ceilings are intentionally higher than ordinary valid content but
remain separate from the user-facing grapheme limits. Pathological combining
sequences can exceed a byte ceiling while remaining under a grapheme limit;
rejecting such payloads is an intentional resource-protection trade-off.

The HTTP layer must also enforce a complete payload-size limit. Aggregate node
count and HTTP payload limits belong to the future API contract rather than to
relational check constraints.

The current model-normalization authority is:

1. trim leading and trailing whitespace;
2. collapse every run of whitespace to one ASCII space; and
3. lowercase the result.

Flutter and Go require shared Unicode and whitespace fixtures before remote
sync ships. Postgres stores and indexes the validated normalized value; it does
not independently derive it with `lower()`.

## 11. Index design

### 11.1 Owner sync and listing

`user_pmcs_checklists`:

- primary key on `id`;
- index on `owner_uid + account_change_version`; and
- partial index on `owner_uid + updated_at DESC + id` where `deleted_at` is
  null.

### 11.2 Revisions

`user_pmcs_revisions`:

- partial unique index on `checklist_id` where state is `draft`;
- partial unique index on `checklist_id` where state is `published`;
- partial unique index on `checklist_id + revision_number` where the number is
  non-null;
- index on `checklist_id + state`; and
- unique key on `checklist_id + id`.

### 11.3 Ordered content

- sections: unique `revision_id + position`;
- items: unique `section_id + position`;
- notices: unique `item_id + position`; and
- procedure steps: unique `item_id + position`.

These indexes enforce sibling uniqueness and cover ordered child reads.

### 11.4 Model discovery

- revision models: primary key `revision_id + normalized_text`;
- revision models: additional `normalized_text + revision_id` index; and
- section models: primary key `section_id + normalized_text`.

The reverse revision-model index supports public discovery beginning with a
normalized equipment model.

### 11.5 Community and subscriptions

Community sources:

- primary key on `checklist_id`;
- partial index on `updated_at DESC + checklist_id` where status is `active`;
  and
- administrator index on `status + updated_at`.

Community releases:

- primary key on `revision_id`;
- unique key on `checklist_id + revision_id`; and
- index on `checklist_id + released_at DESC`.

Subscriptions:

- primary key on `subscriber_uid + checklist_id`;
- index on `subscriber_uid + account_change_version`; and
- partial index on `installed_revision_id` where `deleted_at` is null.

The installed-revision index supports retention checks.

## 12. Synchronization reads

### 12.1 Unified account delta

The account delta combines:

- checklist rows whose `owner_uid` matches the authenticated user; and
- subscription rows whose `subscriber_uid` matches the authenticated user.

Both branches filter on `account_change_version` above the stored cursor and
are merged in version order before pagination. Responses include active rows
and tombstones.

Using one ordered result is required. Querying and paginating the two tables
independently while advancing one shared cursor could skip an older unprocessed
row from the other table.

For each changed active checklist, the client retrieves the current revision
tree required to reconcile local state. An active subscription identifies the
exact immutable release that must exist locally.

Community update availability is a separate read over the active subscriptions
and their source current releases. Releasing a new revision does not fan out a
write to every subscription.

### 12.2 Cursor advancement

The client advances its cursor only after durably applying the full ordered
page. It stores the highest returned `account_change_version`.

## 13. Deletion and retention

### 13.1 Private-only checklist

Deletion:

- tombstones the checklist;
- advances both version fields;
- removes its draft and revision content; and
- retains the root tombstone for the configured offline-device retention
  period.

After the retention period, the tombstone may be physically deleted.

### 13.2 Released checklist

Deletion:

- tombstones the checklist;
- retires the community source;
- clears the current release;
- removes drafts and unreleased content; and
- retains community release records and revision trees required by active
  subscriptions.

Unpinned release rows and content become eligible for cleanup under the
retention policy.

### 13.3 Account deletion

Before the `users` row is deleted, one transaction:

1. deletes private-only checklists and content;
2. tombstones formerly public checklist aggregates;
3. retires their community sources;
4. clears current releases;
5. deletes drafts and unreleased revisions;
6. retains active-subscriber-pinned releases;
7. sets retained checklist ownership to null; and
8. removes creator attribution.

If no active subscription remains, the retired source, release records,
revision content, and checklist tombstone may be purged according to policy.

### 13.4 Subscription

Unsubscribe creates a versioned tombstone. After its offline-device retention
period, the subscription row may be removed. Once no active subscription or
active source references an old release, that release is eligible for cleanup.

Retention durations are deployment policy, not database constraints.

## 14. Existing local data

Existing custom checklists were created without account ownership and remain
device-local permanently.

They are not:

- uploaded automatically;
- assigned to the first user who signs in;
- represented as unowned server rows; or
- included in a server migration backfill.

A future explicit Copy to my account action is a new aggregate creation. It
generates new UUIDs for the checklist, revision, sections, items, notices, and
steps. The server stores no legacy lineage pointer unless a separate future
product requirement justifies it.

## 15. Migration and deployment sequence

The design intentionally does not reserve a migration number. The implementer
must inspect the target server branch because migration 008 work is currently
unfinished.

Recommended delivery order:

1. Create the per-user sync-state table.
2. Create checklist and revision tables.
3. Create normalized revision-content tables.
4. Create community release records.
5. Create community source metadata.
6. Create subscriptions.
7. Add check constraints, foreign keys, and access-pattern indexes.
8. Apply the forward migration to a disposable database.
9. Regenerate Jet models from that canonical schema.
10. Verify rollback and reapplication.
11. Deploy the additive schema before new APIs.
12. Deploy private synchronization APIs.
13. Deploy community release, discovery, moderation, and subscription APIs.
14. Deploy mobile synchronization only after the server contract is stable.

Generated Jet files under `.gen` must be regenerated from Postgres and never
edited manually.

## 16. Required verification

The future implementation plan must prove:

- one draft and one current publication per checklist;
- valid and unique revision numbers;
- client publication numbers are accepted only when they are exactly next;
- every cascade and restriction behaves as designed;
- sibling positions are positive and unique;
- complete publication validation rejects incomplete drafts;
- UUIDs cannot be grafted across checklists, revisions, or owners;
- stale draft and publication writes are rejected;
- concurrent publications cannot both succeed;
- per-user change versions commit in serialization order;
- unified delta pagination cannot skip checklist or subscription rows;
- deletion defeats a later stale upload;
- public release requires ownership and a valid immutable revision;
- public release does not move subscriber installations;
- anonymous users can browse but cannot create subscriptions;
- installed revisions belong to the subscribed source;
- retirement preserves subscriber-pinned revisions;
- suspension removes discovery and server retrieval;
- account deletion removes private content and anonymizes retained releases;
- owner and subscription tombstones appear in delta synchronization;
- public discovery queries never return drafts, unreleased revisions, retired
  sources, or suspended sources;
- forward migration, rollback, and reapplication succeed; and
- Jet regeneration matches the migrated schema.

## 17. Explicitly deferred work

This design does not add:

- direct user sharing;
- shop sharing;
- anonymous community installations;
- collaborative editing;
- automatic tree merges;
- editable forks or lineage tracking;
- ratings, reviews, comments, or reports;
- community search beyond fields and models already present;
- images or attachments;
- custom checklist inspection synchronization;
- automatic publication from the local Publish action;
- remote conversion of existing account-free checklists; or
- a migration-number commitment.
