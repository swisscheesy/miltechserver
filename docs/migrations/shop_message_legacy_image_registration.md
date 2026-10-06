# Legacy shop message image registration (024)

Forward: `migrations/024_register_legacy_shop_message_images.sql`.
Reverse: `migrations/024_rollback_register_legacy_shop_message_images.sql`.
Data only: no table, column, constraint, function or trigger changes, so the
catalog hash and the generated `.gen` models are unchanged. Requires 020.

## Why

Migration 020 deliberately did not adopt images uploaded before it existed. Those
blobs had no `shop_message_uploads` row, so after the new server deployed:

- deleting or editing an old message still worked, but its image blob was never
  queued for cleanup and stayed in Azure until the whole Shop was deleted;
- `DELETE` of an old image via the discard endpoint was refused.

024 is the explicit owner-approved mapping that the 020 design required before
historical images could be cleaned up individually (see ADR-023).

## What it registers

The pre-020 server (`518ef4e`, `UploadMessageImage`) named every blob
`{shop_id}/{uuid.New()}{extension}` in the `shop-message-images` container. That is
exactly the key shape the cleanup worker accepts
(`api/shops/messages/blob_store.go`, `validate`: `BlobKey == ShopID/ID+Extension`),
so the key's UUID becomes the upload ID and no blob is renamed or copied.

Only this canonical marker form is recognized:

```
[IMAGE:https://{account}.blob.core.windows.net/shop-message-images/{shop-uuid}/{uuid}.{jpg|png|gif|webp}]
```

For each distinct image it inserts one `shop_message_uploads` row:

| Column | Value |
|---|---|
| `id` | UUID from the blob key |
| `uploader_id` | `legacy:unattributed` (never inferred from the message author) |
| `account`, `blob_key`, `url` | from the marker; `url` rebuilt exactly as `Reserve` builds it |
| `state` | `ready` |
| `lease_until`, `created_at` | earliest referencing message `created_at` (already expired) |

and one `shop_message_asset_references` row per (message, image).

`uploader_id` only matters to `Finalize` (state `uploading`, never used here) and to
`Discard` of an unreferenced image. Every backfilled image starts referenced and
moves to cleanup automatically when its last reference goes, so the placeholder
removes no capability. A Firebase UID cannot equal it, so only a current Shop admin
could ever discard one.

## Refusals (whole migration rolls back, nothing changes)

A missed reference is the one failure that could lose data: the worker would delete
a blob that some message still shows. The forward therefore refuses when:

| Error | Cause |
|---|---|
| `024: non-canonical historical image reference` | A message mentions `shop-message-images` (case-insensitive) more times than it has canonical markers, e.g. a query string or altered URL |
| `024: historical image path belongs to a different shop` | A marker's key prefix is not the message's own Shop (copied across Shops) |
| `024: conflicting storage targets for one historical image identity` | One UUID appears with different account, Shop or extension |
| `024: historical image identity is registered to a different storage target` | The UUID already exists in the registry with another target |
| `024: historical image identity appears outside a canonical marker` | A newly registered image's UUID appears in a message that has no canonical marker for it |

Each refusal needs an explicit, owner-reviewed resolution (edit the message text, or
leave that image unregistered by a separately reviewed variant). The migration never
repairs data itself.

Targets already in the registry are skipped, so images uploaded through the new
server are left to the runtime and a repeated run is a no-op.

## Inventory before applying

Read-only; run against the intended target first. Zero rows from the first two
queries means the forward will not refuse on canonical-form or Shop checks.

```sql
-- Messages whose container mentions are not all canonical markers
SELECT m.id
FROM shop_messages m
WHERE (length(lower(m.message)) - length(replace(lower(m.message), 'shop-message-images', ''))) / 19
   <> (SELECT count(*) FROM regexp_matches(m.message,
       '\[IMAGE:https://[a-z0-9][a-z0-9-]*\.blob\.core\.windows\.net/shop-message-images/[0-9a-f-]{36}/[0-9a-f-]{36}\.(jpg|png|gif|webp)\]', 'g'));

-- Canonical markers pointing at another Shop's folder
SELECT m.id
FROM shop_messages m,
     regexp_matches(m.message, '/shop-message-images/([0-9a-f-]{36})/', 'g') k
WHERE k[1] <> m.shop_id;

-- How much 024 will register
SELECT count(*) AS markers
FROM shop_messages m,
     regexp_matches(m.message, '\[IMAGE:https://[^\]]*/shop-message-images/[^\]]*\]', 'g');
```

On the 192.168.20.70 `miltech_ng` dev database (2026-10-05, read-only): 62 messages,
5 canonical markers in 3 Shops, all same-Shop, account `miltechng`, no shared images,
no non-canonical mentions.

## Locking and cost

Forward takes `EXCLUSIVE` on `shop_messages`, `shop_message_uploads` and
`shop_message_asset_references`, in the runtime's order. Plain reads continue;
every write and `FOR UPDATE` on those tables waits until commit. `EXCLUSIVE`
(not `SHARE ROW EXCLUSIVE`) is required: the reference FK checks take key-share row
locks on messages, which would deadlock against an editor already holding the
message row `FOR UPDATE`. `lock_timeout` is 5 s per acquisition.

Two checks scan all messages: the occurrence count (one pass) and the identity check
(`strpos` of each new image UUID against every message, i.e. images × messages).
At 5 images × 62 messages this is instant. At 1,000 images × 100,000 messages it is
10^8 substring tests and could hold the lock for tens of seconds: measure on a copy
first and run in a low-traffic window.

## Reverse

Takes `ACCESS EXCLUSIVE` on uploads, references, then cleanup jobs (the 020 reverse
order). It refuses with `024: legacy image cleanup has started` if any
`legacy:unattributed` upload is no longer `ready` or has any cleanup job (queued,
completed or in manual review): that is proof that must be kept. Otherwise it
deletes the references, deletes the cleanup jobs the 020 trigger queued for those
deletions in the same transaction (so no blob is touched), then the upload rows.
The blobs return to unregistered, protected status.

020's reverse refuses while any registry row exists, so 024 must be reversed first.

## Applying

Through the guarded runner like 019–023:
`scripts/apply-shops-remediation-migrations.sh <target> forward-024 ...`, which pins
both files by SHA-256 in `SOURCE_PINS`. The tagged generation checkpoint after it
should show no model change.

**Storage safety:** once 024 is applied, deleting an old message deletes its real
blob. The dev `.env` uses `BLOB_ACCOUNT_NAME=miltechng`, the same account the dev
messages point at. If that is production storage, run dev against a separate
account or with the cleanup worker disabled before testing deletions there.

## Verification

`tests/shops/shops_message_legacy_images_test.go` runs both files against a
disposable 016–023 database:

- registration, uploader placeholder, URL/key shape, no-op repeat;
- shared image survives the first message delete (`references_survive`), deleted after
  the second, and the prepared target satisfies the Azure store key check;
- edit removing the marker moves the image to `cleanup_pending`;
- managed (post-020) uploads are skipped;
- each refusal fires with its own reason and leaves nothing written;
- reverse leaves no uploads, references or jobs; reverse after a delete refuses.

2026-10-05: all pass, and the full `./tests/shops` suite passes with them, on a
scratch PostgreSQL 14.18 instance built from the pinned baseline plus 016–023
(`scripts/test-shops-isolated.sh` could not run locally because `rg` is not on the
non-interactive PATH). Runner unit tests (`apply-shops-remediation-migrations_test.py`)
pass with the `forward-024` envelope.
