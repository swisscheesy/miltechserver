# Shops server changes: client implementation handoff

Date: 2026-10-04. Scope: the server remediation merged into `cleanup`, relative to base `518ef4e`. This document was derived from server routes, handlers, request/response DTOs, services and regression evidence. **No Flutter source was reviewed or changed.** Client work below is required only where the client currently relies on the affected behavior; its implementation has not been verified.

All paths below have prefix **`/api/v1/auth`**. No existing route was removed or renamed by this remediation. The usage PATCH, atomic save, capabilities and messages-v2 routes already existed at the base; their inclusion describes compatibility and hardened behavior, not newly introduced endpoints. Generated Jet models were not edited manually. Automatic tagged Jet generation at application startup remains mandatory.

## Reading the examples

UUIDs, user IDs, timestamps, image URLs and values are synthetic. JSON blocks labeled request body are actual JSON bodies. A GET or bodyless DELETE has **no body**; its query example is shown separately as JSON for readability and must be encoded as URL query parameters, not sent as a JSON body. Multipart upload is described explicitly and is not a JSON request. Omitted fields, explicit `null`, empty strings and `false` have different meanings where specified.

Most responses use `{status, data, message}`. `code` is omitted when empty. A success produced by `response.OK` has `message:""`; several create/settings/aggregate handlers retain their own success text. Bulk-delete success and some legacy validation errors are flat objects. Do not impose one universal envelope on every endpoint.

The optional header `X-MilTech-Shops-Contract: 2` negotiates typed Shops failures on participating handlers. It is required for capabilities, atomic save and messages-v2. Legacy success shapes remain unchanged. Do not add it globally until the client handles negotiated error statuses/codes. Some direct handler errors retain their existing mapping even with that header.

## Endpoint index

Each row maps concrete affected endpoints to the change, reason, client requirement and examples in the numbered sections. All rows also inherit section 1's authentication/failure behavior and section 2's current-resource authorization. Read routes retain their existing JSON schema unless a section explicitly describes a correction.

| Endpoints (relative to prefix) | Section | Main client-visible effect |
| --- | --- | --- |
| Every Shops and equipment-service route below | 1–2 | Current Firebase identity, controlled failure responses, current resource permissions |
| POST `/shops`; PUT `/shops/:shop_id` | 3 | Atomic initial membership; settings preserved; details omission/null preserved |
| DELETE `/shops/:shop_id` | 3 | Creator-only deletion while still a member |
| GET `/shops`, `/shops/:shop_id`, `/shops/user-data`, `/shops/equipment/overview` | 2, 11 | Membership-bound reads; existing shapes preserved |
| GET/PUT `/shops/:shop_id/settings`; GET/PUT `/shops/:shop_id/settings/admin-only-lists`; GET `/shops/:shop_id/is-admin` | 3 | Explicit false accepted; admin-only mutations; omission rejected on setting update |
| POST `/shops/join`; DELETE `/shops/:shop_id/leave`; DELETE `/shops/members/remove`; PUT `/shops/members/promote`; GET `/shops/:shop_id/members` | 4 | Admission preserves roles; last-admin departure requires successor |
| POST `/shops/invite-codes`; GET `/shops/:shop_id/invite-codes`; DELETE `/shops/invite-codes/:code_id`; DELETE `/shops/invite-codes/:code_id/delete` | 4 | Non-null expiry/max-use controls rejected; revocation enforced |
| POST/PUT/DELETE `/shops/lists`; GET `/shops/:shop_id/lists`, `/shops/lists/:list_id` | 2, 5 | Current list policy enforced; referenced-list deletion still conflicts |
| POST/PUT/DELETE `/shops/lists/items`; GET `/shops/lists/:list_id/items`; POST/DELETE `/shops/lists/items/bulk` | 5 | Whole-batch validation; actual unique deletion count; stale bulk IDs tolerated |
| POST/PUT `/shops/vehicles`; GET `/shops/:shop_id/vehicles`, `/shops/vehicles/:vehicle_id`; DELETE `/shops/vehicles/:vehicle_id`; PATCH `/shops/vehicles/:vehicle_id/usage` | 6 | Safe usage classification; metadata field presence; nonnegative base values |
| POST/PUT `/shops/vehicles/notifications`; DELETE/GET `/shops/vehicles/notifications/:notification_id`; GET `/shops/:shop_id/notifications`, `/shops/vehicles/:vehicle_id/notifications`, `/shops/vehicles/:vehicle_id/notifications-with-items` | 7, 11 | Attachment intent preserved; strict type; coherent combined read |
| POST `/shops/notifications/items`, `/shops/notifications/items/bulk`; GET `/shops/notifications/:notification_id/items`, `/shops/:shop_id/notification-items`; DELETE `/shops/notifications/items/:item_id`, `/shops/notifications/items/bulk` | 7 | Metadata retention without row resurrection; whole-batch validation/count |
| POST `/shops/vehicles/notifications/save` | 8 | Independent readiness gate; durable retry identity and receipt semantics |
| POST/PUT `/shops/messages`; DELETE `/shops/messages/:message_id`; GET `/shops/:shop_id/messages`, `/shops/:shop_id/messages/paginated` | 9 | Same-Shop parents; explicit legacy DTO; stable tuple cursors/reload |
| POST `/shops/messages/image/upload`; DELETE `/shops/messages/image/:message_id` | 9 | Bounded multipart; registered asset ownership; asynchronous cleanup |
| GET `/shops/:shop_id/messages-v2/initial`, `/history`, `/catch-up`; POST `/shops/:shop_id/messages-v2/reconcile` | 10 | Integrity/readiness checks; opaque history cursor; string watermarks/reset |
| GET `/shops/capabilities` | 8, 10 | Advertised feature requires flag and readiness; no-store |
| POST/GET `/shops/:shop_id/equipment-services`; GET/PUT/DELETE `/shops/:shop_id/equipment-services/:service_id`; POST `/shops/:shop_id/equipment-services/:service_id/complete` | 12 | Actual Shop authorization; completion timestamp preservation; cancellation |
| GET `/shops/:shop_id/equipment/:equipment_id/services`; GET `/shops/:shop_id/equipment-services/overdue`, `/due-soon`, `/calendar` | 12 | Conjunctive filters/counts; valid dates and bounded query parameters |
| GET `/shops/bootstrap`, `/shops/:shop_id/snapshot`, `/shops/:shop_id/lists-with-items`, `/shops/vehicles/:vehicle_id/maintenance-snapshot`, `/shops/equipment-pmcs-history` | 11 | One authorized read snapshot; large selected sets no longer hit bind ceiling |
| GET `/shops/notifications/:notification_id/changes`, `/shops/:shop_id/notifications/changes`, `/shops/vehicles/:vehicle_id/notifications/changes` | 11 | Enriched event-time audit; nullable deleted-resource references retained |

## 1. Authentication, failures and uncertain mutation outcomes

**Why:** previously insufficient Firebase account checks and unhandled service errors could allow revoked identities or produce an empty apparent success. Firebase verification now checks the current account, deletion/disabled state and token revocation. Infrastructure failures fail closed. The Shops error middleware returns a controlled response and sanitizes internal failures.

**Client requirement:** use `Authorization: Bearer <Firebase ID token>`. A bare token is invalid. Handle 401 as an authentication failure; refresh/re-authenticate as appropriate. A 503 identity dependency failure must not be treated as a logout or empty data. Do not treat a non-2xx response as a successful mutation. Preserve a draft on errors. Many unnegotiated business failures intentionally retain legacy HTTP 500; do not use status alone to classify all legacy permission failures. Contract 2 supplies codes/statuses through the negotiated path, but `typed_errors` capability remains false pending released-client acceptance.

An expired/revoked/invalid token can produce this **contract-2 HTTP 401**:

```json
{"status":401,"code":"unauthorized","data":null,"message":"unauthorized"}
```

An identity dependency failure produces this **contract-2 HTTP 503**:

```json
{"status":503,"code":"unavailable","data":null,"message":"unavailable"}
```

Without negotiation, the corresponding identity failure has HTTP 503 and:

```json
{"status":503,"data":null,"message":"Authentication service unavailable"}
```

Representative negotiated permission failure (HTTP 403):

```json
{"status":403,"code":"denied","data":null,"message":"access denied: not a member of this shop"}
```

All request SQL uses the caller context in the affected packages. A cancellation or lost response can occur **after commit**, including equipment create/update/complete before username enrichment. Legacy mutations and usage adjustments have no replay receipt. Re-read persisted state before retrying; blindly retrying a create, usage increment, message send or completion can duplicate work. Atomic save's stable operation identity is the exception described in section 8.

## 2. Current-resource authorization across reads and writes

**Why:** preflight/cached permissions alone could be stale, and a Shop ID from another resource could authorize a write. Writes now resolve the persisted parent/Shop and recheck current authority under transaction locks. Combined reads authorize in their read snapshot. A subsequent request observes membership revocation; already-started snapshot reads can finish with their snapshot's authority.

**Client requirement:** derive paths and parent IDs from the selected resource, handle permission changes after screens open, and refresh resource/role state after denial. Admin status in Shop A grants no authority in Shop B. Hiding controls is useful UX but does not replace server authorization.

Shop rename/settings/invite/member administration requires current Shop admin. Explicit Shop deletion requires original creator plus current membership. Vehicle metadata/delete requires current member who is vehicle creator or Shop admin; tracked-usage writes allow current members. Equipment-service update/delete/complete requires current member who is service creator or admin of its actual Shop. List/item writes follow the current `admin_only_lists` setting; list deletion additionally requires list creator when admin-only mode is off, or Shop admin. Message edits remain author-only; deletes allow author/admin, with current membership. Notification writes resolve and authorize their actual vehicle/Shop and any linked list; items cannot redirect a mutation by submitting a foreign parent.

Example GET `/shops/11111111-1111-4111-8111-111111111111/members`: no body. An authorized empty-array response has the existing shape:

```json
{"status":200,"data":[],"message":""}
```

Permission failures use section 1's controlled failure path. Existing endpoint-specific status behavior is preserved; a client must not assume every resource denial has one universal legacy status.

## 3. Shop creation, rename, deletion and settings

**Why:** creation and creator membership could partially commit; rename could reset the list setting; false could fail required-field validation; creator/admin policies diverged from the approved behavior.

POST `/shops` now commits the Shop and creator-admin membership together. Omitted `admin_only_lists` defaults false; an explicit true persists. Request body:

```json
{"name":"Maintenance Shop","details":"North bay","admin_only_lists":true}
```

HTTP 201 response (existing model keys):

```json
{"status":201,"message":"Shop created successfully","data":{"id":"11111111-1111-4111-8111-111111111111","name":"Maintenance Shop","details":"North bay","created_by":"firebase-user-a","created_at":"2026-10-04T12:00:00Z","updated_at":"2026-10-04T12:00:00Z","admin_only_lists":true}}
```

PUT `/shops/:shop_id` still requires `name`. Omitted or null `details` preserves stored details; `details:""` clears them. It preserves `admin_only_lists`; use settings routes to change that setting. Request body:

```json
{"name":"Maintenance Shop 2"}
```

HTTP 200 returns the updated Shop in `data` with `message:"Shop updated successfully"` and the same model keys as creation. A promoted current admin can rename even if not creator.

PUT `/shops/:shop_id/settings` and `/settings/admin-only-lists` must send an explicit boolean; **false is valid**. `{}` or null supplies no setting and is rejected. Request body for either:

```json
{"admin_only_lists":false}
```

Unified HTTP 200 response:

```json
{"status":200,"message":"Shop settings updated successfully","data":{"admin_only_lists":false}}
```

Dedicated HTTP 200 response:

```json
{"status":200,"message":"Shop admin_only_lists setting updated successfully","data":{"shop_id":"11111111-1111-4111-8111-111111111111","admin_only_lists":false}}
```

GET equivalents have no body and the same `data` shapes with their existing retrieved-success messages. GET `/shops/:shop_id/is-admin` returns `data:{shop_id,user_id,is_admin}`. Client controls must accept explicit false and not serialize it away.

DELETE `/shops/:shop_id`: no body. Only the original creator while a current member can explicitly delete; promoted admin is insufficient. Success:

```json
{"status":200,"message":"","data":{"message":"Shop deleted successfully"}}
```

Client requirement: expose explicit delete accordingly; after delete retire cached Shop routes/data. A failure is not proof the Shop disappeared.

## 4. Membership, successor promotion and invitations

**Why:** duplicate join could demote an existing admin, revocation could race admission, and departures could leave a Shop without an admin. Invitation expiry/max-use controls were silently ignored.

POST `/shops/join` request body:

```json
{"invite_code":"A1B2C3D4"}
```

Successful join retains its existing success envelope. Already-member does not change the role and returns the existing legacy failure or contract-2 HTTP 409:

```json
{"status":409,"code":"conflict","data":null,"message":"user is already a member of this shop"}
```

Invite admission/revocation is serialized. Removal is **not a ban**; an otherwise valid invitation can admit the user again. Eight-character uppercase hexadecimal codes remain the contract.

PUT `/shops/members/promote` request body:

```json
{"shop_id":"11111111-1111-4111-8111-111111111111","target_user_id":"firebase-user-b"}
```

Success:

```json
{"status":200,"message":"","data":{"message":"Member promoted to admin successfully"}}
```

DELETE `/shops/members/remove` uses the same two-field body and returns `data:{message:"Member removed successfully"}`. DELETE `/shops/:shop_id/leave` has no body and returns `data:{message:"Successfully left shop"}`. **If the last admin would leave while other members remain, promote a successor first.** The sole member can leave and trigger Shop cleanup, including a sole noncreator after the creator has already departed. Existing self-removal/creator-removal protections remain.

POST `/shops/invite-codes`: omit `max_uses` and `expires_at`, or send null. Any non-null value, including zero or empty string, is rejected rather than ignored. Supported request:

```json
{"shop_id":"11111111-1111-4111-8111-111111111111","max_uses":null,"expires_at":null}
```

Unsupported request:

```json
{"shop_id":"11111111-1111-4111-8111-111111111111","max_uses":1}
```

HTTP 400 response without negotiation:

```json
{"status":400,"data":null,"message":"invite expiry and max-use controls are not supported"}
```

Client requirement: hide unsupported limited/expiring-invite controls, avoid sending default 0/empty controls, keep the current role on duplicate join, implement successor promotion before last-admin exit. GET invites has no body and keeps its existing array response. DELETE `/shops/invite-codes/:code_id` deactivates (does not permanently delete); `/.../:code_id/delete` permanently deletes. Both are current-admin operations and retain nested success messages; do not label both operations identically.

## 5. Lists/items: validation, batch deletion and actual counts

**Why:** invalid nested items could escape validation; stale IDs caused an entire bulk removal to fail; requested count differed from actual removed count. Current list policy must be checked at mutation time.

POST `/shops/lists` request body:

```json
{"shop_id":"11111111-1111-4111-8111-111111111111","description":"Filters"}
```

PUT `/shops/lists` uses `{list_id,description}`; DELETE uses `{list_id}`. Existing success/model shapes remain. A linked list cannot be deleted while in use; refresh its references rather than retrying unchanged (`list_in_use` on negotiated failures).

POST `/shops/lists/items` requires nonblank NIIN/nomenclature and positive int32 quantity. Nickname may be null/empty but at most 50 runes; a supplied unit must be nonblank and at most 50 raw runes. Unit codes are not allow-listed. PUT adds required `item_id` and retains the same validation. Request body:

```json
{"list_id":"22222222-2222-4222-8222-222222222222","niin":"012345678","nomenclature":"Filter","quantity":2,"nickname":"Oil filter","unit_of_measure":"EA"}
```

POST `/shops/lists/items/bulk` has outer `list_id` and an array of item bodies. Send the same `list_id` on each nested item to satisfy the existing binding contract. The handler takes targets from the nested items (the required outer ID does not replace them); the repository authorizes every selected list and rejects cross-Shop sets. Multiple authorized lists in one Shop are supported by the repository. All items validate before mutation; one invalid item rejects the whole batch. A valid bulk body:

```json
{"list_id":"22222222-2222-4222-8222-222222222222","items":[{"list_id":"22222222-2222-4222-8222-222222222222","niin":"012345678","nomenclature":"Filter","quantity":2,"nickname":null,"unit_of_measure":null}]}
```

A negotiated invalid-item response is HTTP 400:

```json
{"status":400,"code":"invalid","data":null,"message":"invalid request"}
```

The unnegotiated validation shape on these handlers is flat:

```json
{"message":"invalid request","details":"invalid request"}
```

DELETE `/shops/lists/items/bulk` body:

```json
{"item_ids":["33333333-3333-4333-8333-333333333333","33333333-3333-4333-8333-333333333333","44444444-4444-4444-8444-444444444444"]}
```

If the first row exists and the second unique ID is already gone, HTTP 200 is **flat**, and count is 1:

```json
{"message":"Items removed successfully","count":1}
```

Duplicates are deduplicated; missing IDs are skipped; all missing returns count 0 after authentication. Surviving rows must pass current authorization and belong to one Shop (multiple lists within it are allowed); a foreign survivor rejects the entire transaction. Empty selection is invalid. Single DELETE `/shops/lists/items` uses `{item_id}` and retains strict missing-row behavior and its standard nested success envelope.

Client requirement: validate every item, handle flat validation/delete responses, reconcile by actual count rather than selection length, and tolerate a successful zero-count bulk removal. Do not generalize this idempotency to single deletes or arbitrary other writes.

## 6. Vehicles: usage versus metadata intent

**Why:** legacy tracked-usage payloads could overwrite admin/base metadata with stale snapshots; metadata admin changes were lost; negative base usage was accepted.

POST `/shops/vehicles` still requires `shop_id` and nonempty `admin`; base `mileage`/`hours` must be nonnegative. Existing omitted values/defaults remain. Example body:

```json
{"shop_id":"11111111-1111-4111-8111-111111111111","admin":"A-10","model":"Truck","niin":"012345678","serial":"SN-1","uoc":"UNK","mileage":100,"hours":10,"comment":""}
```

HTTP 201 returns the existing vehicle model in `data` with `message:"Vehicle created successfully"`. The model keys are `id,creator_id,niin,admin,model,serial,uoc,mileage,hours,comment,save_time,last_updated,shop_id,tracked_mileage,tracked_hours`; tracked values can be null. GET routes retain that shape.

PUT `/shops/vehicles` requires `vehicle_id`. A payload with at least one non-null tracked field and all five detail fields (`niin,model,serial,uoc,comment`) absent/null/empty is classified as **tracked usage**. In this branch any supplied `admin,mileage,hours` snapshots are ignored; current membership is required. Prefer a minimal request:

```json
{"vehicle_id":"55555555-5555-4555-8555-555555555555","tracked_mileage":120}
```

Metadata requests require creator/admin authority. Omitted/null pointer fields preserve stored values. Explicit empty detail strings clear them; empty UOC maps to `UNK`; empty `admin` is invalid. Supplied nonnegative base values and a supplied nonempty admin are applied. Example:

```json
{"vehicle_id":"55555555-5555-4555-8555-555555555555","admin":"A-11","model":"Truck","comment":"","mileage":110}
```

Both successful PUT forms return:

```json
{"status":200,"message":"","data":{"message":"Vehicle updated successfully"}}
```

Client requirement: omit fields being preserved. Be particularly careful combining tracked usage with empty detail fields: that combination selects usage-only behavior and cannot be used to clear all metadata. Separate user actions and submit explicit intent through the documented routes. Re-read after an uncertain response.

PATCH `/shops/vehicles/:vehicle_id/usage` remains the explicit relative adjustment route, not an idempotent replay API. Strict request body example:

```json
{"operation":"add","mileage_adjustment":5,"hours_adjustment":null}
```

Operations are `add` or `subtract`; magnitudes follow the existing validation/overflow/nonnegative-result checks. Success is HTTP 200 with the complete updated vehicle model and `message:"Equipment usage adjusted successfully"`. Never automatically resend an uncertain adjustment. DELETE retains `data:{message:"Vehicle deleted successfully"}` and current creator/admin authorization; asynchronous asset cleanup/audit handling is server work.

## 7. Notifications, attachment intent and direct-item metadata

**Why:** invalid type/item inputs, ambiguous parent IDs and omission handling could lose metadata or target the wrong resource. Retention now preserves enrichment across a delete/re-add without resurrecting an item or its quantity.

POST `/shops/vehicles/notifications` requires matching authorized `shop_id`/`vehicle_id`, nonblank title, and exactly `M1`, `PM` or `MW`. Request body:

```json
{"shop_id":"11111111-1111-4111-8111-111111111111","vehicle_id":"55555555-5555-4555-8555-555555555555","title":"Replace filter","description":"Next service","type":"M1","completed":false,"attached_shop_list":null}
```

HTTP 201:

```json
{"status":201,"message":"Notification created successfully","data":{"id":"66666666-6666-4666-8666-666666666666","shop_id":"11111111-1111-4111-8111-111111111111","vehicle_id":"55555555-5555-4555-8555-555555555555","title":"Replace filter","description":"Next service","type":"M1","completed":false,"save_time":"2026-10-04T12:00:00Z","last_updated":"2026-10-04T12:00:00Z","attached_shop_list":null}}
```

PUT `/shops/vehicles/notifications` still sends the complete legacy title/description/type/completed details. **Only attachment has three-state intent:** omit `attached_shop_list` to preserve; null to detach; a valid same-Shop list ID to replace. Example detach body:

```json
{"notification_id":"66666666-6666-4666-8666-666666666666","title":"Replace filter","description":"Next service","type":"M1","completed":false,"attached_shop_list":null}
```

HTTP 200:

```json
{"status":200,"message":"","data":{"message":"Notification updated successfully"}}
```

POST `/shops/notifications/items` body:

```json
{"notification_id":"66666666-6666-4666-8666-666666666666","niin":"012345678","nomenclature":"Filter","quantity":2,"nickname":"Oil filter","unit_of_measure":"EA"}
```

HTTP 201:

```json
{"status":201,"message":"Item added successfully","data":{"id":"77777777-7777-4777-8777-777777777777","shop_id":"11111111-1111-4111-8111-111111111111","notification_id":"66666666-6666-4666-8666-666666666666","niin":"012345678","nomenclature":"Filter","quantity":2,"save_time":"2026-10-04T12:00:00Z","nickname":"Oil filter","unit_of_measure":"EA"}}
```

POST `/shops/notifications/items/bulk` body:

```json
{"notification_id":"66666666-6666-4666-8666-666666666666","items":[{"niin":"012345678","nomenclature":"Filter","quantity":2,"nickname":null,"unit_of_measure":null}]}
```

Nested `notification_id` may be omitted/null (inherits outer) or must match outer. All items validate before effects. Success returns a standard HTTP 201 with the item array and `message:"Items added successfully"`. Validation/flat legacy error examples are those in section 5. NIIN/nomenclature must be nonblank; quantity positive; nickname/unit follow section 5's raw 50-rune rules.

For direct notification items, omitted/null nickname or unit resolves retained values scoped to **the same logical notification and exact NIIN**, even if a new item UUID is used. Explicit values override the respective metadata. It does not restore the deleted row, quantity or unrelated notification's values. Conflicting retained/live candidates refuse rather than choose silently: contract-2 conflict versus the preserved legacy failure. Resolve ambiguity with explicit metadata/owner-approved historical repair; do not retry the unchanged request forever. Null display fallback remains nickname `""`, unit `"EA"`; this does not mean the stored value was rewritten.

DELETE `/shops/notifications/items/bulk` uses `{item_ids:[...]}` and the same **flat** actual-count response as section 5. Deduplicate/stale-ID semantics apply, but surviving items must share **one notification**. Single DELETE `/shops/notifications/items/:item_id` has no body and stays strict. Notification DELETE has no body and returns `data:{message:"Notification deleted successfully"}`. GET item/notification routes keep their existing model/array schemas; combined reads are described in section 11.

Client requirement: preserve attachment omission, send valid exact types, validate every nested item, support raw optional metadata and actual bulk counts, and keep a draft when an ambiguity/permission failure occurs.

## 8. Capabilities and atomic notification save

**Why:** flag-only advertisement and writes without schema readiness could expose an incompatible save path. The handler now independently checks readiness. Receipt replay must survive deployment and older clients omitting enrichment.

GET `/shops/capabilities` requires contract header 2, no body, and responds with `Cache-Control: no-store`. Example HTTP 200 when deployment flags are off:

```json
{"status":200,"message":"","data":{"contract_version":2,"typed_errors":false,"atomic_notification_save":false,"service_dates":false,"service_reads":false,"message_sync":false}}
```

Atomic save is advertised only when its flag **and** readiness succeed; message sync likewise requires configured readiness/flag. The optional legacy typed-error negotiation exists even though its broad capability stays false. `service_dates` and `service_reads` remain false. Do not infer supported date mapping from the presence of equipment-service routes. Refresh discovery for the current authenticated session/deployment; a previous true is not a guarantee every instance is ready.

POST `/shops/vehicles/notifications/save` requires contract 2 and strict JSON at most 1 MiB. Unknown fields/trailing documents are rejected. IDs must be canonical lowercase nonzero UUIDs; direct item IDs are client-provided and unique. `items:[]` is valid complete replacement; omitted/null items is invalid. Linked-list items are represented only through attachment, not copied into the direct items array. Request body:

```json
{"operation_id":"88888888-8888-4888-8888-888888888888","shop_id":"11111111-1111-4111-8111-111111111111","vehicle_id":"55555555-5555-4555-8555-555555555555","notification_id":null,"details":{"title":"Replace filter","description":"Next service","type":"M1","is_completed":false},"attachment":{"intent":"keep","list_id":null},"items":[{"id":"77777777-7777-4777-8777-777777777777","niin":"012345678","nomenclature":"Filter","quantity":2}]}
```

Attachment intents are **`keep`, `remove`, `attach`**. Keep/remove require null list ID; attach requires a valid list ID. These names differ from the legacy nullable attachment field.

HTTP 200 receipt:

```json
{"status":200,"message":"","data":{"operation_id":"88888888-8888-4888-8888-888888888888","notification_id":"66666666-6666-4666-8666-666666666666","committed_at":"2026-10-04T12:00:00Z","replayed":false}}
```

Retrying the same semantic payload and operation identity can return that receipt with `replayed:true`; changing semantic payload under the same operation ID conflicts. The receipt is a commit acknowledgment, not a fresh notification snapshot: fetch authoritative state rather than replacing later edits with stale draft contents. Omitted/null enrichment preserves retained metadata and the old semantic fingerprint.

Unavailable schema/service returns HTTP 503 without a receipt/write:

```json
{"status":503,"code":"unsupported_contract","data":null,"message":"Notification save unavailable"}
```

Client requirement if using this route: gate discovery, persist operation ID/item IDs and draft across unavailable or uncertain outcomes, retry with the same semantic payload, and start a new operation identity only for an intentional new operation. Do not silently fall back to multiple legacy writes after an uncertain atomic call. Parser/released artifact/device acceptance remains unexecuted.

## 9. Legacy messages, parents, timestamp cursors and images

**Why:** generated storage fields could leak into the response; cross-Shop parent IDs and user-selected image deletion targets could violate ownership; equal timestamps and large newer bursts could skip rows; missing anchors needed an explicit reload signal.

POST `/shops/messages` body:

```json
{"shop_id":"11111111-1111-4111-8111-111111111111","message":"Inspection complete","parent_id":null}
```

If supplied, parent must belong to the same Shop. HTTP 201 has exactly the explicit nine message keys:

```json
{"status":201,"message":"Message created successfully","data":{"id":"99999999-9999-4999-8999-999999999999","shop_id":"11111111-1111-4111-8111-111111111111","user_id":"firebase-user-a","message":"Inspection complete","created_at":"2026-10-04T12:00:00Z","updated_at":"2026-10-04T12:00:00Z","is_edited":false,"parent_id":null,"author_username":"Technician"}}
```

Read/send/aggregate message DTOs never include storage `insertion_number`. Nullable fields remain present. Historical IDs are opaque TEXT; do not assume every legacy ID parses as UUID. PUT `/shops/messages` body is `{message_id,message}`; DELETE `/shops/messages/:message_id` has no body. Success retains nested `Message updated successfully`/`Message deleted successfully`. Image-marker references in a send/edit must match registered ready managed assets in that Shop; published/referenced assets are protected from discard. Unknown historical assets are not adopted for deletion.

GET `/shops/:shop_id/messages/paginated`: no body. Page/limit retain their defaults (1/20, max limit 100). Use either `before_id` or `after_id`, never both. Query example:

```json
{"limit":20,"after_id":"99999999-9999-4999-8999-999999999999"}
```

Cursor selection orders by the timestamp/ID tuple, including ties. For newer reads the nearest newer batch is selected, then displayed descending. **Advance using returned `next_cursor`, not the last displayed row.** Pagination metadata and next cursor are optional, not explicit null. Empty terminal cursor-page example:

```json
{"status":200,"message":"","data":{"messages":[]}}
```

Missing/deleted/null-time legacy anchor with header 2 returns HTTP 409:

```json
{"status":409,"code":"reset_required","data":null,"message":"Message cursor is unavailable; reload messages"}
```

Without header 2 it preserves HTTP 500:

```json
{"status":500,"data":null,"message":"Message cursor is unavailable; reload messages"}
```

Client requirement: reset/refetch the message view on that failure, deduplicate by ID while draining, and do not assume legacy timestamp polling is lossless. **Open limitation:** a transaction committing late with an older timestamp can land behind a cursor already observed. This remediation preserves timestamp semantics; a monotonic-write policy or a different synchronization contract requires a separate owner decision.

POST `/shops/messages/image/upload?shop_id=...` is multipart/form-data: exactly one file part named `file`, optional single `shop_id` form value, no extra fields. File is at most 5 MiB; total request at most 6 MiB. Query and form Shop IDs must agree if both supplied and be valid nonzero UUIDs. Failed validation performs no Azure upload. JSON representation of multipart values (documentation only):

```json
{"shop_id":"11111111-1111-4111-8111-111111111111","file":{"filename":"photo.jpg","content_type":"image/jpeg","bytes":"binary multipart content"}}
```

HTTP 200 response:

```json
{"status":200,"message":"Image uploaded successfully","data":{"message_id":"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa","shop_id":"11111111-1111-4111-8111-111111111111","image_url":"https://exampleaccount.blob.core.windows.net/shop-message-images/11111111-1111-4111-8111-111111111111/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa.jpg","file_extension":".jpg"}}
```

`message_id` here is the **upload/asset UUID**, not a posted message's ID. Keep returned URL and ID; use the URL in the existing `[IMAGE:URL]` marker, then send the message. Do not construct a deletion target from URL text.

DELETE `/shops/messages/image/:message_id?shop_id=...` has no body; path ID is that returned upload ID. Only its uploader or current Shop admin, while a current member, can discard a registered unattached upload. A published asset is denied. Success:

```json
{"status":200,"message":"","data":{"message":"Image deleted successfully"}}
```

This acknowledges durable cleanup scheduling, not proof Azure bytes are already gone. Worker retries/recovery and cascade cleanup are server responsibilities. Client requirement: resize/limit uploads, send the narrow multipart shape, keep upload identity until attachment/discard outcome is known, and stop using arbitrary blob/message IDs for discard.

## 10. Numeric message sync: readiness, bounds and recovery

**Why:** numbering/counter corruption, null timestamps and ahead-of-counter cursors could be accepted as a misleading complete stream. The allocator bridge preserves mixed legacy-writer ordering; readers now refuse invalid integrity state.

All four messages-v2 routes require contract 2 and current Shop membership. Use advertised `message_sync`; route presence is not activation. Limits are 1–100 (initial/history default 50, catch-up 100). Duplicate/unknown query keys are rejected. History cursors are opaque server values (max 1024 characters), not message IDs. Decimal watermarks are **strings** to avoid JSON integer precision loss.

GET `/shops/:shop_id/messages-v2/initial`: no body; query example:

```json
{"limit":50}
```

HTTP 200 example for an empty Shop:

```json
{"status":200,"message":"","data":{"rows":[],"older_cursor":null,"has_older":false,"watermark":"0"}}
```

GET `/shops/:shop_id/messages-v2/history`: no body; query uses returned `cursor` and optional limit. HTTP 200 terminal example:

```json
{"status":200,"message":"","data":{"rows":[],"next_cursor":null,"has_more":false}}
```

GET `/shops/:shop_id/messages-v2/catch-up`: no body; query example:

```json
{"after":"10","through":"20","limit":100}
```

HTTP 200 terminal response example where that interval contains no surviving rows:

```json
{"status":200,"message":"","data":{"rows":[],"next_after":"20","through":"20","has_more":false}}
```

Hold the returned `through` bound while draining with returned `next_after` until `has_more:false`. If initially omitted, use the server's returned bound. Reversed after/through is invalid. Rows use section 9's nine-field message DTO.

POST `/shops/:shop_id/messages-v2/reconcile` strict body, at most 16384 bytes and 100 input IDs:

```json
{"ids":["99999999-9999-4999-8999-999999999999"]}
```

Empty array is valid; omitted/null is invalid. UUIDs are normalized/deduplicated. If the requested message is deleted, HTTP 200:

```json
{"status":200,"message":"","data":{"rows":[],"missing_ids":["99999999-9999-4999-8999-999999999999"]}}
```

Numbering/timestamp/counter integrity failure (including corrupt rows outside the selected chunk) returns HTTP 503:

```json
{"status":503,"code":"unsupported_contract","data":null,"message":"Message synchronization is unavailable"}
```

An ahead-of-counter cursor returns HTTP 409:

```json
{"status":409,"code":"message_sync_reset_required","data":null,"message":"Message synchronization requires a new initial snapshot"}
```

Client requirement: treat 503 as unavailable, preserve sync/draft identity, and start a fresh initial snapshot on reset-required rather than retrying the bad cursor. Reconciliation handles edits/deletions; catch-up alone is not a complete mutation log. **Open limitation:** a restore that reuses an in-range numeric watermark is not detectable by this contract (ABA). An epoch/recovery protocol and client recovery design remain an owner decision; operational restore fencing alone does not add that protocol.

## 11. Aggregate, history and audit reads

**Why:** separate reads could combine headers, items, counts and membership from different moments; large complete histories exceeded PostgreSQL's bind-parameter ceiling; legacy audit snapshots lacked item enrichment.

Affected aggregate routes and the legacy notifications-with-items route now use one caller-context read-only repeatable-read snapshot. Large exact selected sets use bounded array bindings. Primary wire schemas, optional limits/unbounded defaults and timestamp interpretation remain. This does not introduce client pagination or reduce unlimited responses.

GET `/shops/:shop_id/lists-with-items`: no body. Optional query example:

```json
{"lists_limit":20,"items_limit":50}
```

HTTP 200 empty response:

```json
{"status":200,"message":"Shop lists with items retrieved successfully","data":{"lists":[],"counts":{"lists":0,"items":0},"limits":{"lists":20,"items_per_list":50}}}
```

GET `/shops/bootstrap` has no body and retains `data:{shops:[...]}`. GET `/shops/:shop_id/snapshot` retains `data:{shop,vehicles,lists,notifications,messages,services,recent_changes,limits}`. GET `/shops/vehicles/:vehicle_id/maintenance-snapshot` retains `data:{vehicle,notifications,recent_changes,services,counts,limits}`. GET `/shops/vehicles/:vehicle_id/notifications-with-items` retains the notification-with-items array. Empty legacy combined-read response:

```json
{"status":200,"message":"","data":[]}
```

GET `/shops/equipment-pmcs-history` has no body, includes complete accessible equipment history, and retains the source-specific optional PMCS fields:

```json
{"status":200,"message":"Equipment PMCS history retrieved successfully","data":{"equipment":[],"count":0}}
```

GET `/shops/equipment/overview` likewise retains its `data:{shops:[...]}` overview. No parser migration is required solely for the snapshot/binding repairs. Avoid assuming counts equal limited-array lengths or that unlimited responses are small: a complete 65536-inspection PMCS fixture produced about 25.9 MB JSON and 656.7 MB cumulative allocation/request; this is not a fleet memory budget.

All three notification changes routes retain nullable deleted-resource IDs and denormalized title/type/admin. Event-time item snapshots now retain identity, nickname/unit including null values. **Client requirement:** accept extra enrichment in `field_changes`, render history even when notification/vehicle/actor references are null, and do not require a deleted parent to fetch/render an otherwise authorized Shop history. Representative array element in a standard HTTP 200 response:

```json
{"status":200,"message":"","data":[{"id":"bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb","notification_id":null,"shop_id":"11111111-1111-4111-8111-111111111111","vehicle_id":null,"changed_by":null,"changed_by_username":"Unknown User","changed_at":"2026-10-04T12:00:00Z","change_type":"delete","field_changes":{"fields_changed":["deleted"]},"notification_title":"Replace filter","notification_type":"M1","vehicle_admin":"A-10","is_deleted":true}]}
```

Legacy audit remains best effort; a committed primary mutation is not rolled back for its audit gap. Common sanitized server warnings require operator alerting. Atomic save requires its transactional audit/receipt. No new mandatory legacy outbox or client retry protocol was added.

## 12. Equipment services: authority, completion history and query filters

**Why:** a foreign Shop admin could modify another Shop's service; repeated completion could overwrite historical dates; some listing filters/counts were ignored; date/cancellation failures were swallowed.

POST `/shops/:shop_id/equipment-services` body example:

```json
{"equipment_id":"55555555-5555-4555-8555-555555555555","list_id":"","description":"Inspect brakes","service_type":"inspection","is_completed":false,"service_date":"2026-10-10T12:00:00Z","service_hours":null,"completion_date":null}
```

Equipment and nonempty list must belong to the path Shop. Description is required and at most 500 characters; supplied service hours must be nonnegative. Service type remains a required free-form string. PUT `/shops/:shop_id/equipment-services/:service_id` sends metadata fields plus required body `service_id`; it does not send `equipment_id`. Binding checks body `service_id` before the handler overwrites it with the authoritative path ID. Send the path's ID in both places; omitting the body field is still invalid. GET returns the existing service DTO. Create/update/complete responses retain that DTO, including `created_by_username`, nullable service/completion dates and hours; no new civil-date mapping is implied.

PUT request body example:

```json
{"service_id":"cccccccc-cccc-4ccc-8ccc-cccccccccccc","list_id":"","description":"Inspect brakes","service_type":"inspection","is_completed":true,"service_date":"2026-10-10T12:00:00Z","service_hours":null,"completion_date":null}
```

POST `/shops/:shop_id/equipment-services/:service_id/complete` body:

```json
{}
```

First completion of an incomplete service defaults completion time inside the locked transaction. Repeated completion with omitted/null date preserves the historical date, including an existing null. An explicit RFC3339 `completion_date` updates it. PUT with `is_completed:false` reopens and clears completion date. **The legacy update field is a plain bool:** omission also defaults false. A metadata editor keeping a service completed must send `is_completed:true`; this remediation does not change PUT into a general PATCH.

Typical successful completion's standard `data` DTO:

```json
{"status":200,"message":"Equipment service completed successfully","data":{"id":"cccccccc-cccc-4ccc-8ccc-cccccccccccc","shop_id":"11111111-1111-4111-8111-111111111111","equipment_id":"55555555-5555-4555-8555-555555555555","list_id":"","description":"Inspect brakes","service_type":"inspection","created_by":"firebase-user-a","created_by_username":"Technician","is_completed":true,"created_at":"2026-10-04T11:00:00Z","updated_at":"2026-10-04T12:00:00Z","service_date":"2026-10-10T12:00:00Z","service_hours":null,"completion_date":"2026-10-04T12:00:00Z"}}
```

Both GET `/shops/:shop_id/equipment-services` and `/shops/:shop_id/equipment/:equipment_id/services` now apply status/type/completion/date/equipment filters together and count the same filtered snapshot. Query example, no body:

```json
{"status":"completed","service_type":"inspection","is_completed":true,"start_date":"2026-10-01T00:00:00Z","end_date":"2026-10-31T23:59:59Z","limit":50,"offset":0}
```

Limit is 1–1000 (default 1000); offset nonnegative (default 0). Status absent/blank means no status filter; unknown nonblank status is invalid. `completed` means completed; `overdue` means incomplete with date strictly before request time; `due_soon` means incomplete date strictly after request time through seven elapsed days; `scheduled` means incomplete future date, including due-soon. Exact equality with now is neither overdue nor future. Filters are **conjunctive**, so contradictory combinations correctly return zero rows. An explicit empty equipment filter matches empty equipment ID, not absence of a filter. Date range comparisons use existing service timestamp semantics; do not substitute completion timestamps.

Empty HTTP 200 shop listing example:

```json
{"status":200,"message":"Services retrieved successfully","data":{"services":[],"total_count":0,"has_more":false}}
```

Rows/count use one snapshot and deterministic ID tie-breakers; offset pages can still shift between separate requests. NULL sort behavior is preserved. `start_date`/`end_date` must be RFC3339 when provided; malformed strings fail safely. Legacy Shop/calendar service errors may still use HTTP 500 while negotiated invalid errors use 400; the equipment-specific query handler has its existing flat binding-error response. Do not infer one uniform error parser/status from just the example above.

GET `/shops/:shop_id/equipment-services/calendar` requires start and end RFC3339 query values, with optional equipment ID. Existing `data` is `{date_range:{start_date,end_date},services,total_count}`. GET `/overdue` and `/due-soon` default limit 50, maximum 200; due-soon `days_ahead` defaults 7 and must be 1–30 (0 invalid). Query example:

```json
{"days_ahead":7,"limit":50}
```

Empty due-soon HTTP 200 example:

```json
{"status":200,"message":"Due soon services retrieved successfully","data":{"due_soon_services":[],"total_count":0}}
```

Overdue uses `data:{overdue_services,total_count}` and per-row `days_overdue`; due-soon rows include `days_until_due`. These fixes honor existing timestamp/date semantics. `service_dates`/`service_reads` activation remains separately gated. Client requirement: preserve completion state/date, send valid conjunctive filters, correctly render smaller filtered results, and handle post-commit response uncertainty as described in section 1.

## Remaining manual implementation and release work

**Completed local server implementation:** the approved remediation and final review fixes. Server-only tests passed before integration; the merged-tree results are recorded separately in [integration verification](shops-server-cleanup-integration.md). This document does not assert client changes are already implemented or that production is ready.

**Potential further code/design work requiring owner decisions:**

1. Legacy late-commit cursor semantics: choose whether to keep the explicit limitation or implement a reviewed monotonic-write/synchronization design. No waiver is inferred.
2. Numeric restore ABA: specify epoch/restore/client-recovery protocol if lossless recovery across in-range watermark reuse is required. Current ahead-watermark reset does not solve it.
3. Invitation abuse: supply active-code population, instance count, authenticated UID/edge limits and acceptable numeric budget. A bounded fleet-aware UID limiter may require implementation if current enforcement is insufficient; none was invented here.
4. Client handling from sections 1–12 must be checked/implemented by the client owner. Atomic/numeric sync upgrades remain optional and capability-gated; released parser, request, retry and device evidence is still required before activation.
5. Date/read capabilities need a separately approved two-way schema/timezone/date mapping and client activation implementation if desired; both flags remain false.

**Operator/release work rather than missing local server fixes:** separately authorized target inventory/data resolution, migrations 019–023 on `miltech_ng_test` first and then separately on `miltech_ng`, tagged Jet regeneration after every database action, actual application-role write/privilege checks, preservation of all 32 tracked PMCS outputs, actual TMDE view/build provenance, fleet/external-writer fencing, actual Docker context/layers/nonroot runtime/Firebase mount/writable startup generation checks, separate confirmations for both credential incidents, observability/alerts, signed/released artifacts and physical device acceptance, and push/deploy/flag activation. Production capacity/concurrency budgets remain unapproved. Named targets are UNPINNED; no named migration, push, deployment or activation occurred in this local integration.

Nonblocking negative-test depth follow-ups are retained in [acceptance](shops-server-remediation-acceptance.md); they are not claimed as implemented. The precise target/operator gates remain in [release gates](shops-server-remediation-release.md). The design/audit remain the historical justification; this handoff describes the resulting server contract.

Integration also identified an intermittent cleanup-test activity assertion (not reproduced with diagnostics) and a container-verification scanner that expands embedded sparse TAR fixtures. Final unmodified server/migration and physical race reruns passed; the scanner repair remains uncommitted in the isolated worktree after the user's tooling-scope correction. See [integration verification](shops-server-cleanup-integration.md). These are verification follow-ups, not additional endpoint changes or a claim that runtime acceptance passed.
