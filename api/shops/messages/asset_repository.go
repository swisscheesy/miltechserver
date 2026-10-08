package messages

import (
	"context"
	"database/sql"
	"miltechserver/bootstrap"
)

type AssetRepository interface {
	Reserve(context.Context, *sql.Tx, *bootstrap.User, string, string) (Asset, error)
	Finalize(context.Context, *sql.Tx, *bootstrap.User, string) error
	FailReserved(context.Context, *sql.Tx, Asset, string) error
	ReplaceReferences(context.Context, *sql.Tx, *bootstrap.User, string, string) error
	Discard(context.Context, *sql.Tx, *bootstrap.User, string, string) error
	EnqueueShopCleanup(context.Context, *sql.Tx, string) error
}
