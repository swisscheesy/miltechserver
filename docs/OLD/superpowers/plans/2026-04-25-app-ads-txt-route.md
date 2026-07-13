# app-ads.txt Root Route Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Serve `app-ads.txt` at the root path `/app-ads.txt` so Google AdMob can verify ad network authorization.

**Architecture:** The file already exists in the correct locations throughout the build pipeline (source → SvelteKit build output → Docker container at `/app/static/app-ads.txt`). The only missing piece is an explicit Gin `StaticFile` route. The SPA `NoRoute` fallback currently intercepts all unmatched non-API paths and returns `index.html`, which is why `/app-ads.txt` returns a 404-like HTML page instead of the file.

**Tech Stack:** Go 1.23, Gin, SvelteKit static adapter, Docker multi-stage build

---

## Current State Analysis

| Location | File Present? | Notes |
|---|---|---|
| `frontend/static/app-ads.txt` | ✅ | SvelteKit static dir — copied verbatim into build output |
| `frontend/build/app-ads.txt` | ✅ | Confirmed present in local build |
| `/app/static/app-ads.txt` (container) | ✅ | Dockerfile copies `frontend/build` → `./static/` |
| `GET /app-ads.txt` route | ❌ | **Missing** — `NoRoute` fallback returns `index.html` |

---

## File Structure

**Modified:**
- `api/route/route.go:83-86` — add one `StaticFile` entry alongside existing static file routes

No files created. No Dockerfile changes needed. No frontend changes needed.

---

### Task 1: Register the `/app-ads.txt` static route

**Files:**
- Modify: `api/route/route.go:85` (add after the existing `StaticFile` entries)

- [ ] **Step 1: Verify the file exists in the build output**

```bash
ls frontend/build/app-ads.txt
```

Expected output: `frontend/build/app-ads.txt`

- [ ] **Step 2: Add the StaticFile route in `route.go`**

In `api/route/route.go`, locate the static file block (lines 82–86):

```go
// Serve static assets (CSS, JS, images, etc.)
router.Static("/_app", "./static/_app")
router.Static("/assets", "./static/assets")
router.StaticFile("/favicon.ico", "./static/favicon.ico")
router.StaticFile("/favicon.svg", "./static/favicon.svg")
```

Add one line so the block becomes:

```go
// Serve static assets (CSS, JS, images, etc.)
router.Static("/_app", "./static/_app")
router.Static("/assets", "./static/assets")
router.StaticFile("/favicon.ico", "./static/favicon.ico")
router.StaticFile("/favicon.svg", "./static/favicon.svg")
router.StaticFile("/app-ads.txt", "./static/app-ads.txt")
```

- [ ] **Step 3: Run the server locally and verify the route**

```bash
go run main.go &
curl -s http://localhost:8080/app-ads.txt | head -3
```

Expected output (first two lines of the file):

```
google.com, pub-4840468445655958, DIRECT, f08c47fec0942fa0
vungle.com,69eda3de8f95e24c6f992fde,DIRECT,c107d686becd2d77
```

Kill the background server after verifying: `kill %1`

- [ ] **Step 4: Verify the SPA fallback still works (regression check)**

```bash
go run main.go &
curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/some-spa-route
```

Expected output: `200` (SPA returns index.html with 200, not 404)

Kill the background server: `kill %1`

- [ ] **Step 5: Commit**

```bash
git add api/route/route.go
git commit -m "fix(routes): serve app-ads.txt at root for AdMob verification"
```

---

## Deployment Note

No Dockerfile changes are required. On next Docker build:
1. `npm run build` copies `frontend/static/app-ads.txt` → `frontend/build/app-ads.txt` (already working)
2. Dockerfile copies `frontend/build/` → `/app/static/` in the container (already working)
3. The new Gin route serves `/app/static/app-ads.txt` at `GET /app-ads.txt` (the fix)

After deploying the new container, `https://api.miltech.app/app-ads.txt` will return the file contents.
