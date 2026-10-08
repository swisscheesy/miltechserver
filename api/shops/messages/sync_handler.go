package messages

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"miltechserver/api/response"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"
)

func registerSyncRoutes(router *gin.RouterGroup, service Service) {
	reader, _ := service.(SyncReader)
	for _, operation := range []string{"initial", "history", "catch-up", "reconcile"} {
		method := "GET"
		if operation == "reconcile" {
			method = "POST"
		}
		router.Handle(method, "/shops/:shop_id/messages-v2/"+operation, messageSyncHandler(reader, operation))
	}
}
func messageSyncHandler(reader SyncReader, operation string) gin.HandlerFunc {
	return func(c *gin.Context) {
		value, _ := c.Get("user")
		user, _ := value.(*bootstrap.User)
		if user == nil || user.UserID == "" {
			response.Error(c, 401, "unauthorized")
			return
		}
		if !shared.RequireContract2(c) {
			return
		}
		fail := func(err error) {
			f := shared.ClassifyFailure(err)
			shared.WriteFailure(c, f, f.Status, f.PublicMessage)
		}
		shopID := c.Param("shop_id")
		parsedShopID, parseErr := uuid.Parse(shopID)
		if parseErr != nil {
			fail(syncInvalid())
			return
		}
		shopID = parsedShopID.String()
		query, parseErr := url.ParseQuery(c.Request.URL.RawQuery)
		if parseErr != nil {
			fail(syncInvalid())
			return
		}
		allowed := map[string]bool{"limit": operation != "reconcile", "cursor": operation == "history", "after": operation == "catch-up", "through": operation == "catch-up"}
		for key, values := range query {
			if !allowed[key] || len(values) != 1 {
				fail(syncInvalid())
				return
			}
		}
		limit := 50
		if operation == "catch-up" {
			limit = 100
		}
		if raw, ok := c.GetQuery("limit"); ok {
			n, err := syncNumber(raw)
			if err != nil || n > 100 || n < 1 {
				fail(syncInvalid())
				return
			}
			limit = int(n)
		} else if _, ok := query["limit"]; ok {
			fail(syncInvalid())
			return
		}
		var result any
		var err error
		switch operation {
		case "initial":
			if reader == nil {
				err = syncUnavailable()
			} else {
				result, err = reader.InitialMessages(c.Request.Context(), user, shopID, limit)
			}
		case "history":
			cursor := c.Query("cursor")
			if _, err = decodeMessageCursor(shopID, cursor); err != nil {
				break
			}
			if reader == nil {
				err = syncUnavailable()
			} else {
				result, err = reader.MessageHistory(c.Request.Context(), user, shopID, cursor, limit)
			}
		case "catch-up":
			after := c.Query("after")
			var n int64
			n, err = syncNumber(after)
			if err != nil {
				break
			}
			var through *string
			if values, ok := query["through"]; ok {
				v := values[0]
				bound, e := syncNumber(v)
				if e != nil || bound < n {
					err = syncInvalid()
					break
				}
				through = &v
			}
			if reader == nil {
				err = syncUnavailable()
			} else {
				result, err = reader.CatchUpMessages(c.Request.Context(), user, shopID, strconv.FormatInt(n, 10), through, limit)
			}
		case "reconcile":
			var body struct {
				IDs []string `json:"ids"`
			}
			d := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 16384))
			d.DisallowUnknownFields()
			if d.Decode(&body) != nil || d.Decode(new(any)) != io.EOF {
				err = syncInvalid()
				break
			}
			var ids []string
			ids, err = normalizeSyncIDs(body.IDs)
			if err != nil {
				break
			}
			if reader == nil {
				err = syncUnavailable()
			} else {
				result, err = reader.ReconcileMessages(c.Request.Context(), user, shopID, ids)
			}
		}
		if err != nil {
			fail(err)
			return
		}
		response.OK(c, result)
	}
}
