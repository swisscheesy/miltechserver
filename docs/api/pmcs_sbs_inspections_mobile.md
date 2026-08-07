# PMCS SBS Shop Inspection API

Base path: /api/v1/auth. Every endpoint requires an authenticated Firebase user.
equipment_id is a Shops vehicle (shop_vehicle.id); pmcs_id is a client-generated
UUID. Guide and custom inspections use the same endpoints. There is no
/custom-pmcs route.

A caller must be a member of the vehicle Shop. An absent vehicle and one the
caller cannot view both return 404 {"message":"pmcs sbs equipment not found"}.
Do not use this API to disclose Shop membership or vehicle data.

## Endpoint index

| Method | Path | Purpose |
| --- | --- | --- |
| PUT | /pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id | Create/update metadata; record zero-fault completion. |
| GET | /pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id | Read inspection, faults, and comments. |
| DELETE | /pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id | Delete inspection and dependent data. |
| GET | /pmcs-sbs/equipment/:equipment_id/pmcs | List vehicle inspections. |
| PUT | /pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id/faults | Create/update a fault; lazily creates inspection. |
| DELETE | /pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id/faults | Delete one fault key. |
| DELETE | /pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id/faults/bulk | Delete 1 through 100 distinct keys. |
| POST | /pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id/comments | Create comment. |
| PUT | /pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id/comments/:comment_id | Edit author-owned comment. |
| DELETE | /pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id/comments/:comment_id | Replace author-owned comment with deleted text. |

Successful inspection, fault, and comment reads/saves use the standard
{"status":200|201,"message":"...","data":...} envelope. Deletes and errors
retain the established {"message":"..."} envelope.

## Source contract and metadata

Every response explicitly emits source_type: guide or custom. Source
provenance is immutable after the first save for a pmcs_id. A retry with the
same vehicle/source tuple is idempotent; changing vehicle, source type, guide
manual, or custom provenance returns 409
{"message":"pmcs sbs inspection conflict"}.

Legacy guide save requests may omit source_type (meaning guide). They require
guide_manual and omit every custom field:

~~~json
{
  "guide_manual": "pmcs_sbs/hmmwv/file.json",
  "performed_date": "2026-08-07T14:30:00Z",
  "notes": "Inspect before dispatch"
}
~~~

Custom save requests require exactly source_type custom, every custom provenance
field (revision number may be 0), and no guide_manual:

~~~json
{
  "source_type": "custom",
  "custom_checklist_id": "22222222-2222-2222-2222-222222222222",
  "custom_revision_id": "33333333-3333-3333-3333-333333333333",
  "custom_revision_number": 7,
  "custom_checklist_name": "Weekly Generator PMCS",
  "performed_date": "2026-08-07T14:30:00Z",
  "notes": null
}
~~~

The authored checklist tree is not copied. Custom id, revision id, revision
number, and name form immutable snapshot provenance that survives source
content retirement/deletion.

performed_date is required RFC 3339 input and is returned in UTC. notes is
inspection-only: omit or send null to clear; blank text clears after trimming;
non-empty notes are limited to 4,000 bytes. Fault saves do not accept notes and
clear persisted notes during their inspection-metadata upsert, so save desired
notes after a fault upsert. Custom checklist name and section_title are trimmed,
limited to 200 Unicode extended grapheme clusters and 8 KiB. Name is required;
title is optional and a blank title becomes absent/null.

## Inspection detail, list, and delete

A guide response includes guide_manual and omits all custom_* fields. A custom
response includes all custom_* fields and omits guide_manual. notes,
performed_by, and performed_by_username are omitted rather than null when
unavailable. faults and comments are arrays on inspection detail; list entries
omit them.

~~~json
{
  "status": 200,
  "message": "",
  "data": {
    "id": "11111111-1111-1111-1111-111111111111",
    "equipment_id": "shop-vehicle-id",
    "source_type": "custom",
    "custom_checklist_id": "22222222-2222-2222-2222-222222222222",
    "custom_revision_id": "33333333-3333-3333-3333-333333333333",
    "custom_revision_number": 7,
    "custom_checklist_name": "Weekly Generator PMCS",
    "performed_date": "2026-08-07T14:30:00Z",
    "performed_by": "firebase-user-id",
    "performed_by_username": "operator",
    "created_at": "2026-08-07T14:31:00Z",
    "updated_at": "2026-08-07T14:31:00Z",
    "faults": [],
    "comments": []
  }
}
~~~

Guide detail shape (guide_manual is present and all custom provenance is
absent):

~~~json
{
  "status": 200,
  "message": "",
  "data": {
    "id": "11111111-1111-1111-1111-111111111111",
    "equipment_id": "shop-vehicle-id",
    "source_type": "guide",
    "guide_manual": "pmcs_sbs/hmmwv/file.json",
    "performed_date": "2026-08-07T14:30:00Z",
    "created_at": "2026-08-07T14:31:00Z",
    "updated_at": "2026-08-07T14:31:00Z",
    "faults": [],
    "comments": []
  }
}
~~~

PUT inspection returns 200 / Inspection saved. Use it to explicitly materialize
a clean completion. GET detail returns faults ordered by section_id,item_index
and comments by created_at. Missing inspection is 404
{"message":"pmcs sbs inspection not found"}. DELETE inspection returns
200 {"message":"Inspection deleted"} and removes dependent faults/comments.

List accepts optional guide_manual, limit, offset. The guide filter selects
guide records only. Default/max limit is 1000 and default offset is 0. List
entries contain source provenance, performed_date, fault_count, comment_count,
created_at, and optional performer fields, newest performed_date first.

## Faults

A fault upsert repeats complete source provenance and performed_date. The first
valid fault atomically materializes an inspection. A fault identity is
(pmcs_id, section_id, item_index); repeating it updates rather than duplicates
the fault while preserving inspection provenance.

Guide fault body:

~~~json
{
  "guide_manual": "pmcs_sbs/hmmwv/file.json",
  "performed_date": "2026-08-07T14:30:00Z",
  "section_id": "before-operation",
  "item_index": 0,
  "item_no": "1",
  "status": "X",
  "fault_text": "Hydraulic leak",
  "corrective_action": "Replace hose"
}
~~~

Custom fault body:

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

section_id, nonnegative item_index, item_no, status, and fault_text are
required; corrective_action may be blank. X/x, / /slash, and -/dash normalize
to x, slash, and dash.

~~~json
{
  "status": 200,
  "message": "Fault saved",
  "data": {
    "pmcs_id": "11111111-1111-1111-1111-111111111111",
    "section_id": "before-operation",
    "section_title": "Before Operation",
    "item_index": 0,
    "item_no": "1",
    "status": "x",
    "fault_text": "Hydraulic leak",
    "corrective_action": "Replace hose",
    "created_at": "2026-08-07T14:31:00Z",
    "updated_at": "2026-08-07T14:31:00Z"
  }
}
~~~

Single delete body is {"section_id":"before-operation","item_index":0}; it
returns 200 {"message":"Fault deleted"}. Inspection must exist and be visible,
but deleting an already-absent key succeeds.

Bulk delete body is {"faults":[{"section_id":"before-operation","item_index":0}]}
and returns {"message":"Faults deleted","requested_count":1,"deleted_count":1}.
Empty, invalid, duplicate, or more-than-100 keys return 400. A replay may have
deleted_count smaller than requested_count.

## Comments

POST comments accepts {"text":"Needs supervisor review"}; text is trimmed,
non-empty, and limited to 2,000 bytes. It returns 201 / Comment created and
id, pmcs_id, author_id, optional author_username, text, created_at, and
optional updated_at.

PUT comment accepts the same body. DELETE comment returns Comment deleted and
the updated object with text Deleted by user; this is a visible history entry,
not removal. Only original author may update/delete (403
{"message":"user not authorized"}). Unknown comment is 404
{"message":"pmcs sbs comment not found"}.

~~~json
{
  "status": 201,
  "message": "Comment created",
  "data": {
    "id": "44444444-4444-4444-4444-444444444444",
    "pmcs_id": "11111111-1111-1111-1111-111111111111",
    "author_id": "firebase-user-id",
    "author_username": "operator",
    "text": "Needs supervisor review",
    "created_at": "2026-08-07T14:31:00Z"
  }
}
~~~

## Errors, parsing, and retries

| Status | Meaning |
| --- | --- |
| 400 | Invalid ID, source/provenance, date, note/fault/comment field, status, limit, or body. |
| 401 | Missing/invalid authenticated user. |
| 403 | Caller is not comment author. |
| 404 | Vehicle missing/not visible, inspection absent for vehicle, or comment absent. |
| 409 | Existing inspection has different vehicle or immutable source tuple. |

Inspection and fault saves accept exactly one JSON value: unknown fields and
trailing JSON are rejected. Raw invalid UTF-8 is rejected before Go JSON
decoding can replace bytes, and errors never echo authored request text.

There is no server queue/outbox, ETag, or conditional-write token. Persist
mutations locally and replay causally: inspection metadata, fault upserts,
fault delete/reset, comments, then final desired notes. User-PMCS content-sync
cursors and ETags are unrelated to Shop inspection mutations and must never be
reused as inspection validators.
