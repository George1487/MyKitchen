package transaction

import "context"

// TxManager runs a callback in a transaction, passing the transaction in its context.
type TxManager interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}
