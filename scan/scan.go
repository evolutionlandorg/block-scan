package scan

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/evolutionlandorg/block-scan/services"

	"github.com/evolutionlandorg/block-scan/util/log"

	"github.com/garyburd/redigo/redis"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

var (
	newTxn = make(chan services.Tnx, 1000)
)

type Polling struct {
	ContractsName        map[services.ContractsAddress]services.ContractsName
	CallbackMethodPrefix []string
	GetCache             services.GetCacheFunc
	ChainIo              services.ChainIo
	Chain                string
	SleepTime            time.Duration
	GetCallbackFunc      services.GetCallbackFunc
}

func (p *Polling) ReceiptDistribution(tx string, BlockTimestamp uint64, receipt *services.Receipts) error {
	var logs []services.Log
	var exist = make(map[string]struct{})

	for index, v := range receipt.Logs {
		eventAddress := strings.ToLower(v.Address)
		key := fmt.Sprintf("%s_%s", eventAddress, v.Data)
		if _, ok := exist[key]; ok {
			continue
		}
		exist[key] = struct{}{}
		logs = append(logs, receipt.Logs[index])
	}
	if len(receipt.Logs) < 1 {
		return nil
	}

	receipt.Logs = logs
	fb := services.FilterBlock{
		Txid:           tx,
		Receipts:       receipt,
		BlockTimestamp: BlockTimestamp,
		Callback:       p.GetCallbackFunc(tx, BlockTimestamp, receipt),
	}

	for _, l := range receipt.Logs {
		eventAddress := strings.ToLower(l.Address)
		if _, ok := exist[eventAddress]; ok {
			continue
		}
		exist[eventAddress] = struct{}{}
		if p.ContractsName[services.ContractsAddress(eventAddress)] != "" {
			contractName := strings.ToLower(p.ContractsName[services.ContractsAddress(eventAddress)].String())
			for _, v := range p.CallbackMethodPrefix {
				if strings.EqualFold(v, contractName) {
					fb.ContractName = v
					_ = fb.Do()
					break
				}
			}

		}
	}
	return nil
}

func (p *Polling) Init(c services.ChainIo, cache services.GetCacheFunc, chain string,
	contractsName map[services.ContractsAddress]services.ContractsName, sleepTime time.Duration,
	getCallbackFunc services.GetCallbackFunc, callbackMethodPrefix []string) error {
	p.GetCache = cache
	p.ContractsName = contractsName
	p.Chain = chain
	p.SleepTime = sleepTime
	p.GetCallbackFunc = getCallbackFunc
	p.ChainIo = c
	p.CallbackMethodPrefix = callbackMethodPrefix
	return nil
}

func (p *Polling) WipeBlock(ctx context.Context, initBlock uint64) error {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case txn := <-newTxn:
				// check Transaction fail
				if status := p.ChainIo.GetTransactionStatus(txn.Tx); status == "Fail" {
					continue
				}
				receipt, _ := p.ChainIo.ReceiptLog(txn.Tx)
				// maybe network abnormal or confirmed delay
				if receipt == nil || len(receipt.Logs) == 0 {
					newTxn <- txn
					continue
				}
				_ = p.ReceiptDistribution(txn.Tx, txn.BlockTimestamp, receipt)
			}
		}
	}()
	log.Debug("start %s wipeBlock", p.Chain)
	var (
		currentBlockNum uint64
		ticker          = time.NewTicker(time.Second)
		filterContracts []string
	)
	for k := range p.ContractsName {
		filterContracts = append(filterContracts, k.String())
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
		span, spanCtx := tracer.StartSpanFromContext(ctx, "daemons.worker",
			tracer.ServiceName("evo-pvp-worker"),
			tracer.SpanType(ext.SpanTypeMessageConsumer),
			tracer.Measured(),
			tracer.Tag("worker-name", "WipeBlock"),
			tracer.Tag("Chain", p.Chain),
		)

		chainCurrentBlockNum := p.ChainIo.BlockNumber()
		if chainCurrentBlockNum == 0 {
			span.Finish()
			continue
		}
		currentBlockNum, _ = redis.Uint64(p.GetCache(spanCtx)("HGET", "WipeBlock", p.Chain))
		if currentBlockNum <= 0 {
			currentBlockNum = initBlock
		}
		chainCurrentBlockNum = chainCurrentBlockNum - currentBlockNum
		if currentBlockNum < chainCurrentBlockNum {
			for i := currentBlockNum + 1; i <= chainCurrentBlockNum; i++ {
				txIDs, contracts, blockTimeStamp, transactionTo := p.ChainIo.FilterTrans(uint64(i), filterContracts)
				if len(txIDs) == 0 {
					continue
				}
				log.Debug("%s find tx id %v; transaction contracts %v", p.Chain, txIDs, transactionTo)
				for index, txID := range txIDs {
					newTxn <- services.Tnx{Tx: txID, BlockTimestamp: blockTimeStamp, Contract: contracts[index]}
				}
			}

			currentBlockNum = chainCurrentBlockNum
			_, _ = p.GetCache(spanCtx)("HSET", "WipeBlock", p.Chain, currentBlockNum)
		}
		span.Finish()
		time.Sleep(p.SleepTime)
	}
}
