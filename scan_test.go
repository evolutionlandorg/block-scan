package block_scan

import (
	"context"
	"testing"
	"time"

	"github.com/evolutionlandorg/block-scan/services"
	"github.com/stretchr/testify/assert"
)

type MockChainIo struct {
}

func (m *MockChainIo) ReceiptLog(_ string) (*services.Receipts, error) {
	return &services.Receipts{
		BlockNumber: "1",
		Logs: []services.Log{
			{
				Topics:  []string{"awD"},
				Data:    "111",
				Address: "222",
			},
		},
		Status:      "0x01",
		ChainSource: "Crab",
		BlockHash:   "1",
	}, nil
}

func (m *MockChainIo) BlockNumber() uint64 {
	return 10
}

func (m *MockChainIo) FilterTrans(blockNum uint64, _ []string) (txn []string, contracts []string, timestamp uint64, transactionTo []string) {
	if blockNum >= 1 {
		transactionTo = append(transactionTo, "222")
		timestamp = 123456789
		contracts = append(contracts, "222")
		txn = append(txn, "1")
		return
	}
	return
}

func (m *MockChainIo) BlockHeader(_ uint64) *services.BlockHeader {
	return &services.BlockHeader{
		BlockTimeStamp: 123456789,
		Hash:           "1",
	}
}

func (m *MockChainIo) GetTransactionStatus(_ string) string {
	return "0x01"
}

type FakeCallback struct {
	tx             string
	blockTimestamp uint64
	receipt        *services.Receipts
}

func (f *FakeCallback) FakeCallback(_ context.Context) error {
	return nil
}

func TestStartScanChainEvents(t *testing.T) {
	f := new(FakeCallback)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	assert.NoError(t, StartScanChainEvents(ctx, POLLING, services.ScanEventsOptions{
		ChainIo: new(MockChainIo),
		GetStartBlock: func() uint64 {
			return 1
		},
		SetStartBlock: func(currentBlockNum uint64) {},
		Chain:         "Crab",
		ContractsName: map[services.ContractsAddress]services.ContractsName{
			services.ContractsAddress("222"): services.ContractsName("fake"),
		},
		GetCallbackFunc: func(tx string, blockTimestamp uint64, receipt *services.Receipts) interface{} {
			f.tx = tx
			f.blockTimestamp = blockTimestamp
			f.receipt = receipt
			return f
		},
		CallbackMethodPrefix: []string{"Fake"},
		InitBlock:            1,
	}))

	assert.Equal(t, f.tx, "1")
	assert.Equal(t, f.blockTimestamp, uint64(123456789))
	assert.Equal(t, f.receipt.BlockHash, "1")
	assert.Equal(t, f.receipt.BlockNumber, "1")

}
