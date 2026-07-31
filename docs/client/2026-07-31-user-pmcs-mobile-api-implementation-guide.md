# User-Created PMCS Mobile API Implementation Guide

**Published:** 2026-07-31

**Server base path:** `/api/v1`

**Audience:** mobile engineers implementing private synchronization, community browsing, and linked community installations

**Verified server baseline:** `517c7c1fd95edc4bd22ed17b4056a8fc69958c92`

## Purpose and scope

This is the mobile team's implementation guide for every server endpoint in the
user-created PMCS feature. It explains when the client should call each route,
why that route exists, what to send, what to persist, and how to react to the
response. The route-by-route contract is followed by complete offline and
community workflows so the endpoints are not implemented as isolated calls.

The feature synchronizes checklist definitions and immutable publication
history. It does not synchronize checklist execution. Inspection progress,
equipment selections, completed steps, faults, notes, exports, and inspection
history remain local or belong to other APIs and must never be added to these
payloads.

## Endpoint map

| # | Method and path | Authentication | Mobile purpose |
|---:|---|---|---|
| 1 | `GET /api/v1/auth/user-pmcs/sync` | Required | Pull all owned-checklist and subscription changes after the local account cursor. |
| 2 | `GET /api/v1/auth/user-pmcs/checklists/{checklist_id}` | Owner | Fetch or reconcile one complete owned checklist aggregate. |
| 3 | `PUT /api/v1/auth/user-pmcs/checklists/{checklist_id}` | Required | Create a server checklist with its initial draft. |
| 4 | `PUT /api/v1/auth/user-pmcs/checklists/{checklist_id}/drafts/{revision_id}` | Owner | Create or completely replace the current draft. |
| 5 | `DELETE /api/v1/auth/user-pmcs/checklists/{checklist_id}/drafts/{revision_id}` | Owner | Discard a draft while preserving published history. |
| 6 | `PUT /api/v1/auth/user-pmcs/checklists/{checklist_id}/publications/{revision_id}` | Owner | Atomically publish the exact next immutable revision. |
| 7 | `GET /api/v1/auth/user-pmcs/checklists/{checklist_id}/revisions/{revision_id}` | Owner | Fetch one immutable historical publication. |
| 8 | `DELETE /api/v1/auth/user-pmcs/checklists/{checklist_id}` | Owner | Permanently delete an owned checklist and retain a lightweight tombstone. |
| 9 | `PUT /api/v1/auth/user-pmcs/checklists/{checklist_id}/community-releases/{revision_id}` | Owner | Make an immutable publication the current public release. |
| 10 | `DELETE /api/v1/auth/user-pmcs/checklists/{checklist_id}/community-source` | Owner | Retire the checklist from new community discovery and installation. |
| 11 | `GET /api/v1/user-pmcs/community` | Public | Browse recent active community releases, optionally filtered by model. |
| 12 | `GET /api/v1/user-pmcs/community/{checklist_id}` | Public | Fetch the complete current public release before preview or installation. |
| 13 | `PUT /api/v1/auth/user-pmcs/subscriptions/{checklist_id}` | Required | Install a linked release or explicitly resubscribe after unsubscribe. |
| 14 | `DELETE /api/v1/auth/user-pmcs/subscriptions/{checklist_id}` | Subscriber | Unsubscribe and retain a per-account tombstone. |
| 15 | `GET /api/v1/auth/user-pmcs/subscriptions/updates` | Required | Check lightweight update availability without downloading full trees. |
| 16 | `PUT /api/v1/auth/user-pmcs/subscriptions/{checklist_id}/installed-releases/{revision_id}` | Subscriber | Accept the current higher community release and advance the pin. |
| 17 | `GET /api/v1/auth/user-pmcs/subscriptions/{checklist_id}/installed-releases/{revision_id}` | Subscriber | Redownload the exact pinned immutable release, including after retirement. |

Before implementing the route DTOs, read the cross-cutting sections on
[success and error envelopes](#success-and-error-envelopes),
[JSON request rules](#json-request-rules),
[ETags](#etags-and-optimistic-concurrency),
[caching and cursors](#caching-gzip-and-cursors), and
[shared JSON structures](#shared-json-structures). The endpoint chapters then
add the workflow-specific requirements and examples for each call.

## Authentication and identity

### Firebase authentication and application account

The two routes without `/auth` are public and do not require a Firebase token.
Every `/api/v1/auth/user-pmcs` route requires a verified Firebase identity. The
server derives ownership and subscription identity from that token. Never send
an owner UID, subscriber UID, email, or Firebase claim in a path, query, or
body. Authenticated requests use the existing application header convention:
`Authorization: Bearer {firebase-id-token}`.

Authentication does not create the application's Postgres account. If a valid
Firebase user has not completed the existing account-initialization flow, the
server returns `409 account_not_initialized`. The mobile client should route
the user through that existing flow, then retry the original operation. It
must not treat this response as an empty PMCS account.

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

## Owned checklist and account synchronization endpoints

### 1. Pull the account delta

`GET /api/v1/auth/user-pmcs/sync`

**What it is used for:** This is the normal incremental synchronization route.
It returns the latest complete state of owned checklists and linked
subscriptions changed after the locally committed account cursor.

**Why it exists:** A single ordered stream prevents the client from racing two
independent feeds. It also avoids downloading every resource on every refresh.
Multiple server mutations to the same root may collapse into one latest state,
so gaps in `account_change_version` are expected.

**Request**

- Authentication: required.
- Query `after`: nonnegative decimal account cursor; defaults to `0`.
- Query `limit`: number of changed roots, `1..25`; defaults to `10`.
- Body: none.
- Conditional header: none.

Example request: `GET /api/v1/auth/user-pmcs/sync?after=41&limit=10`

**Response**

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
        "account_change_version": 45,
        "kind": "checklist",
        "checklist": {
          "id": "60000000-0000-4000-8000-000000000001",
          "sync_version": 7,
          "account_change_version": 45,
          "created_at": "2026-07-31T11:00:00Z",
          "updated_at": "2026-07-31T12:00:00Z",
          "publication": {
            "id": "10000000-0000-4000-8000-000000000001",
            "revision_number": 1,
            "name": "M998 Preventive Maintenance",
            "description": "Operator-authored checklist",
            "models": [],
            "sections": [],
            "state": "published",
            "created_at": "2026-07-31T11:30:00Z",
            "updated_at": "2026-07-31T11:30:00Z",
            "published_at": "2026-07-31T11:30:00Z"
          }
        }
      },
      {
        "account_change_version": 46,
        "kind": "subscription",
        "subscription": {
          "checklist_id": "70000000-0000-4000-8000-000000000001",
          "installed_revision_id": "71000000-0000-4000-8000-000000000001",
          "sync_version": 2,
          "account_change_version": 46,
          "created_at": "2026-07-31T11:50:00Z",
          "updated_at": "2026-07-31T12:05:00Z"
        },
        "installed": {
          "checklist_id": "70000000-0000-4000-8000-000000000001",
          "source_status": "active",
          "creator_display_name": "Maintainer",
          "released_at": "2026-07-31T11:45:00Z",
          "revision": {
            "id": "71000000-0000-4000-8000-000000000001",
            "revision_number": 2,
            "name": "Generator PMCS",
            "description": "Community checklist",
            "models": [],
            "sections": [],
            "state": "published",
            "created_at": "2026-07-31T11:40:00Z",
            "updated_at": "2026-07-31T11:40:00Z",
            "published_at": "2026-07-31T11:40:00Z"
          }
        }
      }
    ]
  }
}
```

`kind` is exactly `checklist` or `subscription`. Checklist changes include
`checklist`. Subscription changes include `subscription` and include
`installed` only while active. Tombstones omit authored or installed content.
An empty page sets `through_cursor` equal to the requested `after` value.
`account_version` is the server snapshot boundary, not necessarily the last
change returned.

The server does not split an aggregate. It normally stops before 20 MiB of
uncompressed canonical envelope JSON, but one otherwise valid large aggregate
may occupy a page alone.

**Mobile persistence rule:** Apply every complete resource in `changes` and
persist `through_cursor` in the same local database transaction. Advance the
cursor only after that transaction commits. Continue requesting pages while
`has_more` is `true`. On an empty response with `has_more: false`, the local
cursor remains `through_cursor`.

**Important failures:** `400 invalid_request` for an invalid cursor/limit,
`409 account_not_initialized`, `413 content_too_large` if an account aggregate
cannot be delivered under configured constraints, and `429 rate_limited`.

### 2. Fetch one current owned checklist

`GET /api/v1/auth/user-pmcs/checklists/{checklist_id}`

**What it is used for:** Fetch the complete canonical aggregate for one owned
checklist.

**Why it exists:** Use it to reconcile a stale write, recover an interrupted
publication-history upload, validate the local cache, or fetch one checklist
without replaying the account feed.

**Request**

- Authentication: owner required.
- Path `checklist_id`: nonzero UUID.
- Optional header: exactly one strong `If-None-Match` containing the last
  checklist ETag.
- Query/body: none.

Example request: `GET /api/v1/auth/user-pmcs/checklists/60000000-0000-4000-8000-000000000001`

**Response when changed**

```json
{
  "status": 200,
  "message": "",
  "data": {
    "id": "60000000-0000-4000-8000-000000000001",
    "sync_version": 7,
    "account_change_version": 15,
    "created_at": "2026-07-31T11:00:00Z",
    "updated_at": "2026-07-31T12:00:00Z",
    "publication": {
      "id": "10000000-0000-4000-8000-000000000001",
      "revision_number": 1,
      "name": "M998 Preventive Maintenance",
      "description": "Operator-authored checklist",
      "models": [],
      "sections": [],
      "state": "published",
      "created_at": "2026-07-31T11:30:00Z",
      "updated_at": "2026-07-31T11:30:00Z",
      "published_at": "2026-07-31T11:30:00Z"
    }
  }
}
```

Store the response `ETag` header with this aggregate. A matching conditional
request returns `304` with no body. The response uses
`Cache-Control: private, no-cache`. Another owner's checklist and an unknown
checklist both return the same safe `404 resource_not_found`.

### 3. Create a checklist and its initial draft

`PUT /api/v1/auth/user-pmcs/checklists/{checklist_id}`

**What it is used for:** Establish a new client-generated checklist UUID on the
server and upload its first draft tree.

**Why it exists:** The client can create and edit offline, retain stable IDs,
and safely retry after ambiguous network failures without creating duplicates.

**Request**

- Authentication: required.
- Path `checklist_id`: new client-generated nonzero UUID.
- Header: `If-None-Match: *`.
- Body: the strict revision request structure with `revision_number` omitted or
  `null`.
- The body revision UUID is independently generated; it is not the checklist
  UUID.

Example request body:

```json
{
  "id": "10000000-0000-4000-8000-000000000001",
  "revision_number": null,
  "name": "M998 Preventive Maintenance",
  "description": "Initial offline draft",
  "models": [],
  "sections": []
}
```

**Response**

```json
{
  "status": 201,
  "message": "",
  "data": {
    "id": "60000000-0000-4000-8000-000000000001",
    "sync_version": 1,
    "account_change_version": 1,
    "created_at": "2026-07-31T11:00:00Z",
    "updated_at": "2026-07-31T11:00:00Z",
    "draft": {
      "id": "10000000-0000-4000-8000-000000000001",
      "name": "M998 Preventive Maintenance",
      "description": "Initial offline draft",
      "models": [],
      "sections": [],
      "state": "draft",
      "created_at": "2026-07-31T11:00:00Z",
      "updated_at": "2026-07-31T11:00:00Z"
    }
  }
}
```

A new resource returns `201`. A proven retry using the identical IDs and
canonical content returns `200` with current canonical data and ETag without
incrementing versions. A conflicting pre-existing UUID or permanent tombstone
returns `412 stale_precondition`; do not generate replacement IDs until the
client has reconciled whether this was its earlier request.

Persist the aggregate and response checklist ETag atomically. Important
failures include account/checklist count ceilings (`413`), draft validation
(`422`), missing create precondition (`428`), and conflicting identity/content
(`412`).

### 4. Create or completely replace the current draft

`PUT /api/v1/auth/user-pmcs/checklists/{checklist_id}/drafts/{revision_id}`

**What it is used for:** Upload the current authored draft, whether newly
created or replacing the prior server draft.

**Why it exists:** Draft synchronization is a complete-tree save. This keeps
the server and offline authoring model deterministic and avoids partial-patch
ordering conflicts.

**Request**

- Authentication: owner required.
- Paths: nonzero checklist and revision UUIDs.
- Header: latest parent checklist `If-Match` ETag.
- Body: full strict revision request with `revision_number` absent or `null`.
- Body `id` must exactly equal path `revision_id`.

Example request body:

```json
{
  "id": "10000000-0000-4000-8000-000000000002",
  "revision_number": null,
  "name": "M998 Preventive Maintenance revision 2",
  "description": "Draft with updated intervals",
  "models": [
    {
      "display_text": "M998 HMMWV"
    }
  ],
  "sections": []
}
```

**Response**

```json
{
  "status": 200,
  "message": "",
  "data": {
    "id": "60000000-0000-4000-8000-000000000001",
    "sync_version": 8,
    "account_change_version": 16,
    "created_at": "2026-07-31T11:00:00Z",
    "updated_at": "2026-07-31T12:15:00Z",
    "draft": {
      "id": "10000000-0000-4000-8000-000000000002",
      "name": "M998 Preventive Maintenance revision 2",
      "description": "Draft with updated intervals",
      "models": [
        {
          "display_text": "M998 HMMWV",
          "normalized_text": "m998 hmmwv"
        }
      ],
      "sections": [],
      "state": "draft",
      "created_at": "2026-07-31T12:15:00Z",
      "updated_at": "2026-07-31T12:15:00Z"
    }
  }
}
```

The response is the complete aggregate and includes a new parent checklist
ETag. Replacing a draft is not a JSON merge or patch: any omitted child is
removed from the current server draft. On `412`, fetch the current aggregate,
reconcile local edits, and submit a deliberate complete replacement using the
new ETag.

### 5. Discard the current draft

`DELETE /api/v1/auth/user-pmcs/checklists/{checklist_id}/drafts/{revision_id}`

**What it is used for:** Remove the work-in-progress revision while retaining
the current publication and all immutable history.

**Why it exists:** It supports an explicit “discard draft” action without
deleting the checklist.

**Request**

- Authentication: owner required.
- Paths: nonzero checklist and exact current-draft revision UUIDs.
- Header: latest parent checklist `If-Match` ETag.
- Query/body: none.

Example request: `DELETE /api/v1/auth/user-pmcs/checklists/60000000-0000-4000-8000-000000000001/drafts/10000000-0000-4000-8000-000000000002`

**Response**

```json
{
  "status": 200,
  "message": "",
  "data": {
    "id": "60000000-0000-4000-8000-000000000001",
    "sync_version": 9,
    "account_change_version": 17,
    "created_at": "2026-07-31T11:00:00Z",
    "updated_at": "2026-07-31T12:20:00Z",
    "publication": {
      "id": "10000000-0000-4000-8000-000000000001",
      "revision_number": 1,
      "name": "M998 Preventive Maintenance",
      "description": "Operator-authored checklist",
      "models": [],
      "sections": [],
      "state": "published",
      "created_at": "2026-07-31T11:30:00Z",
      "updated_at": "2026-07-31T11:30:00Z",
      "published_at": "2026-07-31T11:30:00Z"
    }
  }
}
```

The response omits `draft` and supplies a new checklist ETag. The operation is
not allowed if no publication would remain; that returns
`409 invalid_transition`. If the product wants to remove the only draft-only
checklist, use checklist deletion instead.

### 6. Publish a complete immutable revision

`PUT /api/v1/auth/user-pmcs/checklists/{checklist_id}/publications/{revision_id}`

**What it is used for:** Validate and atomically append the exact next
publication to the checklist's immutable history.

**Why it exists:** Publications are durable versioned snapshots. The server
must validate the complete tree and revision order as one transaction so no
partial publication can become visible.

**Request**

- Authentication: owner required.
- Paths: nonzero checklist and revision UUIDs.
- Header: latest parent checklist `If-Match` ETag.
- Body: full strict revision request.
- Body `id` must equal path `revision_id`.
- `revision_number` must be positive and exactly the next number.

Example request body:

```json
{
  "id": "10000000-0000-4000-8000-000000000002",
  "revision_number": 2,
  "name": "M998 Preventive Maintenance",
  "description": "Updated publication",
  "models": [
    {
      "display_text": "M998 HMMWV"
    }
  ],
  "sections": [
    {
      "id": "20000000-0000-4000-8000-000000000002",
      "position": 1,
      "title": "Before operation",
      "models": [],
      "items": [
        {
          "id": "30000000-0000-4000-8000-000000000002",
          "position": 1,
          "interval": "Before",
          "item_to_be_checked_or_serviced": "Engine compartment",
          "performed_by": "Operator",
          "notices": [],
          "procedure_steps": [
            {
              "id": "50000000-0000-4000-8000-000000000002",
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

**Response**

```json
{
  "status": 200,
  "message": "",
  "data": {
    "id": "60000000-0000-4000-8000-000000000001",
    "sync_version": 10,
    "account_change_version": 18,
    "created_at": "2026-07-31T11:00:00Z",
    "updated_at": "2026-07-31T12:30:00Z",
    "publication": {
      "id": "10000000-0000-4000-8000-000000000002",
      "revision_number": 2,
      "name": "M998 Preventive Maintenance",
      "description": "Updated publication",
      "models": [
        {
          "display_text": "M998 HMMWV",
          "normalized_text": "m998 hmmwv"
        }
      ],
      "sections": [],
      "state": "published",
      "created_at": "2026-07-31T12:30:00Z",
      "updated_at": "2026-07-31T12:30:00Z",
      "published_at": "2026-07-31T12:30:00Z"
    }
  }
}
```

A draft upload is not required before publication. Successful publication
promotes the submitted tree and leaves no current draft; if a different draft
previously existed, it is replaced by the submitted publication. Upload any
newer work-in-progress draft separately after publication. Earlier publication
state becomes `superseded` in history. A byte-for-byte logical retry can return
canonical success without another version. Revision-number gaps, rollback,
divergent content for a reused UUID, or an invalid lifecycle return `409` or
`412`.

### 7. Fetch an immutable owned historical revision

`GET /api/v1/auth/user-pmcs/checklists/{checklist_id}/revisions/{revision_id}`

**What it is used for:** Download a specific publication from an owned
checklist's immutable history.

**Why it exists:** The current aggregate embeds only current draft/publication.
This route reconstructs older local history without making history mutable.

**Request**

- Authentication: owner required.
- Paths: nonzero checklist and revision UUIDs.
- Optional header: exactly one strong `If-None-Match`.
- Query/body: none.

Example request: `GET /api/v1/auth/user-pmcs/checklists/60000000-0000-4000-8000-000000000001/revisions/10000000-0000-4000-8000-000000000001`

**Response**

```json
{
  "status": 200,
  "message": "",
  "data": {
    "id": "10000000-0000-4000-8000-000000000001",
    "revision_number": 1,
    "name": "M998 Preventive Maintenance",
    "description": "Original publication",
    "models": [],
    "sections": [],
    "created_at": "2026-07-31T11:30:00Z",
    "updated_at": "2026-07-31T11:30:00Z",
    "published_at": "2026-07-31T11:30:00Z"
  }
}
```

The historical shape omits `state`. Store the immutable content ETag with the
revision. A matching conditional request returns `304` with no body. The
response uses `Cache-Control: private, max-age=31536000, immutable`, so the
mobile data layer can keep the revision indefinitely unless local storage is
explicitly cleared.

### 8. Permanently delete an owned checklist

`DELETE /api/v1/auth/user-pmcs/checklists/{checklist_id}`

**What it is used for:** Permanently remove owner content while retaining the
minimum tombstone required for synchronization and ID conflict prevention.

**Why it exists:** Other devices must learn that the checklist was deleted,
and a delayed offline request must not resurrect it.

**Request**

- Authentication: owner required.
- Path `checklist_id`: nonzero UUID.
- Header: latest parent checklist `If-Match` ETag.
- Query/body: none.

Example request: `DELETE /api/v1/auth/user-pmcs/checklists/60000000-0000-4000-8000-000000000001`

**Response**

```json
{
  "status": 200,
  "message": "",
  "data": {
    "id": "60000000-0000-4000-8000-000000000001",
    "sync_version": 11,
    "account_change_version": 19,
    "created_at": "2026-07-31T11:00:00Z",
    "updated_at": "2026-07-31T12:45:00Z",
    "deleted_at": "2026-07-31T12:45:00Z"
  }
}
```

Persist the tombstone and final ETag, remove authored local server mirrors, and
prevent queued stale mutations for that checklist from being retried. Deletion
wins permanently over stale create, draft, publish, release, and retire
requests. The response uses a private immutable cache policy. Any product-level
confirmation must occur before this request because there is no restore or
rollback endpoint.

## Success and error envelopes

Every JSON success uses the same outer envelope. `data` is different for each
route. Deletes return `200` with a tombstone or aggregate; no route in this
feature returns `204`.

```json
{
  "status": 200,
  "message": "",
  "data": {}
}
```

Feature-handler JSON errors have a stable machine-readable `error.code`. UI
behavior and retry decisions should use the status and code, not the
human-readable `message`. `error.details` is optional.

```json
{
  "status": 422,
  "message": "revision validation failed",
  "data": null,
  "error": {
    "code": "validation_failed",
    "details": {
      "field": "revision.sections[0].items[0].procedure_steps",
      "reason": "must contain at least one procedure step"
    }
  }
}
```

The application-wide Firebase middleware runs before these feature handlers.
If it rejects a missing, malformed, expired, or invalid token, its existing
legacy `401` response can contain only a `message` field. The shared networking
layer must therefore recognize HTTP `401` before requiring the feature error
envelope. Once authentication reaches a User PMCS handler, errors use the
structured envelope above.

| HTTP | Stable code | Required client behavior |
|---:|---|---|
| `400` | `invalid_request` | Correct malformed UUID, cursor, query, JSON, content type, unknown field, trailing JSON, compression, or UTF-8. Do not retry unchanged. |
| `400` | `invalid_precondition` | Correct the conditional headers. Do not retry unchanged. |
| `401` | `authentication_required` | Refresh or establish authentication, then retry according to the app's auth policy. |
| `403` | `forbidden` | Stop the operation and show an appropriate permission message. |
| `404` | `resource_not_found` | Treat the resource as unavailable. The same response intentionally hides cross-owner resources. |
| `409` | `account_not_initialized` | Complete existing account initialization, then retry. |
| `409` | `invalid_transition` | Refresh canonical state and change the attempted lifecycle action. Blind retry will not help. |
| `412` | `stale_precondition` | Fetch or apply newer canonical state, reconcile, obtain the new ETag, and retry only if still appropriate. |
| `413` | `content_too_large` | Reduce the request/tree/account size. Automatic retry of the same content will fail again. |
| `422` | `validation_failed` | Present field-level authoring feedback from `error.details`; do not retry unchanged. |
| `428` | `precondition_required` | Supply the required `If-Match` or `If-None-Match` header. |
| `429` | `rate_limited` | Back off with jitter and retry later. There is currently no promised `Retry-After` header. |
| `500` | `internal_error` | Preserve local state and retry safely with bounded exponential backoff. |

## JSON request rules

Only checklist creation, draft replacement, and publication accept a JSON
body. For those three routes:

- send uncompressed UTF-8 JSON;
- send `Content-Type: application/json`;
- send exactly one JSON value;
- do not send unknown fields; and
- keep the uncompressed body at or below 8,388,608 bytes.

Every other mutation has no request JSON. Do not send `{}` merely to satisfy a
generic networking wrapper.

All IDs should be canonical lowercase hyphenated nonzero UUIDs. Mobile creates
the checklist, revision, section, item, notice, and procedure-step UUIDs before
upload so retries reuse the same identities. All returned timestamps are RFC
3339 strings.

## ETags and optimistic concurrency

An ETag is an opaque quoted HTTP header value. Store the complete header value,
including quotes, and replay it verbatim. Never calculate an ETag from
`sync_version` or any JSON field.

- New checklist and new subscription creation use `If-None-Match: *`.
- Existing checklist mutations use the latest parent checklist `ETag` in one
  strong `If-Match` header.
- Existing subscription mutations use the latest subscription `ETag` in one
  strong `If-Match` header.
- Subscription installation must send either `If-None-Match` or `If-Match`,
  never both.
- A `304 Not Modified` response has no JSON body; retain the cached body.
- A successful mutation's returned ETag replaces the locally stored ETag only
  when the response and its data have been durably committed together.

The body fields `sync_version` and `account_change_version` have separate jobs.
`sync_version` identifies changes to that root resource. `account_change_version`
orders all owned-checklist and subscription changes for one account. Neither is
a substitute for the ETag required by mutations.

## Caching, gzip, and cursors

The client may request gzip. Full-tree and listing responses can be large, so
the networking layer should transparently decompress before decoding JSON.
Respect response `Cache-Control` and `ETag` headers:

- current private resources use `private, no-cache`;
- immutable owned history uses `private, max-age=31536000, immutable`;
- public resources use `public, no-cache`; and
- deleted checklist tombstones use a private immutable cache policy.

Sync uses a decimal account version. Community browse and update discovery use
opaque cursors. Never decode, edit, compare, or manufacture an opaque cursor.
Persist the exact value and return it only to the endpoint that issued it.

## Shared JSON structures

### Revision request

The same complete-tree structure is used for an initial draft, draft
replacement, and publication. A draft's `revision_number` is absent or `null`.
A publication's number is positive and exactly one greater than the latest
publication. The body `id` must equal the path's `revision_id`.

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
              "notice_text": "Use caution around hot surfaces"
            }
          ],
          "procedure_steps": [
            {
              "id": "50000000-0000-4000-8000-000000000001",
              "position": 1,
              "step_text": "Inspect the compartment for leaks",
              "fault_found_if": "Any fluid leak is present"
            }
          ]
        }
      ]
    }
  ]
}
```

The request sends only `display_text` for models. The server returns both
`display_text` and its derived `normalized_text`. Positions in each ordered
collection must be unique and contiguous starting at `1`. A UUID must be unique
across every node in the submitted tree.

Drafts may be incomplete. Publications additionally require a nonblank name,
at least one nonblank checklist model, at least one titled section, at least one
item per section, a nonblank interval and item description, at least one
nonblank procedure step per item, and a non-null supported type for every
notice. Notice type is one of `warning`, `caution`, or `note`.

### Revision response

Current revisions add state and timestamps. `revision_number` and
`published_at` are omitted when they do not exist. The arrays and nested fields
are always the same tree structure described above.

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
  "created_at": "2026-07-31T12:00:00Z",
  "updated_at": "2026-07-31T12:00:00Z",
  "published_at": "2026-07-31T12:00:00Z"
}
```

Published history may later have state `superseded`. Draft state is `draft`.
The owned historical-revision endpoint intentionally omits `state` because the
historical content itself is immutable.

### Owned checklist aggregate

An aggregate is the synchronization unit. Treat it as a complete replacement
of locally cached server state for that checklist, not as a patch. `draft`,
`publication`, and `community` are independently omitted when absent.

```json
{
  "id": "60000000-0000-4000-8000-000000000001",
  "sync_version": 7,
  "account_change_version": 15,
  "created_at": "2026-07-31T11:00:00Z",
  "updated_at": "2026-07-31T12:00:00Z",
  "draft": {
    "id": "10000000-0000-4000-8000-000000000002",
    "name": "M998 Preventive Maintenance draft",
    "description": "Work in progress",
    "models": [],
    "sections": [],
    "state": "draft",
    "created_at": "2026-07-31T12:00:00Z",
    "updated_at": "2026-07-31T12:00:00Z"
  },
  "publication": {
    "id": "10000000-0000-4000-8000-000000000001",
    "revision_number": 1,
    "name": "M998 Preventive Maintenance",
    "description": "Operator-authored checklist",
    "models": [],
    "sections": [],
    "state": "published",
    "created_at": "2026-07-31T11:30:00Z",
    "updated_at": "2026-07-31T11:30:00Z",
    "published_at": "2026-07-31T11:30:00Z"
  },
  "community": {
    "status": "active",
    "current_release_revision_id": "10000000-0000-4000-8000-000000000001",
    "latest_release_revision_number": 1,
    "first_released_at": "2026-07-31T11:45:00Z",
    "updated_at": "2026-07-31T11:45:00Z"
  }
}
```

Deletion produces a permanent lightweight tombstone with no authored tree.

```json
{
  "id": "60000000-0000-4000-8000-000000000001",
  "sync_version": 8,
  "account_change_version": 16,
  "created_at": "2026-07-31T11:00:00Z",
  "updated_at": "2026-07-31T12:30:00Z",
  "deleted_at": "2026-07-31T12:30:00Z"
}
```

### Subscription and installed release

An active subscription identifies the exact pinned release. A subscription
tombstone omits `installed_revision_id` and adds `deleted_at`.

```json
{
  "checklist_id": "60000000-0000-4000-8000-000000000001",
  "installed_revision_id": "10000000-0000-4000-8000-000000000001",
  "sync_version": 2,
  "account_change_version": 19,
  "created_at": "2026-07-31T12:00:00Z",
  "updated_at": "2026-07-31T12:10:00Z"
}
```

The installed representation embeds the entire immutable revision. It remains
the subscriber's canonical installed content even if the owner later retires
the community source.

```json
{
  "checklist_id": "60000000-0000-4000-8000-000000000001",
  "source_status": "active",
  "creator_display_name": "Maintainer",
  "released_at": "2026-07-31T12:00:00Z",
  "revision": {
    "id": "10000000-0000-4000-8000-000000000001",
    "revision_number": 1,
    "name": "M998 Preventive Maintenance",
    "description": "Operator-authored checklist",
    "models": [],
    "sections": [],
    "state": "published",
    "created_at": "2026-07-31T11:30:00Z",
    "updated_at": "2026-07-31T11:30:00Z",
    "published_at": "2026-07-31T11:30:00Z"
  }
}
```

Install and accept-update responses contain a legacy case-sensitive wrapper.
The four keys shown below are capitalized exactly as implemented. Mobile DTOs
must map `Subscription`, `Installed`, `Created`, and `Idempotent` explicitly.
`Installed` can be `null`. Unsubscribe does not use this wrapper.

```json
{
  "Subscription": {
    "checklist_id": "60000000-0000-4000-8000-000000000001",
    "installed_revision_id": "10000000-0000-4000-8000-000000000001",
    "sync_version": 1,
    "account_change_version": 18,
    "created_at": "2026-07-31T12:00:00Z",
    "updated_at": "2026-07-31T12:00:00Z"
  },
  "Installed": {
    "checklist_id": "60000000-0000-4000-8000-000000000001",
    "source_status": "active",
    "creator_display_name": "Maintainer",
    "released_at": "2026-07-31T12:00:00Z",
    "revision": {
      "id": "10000000-0000-4000-8000-000000000001",
      "revision_number": 1,
      "name": "M998 Preventive Maintenance",
      "description": "Operator-authored checklist",
      "models": [],
      "sections": [],
      "state": "published",
      "created_at": "2026-07-31T11:30:00Z",
      "updated_at": "2026-07-31T11:30:00Z",
      "published_at": "2026-07-31T11:30:00Z"
    }
  },
  "Created": true,
  "Idempotent": false
}
```

## Community publishing endpoints for checklist owners

### 9. Release an immutable revision to the community

`PUT /api/v1/auth/user-pmcs/checklists/{checklist_id}/community-releases/{revision_id}`

**What it is used for:** Select an already published revision as the owner's
current public community release.

**Why it exists:** Publishing creates private immutable history; releasing is a
separate, explicit distribution decision. This prevents every private
publication from becoming public automatically.

**Request**

- Authentication: owner required.
- Paths: nonzero checklist and published/superseded revision UUIDs.
- Header: latest parent checklist `If-Match` ETag.
- Query/body: none.

Example request: `PUT /api/v1/auth/user-pmcs/checklists/60000000-0000-4000-8000-000000000001/community-releases/10000000-0000-4000-8000-000000000002`

**Response**

```json
{
  "status": 200,
  "message": "",
  "data": {
    "id": "60000000-0000-4000-8000-000000000001",
    "sync_version": 12,
    "account_change_version": 20,
    "created_at": "2026-07-31T11:00:00Z",
    "updated_at": "2026-07-31T13:00:00Z",
    "publication": {
      "id": "10000000-0000-4000-8000-000000000002",
      "revision_number": 2,
      "name": "M998 Preventive Maintenance",
      "description": "Updated publication",
      "models": [],
      "sections": [],
      "state": "published",
      "created_at": "2026-07-31T12:30:00Z",
      "updated_at": "2026-07-31T12:30:00Z",
      "published_at": "2026-07-31T12:30:00Z"
    },
    "community": {
      "status": "active",
      "current_release_revision_id": "10000000-0000-4000-8000-000000000002",
      "latest_release_revision_number": 2,
      "first_released_at": "2026-07-31T11:45:00Z",
      "updated_at": "2026-07-31T13:00:00Z"
    }
  }
}
```

Persist the complete aggregate and new checklist ETag. The revision must belong
to this checklist and be immutable. A new release number must be strictly
higher than every prior community release. Repeating the already-current
release with matching state can succeed idempotently. Selecting an older
release is a forbidden rollback and returns `409 invalid_transition`.

This endpoint is additionally subject to release-specific rate limits. The
mobile UI should disable repeated taps while a release request is in flight and
back off after `429 rate_limited`.

### 10. Retire a community source

`DELETE /api/v1/auth/user-pmcs/checklists/{checklist_id}/community-source`

**What it is used for:** Hide an owned checklist from public browse/detail and
prevent new installations.

**Why it exists:** Retirement stops future distribution without breaking
existing subscribers' pinned immutable content.

**Request**

- Authentication: owner required.
- Path `checklist_id`: nonzero UUID.
- Header: latest parent checklist `If-Match` ETag.
- Query/body: none.

Example request: `DELETE /api/v1/auth/user-pmcs/checklists/60000000-0000-4000-8000-000000000001/community-source`

**Response**

```json
{
  "status": 200,
  "message": "",
  "data": {
    "id": "60000000-0000-4000-8000-000000000001",
    "sync_version": 13,
    "account_change_version": 21,
    "created_at": "2026-07-31T11:00:00Z",
    "updated_at": "2026-07-31T13:10:00Z",
    "publication": {
      "id": "10000000-0000-4000-8000-000000000002",
      "revision_number": 2,
      "name": "M998 Preventive Maintenance",
      "description": "Updated publication",
      "models": [],
      "sections": [],
      "state": "published",
      "created_at": "2026-07-31T12:30:00Z",
      "updated_at": "2026-07-31T12:30:00Z",
      "published_at": "2026-07-31T12:30:00Z"
    },
    "community": {
      "status": "retired",
      "latest_release_revision_number": 2,
      "first_released_at": "2026-07-31T11:45:00Z",
      "updated_at": "2026-07-31T13:10:00Z",
      "retired_at": "2026-07-31T13:10:00Z"
    }
  }
}
```

Retirement clears `current_release_revision_id`; do not infer that the owner's
publication was deleted. Existing active subscribers remain pinned and can
redownload their exact installed release. A voluntarily retired owned source
may later release only a strictly higher revision. An owner-deleted or
tombstoned source can never reactivate.

## Public community discovery endpoints

### 11. Browse active community releases

`GET /api/v1/user-pmcs/community`

**What it is used for:** Populate the public community library with recent
current releases and optional exact model filtering.

**Why it exists:** The list carries lightweight metadata only, which keeps
browsing fast. The mobile client downloads the complete tree only when the user
opens a result.

**Request**

- Authentication: none.
- Query `after`: optional opaque cursor from the preceding page.
- Query `limit`: `1..50`; defaults to `20`.
- Query `model`: optional model text. The server normalizes it and requires an
  exact normalized match.
- Body/conditional header: none.

Example request, first page: `GET /api/v1/user-pmcs/community?limit=20&model=M998%20HMMWV`

Example request, next page: `GET /api/v1/user-pmcs/community?after=opaque-value&limit=20&model=M998%20HMMWV`

**Response**

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
        "revision_id": "10000000-0000-4000-8000-000000000002",
        "revision_number": 2,
        "name": "M998 Preventive Maintenance",
        "description": "Updated publication",
        "models": [
          {
            "display_text": "M998 HMMWV",
            "normalized_text": "m998 hmmwv"
          }
        ],
        "creator_display_name": "Maintainer",
        "released_at": "2026-07-31T13:00:00Z",
        "updated_at": "2026-07-31T13:00:00Z"
      }
    ]
  }
}
```

`next_cursor` is omitted on the final page. Active sources sort by
`updated_at` descending and then checklist UUID. This is a mutable recency
feed: a concurrent release can move an item ahead of the current cursor. Pull
to refresh must discard the pagination chain and start again without `after`.
Do not merge a refreshed first page into an old cursor chain.

The displayed creator name is current at read time. It is never a UID or
email. Retained content whose owner account was deleted displays
`"Deleted user"`. A filter that yields no results is a normal `200` with an
empty `items` array and `has_more: false`.

### 12. Fetch the current public release

`GET /api/v1/user-pmcs/community/{checklist_id}`

**What it is used for:** Download the complete current release for preview and
as the public source immediately before installation.

**Why it exists:** Browse results intentionally omit the large tree. This route
provides one cacheable immutable release while still following the source's
current public pointer.

**Request**

- Authentication: none.
- Path `checklist_id`: nonzero UUID.
- Optional `If-None-Match`: supports `*`, weak or strong validators,
  comma-separated values, and repeated header fields using weak comparison.
- Query/body: none.

Example request: `GET /api/v1/user-pmcs/community/60000000-0000-4000-8000-000000000001`

**Response**

```json
{
  "status": 200,
  "message": "",
  "data": {
    "checklist_id": "60000000-0000-4000-8000-000000000001",
    "creator_display_name": "Maintainer",
    "released_at": "2026-07-31T13:00:00Z",
    "revision": {
      "id": "10000000-0000-4000-8000-000000000002",
      "revision_number": 2,
      "name": "M998 Preventive Maintenance",
      "description": "Updated publication",
      "models": [
        {
          "display_text": "M998 HMMWV",
          "normalized_text": "m998 hmmwv"
        }
      ],
      "sections": [],
      "state": "published",
      "created_at": "2026-07-31T12:30:00Z",
      "updated_at": "2026-07-31T12:30:00Z",
      "published_at": "2026-07-31T12:30:00Z"
    }
  }
}
```

Store the response ETag with the public representation. A conditional match
returns `304` with no body. Retired, deleted, never-released, and
superseded-but-not-current resources all return safe `404 resource_not_found`.
If the detail becomes unavailable between browse and open, remove or mark the
stale browse card and let the user refresh.

## Linked subscription endpoints

### 13. Install a community checklist or resubscribe

`PUT /api/v1/auth/user-pmcs/subscriptions/{checklist_id}`

**What it is used for:** Create a linked read-only installation pinned to the
source's current release, or reactivate a retained subscription tombstone.

**Why it exists:** Installation retains provenance and update linkage. It is
not an editable copy or fork of the owner's checklist.

**Request for a first installation**

- Authentication: required; an owner cannot subscribe to their own source.
- Path `checklist_id`: source checklist UUID.
- Header: `If-None-Match: *`.
- Query/body: none.

**Request for resubscription**

- Use the same route and no body.
- Header: `If-Match` with the current subscription tombstone ETag.
- Never send `If-None-Match` and `If-Match` together.

Example request: `PUT /api/v1/auth/user-pmcs/subscriptions/60000000-0000-4000-8000-000000000001`

**Response**

```json
{
  "status": 201,
  "message": "",
  "data": {
    "Subscription": {
      "checklist_id": "60000000-0000-4000-8000-000000000001",
      "installed_revision_id": "10000000-0000-4000-8000-000000000002",
      "sync_version": 1,
      "account_change_version": 22,
      "created_at": "2026-07-31T13:15:00Z",
      "updated_at": "2026-07-31T13:15:00Z"
    },
    "Installed": {
      "checklist_id": "60000000-0000-4000-8000-000000000001",
      "source_status": "active",
      "creator_display_name": "Maintainer",
      "released_at": "2026-07-31T13:00:00Z",
      "revision": {
        "id": "10000000-0000-4000-8000-000000000002",
        "revision_number": 2,
        "name": "M998 Preventive Maintenance",
        "description": "Updated publication",
        "models": [],
        "sections": [],
        "state": "published",
        "created_at": "2026-07-31T12:30:00Z",
        "updated_at": "2026-07-31T12:30:00Z",
        "published_at": "2026-07-31T12:30:00Z"
      }
    },
    "Created": true,
    "Idempotent": false
  }
}
```

A new subscription row returns `201`. Resubscription and a proven retry return
`200`. Decode the capitalized wrapper keys exactly. Persist `Subscription`,
`Installed`, and the response subscription ETag atomically before showing the
checklist as installed.

A retired source cannot be newly installed. A create-style request against a
retained tombstone returns `412`; resubscription must use that tombstone's ETag.
If a previewed release changes before installation, the install pins whatever
release is current when the installation transaction succeeds; always display
the canonical returned `Installed` tree rather than assuming it matches the
preview.

### 14. Unsubscribe from an installed checklist

`DELETE /api/v1/auth/user-pmcs/subscriptions/{checklist_id}`

**What it is used for:** Remove the active linked installation and retain a
lightweight subscription tombstone for synchronization and explicit future
resubscription.

**Why it exists:** Other devices need to learn that the user unsubscribed, and
a delayed install retry must not silently reactivate the link.

**Request**

- Authentication: active subscriber required.
- Path `checklist_id`: source checklist UUID.
- Header: current subscription `If-Match` ETag.
- Query/body: none.

Example request: `DELETE /api/v1/auth/user-pmcs/subscriptions/60000000-0000-4000-8000-000000000001`

**Response**

```json
{
  "status": 200,
  "message": "",
  "data": {
    "checklist_id": "60000000-0000-4000-8000-000000000001",
    "sync_version": 2,
    "account_change_version": 23,
    "created_at": "2026-07-31T13:15:00Z",
    "updated_at": "2026-07-31T13:30:00Z",
    "deleted_at": "2026-07-31T13:30:00Z"
  }
}
```

This route returns the snake_case subscription directly in `data`; it does not
return the capitalized mutation wrapper. Persist the tombstone and new ETag,
then remove the installed tree from active local views. A repeated delete with
the current tombstone ETag is idempotent. A tombstoned subscription cannot read
its former pin or accept an update.

### 15. Discover available subscription updates

`GET /api/v1/auth/user-pmcs/subscriptions/updates`

**What it is used for:** List version/status metadata for active subscriptions
without downloading any complete revision trees.

**Why it exists:** Community releases do not fan out writes to subscribers and
do not automatically move their pins. This lightweight read lets mobile show
update badges efficiently.

**Request**

- Authentication: required.
- Query `after`: optional opaque cursor from the previous update page.
- Query `limit`: `1..100`; defaults to `50`.
- Body/conditional header: none.

Example request: `GET /api/v1/auth/user-pmcs/subscriptions/updates?limit=50`

**Response**

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
        "source_status": "active",
        "installed_revision_id": "10000000-0000-4000-8000-000000000001",
        "installed_revision_number": 1,
        "current_release_revision_id": "10000000-0000-4000-8000-000000000002",
        "current_release_revision_number": 2,
        "update_available": true
      },
      {
        "checklist_id": "70000000-0000-4000-8000-000000000001",
        "source_status": "retired",
        "installed_revision_id": "71000000-0000-4000-8000-000000000003",
        "installed_revision_number": 3,
        "update_available": false
      }
    ]
  }
}
```

`next_cursor` is omitted on the final page. Pages use stable ascending
checklist UUID order. An active row includes current release fields and sets
`update_available` only when the current number is higher than the installed
number. A retired row omits both current release fields and always reports
`false`. Tombstoned subscriptions are omitted.

This route does not mutate the subscription or provide the full update tree.
Refresh from the first page when the app needs a new snapshot. The client may
cache these rows as presentation metadata, but the subscription object and
installed release remain authoritative.

### 16. Accept the current higher release

`PUT /api/v1/auth/user-pmcs/subscriptions/{checklist_id}/installed-releases/{revision_id}`

**What it is used for:** Explicitly advance an active linked installation to
the source's current higher release.

**Why it exists:** Updates are user-controlled. Publishing or releasing never
silently changes a subscriber's pinned content.

**Request**

- Authentication: active subscriber required.
- Path `checklist_id`: source UUID.
- Path `revision_id`: exact `current_release_revision_id` obtained from update
  discovery or a refreshed public source.
- Header: current subscription `If-Match` ETag.
- Query/body: none.

Example request: `PUT /api/v1/auth/user-pmcs/subscriptions/60000000-0000-4000-8000-000000000001/installed-releases/10000000-0000-4000-8000-000000000002`

**Response**

```json
{
  "status": 200,
  "message": "",
  "data": {
    "Subscription": {
      "checklist_id": "60000000-0000-4000-8000-000000000001",
      "installed_revision_id": "10000000-0000-4000-8000-000000000002",
      "sync_version": 3,
      "account_change_version": 24,
      "created_at": "2026-07-31T13:15:00Z",
      "updated_at": "2026-07-31T14:00:00Z"
    },
    "Installed": {
      "checklist_id": "60000000-0000-4000-8000-000000000001",
      "source_status": "active",
      "creator_display_name": "Maintainer",
      "released_at": "2026-07-31T13:00:00Z",
      "revision": {
        "id": "10000000-0000-4000-8000-000000000002",
        "revision_number": 2,
        "name": "M998 Preventive Maintenance",
        "description": "Updated publication",
        "models": [],
        "sections": [],
        "state": "published",
        "created_at": "2026-07-31T12:30:00Z",
        "updated_at": "2026-07-31T12:30:00Z",
        "published_at": "2026-07-31T12:30:00Z"
      }
    },
    "Created": false,
    "Idempotent": false
  }
}
```

Decode the capitalized wrapper keys exactly. Replace the local subscription,
installed tree, and ETag in one transaction. Do not discard the old local tree
until the canonical response is durable.

The target must still be the source's current active release and must advance
the installed revision. If the source changed again, retired, or otherwise
cannot make the transition, the server returns `409 invalid_transition`; rerun
update discovery. A stale subscription ETag returns `412`; reconcile the
subscription first. Missing or deleted subscriptions return safe `404` before
source state is evaluated.

### 17. Redownload the exact pinned release

`GET /api/v1/auth/user-pmcs/subscriptions/{checklist_id}/installed-releases/{revision_id}`

**What it is used for:** Recover the complete immutable tree for the exact
revision currently pinned by an active subscription.

**Why it exists:** It supports cache eviction, device restoration, and
interrupted local persistence without allowing arbitrary access to the
creator's release history.

**Request**

- Authentication: active subscriber required.
- Path `checklist_id`: source UUID.
- Path `revision_id`: must exactly equal the subscription's
  `installed_revision_id`.
- Optional `If-None-Match`: accepts weak or strong validators, `*`, lists, and
  repeated fields using weak comparison.
- Query/body: none.

Example request: `GET /api/v1/auth/user-pmcs/subscriptions/60000000-0000-4000-8000-000000000001/installed-releases/10000000-0000-4000-8000-000000000002`

**Response**

```json
{
  "status": 200,
  "message": "",
  "data": {
    "checklist_id": "60000000-0000-4000-8000-000000000001",
    "source_status": "retired",
    "creator_display_name": "Maintainer",
    "released_at": "2026-07-31T13:00:00Z",
    "revision": {
      "id": "10000000-0000-4000-8000-000000000002",
      "revision_number": 2,
      "name": "M998 Preventive Maintenance",
      "description": "Updated publication",
      "models": [],
      "sections": [],
      "state": "published",
      "created_at": "2026-07-31T12:30:00Z",
      "updated_at": "2026-07-31T12:30:00Z",
      "published_at": "2026-07-31T12:30:00Z"
    }
  }
}
```

The exact pin remains readable after voluntary source retirement, which is why
`source_status` can be `retired`. A conditional match returns `304` with no
body. A different revision UUID returns safe `404`; this route is not a public
or subscriber history browser. Tombstoned subscriptions cannot use it.

## End-to-end mobile workflows

The route contracts above define individual calls. The sequences below define
how those calls fit together without losing local work or corrupting cursors.

### First upload of a checklist with local publication history

Use this sequence when a checklist was authored offline and may already have
several local immutable publications plus a newer incomplete draft.

1. Keep all existing client-generated UUIDs. Do not regenerate IDs just because
   the server has not seen them.
2. Call endpoint 3 with publication 1's complete tree represented as a draft:
   `revision_number: null` and `If-None-Match: *`.
3. Persist the returned aggregate and parent checklist ETag.
4. Call endpoint 6 with the same revision UUID and tree, now with
   `revision_number: 1` and the stored parent ETag in `If-Match`.
5. Persist the returned aggregate and replacement ETag.
6. For publication 2 and every later local publication in ascending number
   order, call endpoint 4 to upload that complete tree as the current draft,
   then endpoint 6 to publish the same UUID/tree as the exact next number.
   Always use the newest ETag from the preceding successful mutation.
7. After the entire immutable publication prefix exists on the server, upload
   the current incomplete local draft with endpoint 4, if one exists.
8. Fetch endpoint 2 once to confirm the current aggregate, then start endpoint
   1 from the locally committed account cursor.

Never upload only the latest publication and assume older history can be added
later. The server enforces ascending immutable publication order. Never use a
community release endpoint until the target publication exists.

### Recovering an interrupted first upload

Network loss can occur after the server commits but before mobile records the
response. Recovery must be state-based:

1. Fetch endpoint 2 for the checklist UUID.
2. Compare the canonical draft/publication UUID and revision number with local
   history.
3. If the previously attempted operation is already represented identically,
   accept the canonical aggregate and continue from the next missing step.
4. If it is not represented, retry the original request with the same IDs and
   content after obtaining the correct ETag.
5. If the server reports divergent content, a reused UUID, an unexpected
   revision number, or `412`, stop automatic upload and surface a reconciliation
   state. The server will not renumber or overwrite immutable history.

Create, publish, release, unsubscribe, and update operations have idempotent
cases, but that does not make arbitrary changed retries safe. Reuse the exact
original identifiers and logical content for ambiguity recovery.

### Normal startup and foreground synchronization

1. Load the last durably committed account cursor, defaulting to `0` for the
   first account sync.
2. Pull endpoint 1 with a conservative page limit.
3. In one local transaction, replace each checklist/subscription root with its
   complete returned aggregate, apply tombstones, store embedded installed
   content, and update the cursor to `through_cursor`.
4. Commit the transaction before requesting the next page.
5. Continue while `has_more` is `true`.
6. Separately call endpoint 15 when the product wants community update badges.
   Update discovery is presentation metadata and does not replace account
   delta synchronization.

On process death before commit, retry the same page. On process death after
commit, resume from the committed cursor. Do not advance directly to
`account_version`; doing so could skip roots not returned on the current page.

### Saving a normal draft

1. Edit locally without issuing per-field server requests.
2. Build one complete draft tree with stable node UUIDs and contiguous
   positions.
3. Call endpoint 4 with the latest stored parent checklist ETag.
4. On `200`, atomically replace the local server mirror and ETag.
5. On `412`, keep local edits, fetch endpoint 2, and require a deliberate merge
   or overwrite decision before resubmission.

The server response is canonical. In particular, store server model
`normalized_text` and timestamps rather than recalculating them for equality.

### Publishing and optionally releasing

1. Validate the publication requirements locally for immediate author feedback.
2. Call endpoint 6 with the exact next revision number and latest checklist
   ETag.
3. Persist the aggregate and new ETag before updating UI state.
4. If the user separately chooses to share publicly, call endpoint 9 with that
   publication UUID and the new ETag.
5. Persist the released aggregate and newest ETag.

Publication and community release are intentionally separate confirmations.
A failed release does not roll back a successful private publication. Retrying
release must use refreshed checklist state when the failure is `412`.

### Browsing, previewing, and installing community content

1. Use endpoint 11 for recent cards. Restart at page one for pull-to-refresh.
2. Use endpoint 12 to fetch a selected card's current full tree and preview it.
3. If the user installs, call endpoint 13 with `If-None-Match: *`.
4. Treat endpoint 13's returned `Installed` object as authoritative, because a
   newer release may have become current after preview.
5. Persist the subscription, installed release, and ETag in one transaction.
6. Display the installed checklist as linked and read-only. Do not expose it as
   an owned editable fork.

If install returns `412` because a tombstone exists, load that tombstone and
require an explicit resubscribe action using its ETag. Do not silently turn a
create attempt into resubscription.

### Discovering and accepting a community update

1. Page through endpoint 15 from the beginning to build a fresh lightweight
   snapshot.
2. Display update UI only when `source_status` is `active` and
   `update_available` is `true`.
3. When the user accepts, call endpoint 16 with the advertised current release
   UUID and the latest subscription ETag.
4. On `200`, atomically replace the subscription, full installed release, and
   ETag from the capitalized response wrapper.
5. On `409`, refresh update discovery because the current source release or
   status changed.
6. On `412`, reconcile the subscription via endpoint 1 and retry only if it is
   still active and still pinned to the expected older revision.

Never change local installed content merely because update discovery reports a
higher number. The pin advances only after endpoint 16 succeeds and its full
canonical response is durable.

### Retirement, redownload, unsubscribe, and resubscribe

- When update discovery says `source_status: "retired"`, keep the installed
  content available, remove update actions, and show an appropriate source
  status.
- If the pinned local tree is missing, endpoint 17 can redownload that exact
  pin even after retirement.
- Endpoint 17 cannot fetch any other release, and endpoint 12 will no longer
  expose a retired current public release.
- Endpoint 14 unsubscribes. Persist its tombstone before removing active local
  presentation state.
- Resubscription uses endpoint 13 with the tombstone ETag and succeeds only
  when the source again has an active current release.

## Validation and payload limits

The mobile authoring layer should enforce these implemented server ceilings
before upload, while still displaying server `422` or `413` details if its
validation differs:

| Limit | Maximum |
|---|---:|
| Active owned checklists per account | 250 |
| Active subscriptions per account | 500 |
| Checklist models | 100 |
| Sections | 100 |
| Models per section | 100 |
| Models across all sections | 1,000 |
| Items per section | 500 |
| Items across all sections | 2,000 |
| Notices per item | 100 |
| Notices across the revision | 4,000 |
| Procedure steps per item | 250 |
| Procedure steps across the revision | 10,000 |
| Short text | 200 Unicode extended grapheme clusters and 8 KiB |
| Long text | 4,000 Unicode extended grapheme clusters and 64 KiB |
| Mutation request body | 8 MiB uncompressed |
| Account delta roots per page | Default 10, maximum 25 |
| Account delta response | 20 MiB soft boundary without splitting an aggregate |
| Community browse items per page | Default 20, maximum 50 |
| Subscription update items per page | Default 50, maximum 100 |

Short text fields are revision name, model display text, section title, item
interval, and performed-by. Long text fields are revision description,
item-to-be-checked-or-serviced, notice text, procedure-step text, and
fault-found-if.

Count user-visible text by Unicode extended grapheme clusters, not bytes,
UTF-16 code units, or Unicode scalar values. Both the grapheme and byte limit
must pass. Model uniqueness uses the server's normalized form, so mobile should
handle a server duplicate-model error even when the display strings differ.

The default operational limits are 2 public requests per second with burst 20,
10 authenticated reads per second with burst 30, and 2 authenticated mutations
per second with burst 10. Community release additionally defaults to 12 per
user per hour with burst 3 and 60 per IP per hour with burst 10. Deployments can
configure these values, so mobile must rely on `429`, not hard-code timing based
on the defaults.

## Retry and reconciliation matrix

| Situation | Safe mobile action |
|---|---|
| Request never left the device | Retry with the same request identifiers and precondition. |
| Connection ended before a mutation response | Fetch canonical root when possible; otherwise retry the exact logical request with the same IDs. |
| `401 authentication_required` | Refresh authentication once, then retry according to existing auth policy. |
| `409 account_not_initialized` | Complete existing account initialization, then retry. |
| `409 invalid_transition` | Refresh state and change the requested action; do not blind-retry. |
| `412 stale_precondition` | Fetch/apply canonical state, reconcile, and use the returned current ETag. |
| `413 content_too_large` | Reduce content/count; do not retry unchanged. |
| `422 validation_failed` | Attach `details.field` and `details.reason` to authoring UI; do not retry unchanged. |
| `429 rate_limited` | Retry later with exponential backoff and jitter; do not spin. |
| `500 internal_error` | Preserve local work and retry exact idempotent operations with bounded exponential backoff. |
| `304 Not Modified` | Keep the cached body and metadata; there is no JSON response to decode. |

## Mobile implementation checklist

- Implement DTOs that distinguish absent fields from `null` where documented.
- Preserve unknown future response fields if the mobile serialization approach
  supports forward-compatible decoding, but never send unknown request fields.
- Map the capitalized install/update wrapper keys separately from snake_case
  resource objects.
- Store owned-checklist and subscription ETags separately.
- Store ETags with their exact resource representation and replace them only in
  the same transaction as a successful mutation response.
- Store one account delta cursor per authenticated application account.
- Keep community and update cursors scoped to their query/filter and pagination
  session; never reuse one endpoint's cursor on another endpoint.
- Treat aggregates and installed releases as complete replacements, not patches.
- Apply delta changes and `through_cursor` atomically.
- Retain tombstones so delayed offline operations cannot resurrect deleted
  state.
- Keep owned checklists and linked subscriptions as distinct local resource
  types, even if they share rendering components.
- Keep linked installed content read-only and pinned until explicit update
  acceptance succeeds.
- Make public browse refresh restart from the first page.
- Handle `304` before attempting JSON decoding.
- Allow gzip and test decoding of large full-tree responses.
- Never log Firebase tokens, authored checklist bodies, or private ETags in
  production diagnostics.
- Test process death before and after each local transaction boundary used for
  cursor advancement and mutation persistence.

## Explicit exclusions

The mobile implementation must not infer or build unimplemented server
features. This API has no moderation, rollback release, direct sharing,
editable fork, JSONB revision document, generic idempotency key, or server-side
inspection execution/progress. It does not provision Firebase/application
accounts. It does not synchronize faults, notes, equipment selections,
completions, exports, or inspection history.

Creating an editable local copy of community content, if ever approved, would
be a separate product feature with new client-generated identities. It is not
part of linked subscription installation and is not authorized by these
endpoints.

## Source of truth and change control

This guide describes the server behavior verified at the baseline at the top
of the document. The lower-level server contract remains in
`docs/client/2026-07-29-user-pmcs-server-api-contract.md`. If mobile observes a
wire response that differs from this guide, capture the HTTP method, redacted
path, status, headers, and response shape and resolve the server/client contract
before adding a client-side workaround.
