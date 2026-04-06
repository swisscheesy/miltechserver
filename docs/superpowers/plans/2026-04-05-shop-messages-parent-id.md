# Shop Messages parent_id Reply Support Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire the existing `parent_id` column through the `CreateShopMessage` write path so clients can attach a reply to a previous message.

**Architecture:** Add an optional `parent_id` field to `CreateShopMessageRequest`, map it in the handler, and include `ShopMessages.ParentID` in the Jet INSERT column list. All read endpoints already return `parent_id` via `AllColumns` — no read-side changes needed.

**Tech Stack:** Go, Gin, go-jet/jet (Postgres query builder), PostgreSQL, testify (integration tests against a real test DB)

---

## File Map

| File | Change |
|------|--------|
| `api/request/shops_request.go` | Add `ParentID *string` to `CreateShopMessageRequest` |
| `api/shops/messages/handler.go` | Map `req.ParentID` into the `model.ShopMessages` struct |
| `api/shops/messages/repository_impl.go` | Add `ShopMessages.ParentID` to INSERT column list |
| `tests/shops/shops_messages_test.go` | Add tests for reply creation and parent_id in responses |

---

### Task 1: Write the failing integration tests

**Files:**
- Modify: `tests/shops/shops_messages_test.go`

- [ ] **Step 1: Add two test functions to the bottom of `tests/shops/shops_messages_test.go`**

Open `tests/shops/shops_messages_test.go` and append the following two test functions after the existing `TestMessagesCreateAndGet` function:

```go
func TestCreateMessageReply(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")

	router := newTestRouter(t)
	shopID := createShop(t, router, "user-1", "Reply Shop")

	// Create a top-level message to reply to
	parentID := createMessage(t, router, "user-1", shopID, "Parent message")

	// Create a reply with parent_id set
	replyBody := map[string]interface{}{
		"shop_id":   shopID,
		"message":   "Reply to parent",
		"parent_id": parentID,
	}

	replyResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/messages", replyBody, "user-1")
	require.Equal(t, http.StatusCreated, replyResp.Code)

	resp := decodeStandardResponse(t, replyResp.Body)
	replyData := decodeMap(t, resp.Data)

	require.Equal(t, parentID, replyData["parent_id"], "reply should have parent_id set to the parent message's id")
}

func TestMessageParentIDInGetMessages(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")

	router := newTestRouter(t)
	shopID := createShop(t, router, "user-1", "Get Reply Shop")

	parentID := createMessage(t, router, "user-1", shopID, "Parent message")

	replyBody := map[string]interface{}{
		"shop_id":   shopID,
		"message":   "Reply message",
		"parent_id": parentID,
	}
	replyResp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/shops/messages", replyBody, "user-1")
	require.Equal(t, http.StatusCreated, replyResp.Code)

	getResp := doJSONRequest(t, router, http.MethodGet, "/api/v1/auth/shops/"+shopID+"/messages", nil, "user-1")
	require.Equal(t, http.StatusOK, getResp.Code)

	messages := decodeStandardResponse(t, getResp.Body)
	messagesList := decodeSlice(t, messages.Data)
	require.Len(t, messagesList, 2, "shop should have 2 messages: parent and reply")

	// Find the reply message and assert its parent_id
	foundReply := false
	for _, m := range messagesList {
		msg := m.(map[string]interface{})
		if msg["message"] == "Reply message" {
			require.Equal(t, parentID, msg["parent_id"], "reply message should have parent_id set")
			foundReply = true
		}
	}
	require.True(t, foundReply, "reply message should be present in get response")
}
```

- [ ] **Step 2: Run the tests to confirm they fail**

```bash
go test ./tests/shops/... -v -run "TestCreateMessageReply|TestMessageParentIDInGetMessages"
```

Expected: Both tests FAIL. `TestCreateMessageReply` will fail because `parent_id` sent in the request body is ignored — the created message will have `parent_id: null` instead of the expected parent ID.

---

### Task 2: Add `parent_id` to the request struct

**Files:**
- Modify: `api/request/shops_request.go`

- [ ] **Step 1: Add `ParentID` field to `CreateShopMessageRequest`**

In `api/request/shops_request.go`, find `CreateShopMessageRequest` (currently at line 30) and add the `ParentID` field:

```go
type CreateShopMessageRequest struct {
	ShopID   string  `json:"shop_id" binding:"required"`
	Message  string  `json:"message" binding:"required"`
	ParentID *string `json:"parent_id"`
}
```

The field is `*string` (pointer) so it is always optional. Clients that don't send `parent_id` get `nil`. No `binding` tag — it must never be required.

- [ ] **Step 2: Run the tests (still failing — handler not yet updated)**

```bash
go test ./tests/shops/... -v -run "TestCreateMessageReply|TestMessageParentIDInGetMessages"
```

Expected: Still FAIL. The request struct now accepts the field but the handler doesn't map it into the model yet.

---

### Task 3: Map `ParentID` in the handler

**Files:**
- Modify: `api/shops/messages/handler.go`

- [ ] **Step 1: Pass `ParentID` through to the model in `CreateShopMessage`**

In `api/shops/messages/handler.go`, find the `CreateShopMessage` handler (around line 38). Update the model construction to include `ParentID`:

```go
message := model.ShopMessages{
	ShopID:   req.ShopID,
	Message:  req.Message,
	ParentID: req.ParentID,
}
```

This is the only change in this file.

- [ ] **Step 2: Run the tests (still failing — repository not yet updated)**

```bash
go test ./tests/shops/... -v -run "TestCreateMessageReply|TestMessageParentIDInGetMessages"
```

Expected: Still FAIL. The model now carries `ParentID` but the INSERT statement does not write it to the database.

---

### Task 4: Add `ParentID` to the INSERT statement

**Files:**
- Modify: `api/shops/messages/repository_impl.go`

- [ ] **Step 1: Add `ShopMessages.ParentID` to the INSERT column list in `CreateShopMessage`**

In `api/shops/messages/repository_impl.go`, find `CreateShopMessage` (starting at line 44). Update the INSERT statement:

```go
stmt := ShopMessages.INSERT(
	ShopMessages.ID,
	ShopMessages.ShopID,
	ShopMessages.UserID,
	ShopMessages.Message,
	ShopMessages.CreatedAt,
	ShopMessages.UpdatedAt,
	ShopMessages.IsEdited,
	ShopMessages.ParentID,
).MODEL(message).RETURNING(ShopMessages.AllColumns)
```

When `message.ParentID` is `nil`, Jet writes `NULL` to the column. When it is set, the UUID string is written. No conditional logic is needed.

- [ ] **Step 2: Run the tests — expect both to pass**

```bash
go test ./tests/shops/... -v -run "TestCreateMessageReply|TestMessageParentIDInGetMessages"
```

Expected:
```
--- PASS: TestCreateMessageReply
--- PASS: TestMessageParentIDInGetMessages
```

- [ ] **Step 3: Run the full shops test suite to confirm no regressions**

```bash
go test ./tests/shops/... -v
```

Expected: All tests pass.

- [ ] **Step 4: Commit**

```bash
git add api/request/shops_request.go \
        api/shops/messages/handler.go \
        api/shops/messages/repository_impl.go \
        tests/shops/shops_messages_test.go
git commit -m "feat(shop-messages): support parent_id for message replies"
```
