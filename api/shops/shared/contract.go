package shared

import (
	"errors"
	"github.com/gin-gonic/gin"
	"miltechserver/api/response"
	"strings"
)

const ContractHeader = "X-MilTech-Shops-Contract"
const contractScopeKey = "shops.errorContract"

func UsesContract2(c *gin.Context) bool {
	return c.GetBool(contractScopeKey) && c.GetHeader(ContractHeader) == "2"
}
func ContractMiddleware(c *gin.Context) {
	c.Set(contractScopeKey, true)
	c.Set(response.ErrorWriterKey, func(status int, message string) {
		if !publicResponseMessages[message] {
			message = "Unable to complete Shops request"
		}
		code := "internal_error"
		switch status {
		case 400, 422:
			code = "invalid"
		case 401:
			code = "unauthorized"
		case 403:
			code = "denied"
		case 404:
			code = "not_found"
		case 409:
			code = "conflict"
		}
		WriteFailure(c, &Failure{Code: code, PublicMessage: message, Status: status}, status, message)
	})
	c.Next()
}
func RequireContract2(c *gin.Context) bool {
	if UsesContract2(c) {
		return true
	}
	c.JSON(400, response.StandardResponse{Status: 400, Code: "unsupported_contract", Message: "Unsupported Shops contract"})
	return false
}
func WriteFailure(c *gin.Context, f *Failure, legacyStatus int, legacyMessage string) {
	status, message, code := legacyStatus, legacyMessage, ""
	if UsesContract2(c) {
		status, message, code = f.Status, f.PublicMessage, f.Code
	}
	c.JSON(status, response.StandardResponse{Status: status, Message: message, Code: code})
}

// HandleContractError only handles registered Shops routes. Router/proxy 404s
// cannot acquire resource-specific codes from a URL resemblance.
func HandleContractError(c *gin.Context, err error) bool {
	if !c.GetBool(contractScopeKey) {
		return false
	}
	if c.Writer.Written() {
		return true
	}
	status := 500
	if strings.Contains(err.Error(), "no item found") {
		status = 404
	}
	f := ClassifyFailure(err)
	message := legacyPublicMessage(err)
	WriteFailure(c, f, status, message)
	return true
}

var publicResponseMessages = map[string]bool{
	"invalid limit": true, "invalid include": true, "failed to retrieve shops aggregate": true, "failed to retrieve shop equipment overview": true,
	"access denied":                                  true,
	"at least one setting must be provided":          true,
	"before_id and after_id cannot be used together": true,
	"code_id is required":                            true,
	"equipment_id is required":                       true,
	"failed to get uploaded file":                    true,
	"failed to read file data":                       true,
	"file size exceeds maximum allowed size of 5MB":  true,
	"invalid query parameters":                       true,
	"invalid request":                                true,
	"item_id is required":                            true,
	"list_id is required":                            true,
	"message_id is required":                         true,
	"notification_id is required":                    true,
	"service_id is required":                         true,
	"shop_id is required":                            true,
	"unauthorized":                                   true,
	"vehicle_id is required":                         true,

	// The usage handler emits these fixed public messages for legacy requests.
	"shop access denied":     true,
	"shop vehicle not found": true,
	"usage adjustment would move tracked usage outside the supported range": true,
}

func WriteValidationError(c *gin.Context, message string) {
	if UsesContract2(c) {
		c.JSON(400, response.StandardResponse{Status: 400, Code: "invalid", Message: message})
		return
	}
	c.JSON(400, gin.H{"message": message, "details": "invalid request"})
}

func legacyPublicMessage(err error) string {
	var typed *Failure
	if errors.As(err, &typed) {
		return typed.PublicMessage
	}
	if _, ok := publicFailures[err.Error()]; ok {
		return err.Error()
	}
	if publicResponseMessages[err.Error()] {
		return err.Error()
	}
	if child := errors.Unwrap(err); child != nil {
		prefix := strings.TrimSuffix(err.Error(), child.Error())
		if legacyErrorPrefixes[prefix] {
			message := legacyPublicMessage(child)
			if message != "Unable to complete Shops request" {
				return prefix + message
			}
		}
	}
	return "Unable to complete Shops request"
}

// Only fixed source prefixes can surround an allowlisted public error.
var legacyErrorPrefixes = map[string]bool{
	"equipment access validation failed: ":                             true,
	"error iterating change rows: ":                                    true,
	"error iterating rows: ":                                           true,
	"failed to add list item: ":                                        true,
	"failed to add list items: ":                                       true,
	"failed to add member to shop: ":                                   true,
	"failed to add notification item: ":                                true,
	"failed to add notification items: ":                               true,
	"failed to adjust shop vehicle usage: ":                            true,
	"failed to check admin status: ":                                   true,
	"failed to check membership: ":                                     true,
	"failed to complete equipment service: ":                           true,
	"failed to count pmcs faults for equipment history: ":              true,
	"failed to count pmcs inspection comments for equipment history: ": true,
	"failed to count services: ":                                       true,
	"failed to create equipment service: ":                             true,
	"failed to create invite code: ":                                   true,
	"failed to create notification change record: ":                    true,
	"failed to create notification item: ":                             true,
	"failed to create notification items: ":                            true,
	"failed to create shop list: ":                                     true,
	"failed to create shop message: ":                                  true,
	"failed to create shop vehicle: ":                                  true,
	"failed to create shop: ":                                          true,
	"failed to create vehicle notification: ":                          true,
	"failed to deactivate invite code: ":                               true,
	"failed to delete equipment service: ":                             true,
	"failed to delete invite code: ":                                   true,
	"failed to delete message image: ":                                 true,
	"failed to delete notification item: ":                             true,
	"failed to delete notification items: ":                            true,
	"failed to delete shop list: ":                                     true,
	"failed to delete shop message: ":                                  true,
	"failed to delete shop vehicle: ":                                  true,
	"failed to delete shop: ":                                          true,
	"failed to delete vehicle notification: ":                          true,
	"failed to fetch updated settings: ":                               true,
	"failed to generate invite code: ":                                 true,
	"failed to get admin_only_lists setting: ":                         true,
	"failed to get attached shop list: ":                               true,
	"failed to get created list item with username: ":                  true,
	"failed to get created list items with usernames: ":                true,
	"failed to get created shop list with username: ":                  true,
	"failed to get created shop message: ":                             true,
	"failed to get current notification: ":                             true,
	"failed to get current vehicle: ":                                  true,
	"failed to get cursor-based shop messages: ":                       true,
	"failed to get due soon services: ":                                true,
	"failed to get equipment service: ":                                true,
	"failed to get equipment services: ":                               true,
	"failed to get invite code: ":                                      true,
	"failed to get invite codes: ":                                     true,
	"failed to get item: ":                                             true,
	"failed to get list item: ":                                        true,
	"failed to get list items with usernames: ":                        true,
	"failed to get list: ":                                             true,
	"failed to get member count: ":                                     true,
	"failed to get message: ":                                          true,
	"failed to get notification changes: ":                             true,
	"failed to get notification item: ":                                true,
	"failed to get notification items: ":                               true,
	"failed to get notification: ":                                     true,
	"failed to get overdue services: ":                                 true,
	"failed to get paginated shop messages: ":                          true,
	"failed to get rows affected: ":                                    true,
	"failed to get services by equipment: ":                            true,
	"failed to get services in date range: ":                           true,
	"failed to get services: ":                                         true,
	"failed to get shop list: ":                                        true,
	"failed to get shop lists with usernames: ":                        true,
	"failed to get shop members: ":                                     true,
	"failed to get shop message: ":                                     true,
	"failed to get shop messages count: ":                              true,
	"failed to get shop messages: ":                                    true,
	"failed to get shop notification changes: ":                        true,
	"failed to get shop notification items: ":                          true,
	"failed to get shop notifications: ":                               true,
	"failed to get shop settings: ":                                    true,
	"failed to get shop vehicle: ":                                     true,
	"failed to get shop vehicles: ":                                    true,
	"failed to get shop with stats: ":                                  true,
	"failed to get shop: ":                                             true,
	"failed to get shops for user: ":                                   true,
	"failed to get shops with stats: ":                                 true,
	"failed to get shops: ":                                            true,
	"failed to get user role: ":                                        true,
	"failed to get user shops with stats: ":                            true,
	"failed to get vehicle notification changes: ":                     true,
	"failed to get vehicle notification: ":                             true,
	"failed to get vehicle notifications with items: ":                 true,
	"failed to get vehicle notifications: ":                            true,
	"failed to get vehicle: ":                                          true,
	"failed to iterate notification items: ":                           true,
	"failed to iterate shop lists with items: ":                        true,
	"failed to iterate shop snapshot changes: ":                        true,
	"failed to iterate shop snapshot messages: ":                       true,
	"failed to iterate shop snapshot notifications: ":                  true,
	"failed to iterate shop snapshot services: ":                       true,
	"failed to iterate shop snapshot vehicles: ":                       true,
	"failed to iterate shops bootstrap equipment: ":                    true,
	"failed to iterate shops bootstrap summaries: ":                    true,
	"failed to iterate vehicle equipment services: ":                   true,
	"failed to iterate vehicle notification changes: ":                 true,
	"failed to iterate vehicle notifications: ":                        true,
	"failed to leave shop: ":                                           true,
	"failed to load cursor message: ":                                  true,
	"failed to marshal field changes: ":                                true,
	"failed to promote member to admin: ":                              true,
	"failed to query equipment for pmcs history: ":                     true,
	"failed to query notification items: ":                             true,
	"failed to query pmcs inspections for equipment history: ":         true,
	"failed to query shop equipment overview: ":                        true,
	"failed to query shop lists with items: ":                          true,
	"failed to query shop snapshot changes: ":                          true,
	"failed to query shop snapshot messages: ":                         true,
	"failed to query shop snapshot notifications: ":                    true,
	"failed to query shop snapshot services: ":                         true,
	"failed to query shop snapshot summary: ":                          true,
	"failed to query shop snapshot vehicles: ":                         true,
	"failed to query shops bootstrap equipment: ":                      true,
	"failed to query shops bootstrap summaries: ":                      true,
	"failed to query vehicle equipment services: ":                     true,
	"failed to query vehicle maintenance snapshot vehicle: ":           true,
	"failed to query vehicle notification changes: ":                   true,
	"failed to query vehicle notifications: ":                          true,
	"failed to remove list item: ":                                     true,
	"failed to remove list items: ":                                    true,
	"failed to remove member from shop: ":                              true,
	"failed to remove member: ":                                        true,
	"failed to remove notification item: ":                             true,
	"failed to remove notification items: ":                            true,
	"failed to scan change row: ":                                      true,
	"failed to scan equipment service: ":                               true,
	"failed to scan member row: ":                                      true,
	"failed to scan notification change: ":                             true,
	"failed to scan notification item: ":                               true,
	"failed to scan shop list item aggregate row: ":                    true,
	"failed to scan shop row: ":                                        true,
	"failed to scan shop snapshot message: ":                           true,
	"failed to scan shop snapshot vehicle: ":                           true,
	"failed to scan shops bootstrap equipment: ":                       true,
	"failed to scan shops bootstrap summary: ":                         true,
	"failed to scan vehicle notification: ":                            true,
	"failed to update admin_only_lists setting: ":                      true,
	"failed to update equipment service: ":                             true,
	"failed to update list item: ":                                     true,
	"failed to update member role: ":                                   true,
	"failed to update shop list: ":                                     true,
	"failed to update shop message: ":                                  true,
	"failed to update shop settings: ":                                 true,
	"failed to update shop vehicle usage: ":                            true,
	"failed to update shop vehicle: ":                                  true,
	"failed to update shop: ":                                          true,
	"failed to update vehicle notification: ":                          true,
	"failed to upload image: ":                                         true,
	"failed to upload message image: ":                                 true,
	"failed to validate service ownership: ":                           true,
	"failed to verify admin status: ":                                  true,
	"failed to verify delete permissions: ":                            true,
	"failed to verify list modification permissions: ":                 true,
	"failed to verify membership: ":                                    true,
	"failed to verify modify permissions: ":                            true,
	"failed to verify service access: ":                                true,
	"failed to verify shop membership: ":                               true,
	"failed to verify target user membership: ":                        true,
	"invalid end_date format: ":                                        true,
	"invalid invite code: ":                                            true,
	"invalid start_date format: ":                                      true,
	"invite code not found: ":                                          true,
	"list access validation failed: ":                                  true,
	"lock shop vehicle for usage adjustment: ":                         true,
	"shop vehicle not found: ":                                         true,
	"update shop vehicle usage: ":                                      true,
	"vehicle notification lookup failed: ":                             true,
	"vehicle notification not found: ":                                 true,
}
