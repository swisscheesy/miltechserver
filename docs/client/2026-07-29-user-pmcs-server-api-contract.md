# User-Created PMCS Server API Contract

**Published:** 2026-07-30

**Server base path:** `/api/v1`

**Audience:** mobile sync and community-library clients

**Implementation baseline:** `ffe7b8295dfc7633335adceb209f0da460314217`

## Scope

This document describes the implemented JSON and HTTP contract for private
user-created PMCS synchronization, public community releases, and linked
read-only installations.

The server does not store custom PMCS execution or progress. This API has no
inspection, equipment-selection, procedure-completion, fault, note, export, or
inspection-history fields or routes.

## Common HTTP rules

### Authentication and identity

The two public community routes require no authentication. Every route under
`/api/v1/auth` requires a verified Firebase identity and derives the owner or
subscriber UID from that identity. Clients never send an owner UID,
subscriber UID, email, or Firebase claim in a path, query, or request body.

An authenticated Firebase identity without a matching Postgres `users` row
receives:

```json
{
  "status": 409,
  "message": "account is not initialized",
  "data": null,
  "error": {
    "code": "account_not_initialized"
  }
}
```

### JSON success envelope

Every `200` or `201` JSON response uses:

```json
{
  "status": 200,
  "data": {},
  "message": ""
}
```

The `data` shape is route-specific. Deletes return a JSON representation with
`200`; they do not return `204`.

### JSON error envelope

Errors use:

```json
{
  "status": 412,
  "message": "resource changed",
  "data": null,
  "error": {
    "code": "stale_precondition"
  }
}
```

When present, `error.details` contains only safe structured values such as a
field name or configured limit:

```json
{
  "status": 422,
  "message": "publication revision_number must be positive",
  "data": null,
  "error": {
    "code": "validation_failed",
    "details": {
      "field": "revision.revision_number"
    }
  }
}
```

Stable codes are:

| Status | Code | Meaning |
|---:|---|---|
| `400` | `invalid_request` | Invalid UUID, cursor, query, content type, JSON, unknown JSON field, trailing JSON, compressed body, or invalid UTF-8 |
| `400` | `invalid_precondition` | A conditional header is malformed, unsupported, or conflicts with another conditional header |
| `401` | `authentication_required` | Authentication is missing or invalid |
| `403` | `forbidden` | The operation is forbidden and existence can safely be revealed |
| `404` | `resource_not_found` | The resource is absent or intentionally hidden |
| `409` | `account_not_initialized` | The verified Firebase UID has no Postgres user |
| `409` | `invalid_transition` | The current resource cannot make the requested domain transition |
| `412` | `stale_precondition` | A valid conditional does not match current state |
| `413` | `content_too_large` | A request, node count, account count, or aggregate exceeds its configured ceiling |
| `422` | `validation_failed` | Authored fields or a publication tree fail validation |
| `428` | `precondition_required` | A required conditional header is absent |
| `429` | `rate_limited` | A configured request limiter rejected the request |
| `500` | `internal_error` | An unexpected server failure occurred |

Private cross-owner, unknown, and intentionally hidden resources use the same
safe `404 resource_not_found` envelope.

### JSON request rules

The three body-bearing routes accept only an uncompressed
`Content-Type: application/json` request. The uncompressed body ceiling is
8,388,608 bytes. Unknown fields, a second JSON value, malformed JSON, and
invalid UTF-8 are rejected. The other mutation routes have no request JSON.

UUIDs are nonzero canonical UUID strings. Server timestamps are RFC 3339
values.

### ETags and conditionals

ETags are opaque quoted values. Clients store and replay them; they do not
construct them from `sync_version`.

- Creating a new client-ID resource uses `If-None-Match: *`.
- Mutating an existing checklist or subscription uses exactly one strong
  `If-Match` value.
- Draft, publication, community-release, community-retirement, and checklist
  deletion mutations all use the parent checklist ETag.
- Subscription unsubscribe, resubscribe, and update acceptance use the
  subscription ETag.
- Supplying both `If-None-Match` and `If-Match` to subscription installation is
  invalid.
- A proven idempotent retry returns `200`, the canonical current data, and the
  current ETag without consuming another version.

`GET` conditional behavior differs by implemented handler:

- owned current and owned historical GETs accept either no `If-None-Match` or
  exactly one strong ETag;
- public current-release and pinned-release GETs implement weak comparison and
  accept `*`, a weak validator, a comma-separated list, or repeated header
  fields;
- a match returns `304` with no body.

The sync and listing routes are cursor-driven and do not use ETags.

## Revision request JSON

Checklist creation and draft replacement use the same strict revision body.
`revision_number` must be absent or `null` for a draft.

```json
{
  "id": "10000000-0000-4000-8000-000000000001",
  "revision_number": null,
  "name": "M998 Preventive Maintenance",
  "description": "Operator-authored checklist",
  "models": [
    {
      "display_text": "M998 HMMWV"
    }
  ],
  "sections": [
    {
      "id": "20000000-0000-4000-8000-000000000001",
      "position": 1,
      "title": "Before operation",
      "models": [
        {
          "display_text": "M998"
        }
      ],
      "items": [
        {
          "id": "30000000-0000-4000-8000-000000000001",
          "position": 1,
          "interval": "Before",
          "item_to_be_checked_or_serviced": "Engine compartment",
          "performed_by": "Operator",
          "notices": [
            {
              "id": "40000000-0000-4000-8000-000000000001",
              "position": 1,
              "type": "warning",
              "notice_text": "Use caution"
            }
          ],
          "procedure_steps": [
            {
              "id": "50000000-0000-4000-8000-000000000001",
              "position": 1,
              "step_text": "Inspect for leaks",
              "fault_found_if": "Any leak is present"
            }
          ]
        }
      ]
    }
  ]
}
```

Publication uses the same body with a positive exact-next number:

```json
{
  "id": "10000000-0000-4000-8000-000000000001",
  "revision_number": 1,
  "name": "M998 Preventive Maintenance",
  "description": "Operator-authored checklist",
  "models": [
    {
      "display_text": "M998 HMMWV"
    }
  ],
  "sections": [
    {
      "id": "20000000-0000-4000-8000-000000000001",
      "position": 1,
      "title": "Before operation",
      "models": [],
      "items": [
        {
          "id": "30000000-0000-4000-8000-000000000001",
          "position": 1,
          "interval": "Before",
          "item_to_be_checked_or_serviced": "Engine compartment",
          "performed_by": "Operator",
          "notices": [],
          "procedure_steps": [
            {
              "id": "50000000-0000-4000-8000-000000000001",
              "position": 1,
              "step_text": "Inspect for leaks",
              "fault_found_if": "Any leak is present"
            }
          ]
        }
      ]
    }
  ]
}
```

The body `id` must equal the route `revision_id`. Clients send only
`display_text` for models. The response adds server-derived
`normalized_text`.

Drafts may be incomplete. Publication revalidates nonblank required content,
positions, notice types, at least one procedure step per item, the exact next
revision number, UUID ownership, field limits, and node limits.

## Response shapes

### Revision and tree

Current draft/publication revisions use:

```json
{
  "id": "10000000-0000-4000-8000-000000000001",
  "revision_number": 1,
  "name": "M998 Preventive Maintenance",
  "description": "Operator-authored checklist",
  "models": [
    {
      "display_text": "M998 HMMWV",
      "normalized_text": "m998 hmmwv"
    }
  ],
  "sections": [],
  "state": "published",
  "created_at": "2026-07-30T12:00:00Z",
  "updated_at": "2026-07-30T12:00:00Z",
  "published_at": "2026-07-30T12:00:00Z"
}
```

`revision_number` and `published_at` are omitted when absent. Nested section,
item, notice, and procedure-step response fields match the request names,
except model objects also contain `normalized_text`.

Owned historical revision responses deliberately omit mutable `state`:

```json
{
  "id": "10000000-0000-4000-8000-000000000001",
  "revision_number": 1,
  "name": "M998 Preventive Maintenance",
  "description": "Operator-authored checklist",
  "models": [],
  "sections": [],
  "created_at": "2026-07-30T12:00:00Z",
  "updated_at": "2026-07-30T12:00:00Z",
  "published_at": "2026-07-30T12:00:00Z"
}
```

### Owned checklist aggregate

An active aggregate is:

```json
{
  "id": "60000000-0000-4000-8000-000000000001",
  "sync_version": 7,
  "account_change_version": 15,
  "created_at": "2026-07-30T11:00:00Z",
  "updated_at": "2026-07-30T12:00:00Z",
  "draft": {},
  "publication": {},
  "community": {
    "status": "active",
    "current_release_revision_id": "10000000-0000-4000-8000-000000000001",
    "latest_release_revision_number": 1,
    "first_released_at": "2026-07-30T12:00:00Z",
    "updated_at": "2026-07-30T12:00:00Z"
  }
}
```

`draft`, `publication`, and `community` are independently omitted when absent.
An owned tombstone contains no authored tree:

```json
{
  "id": "60000000-0000-4000-8000-000000000001",
  "sync_version": 8,
  "account_change_version": 16,
  "created_at": "2026-07-30T11:00:00Z",
  "updated_at": "2026-07-30T12:30:00Z",
  "deleted_at": "2026-07-30T12:30:00Z"
}
```

### Subscription and installed release

The tagged subscription object is:

```json
{
  "checklist_id": "60000000-0000-4000-8000-000000000001",
  "installed_revision_id": "10000000-0000-4000-8000-000000000001",
  "sync_version": 2,
  "account_change_version": 19,
  "created_at": "2026-07-30T12:00:00Z",
  "updated_at": "2026-07-30T12:10:00Z"
}
```

An unsubscribe tombstone omits `installed_revision_id` and adds `deleted_at`.

An installed immutable release is:

```json
{
  "checklist_id": "60000000-0000-4000-8000-000000000001",
  "source_status": "active",
  "creator_display_name": "Maintainer",
  "released_at": "2026-07-30T12:00:00Z",
  "revision": {}
}
```

The implemented install and accept-update handlers expose an untagged Go
result wrapper. Its exact case-sensitive JSON keys are:

```json
{
  "Subscription": {
    "checklist_id": "60000000-0000-4000-8000-000000000001",
    "installed_revision_id": "10000000-0000-4000-8000-000000000001",
    "sync_version": 1,
    "account_change_version": 18,
    "created_at": "2026-07-30T12:00:00Z",
    "updated_at": "2026-07-30T12:00:00Z"
  },
  "Installed": {
    "checklist_id": "60000000-0000-4000-8000-000000000001",
    "source_status": "active",
    "creator_display_name": "Maintainer",
    "released_at": "2026-07-30T12:00:00Z",
    "revision": {}
  },
  "Created": true,
  "Idempotent": false
}
```

These four wrapper keys are uppercase. `Installed` may be `null`. In contrast,
the unsubscribe route returns the tagged subscription object directly.

## Route contract

### 1. Account delta

`GET /api/v1/auth/user-pmcs/sync`

- Auth: required.
- Query: `after` is a decimal nonnegative account version, default `0`;
  `limit` is `1..25`, default `10`.
- Body: none.
- Conditional: none.
- Response: `200`; gzip is supported and `Vary: Accept-Encoding` is set.

```json
{
  "status": 200,
  "message": "",
  "data": {
    "from_cursor": 41,
    "through_cursor": 46,
    "account_version": 49,
    "has_more": true,
    "changes": [
      {
        "account_change_version": 46,
        "kind": "checklist",
        "checklist": {}
      }
    ]
  }
}
```

`kind` is `checklist` or `subscription`. A checklist change has `checklist`.
A subscription change has `subscription` and, only when active, `installed`.
Tombstones omit authored/installed content. Changes are ordered by
`account_change_version`. Multiple mutations to one root between pulls
collapse to its latest complete state, so version gaps are valid.

The server never splits an aggregate. It stops before 20 MiB of uncompressed
canonical envelope JSON, except that one otherwise-valid aggregate may occupy
a page alone. `through_cursor` is the last returned change version, or
`after` for an empty page. `account_version` is the snapshot version.

### 2. Current owned checklist

`GET /api/v1/auth/user-pmcs/checklists/{checklist_id}`

- Auth: owner required; other owners receive safe `404`.
- Path: nonzero UUID `checklist_id`.
- Body/query: none.
- Optional header: one strong `If-None-Match`.
- Response: `200` owned aggregate or `304`; `ETag`,
  `Cache-Control: private, no-cache`; gzip supported.

### 3. Create checklist and initial draft

`PUT /api/v1/auth/user-pmcs/checklists/{checklist_id}`

- Auth: required; authenticated account must exist.
- Path: nonzero client-generated checklist UUID.
- Header: `If-None-Match: *`.
- Body: strict draft revision JSON; `revision_number` absent or `null`.
- Response: `201` when created, `200` for a proven identical retry; owned
  aggregate, checklist `ETag`, `Cache-Control: private, no-cache`.
- Common errors: `400`, `409 account_not_initialized`,
  `412 stale_precondition`, `413`, `422`, `428`, `429`.

### 4. Create or replace current draft

`PUT /api/v1/auth/user-pmcs/checklists/{checklist_id}/drafts/{revision_id}`

- Auth: owner required.
- Paths: nonzero UUIDs; body `id` must equal `revision_id`.
- Header: parent checklist `If-Match`.
- Body: strict draft revision JSON; `revision_number` absent or `null`.
- Response: `200` owned aggregate plus new checklist ETag.

This is complete replacement, not a patch.

### 5. Discard current draft

`DELETE /api/v1/auth/user-pmcs/checklists/{checklist_id}/drafts/{revision_id}`

- Auth: owner required.
- Paths: nonzero UUIDs.
- Header: parent checklist `If-Match`.
- Body/query: none.
- Response: `200` owned aggregate plus new checklist ETag.
- Transition: a publication must remain; an invalid discard is
  `409 invalid_transition`.

### 6. Publish complete revision

`PUT /api/v1/auth/user-pmcs/checklists/{checklist_id}/publications/{revision_id}`

- Auth: owner required.
- Paths: nonzero UUIDs; body `id` must equal `revision_id`.
- Header: parent checklist `If-Match`.
- Body: strict complete revision JSON with a positive exact-next
  `revision_number`.
- Response: `200` owned aggregate plus new checklist ETag.

The complete tree is validated and committed atomically. A draft upload is not
required first.

### 7. Owned immutable historical revision

`GET /api/v1/auth/user-pmcs/checklists/{checklist_id}/revisions/{revision_id}`

- Auth: owner required.
- Paths: nonzero UUIDs.
- Optional header: one strong `If-None-Match`.
- Response: `200` historical revision without `state`, or `304`; immutable
  content ETag; `Cache-Control: private, max-age=31536000, immutable`; gzip
  supported.

### 8. Delete owned checklist

`DELETE /api/v1/auth/user-pmcs/checklists/{checklist_id}`

- Auth: owner required.
- Path: nonzero UUID.
- Header: parent checklist `If-Match`.
- Body/query: none.
- Response: `200` content-free owned tombstone, final checklist ETag,
  `Cache-Control: private, max-age=31536000, immutable`.

Deletion wins permanently over stale create, draft, publication, release, and
retirement requests for that checklist UUID.

### 9. Release immutable revision to community

`PUT /api/v1/auth/user-pmcs/checklists/{checklist_id}/community-releases/{revision_id}`

- Auth: owner required.
- Paths: nonzero UUIDs.
- Header: parent checklist `If-Match`.
- Body/query: none.
- Response: `200` owned aggregate and new checklist ETag;
  `Cache-Control: private, no-cache`.

The revision must be published or superseded and must be strictly higher than
the source's highest prior release. Repeating the same current release can be
idempotent. Release rollback is `409 invalid_transition`.

### 10. Retire community source

`DELETE /api/v1/auth/user-pmcs/checklists/{checklist_id}/community-source`

- Auth: owner required.
- Path: nonzero UUID.
- Header: parent checklist `If-Match`.
- Body/query: none.
- Response: `200` owned aggregate and new checklist ETag.

Retirement hides the source from public browse/detail and clears its current
release. Existing active subscribers remain pinned. A voluntarily retired
owned source can be released again only at a strictly higher revision. A
tombstoned or owner-deleted source cannot reactivate.

### 11. Browse public community

`GET /api/v1/user-pmcs/community`

- Auth: none.
- Query: optional opaque `after`; optional `limit` `1..50`, default `20`;
  optional `model`, normalized by the server and matched exactly.
- Body/conditional: none.
- Response: `200`; `Cache-Control: public, no-cache`;
  `Vary: Accept-Encoding`; gzip supported.

```json
{
  "status": 200,
  "message": "",
  "data": {
    "next_cursor": "opaque-value",
    "has_more": true,
    "items": [
      {
        "checklist_id": "60000000-0000-4000-8000-000000000001",
        "revision_id": "10000000-0000-4000-8000-000000000001",
        "revision_number": 1,
        "name": "M998 Preventive Maintenance",
        "description": "Operator-authored checklist",
        "models": [],
        "creator_display_name": "Maintainer",
        "released_at": "2026-07-30T12:00:00Z",
        "updated_at": "2026-07-30T12:00:00Z"
      }
    ]
  }
}
```

`next_cursor` is omitted when absent. The opaque cursor internally anchors
version `1`, `updated_at`, and `checklist_id`; clients must not decode or
construct it. Active sources sort by `updated_at DESC`, then checklist UUID.
Because this is a mutable recency feed, a concurrent release can move an item
ahead of an in-progress cursor; restart at the first page to refresh.

Public output contains current `creator_display_name`, never UID or email.
After retained-owner account deletion it is `"Deleted user"`.

### 12. Get current public release

`GET /api/v1/user-pmcs/community/{checklist_id}`

- Auth: none.
- Path: nonzero UUID.
- Optional header: `If-None-Match` using weak comparison.
- Response: `200` public checklist release or `304`; public immutable-content
  ETag; `Cache-Control: public, no-cache`; `Vary: Accept-Encoding`; gzip
  supported.

Retired, deleted, never-released, and superseded-unreleased resources return
safe `404`.

### 13. Install or resubscribe

`PUT /api/v1/auth/user-pmcs/subscriptions/{checklist_id}`

- Auth: required; the owner cannot subscribe to their own source.
- Path: nonzero source checklist UUID.
- Body/query: none.
- New installation header: `If-None-Match: *`.
- Explicit resubscription header: current tombstone `If-Match`.
- Response: `201` for a new row; `200` for resubscription or a proven retry;
  the uppercase-key mutation wrapper, subscription ETag,
  `Cache-Control: private, no-cache`.

A new install or resubscribe pins the source's current release. A retired
source cannot be newly installed. A create-style attempt against a retained
subscription tombstone returns `412`; resubscription must mutate that
tombstone with its ETag.

### 14. Unsubscribe

`DELETE /api/v1/auth/user-pmcs/subscriptions/{checklist_id}`

- Auth: active subscriber required.
- Path: nonzero source checklist UUID.
- Header: subscription `If-Match`.
- Body/query: none.
- Response: `200` tagged snake_case subscription tombstone directly in
  `data`, subscription ETag, `Cache-Control: private, no-cache`.

The tombstone clears `installed_revision_id` and remains while the subscriber
account exists. A repeated delete with the current tombstone ETag is
idempotent. A tombstoned subscription cannot read its former pin or accept an
update.

When this is the final pin for an already owner-null, checklist-tombstoned,
retired source, unsubscribe also removes the otherwise unreachable retained
release/tree. It does not reclaim active or owned checklist history.

### 15. Discover subscription updates

`GET /api/v1/auth/user-pmcs/subscriptions/updates`

- Auth: required.
- Query: optional opaque `after`; optional `limit` `1..100`, default `50`.
- Body/conditional: none.
- Response: `200`; `Cache-Control: private, no-cache`; gzip supported.

```json
{
  "status": 200,
  "message": "",
  "data": {
    "next_cursor": "opaque-value",
    "has_more": false,
    "items": [
      {
        "checklist_id": "60000000-0000-4000-8000-000000000001",
        "source_status": "active",
        "installed_revision_id": "10000000-0000-4000-8000-000000000001",
        "installed_revision_number": 1,
        "current_release_revision_id": "10000000-0000-4000-8000-000000000002",
        "current_release_revision_number": 2,
        "update_available": true
      }
    ]
  }
}
```

The opaque cursor internally anchors version `1` and `checklist_id`; clients
must not decode or construct it. Pages use stable ascending checklist UUID
order. Retired sources have `source_status: "retired"`, omit both current
release fields, and report `update_available: false`. Tombstoned subscriptions
are omitted. This read does not mutate versions or fan out release writes.

### 16. Accept current higher release

`PUT /api/v1/auth/user-pmcs/subscriptions/{checklist_id}/installed-releases/{revision_id}`

- Auth: active subscriber required.
- Paths: nonzero UUIDs.
- Header: subscription `If-Match`.
- Body/query: none.
- Response: `200` uppercase-key mutation wrapper, new subscription ETag,
  `Cache-Control: private, no-cache`.

The target must be the source's current active release and must advance the
installed revision. A missing or deleted subscription returns safe `404`
before source transition state is evaluated. An existing subscription whose
source cannot transition returns `409 invalid_transition`.

### 17. Redownload exact pinned release

`GET /api/v1/auth/user-pmcs/subscriptions/{checklist_id}/installed-releases/{revision_id}`

- Auth: active subscriber required.
- Paths: nonzero UUIDs. `revision_id` must equal the subscription's exact
  installed revision.
- Optional header: `If-None-Match` using weak comparison.
- Response: `200` installed checklist release or `304`; representation ETag,
  `Cache-Control: private, no-cache`; gzip supported.

The exact pin remains readable after source retirement. It does not provide
arbitrary historical-release access.

## Offline first synchronization

For a local checklist with publication history that has never synchronized:

1. Create the checklist with revision 1's complete tree as the initial draft,
   using its client-generated checklist/revision/content UUIDs and
   `If-None-Match: *`.
2. Store the returned parent checklist ETag.
3. Publish revision 1 with that ETag and the positive
   `revision_number: 1`.
4. For each later local publication in ascending revision-number order, upload
   its complete tree as the current draft with the latest parent ETag, then
   publish that same UUID/tree as the exact next revision.
5. After the local publication prefix is reconstructed, upload the current
   incomplete local draft, when present.
6. Fetch the current owned aggregate and then pull account delta pages.

After interruption, fetch the current owned aggregate, identify the next
missing revision, and continue. Identical proven retries return canonical
success. UUID, content, or revision-number divergence returns `412` or `409`
and requires reconciliation; the server does not renumber history.

For ongoing delta sync, apply every page and persist `through_cursor` in one
durable local transaction. Advance the cursor only after that transaction
commits. Continue while `has_more` is true.

## Installed release and update behavior

An installation is a linked read-only subscription, not a copied editable
checklist. It stays pinned to one immutable release until the subscriber
explicitly accepts the current higher release. Community release does not
change subscriber rows.

Clients should:

1. use update discovery for lightweight availability;
2. display `source_status`, current and installed revision numbers, and
   `update_available`;
3. send the current subscription ETag when accepting an update;
4. replace local installed content only after the canonical success response
   is durably stored; and
5. retain pinned content when the source is retired.

Unsubscribe retains a durable linked-installation tombstone for that account.
Resubscribe is explicit and installs the then-current active release.

## Implemented v1 limits

- Active owned checklists per account: 250.
- Active subscriptions per account: 500.
- Checklist models: 100.
- Sections: 100.
- Section models: 100 per section, 1,000 total.
- Items: 500 per section, 2,000 total.
- Notices: 100 per item, 4,000 total.
- Procedure steps: 250 per item, 10,000 total.
- Short fields: 200 Unicode extended grapheme clusters and 8 KiB.
- Long fields: 4,000 Unicode extended grapheme clusters and 64 KiB.
- Mutation body: 8 MiB uncompressed.
- Delta page: 10 roots by default, 25 maximum, and a 20 MiB soft envelope
  boundary without aggregate splitting.

There is no moderation, rollback release, direct sharing, editable fork,
server-side inspection execution/progress, generic idempotency key, or mobile
remote-sync implementation in this server contract.
