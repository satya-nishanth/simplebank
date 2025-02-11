package db

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTransferTx(t *testing.T) {
	store := NewStore(testDB)

	fAccount := createRandomAccount(t)
	tAccount := createRandomAccount(t)
	amount := int64(10)
	n := 5
	errsChan := make(chan error)
	resultChan := make(chan TransferTxResult)
	for i := 0; i < n; i++ {
		txName := fmt.Sprintf("tx %d", i+1)
		go func() {
			ctx := context.WithValue(context.Background(), txKey, txName)
			result, err := store.TransferTx(ctx, TransferTxParams{
				FromAccount: fAccount.ID,
				ToAccount:   tAccount.ID,
				Amount:      amount,
			})

			if err != nil {
				errsChan <- err
				return
			}
			//fmt.Println(txName, " commited")
			resultChan <- result
		}()
	}
	existed := make(map[int]bool)
	fmt.Println(">> before balance: ", fAccount.Balance, tAccount.Balance)
	for i := 0; i < n; i++ {
		select {
		case err := <-errsChan:
			require.NoError(t, err)
		case transferResult := <-resultChan:
			require.NotEmpty(t, transferResult)

			transfer := transferResult.Transfer

			require.NotEmpty(t, transfer)
			require.Equal(t, fAccount.ID, transfer.FromAccountID)
			require.Equal(t, tAccount.ID, transfer.ToAccountID)
			require.Equal(t, amount, transfer.Amount)

			_, err := store.GetTransfer(context.Background(), transfer.ID)
			//fmt.Printf("transfer %+v \n", transfer)
			require.NoError(t, err)

			fromEntry := transferResult.FromEntry

			require.Equal(t, fAccount.ID, fromEntry.AccountID)
			require.Equal(t, -amount, fromEntry.Amount)
			require.NotZero(t, fromEntry.ID)

			_, err = store.GetEntry(context.Background(), fromEntry.ID)
			require.NoError(t, err)

			toEntry := transferResult.ToEntry
			require.Equal(t, tAccount.ID, toEntry.AccountID)
			require.Equal(t, amount, toEntry.Amount)
			require.NotZero(t, toEntry.ID)

			_, err = store.GetEntry(context.Background(), toEntry.ID)
			require.NoError(t, err)

			fromAccount := transferResult.FromAccount
			require.NotEmpty(t, fromAccount)
			require.Equal(t, fAccount.ID, fromAccount.ID)

			toAccount := transferResult.ToAccount
			require.NotEmpty(t, toAccount)
			require.Equal(t, tAccount.ID, toAccount.ID)

			//fmt.Println(">>",, fromAccount.Balance, toAccount.Balance)

			diff1 := fAccount.Balance - fromAccount.Balance
			diff2 := toAccount.Balance - tAccount.Balance
			require.Equal(t, diff1, diff2)
			require.True(t, diff1 > 1)
			require.True(t, diff1%amount == 0)

			k := int(diff1 / amount)
			require.True(t, k >= 1 && k <= n)
			require.NotContains(t, existed, k)
			existed[k] = true

		}

	}
	updateAccountFrom, err := store.GetAccount(context.Background(), fAccount.ID)
	require.NoError(t, err)

	updateAccountTo, err := store.GetAccount(context.Background(), tAccount.ID)
	require.NoError(t, err)

	fmt.Println(">> after bal: ", updateAccountFrom.Balance, updateAccountTo.Balance)
	require.Equal(t, fAccount.Balance-int64(n)*amount, updateAccountFrom.Balance)
	require.Equal(t, tAccount.Balance+int64(n)*amount, updateAccountTo.Balance)

}

func TestTransferDeadlockTx(t *testing.T) {
	store := NewStore(testDB)

	account1 := createRandomAccount(t)
	account2 := createRandomAccount(t)
	amount := int64(10)
	n := 10
	errsChan := make(chan error)
	resultChan := make(chan TransferTxResult)
	for i := 0; i < n; i++ {
		fAccount := account1
		tAccount := account2
		if i%2 == 1 {
			fAccount = account2
			tAccount = account1
		}
		txName := fmt.Sprintf("tx %d", i+1)
		go func() {
			ctx := context.WithValue(context.Background(), txKey, txName)
			result, err := store.TransferTx(ctx, TransferTxParams{
				FromAccount: fAccount.ID,
				ToAccount:   tAccount.ID,
				Amount:      amount,
			})

			if err != nil {
				errsChan <- err
				return
			}
			//fmt.Println(txName, " commited")
			resultChan <- result
		}()
	}

	for i := 0; i < n; i++ {
		select {
		case err := <-errsChan:
			require.NoError(t, err)
		case transferResult := <-resultChan:
			require.NotEmpty(t, transferResult)
		}

	}
	updateAccountFrom, err := store.GetAccount(context.Background(), account1.ID)
	require.NoError(t, err)

	updateAccountTo, err := store.GetAccount(context.Background(), account2.ID)
	require.NoError(t, err)

	fmt.Println(">> after bal: ", updateAccountFrom.Balance, updateAccountTo.Balance)
	require.Equal(t, account1.Balance, updateAccountFrom.Balance)
	require.Equal(t, account2.Balance, updateAccountTo.Balance)
}
