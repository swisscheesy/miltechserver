package shared

import (
	"database/sql"
	"errors"
	"github.com/go-jet/jet/v2/qrm"
)

var ErrNotFound = errors.New("resource not found")

type Failure struct {
	Code          string
	PublicMessage string
	Status        int
	Cause         error
}

func (f *Failure) Error() string { return f.PublicMessage }
func (f *Failure) Unwrap() error { return f.Cause }
func NormalizeNoRows(err error) error {
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, qrm.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
func ClassifyFailure(err error) *Failure {
	var failure *Failure
	if errors.As(err, &failure) {
		return failure
	}
	for _, entry := range []struct {
		err    error
		code   string
		status int
	}{
		{ErrShopAccessDenied, "denied", 403}, {ErrShopAdminRequired, "denied", 403}, {ErrShopCreatorOnly, "denied", 403},
		{ErrListAccessDenied, "denied", 403}, {ErrVehicleAccessDenied, "denied", 403}, {ErrAdminOnlyLists, "denied", 403},
		{ErrListNotFound, "list_not_found", 404}, {ErrNotificationNotFound, "notification_not_found", 404},
		{ErrShopNotFound, "shop_not_found", 404}, {ErrVehicleNotFound, "vehicle_not_found", 404},
		{ErrMemberNotFound, "member_not_found", 404}, {ErrAlreadyMember, "conflict", 409},
		{ErrInviteCodeInvalid, "invalid", 400}, {ErrInviteCodeExpired, "invalid", 400}, {ErrInviteCodeUsed, "conflict", 409},
		{ErrCannotRemoveSelf, "invalid", 422}, {ErrCannotRemoveCreator, "denied", 403},
	} {
		if errors.Is(err, entry.err) {
			return &Failure{entry.code, entry.err.Error(), entry.status, err}
		}
	}
	// Match only exact server-owned public messages, including wrapped leaves.
	for current := err; current != nil; current = errors.Unwrap(current) {
		if known, ok := publicFailures[current.Error()]; ok {
			known.PublicMessage = current.Error()
			known.Cause = err
			return &known
		}
	}
	if errors.Is(NormalizeNoRows(err), ErrNotFound) {
		return &Failure{"not_found", "resource not found", 404, err}
	}
	return &Failure{"internal_error", "Unable to complete Shops request", 500, err}
}

// Explicit legacy public-message allowlist; never expose an unknown error string.
var publicFailures = map[string]Failure{
	"no item found":                            {Code: "not_found", Status: 404},
	"No item found":                            {Code: "not_found", Status: 404},
	"access denied":                            {Code: "denied", Status: 403},
	"access denied to list":                    {Code: "denied", Status: 403},
	"access denied to vehicle":                 {Code: "denied", Status: 403},
	"access denied: admin privileges required": {Code: "denied", Status: 403},
	"access denied: insufficient permissions to create lists":                        {Code: "denied", Status: 403},
	"access denied: insufficient permissions to delete lists":                        {Code: "denied", Status: 403},
	"access denied: insufficient permissions to modify list items":                   {Code: "denied", Status: 403},
	"access denied: insufficient permissions to modify lists":                        {Code: "denied", Status: 403},
	"access denied: not a member of this shop":                                       {Code: "denied", Status: 403},
	"access denied: only service creators or shop admins can delete services":        {Code: "denied", Status: 403},
	"access denied: only service creators or shop admins can modify services":        {Code: "denied", Status: 403},
	"access denied: only shop administrators can modify settings":                    {Code: "denied", Status: 403},
	"access denied: only shop administrators can modify this setting":                {Code: "denied", Status: 403},
	"access denied: only shop admins can update shops":                               {Code: "denied", Status: 403},
	"access denied: only shop creator can perform this action":                       {Code: "denied", Status: 403},
	"access denied: only vehicle creator or shop admin can delete vehicles":          {Code: "denied", Status: 403},
	"access denied: only vehicle creator or shop admin can update equipment details": {Code: "denied", Status: 403},
	"access denied: user is not a member of this shop":                               {Code: "denied", Status: 403},
	"before_id and after_id cannot be used together":                                 {Code: "invalid", Status: 400},
	"cannot delete items from multiple notifications in a single operation":          {Code: "invalid", Status: 400},
	"cannot remove shop creator":                                                     {Code: "invalid", Status: 400},
	"cannot remove yourself from shop":                                               {Code: "invalid", Status: 400},
	"cursor message does not belong to this shop":                                    {Code: "invalid", Status: 400},
	"equipment and list must belong to the same shop":                                {Code: "invalid", Status: 400},
	"equipment not found or access denied":                                           {Code: "denied", Status: 403},
	"equipment service not found or access denied":                                   {Code: "denied", Status: 403},
	"invalid attached_shop_list":                                                     {Code: "invalid", Status: 400},
	"invalid include":                                                                {Code: "invalid", Status: 400},
	"invalid invite code":                                                            {Code: "invalid", Status: 400},
	"invalid limit":                                                                  {Code: "invalid", Status: 400},
	"invalid notification type: must be M1, PM, or MW":                               {Code: "invalid", Status: 400},
	"invalid usage adjustment":                                                       {Code: "invalid", Status: 400},
	"invite code has already been used":                                              {Code: "conflict", Status: 409},
	"invite code has expired":                                                        {Code: "invalid", Status: 400},
	"invite code is inactive":                                                        {Code: "invalid", Status: 400},
	"invite code not found":                                                          {Code: "not_found", Status: 404},
	"list is in use":                                                                 {Code: "list_in_use", Status: 409},
	"list item not found":                                                            {Code: "not_found", Status: 404},
	"list not found":                                                                 {Code: "list_not_found", Status: 404},
	"list not found or access denied":                                                {Code: "denied", Status: 403},
	"member not found":                                                               {Code: "not_found", Status: 404},
	"member not found in shop":                                                       {Code: "not_found", Status: 404},
	"message not found":                                                              {Code: "not_found", Status: 404},
	"message not found or user not authorized to delete":                             {Code: "denied", Status: 403},
	"message not found or user not authorized to update":                             {Code: "denied", Status: 403},
	"no items to add":                                                                {Code: "invalid", Status: 400},
	"no items to remove":                                                             {Code: "invalid", Status: 400},
	"no notification items found":                                                    {Code: "invalid", Status: 400},
	"no settings to update":                                                          {Code: "invalid", Status: 400},
	"notification item not found":                                                    {Code: "not_found", Status: 404},
	"notification not found":                                                         {Code: "notification_not_found", Status: 404},
	"only admins can create lists in this shop":                                      {Code: "denied", Status: 403},
	"only shop administrators can deactivate invite codes":                           {Code: "denied", Status: 403},
	"only shop administrators can delete invite codes":                               {Code: "denied", Status: 403},
	"only shop administrators can delete shops":                                      {Code: "denied", Status: 403},
	"only shop administrators can promote members":                                   {Code: "denied", Status: 403},
	"only shop administrators can remove members":                                    {Code: "denied", Status: 403},
	"operation payload conflicts with previous request":                              {Code: "operation_payload_conflict", Status: 409},
	"request body must contain exactly one JSON value":                               {Code: "invalid", Status: 400},
	"resource not found":                                                             {Code: "not_found", Status: 404},
	"service not found":                                                              {Code: "service_not_found", Status: 404},
	"service_hours must be non-negative":                                             {Code: "invalid", Status: 400},
	"shop list not found":                                                            {Code: "list_not_found", Status: 404},
	"shop not found":                                                                 {Code: "not_found", Status: 404},
	"shop not found or user not authorized to delete":                                {Code: "denied", Status: 403},
	"target user is not a member of this shop":                                       {Code: "invalid", Status: 400},
	"unauthorized":      {Code: "unauthorized", Status: 401},
	"unauthorized user": {Code: "unauthorized", Status: 401},
	"usage adjustment would move tracked usage outside the supported range": {Code: "invalid", Status: 400},
	"use leave shop endpoint to remove yourself":                            {Code: "invalid", Status: 400},
	"user is already a member of this shop":                                 {Code: "conflict", Status: 409},
	"user is not a member of this shop":                                     {Code: "denied", Status: 403},
	"vehicle not found":                                                     {Code: "not_found", Status: 404},
}
