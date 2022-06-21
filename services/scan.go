package services

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/itering/go-workers"
)

type (
	CacheFunc        func(commandName string, args ...interface{}) (reply interface{}, err error)
	GetCacheFunc     func(ctx context.Context) CacheFunc
	ContractsAddress string
	ContractsName    string
)

type GetCallbackFunc func(tx string, receipt *Receipts) interface{}

type ChainIo interface {
	ReceiptLog(tx string) (*Receipts, error)
	BlockNumber() uint64
	FilterTrans(blockNum uint64, filter []string) (txn []string, contracts []string, timestamp uint64, transactionTo []string)
	BlockHeader(blockNum uint64) *BlockHeader
	GetTransactionStatus(tx string) string
}

type Tnx struct {
	Tx             string
	BlockTimestamp uint64
	Contract       string
}

type Receipts struct {
	BlockNumber      string `json:"block_number"`
	Logs             []Log  `json:"logs"`
	Status           string `json:"status"`
	ChainSource      string `json:"chainSource"`
	GasUsed          string `json:"gasUsed"`
	LogsBloom        string `json:"logsBloom"`
	Solidity         bool   `json:"solidity"`
	TransactionIndex string `json:"transactionIndex"`
	BlockHash        string `json:"blockHash"`
}

type Log struct {
	Topics  []string `json:"topics"`
	Data    string   `json:"data"`
	Address string   `json:"address"`
}

type BlockHeader struct {
	BlockTimeStamp uint64
	Hash           string
}

type Scan interface {
	Init(c ChainIo, cache GetCacheFunc, chain string, contractsName map[ContractsAddress]ContractsName, sleepTime time.Duration, getCallbackFunc GetCallbackFunc) error
	WipeBlock(ctx context.Context, initBlock uint64) error
}

type FilterBlock struct {
	ContractName   string
	Txid           string
	Receipts       *Receipts
	BlockTimestamp uint64
	Callback       interface{}
}

func (c ContractsAddress) String() string {
	return string(c)
}

func (c ContractsName) String() string {
	return string(c)
}

func (fb *FilterBlock) Do() error {
	var startTask = func(tx, contractName string, blockTimestamp uint64) map[string]interface{} {
		taskId := uuid.New().String()
		return map[string]interface{}{
			"tx":              tx,
			"contract_name":   contractName,
			"task_id":         taskId,
			"block_timestamp": blockTimestamp,
			"chain":           fb.Receipts.ChainSource,
			"receipts":        fb.Receipts,
		}
	}

	if !fb.Receipts.Solidity {
		fb.TronSolidityProcess()
		return nil
	}
	queueName := fmt.Sprintf("%sProcess", strings.ToLower(fb.Receipts.ChainSource))
	_, err := workers.Enqueue(queueName, queueName, startTask(fb.Txid, fb.ContractName, fb.BlockTimestamp))
	return err
}

func (fb *FilterBlock) TronSolidityProcess() {
	if fb.Callback == nil {
		return
	}
	ecInstant := fb.Callback
	wReflect := reflect.ValueOf(&ecInstant).Elem()
	wReflect.MethodByName(fb.ContractName + "Callback").Call(nil)
}
