package db

import (
	"context"
	"database/sql"
	"fmt"
)

type Store struct {
	*Queries
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {

	return &Store{Queries: New(db), db: db}
}

var txKey = struct{}{}

func (store *Store) execTransaction(ctx context.Context, cb func(q *Queries) error) error {
	tx, _ := store.db.BeginTx(ctx, nil)

	q := New(tx)

	err := cb(q)
	if err != nil {
		rbErr := tx.Rollback()
		if rbErr != nil {
			return fmt.Errorf("txErr: %s, rbErr: %s", err, rbErr)
		}
		fmt.Println(ctx.Value(txKey), err)
		//os.Exit(1)
		return err
	}

	return tx.Commit()
}

type TransferTxParams struct {
	FromAccount int64 `json:"from_account"`
	ToAccount   int64 `json:"to_account"`
	Amount      int64 `json:"amount"`
}

type TransferTxResult struct {
	Transfer    Transfer `json:"transfer"`
	FromAccount Account  `json:"from_account"`
	ToAccount   Account  `json:"to_account"`
	FromEntry   Entry    `json:"from_entry"`
	ToEntry     Entry    `json:"to_entry"`
}

func (store *Store) TransferTx(ctx context.Context, arg TransferTxParams) (TransferTxResult, error) {

	var result TransferTxResult
	err := store.execTransaction(ctx, func(q *Queries) error {
		var err error
		//ctxVal := ctx.Value(txKey)
		//fmt.Println(ctxVal, " create transfer")
		result.Transfer, err = q.CreateTransfer(ctx, CreateTransferParams{
			FromAccountID: arg.FromAccount,
			ToAccountID:   arg.ToAccount,
			Amount:        arg.Amount,
		})

		if err != nil {
			return err
		}
		//fmt.Println(ctxVal, " create fromEntry")
		result.FromEntry, err = q.CreateEntry(ctx, CreateEntryParams{AccountID: arg.FromAccount,
			Amount: -arg.Amount})

		if err != nil {
			return err
		}
		//fmt.Println(ctxVal, " create toEntry")
		result.ToEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: arg.ToAccount,
			Amount:    arg.Amount,
		})

		if err != nil {
			return err
		}

		if arg.FromAccount < arg.ToAccount {
			result.FromAccount, result.ToAccount, err = AddMoney(ctx, q, arg.FromAccount, -arg.Amount, arg.ToAccount, arg.Amount)
		} else {
			result.FromAccount, result.ToAccount, err = AddMoney(ctx, q, arg.ToAccount, arg.Amount, arg.FromAccount, -arg.Amount)
		}

		return err
	})

	return result, err
}

func AddMoney(ctx context.Context, q *Queries, account1Id int64, amount1 int64,
	account2Id int64, amount2 int64) (account1 Account, account2 Account, err error) {
	account1, err = q.AddAccountBalance(ctx, AddAccountBalanceParams{
		ID:     account1Id,
		Amount: amount1,
	})

	if err != nil {
		return
	}

	account2, err = q.AddAccountBalance(ctx, AddAccountBalanceParams{
		ID:     account2Id,
		Amount: amount2,
	})
	return
}
