# Custom PMCS Shop Inspection: Mobile Workflow Handoff

This is the consumer workflow for recording a User-PMCS checklist against a
Shop vehicle. The API reference is docs/api/pmcs_sbs_inspections_mobile.md;
this document focuses on durable client behavior, not a duplicate endpoint list.

## Domain boundary and captured provenance

Use /api/v1/auth/pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id for guide and
custom inspections. There is no /custom-pmcs route. equipment_id is a Shops
vehicle, and missing/non-visible vehicles intentionally share 404 behavior.

## Workflow operation map

| Method | Path suffix | Workflow use |
| --- | --- | --- |
| PUT | /pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id | Save metadata or an explicit clean completion. |
| GET | /pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id | Reconcile detail, faults, and comments. |
| DELETE | /pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id | Delete inspection and dependent faults/comments. |
| GET | /pmcs-sbs/equipment/:equipment_id/pmcs | History list; limit defaults to and caps at 1000, offset defaults to 0. |
| PUT | /pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id/faults | Create/update fault; first valid write materializes inspection. |
| DELETE | /pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id/faults | Delete one fault key; message-only success. |
| DELETE | /pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id/faults/bulk | Delete 1-100 keys; message plus requested/deleted counts. |
| POST | /pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id/comments | Create a non-idempotent comment. |
| PUT | /pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id/comments/:comment_id | Edit caller-owned comment. |
| DELETE | /pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id/comments/:comment_id | Replace caller-owned text with deleted text; standard status/message/data response. |

User-PMCS content-sync cursors and ETags are opaque content-sync values. They
are unrelated to Shop inspection mutations and must never be reused as
inspection validators, retry tokens, or ordering keys. Shop inspection writes
have no ETag, conditional write, server queue, or outbox.

At inspection start, durably capture Shop vehicle id, generated pmcs_id, custom
checklist id, revision id, revision number, and displayed checklist name. The
authored checklist tree is not copied. That immutable snapshot provenance
survives source content retirement/deletion, but detail cannot reconstruct the
retired authored tree.

## Start, clean completion, and notes

1. Generate/persist one UUID pmcs_id before any write.
2. Persist the complete source tuple with it; never replace a started custom
   inspection with a newer revision or renamed checklist.
3. Use inspection PUT even with zero faults to record a clean completion.
4. A first fault PUT can lazily materialize the inspection atomically.
5. Retry with same UUID and source tuple. A 409 means stop automatic replay.
6. GET detail to reconcile faults/comments; list is newest performed_date first.

Custom inspection save:

~~~json
{
  "source_type": "custom",
  "custom_checklist_id": "22222222-2222-2222-2222-222222222222",
  "custom_revision_id": "33333333-3333-3333-3333-333333333333",
  "custom_revision_number": 7,
  "custom_checklist_name": "Weekly Generator PMCS",
  "performed_date": "2026-08-07T14:30:00Z",
  "notes": "Finished after generator warmup"
}
~~~

Guide remains backward compatible: omit source_type and send guide_manual plus
performed_date/optional notes. Custom responses emit source_type custom and
custom_* fields but omit guide_manual. Guide responses emit source_type guide,
include guide_manual, and omit custom provenance. Nullable notes and performer
fields are omitted, not null.

~~~json
{
  "guide_manual": "pmcs_sbs/hmmwv/file.json",
  "performed_date": "2026-08-07T14:30:00Z"
}
~~~

The guide result includes source_type guide and guide_manual only. The custom
result includes source_type custom and custom_checklist_id,
custom_revision_id, custom_revision_number, and custom_checklist_name only.

performed_date is required RFC 3339 input. Notes are inspection-only:
omitted/null/blank clears them; non-empty text is limited to 4,000 bytes. A
fault save has no notes and clears saved notes, so make the final inspection
PUT after fault writes if notes matter.

## Custom fault, reset, and retries

Every custom fault repeats the exact source tuple and custom section context:

~~~json
{
  "source_type": "custom",
  "custom_checklist_id": "22222222-2222-2222-2222-222222222222",
  "custom_revision_id": "33333333-3333-3333-3333-333333333333",
  "custom_revision_number": 7,
  "custom_checklist_name": "Weekly Generator PMCS",
  "performed_date": "2026-08-07T14:30:00Z",
  "section_id": "before-operation",
  "section_title": "Before Operation",
  "item_index": 0,
  "item_no": "1",
  "status": "X",
  "fault_text": "Hydraulic leak",
  "corrective_action": "Replace hose"
}
~~~

A guide fault omits source_type/custom fields/unavailable section title and
uses guide_manual. Fault identity is (pmcs_id, section_id, item_index);
replaying it updates in place. Title is optional; blank omits it. Checklist
name/title are trimmed and capped at 200 Unicode grapheme clusters and 8 KiB.
Status normalizes to x, slash, or dash.

Delete one fault using {"section_id":"before-operation","item_index":0}. For
batch reset send 1 through 100 unique keys:
{"faults":[{"section_id":"before-operation","item_index":0}]}. Both delete
operations are retry-tolerant; individual delete is message-only success, while
bulk returns requested_count and actual deleted_count, which may be lower on
replay. Invalid, duplicate, empty, negative, or too-many keys are 400.
Inspection deletion uses DELETE /pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id
and returns {"message":"Inspection deleted"}; use it only when pending
fault/comment work for that inspection is intentionally abandoned.

## Comments, privacy, and offline order

Detail returns faults/comments only for a caller who can see the Shop vehicle.
Create comment with {"text":"Needs supervisor review"}; text is trimmed,
required, and capped at 2,000 bytes. Comment POST is not idempotent: it creates
a server-generated id. On an ambiguous timeout or lost response, do not blindly
retry. First GET the inspection and reconcile a locally persisted author, text,
and approximate creation time; if that cannot establish whether it succeeded,
ask the operator before a duplicate-risk retry. Persist the returned comment id
before update/delete. Only original author can update/delete; comment delete
visibly changes text to Deleted by user rather than erasing history and returns
the standard status/message/data envelope. Non-author is 403, unknown comment
404. Do not display cached Shop inspection data after visibility changes without
fresh detail.

Keep a durable local queue in exact causal order per inspection and fault key;
do not globally replay all upserts before all deletes. For example, upsert A,
delete A, upsert A must remain in that order. Coalesce only when the final
desired state for the same unchanged key is known exactly. A final inspection
note write follows fault writes because they clear notes. A comment update/delete
follows its confirmed create because it needs the returned server id. There is
no server outbox to recover a client queue.

On 400 retain local state for correction, not blind retry. On 404 refresh Shop
visibility without inferring missing data. On 409 halt automatic replay and ask
the operator to reconcile identity; never silently substitute a current
User-PMCS revision.

Inspection/fault bodies are capped at 8 MiB, require one JSON value with no
unknown/trailing content, and reject oversized or raw-invalid UTF-8 input as
{"message":"invalid request body"} without echoing authored text. This
describes the current server contract; it does not claim deployment, production
migration, mobile implementation, or end-to-end device verification.
