# PMCS SBS Image Loading Design

Date: 2026-06-27

## Goal

Add a public PMCS SBS image endpoint that lets the mobile app load PNG images referenced by a loaded PMCS SBS guide item. The guide JSON already contains extensionless image names in item-level `images` arrays. The server will use the selected guide `blob_path` plus the requested image name to derive the matching PNG blob path and stream the image bytes.

## Current Context

The existing PMCS SBS public library API lives in `api/library/pmcs_sbs` and is registered under `/api/v1/library/pmcs-sbs`. It currently supports:

- `GET /library/pmcs-sbs/folders`
- `GET /library/pmcs-sbs/:folder/files`
- `GET /library/pmcs-sbs/content?blob_path=...`

The current content endpoint returns guide JSON from Azure Blob Storage container `library` under the `pmcs_sbs/` prefix. Mobile already receives the selected guide's full `blob_path` from the files endpoint and uses it to load the guide.

The new image folder layout mirrors the guide name under the same PMCS SBS vehicle folder:

```text
pmcs_sbs/HMMWV/HMMWV NoArmor (SEPT13).json
pmcs_sbs/HMMWV/images/HMMWV NoArmor (SEPT13)/Before_12.png
```

## Endpoint

```http
GET /api/v1/library/pmcs-sbs/image?blob_path=<guide_json_blob_path>&image_name=<extensionless_image_name>
```

Example:

```http
GET /api/v1/library/pmcs-sbs/image?blob_path=pmcs_sbs/HMMWV/HMMWV%20NoArmor%20(SEPT13).json&image_name=Before_12
```

Success returns raw PNG bytes, not a JSON envelope.

Required success headers:

- `Content-Type: image/png`
- `Content-Length` when Azure provides it
- `Content-Disposition: inline; filename="<image_name>.png"`
- `Cache-Control: public, max-age=86400`

The route should use the existing rate limiter, matching the JSON content endpoint's public blob-read protection.

## Blob Path Derivation

The server derives the PNG blob path from the selected guide JSON path and the extensionless image name.

Algorithm:

1. Validate and clean `blob_path`.
2. Validate `image_name`.
3. Compute `guideDir = path.Dir(blob_path)`.
4. Compute `guideFile = path.Base(blob_path)`.
5. Compute `guideName = strings.TrimSuffix(guideFile, path.Ext(guideFile))`.
6. Compute `imageBlobPath = path.Join(guideDir, "images", guideName, imageName + ".png")`.

Example:

```text
blob_path:  pmcs_sbs/HMMWV/HMMWV NoArmor (SEPT13).json
image_name: Before_12
result:     pmcs_sbs/HMMWV/images/HMMWV NoArmor (SEPT13)/Before_12.png
```

## Validation

`blob_path` validation should follow the current guide content endpoint rules:

- required and not whitespace-only;
- `path.Clean` must not move it outside the PMCS SBS prefix;
- must start with `pmcs_sbs/`;
- must end with `.json`, case-insensitive;
- must not contain Windows path separators.

`image_name` validation:

- required and not whitespace-only;
- extensionless: `Before_12`, not `Before_12.png`;
- must not contain `/`, `\`, `.`, or path traversal tokens;
- after trimming, it is used exactly as the PNG basename.

The server will not parse the guide JSON or verify that `image_name` appears in an item `images` array. Path validation plus Azure blob existence are enough for this feature.

## Error Handling

| Condition | Status | Body |
|---|---:|---|
| Missing or blank `blob_path` | `400` | `{"error":"blob_path query parameter is required"}` |
| Missing or blank `image_name` | `400` | `{"error":"image_name query parameter is required"}` |
| Invalid guide path | `400` | `{"error":"Invalid request","details":"..."}` |
| Invalid image name | `400` | `{"error":"Invalid request","details":"..."}` |
| Derived PNG blob does not exist | `404` | `{"error":"Image not found","details":"The requested image does not exist or is not accessible"}` |
| Azure download/read failure | `500` | `{"error":"Failed to retrieve image"}` |

Errors should be logged with structured `slog` fields for `blobPath`, `imageName`, and the derived `imageBlobPath` when available. Error responses must not expose Azure credential or SDK internals.

## Architecture

Keep the work inside the existing public PMCS SBS library package:

- Extend `api/library/pmcs_sbs.Service` with an image download method.
- Implement the method in `ServiceImpl` using Azure Blob Storage `DownloadStream`.
- Add a focused image response/internal type that carries the stream, content length, content type, filename, and derived blob path.
- Add a new handler method in `api/library/pmcs_sbs/route.go`.
- Register the endpoint beside the existing public PMCS SBS routes.

No Postgres, Jet, migrations, authenticated routes, or PMCS SBS fault API changes are needed.

## Mobile Contract

Mobile should:

1. Load the guide through the existing `content` endpoint.
2. Read each item's `images` array.
3. For each image name, call the image endpoint with:
   - `blob_path`: the same selected guide blob path used to load the JSON;
   - `image_name`: the extensionless string from the guide item.
4. Render the returned PNG bytes.
5. Treat `404` as an image missing from content storage and continue showing the guide item without that image.

## Documentation

Update `docs/api/pmcs-sbs-api.md`:

- Change the overview from three endpoints to four.
- Add the image endpoint after Fetch Content.
- Include the HMMWV `Before_12` example.
- Document binary PNG success behavior and error responses.
- Update the recommended mobile workflow to load images on demand from the guide item's `images` array.

Do not update the PMCS SBS fault docs because this endpoint is part of the public guide-content API, not authenticated progress/fault persistence.

## Testing Plan

Handler tests in `api/library/pmcs_sbs/route_test.go`:

- success returns `200`, `image/png`, and streamed bytes;
- missing `blob_path` returns `400`;
- missing `image_name` returns `400`;
- invalid guide path returns `400`;
- invalid image name returns `400`;
- missing image returns `404`;
- unexpected service error returns `500`;
- handler passes both `blob_path` and `image_name` to the service.

Service tests in `api/library/pmcs_sbs/service_impl_test.go`:

- derives `pmcs_sbs/HMMWV/images/HMMWV NoArmor (SEPT13)/Before_12.png`;
- rejects path traversal in `blob_path`;
- rejects non-JSON guide paths;
- rejects image names with `.png`;
- rejects image names containing `/`, `\`, `.`, or traversal tokens.

Route setup test in `api/route/route_test.go`:

- verify `GET /api/v1/library/pmcs-sbs/image` is registered as a public route.

Focused verification command:

```bash
go test ./api/library/pmcs_sbs ./api/library ./api/route -count=1
```

## Implementation Notes

Use Gin `DataFromReader` for the successful image response. Ensure the Azure response body is closed after streaming completes. The handler should be responsible for closing the returned reader after `DataFromReader` returns.

The existing JSON content endpoint reads and validates complete JSON documents. The image endpoint should stream bytes instead of reading the entire image into memory, because image content does not need server-side parsing.

## Out Of Scope

- Server-side guide JSON parsing to verify `images[]` membership.
- Returning SAS URLs instead of raw PNG bytes.
- Supporting non-PNG formats.
- Listing available images for a guide.
- Database-backed image metadata.
- Authenticated PMCS SBS progress or fault changes.
