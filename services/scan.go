package services

import (
	"context"
	"fmt"
	"github.com/evolutionlandorg/block-scan/metrics"
	"reflect"
	"strings"
	"time"

	"github.com/evolutionlandorg/block-scan/util/log"
	"github.com/google/uuid"
	"github.com/itering/go-workers"
	"github.com/pkg/errors"
)

type (
	CacheFunc        func(commandName string, args ...interface{}) (reply interface{}, err error)
	GetCacheFunc     func(ctx context.Context) CacheFunc
	ContractsAddress string
	ContractsName    string
)

type GetCallbackFunc func(tx string, blockTimestamp uint64, receipt *Receipts) interface{}
type BeforePushFunc func(tx string, BlockTimestamp uint64, receipt *Receipts) bool

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

type ScanEventsOptions struct {
	ChainIo              ChainIo
	Chain                string
	ContractsName        map[ContractsAddress]ContractsName
	SleepTime            time.Duration
	GetCallbackFunc      GetCallbackFunc
	CallbackMethodPrefix []string
	InitBlock            uint64
	RunForever           bool
	BeforePushMiddleware []BeforePushFunc
	GetStartBlock        func() uint64
	SetStartBlock        func(currentBlockNum uint64)
}

func (s *ScanEventsOptions) Check() error {
	if s.ChainIo == nil {
		return errors.New("chainIo must be not nil")
	}
	if s.GetStartBlock == nil {
		return errors.New("GetStartBlock must be not nil")
	}
	if s.SetStartBlock == nil {
		return errors.New("SetStartBlock must be not nil")
	}
	if s.Chain == "" {
		return errors.New("chain must be not nil")
	}
	if len(s.ContractsName) == 0 {
		return errors.New("contractsName must be not nil")
	}
	if s.GetCallbackFunc == nil {
		return errors.New("getCallbackFunc must be not nil")
	}
	if s.SleepTime == 0 {
		s.SleepTime = time.Second * 5
	}
	return nil
}

type Scan interface {
	Init(opt ScanEventsOptions) error
	WipeBlock(ctx context.Context) error
	SetMetrics(metrics metrics.Metrics)
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
	methodName := fmt.Sprintf("%sCallback", fb.ContractName)
	methodFunc := wReflect.MethodByName(methodName)
	if !methodFunc.IsValid() {
		return
	}
	res := methodFunc.Call([]reflect.Value{reflect.ValueOf(context.Background())})
	if v := res[0].Interface(); v != nil {
		if err, ok := v.(error); ok && !strings.EqualFold(err.Error(), "tx exist") {
			log.DPanic("Process error. %s", err)
		}
	}
}
