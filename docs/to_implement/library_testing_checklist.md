# Library Feature Testing Checklist

**Feature**: PMCS/BII Library - Browse and Download Documents
**Phases Implemented**: 1, 2, 3
**Status**: Ready for Manual Testing with Live Azure Blob Storage
**Last Updated**: 2025-11-14

---

## Prerequisites

- [ ] Server deployed and running with access to Azure Blob Storage
- [ ] Azure Blob Storage account: `miltechng`
- [ ] Container `library` exists with proper permissions
- [ ] Test PDF files uploaded to known paths in Azure:
  - [ ] At least 2 files in `pmcs/TRACK/`
  - [ ] At least 2 files in `pmcs/HMMWV/`
  - [ ] At least 1 file in `pmcs/GENERATOR/`
- [ ] Testing tools available: `curl`, `jq` (optional for JSON formatting)
- [ ] Network access to deployed server

---

## Phase 1: Vehicle Folder Listing

### Test 1.1: Basic Vehicle List Retrieval

**Endpoint**: `GET /api/v1/library/pmcs/vehicles`

**Command**:
```bash
curl http://YOUR_SERVER_URL/api/v1/library/pmcs/vehicles | jq
```

**Expected Response**: 200 OK
```json
{
  "vehicles": [
    {
      "name": "GENERATOR",
      "full_path": "pmcs/GENERATOR/",
      "display_name": "GENERATOR"
    },
    {
      "name": "HMMWV",
      "full_path": "pmcs/HMMWV/",
      "display_name": "HMMWV"
    },
    {
      "name": "TRACK",
      "full_path": "pmcs/TRACK/",
      "display_name": "TRACK"
    }
  ],
  "count": 3
}
```

**Validation Checklist**:
- [ ] Response status is 200 OK
- [ ] JSON structure matches expected format
- [ ] All known vehicle folders are listed
- [ ] `count` field matches array length
- [ ] `display_name` properly formats underscores as spaces
- [ ] `full_path` includes trailing slash
- [ ] Response time < 2 seconds

**Known Good Vehicles** (based on your Phase 1 test):
- [ ] GENERATOR
- [ ] HEMTT_LET (displays as "HEMTT LET")
- [ ] HMMWV
- [ ] LMTV_MTV (displays as "LMTV MTV")
- [ ] MATERIAL_HANDLING_EQUIPMENT (displays as "MATERIAL HANDLING EQUIPMENT")
- [ ] MISCELLANEOUS
- [ ] OTHER_VEHICLES (displays as "OTHER VEHICLES")
- [ ] RECOVERY
- [ ] TRACK
- [ ] TRAILER
- [ ] WEAPONS_AND_ELECTRONICS (displays as "WEAPONS AND ELECTRONICS")

### Test 1.2: Error Handling

**Command** (Invalid endpoint):
```bash
curl http://YOUR_SERVER_URL/api/v1/library/pmcs/invalid
```

**Expected**: 404 Not Found

**Validation**:
- [ ] Proper error response returned
- [ ] Server doesn't crash

---

## Phase 2: Document Listing

### Test 2.1: List Documents in TRACK Folder

**Endpoint**: `GET /api/v1/library/pmcs/:vehicle/documents`

**Command**:
```bash
curl http://YOUR_SERVER_URL/api/v1/library/pmcs/TRACK/documents | jq
```

**Expected Response**: 200 OK
```json
{
  "vehicle_name": "TRACK",
  "documents": [
    {
      "name": "m1-abrams-pmcs.pdf",
      "blob_path": "pmcs/TRACK/m1-abrams-pmcs.pdf",
      "size_bytes": 2457600,
      "last_modified": "2024-11-10T14:30:00Z"
    },
    {
      "name": "m2-bradley-pmcs.pdf",
      "blob_path": "pmcs/TRACK/m2-bradley-pmcs.pdf",
      "size_bytes": 1843200,
      "last_modified": "2024-10-25T09:15:00Z"
    }
  ],
  "count": 2
}
```

**Validation Checklist**:
- [ ] Response status is 200 OK
- [ ] `vehicle_name` matches URL parameter
- [ ] All PDF files in folder are listed
- [ ] Only PDF files are listed (no .jpg, .txt, etc.)
- [ ] `count` matches array length
- [ ] `size_bytes` is a positive integer
- [ ] `last_modified` is valid ISO 8601 timestamp
- [ ] `blob_path` is correct full path
- [ ] Response time < 2 seconds

### Test 2.2: List Documents for Each Vehicle Type

Run for each known vehicle:
```bash
# HMMWV
curl http://YOUR_SERVER_URL/api/v1/library/pmcs/HMMWV/documents | jq

# GENERATOR
curl http://YOUR_SERVER_URL/api/v1/library/pmcs/GENERATOR/documents | jq

# HEMTT_LET
curl http://YOUR_SERVER_URL/api/v1/library/pmcs/HEMTT_LET/documents | jq
```

**Validation**:
- [ ] All vehicles return 200 OK
- [ ] Each shows correct vehicle_name
- [ ] Document counts match Azure Storage contents

### Test 2.3: Empty Folder Handling

**Command** (for vehicle with no documents):
```bash
curl http://YOUR_SERVER_URL/api/v1/library/pmcs/EMPTY_VEHICLE/documents | jq
```

**Expected Response**: 200 OK
```json
{
  "vehicle_name": "EMPTY_VEHICLE",
  "documents": [],
  "count": 0
}
```

**Validation**:
- [ ] Returns 200 OK (not 404)
- [ ] Empty documents array
- [ ] Count is 0

### Test 2.4: Non-existent Vehicle

**Command**:
```bash
curl http://YOUR_SERVER_URL/api/v1/library/pmcs/FAKE_VEHICLE/documents | jq
```

**Expected**: 200 OK with empty array

**Validation**:
- [ ] Returns 200 OK
- [ ] Empty documents array
- [ ] Count is 0

### Test 2.5: Mixed File Types (If Applicable)

If you have non-PDF files in a folder:

**Expected Behavior**:
- [ ] Only .pdf files appear in response
- [ ] .jpg, .png, .txt files are filtered out
- [ ] Filtering is case-insensitive (.PDF, .pdf both work)

---

## Phase 3: Download URL Generation

### Test 3.1: Generate Download URL for Valid Document

**Endpoint**: `GET /api/v1/library/download?blob_path=BLOB_PATH`

**Command** (replace with actual blob_path from Test 2.1):
```bash
curl "http://YOUR_SERVER_URL/api/v1/library/download?blob_path=pmcs/TRACK/m1-abrams-pmcs.pdf" | jq
```

**Expected Response**: 200 OK
```json
{
  "blob_path": "pmcs/TRACK/m1-abrams-pmcs.pdf",
  "download_url": "https://miltechng.blob.core.windows.net/library/pmcs/TRACK/m1-abrams-pmcs.pdf?sv=2023-11-03&se=2024-11-14T15:30:00Z&sr=b&sp=r&sig=SIGNATURE_HASH",
  "expires_at": "2024-11-14T15:30:00Z"
}
```

**Validation Checklist**:
- [ ] Response status is 200 OK
- [ ] `blob_path` matches request parameter
- [ ] `download_url` is a valid HTTPS URL
- [ ] URL contains Azure Blob Storage host
- [ ] URL contains SAS query parameters (`sv`, `se`, `sr`, `sp`, `sig`)
- [ ] `expires_at` is approximately 1 hour in the future
- [ ] Response time < 500ms

### Test 3.2: Download File Using SAS URL

**Command** (copy download_url from Test 3.1):
```bash
curl "PASTE_DOWNLOAD_URL_HERE" --output test_download.pdf
```

**Validation Checklist**:
- [ ] File downloads successfully
- [ ] Downloaded file is a valid PDF
- [ ] File size matches `size_bytes` from Phase 2
- [ ] File content is correct (open in PDF reader)

**Verify PDF**:
```bash
file test_download.pdf
# Expected output: test_download.pdf: PDF document, version X.X
```

### Test 3.3: SAS URL Security - HTTPS Enforcement

**Command** (modify URL to use HTTP instead of HTTPS):
```bash
# Copy download_url from Test 3.1 and change https:// to http://
curl "http://miltechng.blob.core.windows.net/library/..." --output should_fail.pdf
```

**Expected**: Download fails (Azure rejects HTTP requests with SAS tokens)

**Validation**:
- [ ] Download fails with authentication error
- [ ] Confirms HTTPS-only enforcement

### Test 3.4: SAS URL Security - Read-Only Permission

**Command** (try to DELETE using SAS URL):
```bash
# Copy download_url from Test 3.1
curl -X DELETE "PASTE_DOWNLOAD_URL_HERE"
```

**Expected**: 403 Forbidden (SAS token doesn't have delete permission)

**Validation**:
- [ ] DELETE request fails
- [ ] Confirms read-only permission

### Test 3.5: Missing blob_path Parameter

**Command**:
```bash
curl "http://YOUR_SERVER_URL/api/v1/library/download" | jq
```

**Expected Response**: 400 Bad Request
```json
{
  "error": "blob_path query parameter is required"
}
```

**Validation**:
- [ ] Returns 400 status
- [ ] Error message is clear

### Test 3.6: Invalid Path Prefix

**Command**:
```bash
curl "http://YOUR_SERVER_URL/api/v1/library/download?blob_path=unauthorized/secret.pdf" | jq
```

**Expected Response**: 400 Bad Request
```json
{
  "error": "Invalid request",
  "details": "invalid blob path: must start with pmcs/ or bii/"
}
```

**Validation**:
- [ ] Returns 400 status
- [ ] Validates path prefix restriction

### Test 3.7: Non-PDF File Type

**Command**:
```bash
curl "http://YOUR_SERVER_URL/api/v1/library/download?blob_path=pmcs/TRACK/image.jpg" | jq
```

**Expected Response**: 400 Bad Request
```json
{
  "error": "Invalid request",
  "details": "invalid file type: only PDF files can be downloaded"
}
```

**Validation**:
- [ ] Returns 400 status
- [ ] Enforces PDF-only restriction

### Test 3.8: Non-existent Document

**Command**:
```bash
curl "http://YOUR_SERVER_URL/api/v1/library/download?blob_path=pmcs/TRACK/fake-document.pdf" | jq
```

**Expected Response**: 404 Not Found
```json
{
  "error": "Document not found",
  "details": "The requested document does not exist or is not accessible"
}
```

**Validation**:
- [ ] Returns 404 status
- [ ] Clear error message

### Test 3.9: URL Expiry Behavior (Time-based Test)

**Setup**: Generate a download URL and wait 1+ hours

**Command** (after 1 hour):
```bash
# Use download_url from Test 3.1 after waiting 1+ hours
curl "EXPIRED_DOWNLOAD_URL" --output should_fail.pdf
```

**Expected**: 403 Forbidden (Azure rejects expired SAS tokens)

**Validation**:
- [ ] Download fails after expiry time
- [ ] Confirms 1-hour expiry works

**Note**: This test requires waiting. Alternative: Manually inspect `expires_at` timestamp to verify it's ~1 hour from generation time.

---

## End-to-End Workflow Tests

### Test E2E-1: Complete User Journey

Simulate a mobile app user's complete workflow:

**Step 1**: List available vehicles
```bash
VEHICLES=$(curl -s http://YOUR_SERVER_URL/api/v1/library/pmcs/vehicles)
echo $VEHICLES | jq '.vehicles[0].name'
# Note the first vehicle name
```

**Step 2**: List documents for that vehicle
```bash
VEHICLE_NAME="TRACK"  # Use name from Step 1
DOCS=$(curl -s "http://YOUR_SERVER_URL/api/v1/library/pmcs/$VEHICLE_NAME/documents")
echo $DOCS | jq '.documents[0].blob_path'
# Note the first document's blob_path
```

**Step 3**: Generate download URL
```bash
BLOB_PATH="pmcs/TRACK/m1-abrams-pmcs.pdf"  # Use blob_path from Step 2
DOWNLOAD=$(curl -s "http://YOUR_SERVER_URL/api/v1/library/download?blob_path=$BLOB_PATH")
echo $DOWNLOAD | jq '.download_url'
```

**Step 4**: Download the file
```bash
DOWNLOAD_URL=$(echo $DOWNLOAD | jq -r '.download_url')
curl "$DOWNLOAD_URL" --output final_test.pdf
file final_test.pdf
```

**Validation**:
- [ ] All 4 steps complete successfully
- [ ] Final PDF is valid and correct
- [ ] Total workflow time < 5 seconds

### Test E2E-2: Multiple Document Downloads

**Command**:
```bash
# Get list of documents
DOCS=$(curl -s "http://YOUR_SERVER_URL/api/v1/library/pmcs/TRACK/documents")

# Extract first 3 blob paths
BLOB1=$(echo $DOCS | jq -r '.documents[0].blob_path')
BLOB2=$(echo $DOCS | jq -r '.documents[1].blob_path')
BLOB3=$(echo $DOCS | jq -r '.documents[2].blob_path')

# Generate download URLs for all 3
curl -s "http://YOUR_SERVER_URL/api/v1/library/download?blob_path=$BLOB1" | jq '.download_url'
curl -s "http://YOUR_SERVER_URL/api/v1/library/download?blob_path=$BLOB2" | jq '.download_url'
curl -s "http://YOUR_SERVER_URL/api/v1/library/download?blob_path=$BLOB3" | jq '.download_url'
```

**Validation**:
- [ ] All 3 URLs generated successfully
- [ ] Each URL has different signature
- [ ] Each URL has similar expiry time (~1 hour from now)

---

## Performance Tests

### Test P-1: Response Time - Vehicle List

**Command** (with timing):
```bash
time curl -s http://YOUR_SERVER_URL/api/v1/library/pmcs/vehicles -o /dev/null
```

**Validation**:
- [ ] Total time < 2 seconds
- [ ] Consistent across 5 requests

### Test P-2: Response Time - Document List

**Command**:
```bash
time curl -s "http://YOUR_SERVER_URL/api/v1/library/pmcs/TRACK/documents" -o /dev/null
```

**Validation**:
- [ ] Total time < 2 seconds for folders with < 50 files
- [ ] Total time < 5 seconds for folders with 100+ files

### Test P-3: Response Time - Download URL Generation

**Command**:
```bash
time curl -s "http://YOUR_SERVER_URL/api/v1/library/download?blob_path=pmcs/TRACK/m1-abrams-pmcs.pdf" -o /dev/null
```

**Validation**:
- [ ] Total time < 500ms
- [ ] Consistent across 10 requests
- [ ] No significant variation in timing

### Test P-4: Large File Download

**Command** (for a large PDF, e.g., 50+ MB):
```bash
# Generate URL
DOWNLOAD=$(curl -s "http://YOUR_SERVER_URL/api/v1/library/download?blob_path=pmcs/TRACK/large-manual.pdf")
URL=$(echo $DOWNLOAD | jq -r '.download_url')

# Time the download
time curl "$URL" --output large_file.pdf
```

**Validation**:
- [ ] Large files (50+ MB) download successfully
- [ ] Download completes within 1-hour expiry window
- [ ] No timeout errors

---

## Security Tests

### Test S-1: Path Traversal Attempt

**Command**:
```bash
curl "http://YOUR_SERVER_URL/api/v1/library/download?blob_path=pmcs/../../etc/passwd" | jq
```

**Expected**: 400 Bad Request (path validation blocks this)

**Validation**:
- [ ] Returns 400 error
- [ ] No system files accessible

### Test S-2: SQL Injection Attempt (Defensive)

**Command**:
```bash
curl "http://YOUR_SERVER_URL/api/v1/library/download?blob_path=pmcs/TRACK/'; DROP TABLE users; --.pdf" | jq
```

**Expected**: 400 or 404 (treated as invalid filename)

**Validation**:
- [ ] No database errors
- [ ] Server handles gracefully

### Test S-3: Cross-Origin Request

**Command** (from browser console on different domain):
```javascript
fetch('http://YOUR_SERVER_URL/api/v1/library/pmcs/vehicles')
  .then(r => r.json())
  .then(console.log)
```

**Validation**:
- [ ] CORS headers configured appropriately
- [ ] Mobile app can make requests

---

## Logging & Monitoring Tests

### Test L-1: Verify Structured Logging

**Check server logs** for each request type:

**Vehicle List Request**:
- [ ] Logs show: "Fetching PMCS vehicles from Azure Blob Storage"
- [ ] Logs show: "Successfully fetched PMCS vehicles" with count

**Document List Request**:
- [ ] Logs show: "Fetching PMCS documents from Azure Blob Storage" with vehicle name
- [ ] Logs show: "Successfully fetched PMCS documents" with count

**Download URL Request**:
- [ ] Logs show: "Generating download URL for blob" with blob_path
- [ ] Logs show: "Successfully generated download URL" with expiry time

**Error Cases**:
- [ ] Logs show appropriate ERROR level for failures
- [ ] Logs include error context (blob_path, vehicle_name, etc.)

---

## Regression Tests

Run these after any code changes:

### Test R-1: Phase 1 Still Works
```bash
curl http://YOUR_SERVER_URL/api/v1/library/pmcs/vehicles | jq
```
- [ ] Returns vehicle list correctly

### Test R-2: Phase 2 Still Works
```bash
curl http://YOUR_SERVER_URL/api/v1/library/pmcs/TRACK/documents | jq
```
- [ ] Returns document list correctly

### Test R-3: Phase 3 Still Works
```bash
curl "http://YOUR_SERVER_URL/api/v1/library/download?blob_path=pmcs/TRACK/test.pdf" | jq
```
- [ ] Generates SAS URL correctly

---

## Known Test Data

Based on your Azure Blob Storage container `library`:

### PMCS Vehicles (from Phase 1 test response):
1. GENERATOR
2. HEMTT_LET
3. HMMWV
4. LMTV_MTV
5. MATERIAL_HANDLING_EQUIPMENT
6. MISCELLANEOUS
7. OTHER_VEHICLES
8. RECOVERY
9. TRACK
10. TRAILER
11. WEAPONS_AND_ELECTRONICS

### Recommended Test Documents

For thorough testing, upload these test files if not already present:

**In `pmcs/TRACK/`**:
- `m1-abrams-pmcs.pdf` (small, ~2MB)
- `m2-bradley-pmcs.pdf` (medium, ~5MB)
- `test-large-manual.pdf` (large, 50+ MB) - for performance testing

**In `pmcs/HMMWV/`**:
- `hmmwv-operators-manual.pdf`
- `hmmwv-pmcs-checklist.pdf`

**In `pmcs/GENERATOR/`**:
- `generator-maintenance.pdf`

---

## Test Results Summary

**Date Tested**: ___________
**Tester**: ___________
**Environment**: ___________ (development/staging/production)

| Test Category | Tests Run | Passed | Failed | Notes |
|---------------|-----------|--------|--------|-------|
| Phase 1 - Vehicle List | | | | |
| Phase 2 - Document List | | | | |
| Phase 3 - Download URLs | | | | |
| End-to-End Workflows | | | | |
| Performance | | | | |
| Security | | | | |
| Logging | | | | |

**Overall Status**: ⬜ Pass / ⬜ Fail / ⬜ Needs Investigation

**Critical Issues Found**: (list any blocking issues)

**Non-Critical Issues**: (list minor issues or improvements)

**Deployment Recommendation**: ⬜ Ready for Production / ⬜ Needs Fixes

---

## Quick Test Script

For rapid testing, use this script:

```bash
#!/bin/bash
# Quick Library Feature Test Script

SERVER_URL="http://localhost:8080"  # Update with your server URL

echo "=== Phase 1: List Vehicles ==="
curl -s "$SERVER_URL/api/v1/library/pmcs/vehicles" | jq '.count'

echo ""
echo "=== Phase 2: List TRACK Documents ==="
curl -s "$SERVER_URL/api/v1/library/pmcs/TRACK/documents" | jq '.count'

echo ""
echo "=== Phase 3: Generate Download URL ==="
# Replace with actual blob_path from your storage
BLOB_PATH="pmcs/TRACK/m1-abrams-pmcs.pdf"
RESPONSE=$(curl -s "$SERVER_URL/api/v1/library/download?blob_path=$BLOB_PATH")
echo $RESPONSE | jq '.expires_at'

echo ""
echo "=== Download File ==="
URL=$(echo $RESPONSE | jq -r '.download_url')
curl -s "$URL" --output /tmp/test_library_download.pdf
file /tmp/test_library_download.pdf

echo ""
echo "=== All tests complete ==="
```

Save as `test_library.sh`, make executable (`chmod +x test_library.sh`), and run.

---

## Notes

- Replace `YOUR_SERVER_URL` with your actual server address
- Replace blob paths with actual files from your Azure Storage
- Some tests require actual PDF files to be present in Azure Storage
- SAS URL expiry test requires waiting 1+ hours
- Large file test requires a 50+ MB PDF in storage
- All curl commands include `| jq` for pretty JSON output (optional)
