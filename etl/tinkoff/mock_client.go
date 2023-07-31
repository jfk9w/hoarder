package tinkoff

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/jfk9w-go/tinkoff-api"
)

type mockClient struct {
}

func (m *mockClient) AccountsLightIb(ctx context.Context) (tinkoff.AccountsLightIbOut, error) {
	file, err := os.Open("etl/tinkoff/data/accounts.json")
	if err != nil {
		return nil, err
	}

	defer file.Close()

	var accounts tinkoff.AccountsLightIbOut
	if err := json.NewDecoder(file).Decode(&accounts); err != nil {
		return nil, err
	}

	return accounts, nil
}

func (m *mockClient) Operations(ctx context.Context, in *tinkoff.OperationsIn) (tinkoff.OperationsOut, error) {
	file, err := os.Open(fmt.Sprintf("etl/tinkoff/data/operations_%s.json", in.Account))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, err
	}

	defer file.Close()

	var operations tinkoff.OperationsOut
	if err := json.NewDecoder(file).Decode(&operations); err != nil {
		return nil, err
	}

	return operations, nil
}

func (m *mockClient) ShoppingReceipt(ctx context.Context, in *tinkoff.ShoppingReceiptIn) (*tinkoff.ShoppingReceiptOut, error) {
	return nil, tinkoff.ErrNoDataFound
}

func (m *mockClient) InvestOperationTypes(ctx context.Context) (*tinkoff.InvestOperationTypesOut, error) {
	//TODO implement me
	panic("implement me")
}

func (m *mockClient) InvestAccounts(ctx context.Context, in *tinkoff.InvestAccountsIn) (*tinkoff.InvestAccountsOut, error) {
	//TODO implement me
	panic("implement me")
}

func (m *mockClient) InvestOperations(ctx context.Context, in *tinkoff.InvestOperationsIn) (*tinkoff.InvestOperationsOut, error) {
	//TODO implement me
	panic("implement me")
}

func (m *mockClient) Close() {
	//TODO implement me
	panic("implement me")
}
