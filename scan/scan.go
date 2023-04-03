package scan

import (
	"context"
	"fmt"
	"github.com/evolutionlandorg/block-scan/metrics"
	"strings"
	"time"

	"github.com/evolutionlandorg/block-scan/services"
	"github.com/evolutionlandorg/block-scan/util"
	"github.com/evolutionlandorg/block-scan/util/log"
	"github.com/spf13/cast"
)

var (
	newTxn = make(chan services.Tnx, 1000)
)

type Polling struct {
	Opt     services.ScanEventsOptions
	metrics metrics.Metrics
}

func (p *Polling) SetMetrics(metrics metrics.Metrics) {
	p.metrics = metrics
}

func (p *Polling) RunBeforePushMiddleware(tx string, BlockTimestamp uint64, receipt *services.Receipts) bool {
	for _, v := range p.Opt.BeforePushMiddleware {
		if v == nil {
			continue
		}
		if !v(tx, BlockTimestamp, receipt) {
			return false
		}
	}
	return true
}

func (p *Polling) ReceiptDistribution(tx string, BlockTimestamp uint64, receipt *services.Receipts) error {
	var logs []services.Log
	var exist = make(map[string]struct{})

	for index, v := range receipt.Logs {
		eventAddress := strings.ToLower(v.Address)
		key := fmt.Sprintf("%s_%s_%s", eventAddress, v.Data, strings.Join(v.Topics, ""))
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
		Callback:       p.Opt.GetCallbackFunc(tx, BlockTimestamp, receipt),
	}

	for _, l := range receipt.Logs {
		eventAddress := strings.ToLower(l.Address)
		if _, ok := exist[eventAddress]; ok {
			continue
		}
		exist[eventAddress] = struct{}{}
		if p.Opt.ContractsName[services.ContractsAddress(eventAddress)] != "" {
			contractName := strings.ToLower(p.Opt.ContractsName[services.ContractsAddress(eventAddress)].String())
			for _, v := range p.Opt.CallbackMethodPrefix {
				if strings.EqualFold(v, contractName) {
					fb.ContractName = v
					p.metrics.ScanCallbackTotal(v)
					_ = fb.Do()
					break
				}
			}
		}
	}
	return nil
}

func (p *Polling) Init(opt services.ScanEventsOptions) error {
	p.Opt = opt
	return opt.Check()
}

func (p *Polling) WipeBlock(ctx context.Context) error {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case txn := <-newTxn:
				// check Transaction fail
				if status := p.Opt.ChainIo.GetTransactionStatus(txn.Tx); status == "Fail" {
					continue
				}
				receipt, _ := p.Opt.ChainIo.ReceiptLog(txn.Tx)
				// maybe network abnormal or confirmed delay
				if receipt == nil || len(receipt.Logs) == 0 {
					newTxn <- txn
					continue
				}
				p.metrics.ScanTxTotal(p.Opt.Chain)
				if p.RunBeforePushMiddleware(txn.Tx, txn.BlockTimestamp, receipt) {
					_ = p.ReceiptDistribution(txn.Tx, txn.BlockTimestamp, receipt)
					p.Opt.SetStartBlock(cast.ToUint64(receipt.BlockNumber))
				}
			}
		}
	}()
	log.Debug("start %s wipeBlock", p.Opt.Chain)
	var (
		currentBlockNum uint64
		filterContracts []string
	)
	for k := range p.Opt.ContractsName {
		filterContracts = append(filterContracts, k.String())
	}
	sleepTime := util.GetSleepTime()
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		chainCurrentBlockNum := p.Opt.ChainIo.BlockNumber()
		if chainCurrentBlockNum == 0 {
			continue
		}
		if currentBlockNum <= 0 {
			currentBlockNum = p.Opt.GetStartBlock()
		}
		if currentBlockNum <= 0 {
			currentBlockNum = p.Opt.InitBlock
		}

		if currentBlockNum < chainCurrentBlockNum {
			for i := currentBlockNum + 1; i <= chainCurrentBlockNum; i++ {
				txIDs, contracts, blockTimeStamp, transactionTo := p.Opt.ChainIo.FilterTrans(uint64(i), filterContracts)
				if i%100 == 0 && len(txIDs) == 0 {
					log.Debug("scan %s current block %d", p.Opt.Chain, i)
					continue
				}
				if len(txIDs) == 0 {
					continue
				}
				log.Debug("%s %d find tx id %v; transaction contracts %v", p.Opt.Chain, i, txIDs, transactionTo)
				for index, txID := range txIDs {
					newTxn <- services.Tnx{Tx: txID, BlockTimestamp: blockTimeStamp, Contract: contracts[index]}
				}
			}
			currentBlockNum = chainCurrentBlockNum
		}
		time.Sleep(sleepTime)
	}
}
