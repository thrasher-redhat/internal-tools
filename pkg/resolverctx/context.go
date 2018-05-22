package resolverctx

import (
	"context"

	"github.com/thrasher-redhat/internal-tools/pkg/db"
)

type key int

var txKey key

// WithTx returns a new context with the given transaction
func WithTx(ctx context.Context, tx db.ReadQuerier) context.Context {
	return context.WithValue(ctx, txKey, tx)
}

// GetTx returns the transaction stored in ctx, if any
func GetTx(ctx context.Context) (db.ReadQuerier, bool) {
	tx, ok := ctx.Value(txKey).(db.ReadQuerier)
	return tx, ok
}
