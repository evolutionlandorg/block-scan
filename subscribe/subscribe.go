package subscribe

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/evolutionlandorg/block-scan/scan"
	"github.com/evolutionlandorg/block-scan/services"
	"github.com/evolutionlandorg/block-scan/util"
	"github.com/evolutionlandorg/block-scan/util/log"

	"github.com/evolutionlandorg/block-scan/util/ethclient"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/spf13/cast"
)

type Receipts struct {
	*services.Receipts
	Tx          string
	Timestamp   uint64
	BlockNumber uint64
}

type Subscribe struct {
	*scan.Polling
	wss string
}

func (p *Subscribe) Init(opt services.ScanEventsOptions) error {
	if p.Polling == nil {
		p.Polling = new(scan.Polling)
	}
	wss := os.Getenv(fmt.Sprintf("%s_WSS_RPC", strings.ToUpper(opt.Chain)))
	if !strings.HasPrefix(wss, "ws") || wss == "" {
		return fmt.Errorf("check if %s_WSS_RPC is a valid websocket connection", strings.ToUpper(opt.Chain))
	}
	p.wss = wss
	return p.Polling.Init(opt)
}

func (p *Subscribe) filterLogs(ctx context.Context, startBlock uint64, client *ethclient.Client) uint64 {
	query := new(ethereum.FilterQuery)
	for k := range p.Opt.ContractsName {
		contractsAddress := strings.ToLower(k.String())
		query.Addresses = append(query.Addresses, common.HexToAddress(contractsAddress))
	}
	for {
		endBlock, _ := client.BlockNumber(context.Background())
		if endBlock == 0 {
			time.Sleep(time.Second)
			continue
		}
		if endBlock <= startBlock {
			return endBlock
		}

		if endBlock-startBlock >= 500 {
			endBlock = startBlock + 500
		}

		query.FromBlock = big.NewInt(int64(startBlock))
		query.ToBlock = big.NewInt(int64(endBlock))
		rawLogs, err := client.FilterLogs(ctx, *query)

		if err != nil {
			log.Warn("%s FilterLogs error: %v. trying again.", p.Opt.Chain, err)
			continue
		}

		var (
			data        = make(map[string]*Receipts)
			blockNumber = make(map[uint64]uint64)
		)

		for _, v := range rawLogs {
			tx := v.TxHash.Hex()
			if _, ok := blockNumber[v.BlockNumber]; !ok {
				result, err := util.TryReturn(func() (result interface{}, err error) {
					return client.BlockByNumber(ctx, big.NewInt(int64(v.BlockNumber)))
				}, 10)
				if err != nil {
					log.Error("%s %s get block by number error: %s", p.Opt.Chain, tx, err)
					continue
				}
				blockNumber[v.BlockNumber] = result.(*types.Block).Time()
			}
			if _, ok := data[tx]; !ok {
				data[tx] = &Receipts{
					Tx:          v.TxHash.Hex(),
					Timestamp:   blockNumber[v.BlockNumber],
					BlockNumber: v.BlockNumber,
				}
			}
		}

		for _, v := range data {
			result, err := util.TryReturn(func() (result interface{}, err error) {
				resp, err := p.Opt.ChainIo.ReceiptLog(v.Tx)
				if err != nil {
					time.Sleep(time.Second)
					return nil, err
				}
				return resp, nil
			}, 10)
			util.Panic(err)
			data[v.Tx].Receipts = result.(*services.Receipts)
			if p.RunBeforePushMiddleware(v.Tx, v.Timestamp, data[v.Tx].Receipts) {
				_ = p.ReceiptDistribution(v.Tx, v.Timestamp, data[v.Tx].Receipts)
				p.Opt.SetStartBlock(cast.ToUint64(v.Receipts.BlockNumber))
			}
		}
		log.Warn("%s %d-%d block high filter logs %d", p.Opt.Chain, startBlock, endBlock, len(data))
		startBlock = endBlock
	}
}

func (p *Subscribe) WipeBlock(ctx context.Context) error {
	query := new(ethereum.FilterQuery)
	for k := range p.Opt.ContractsName {
		contractsAddress := strings.ToLower(k.String())
		query.Addresses = append(query.Addresses, common.HexToAddress(contractsAddress))
	}

	client, err := ethclient.DialContext(ctx, p.wss)
	if err != nil {
		return err
	}

	// 先筛选
	currentBlockNum := p.Opt.GetStartBlock()
	if currentBlockNum == 0 {
		currentBlockNum = p.Opt.InitBlock
	}

	query.FromBlock = big.NewInt(int64(p.filterLogs(ctx, currentBlockNum, client)))
	log.Debug("%s start subscribe latest block info", p.Opt.Chain)
	logs := make(chan types.Log)

	sub, err := client.SubscribeFilterLogs(ctx, *query, logs)
	if err != nil {
		return err
	}

	sleepTime := util.GetSleepTime()
	waitTime := util.GetDelayTime()
	t := time.NewTicker(sleepTime)
	defer t.Stop()

	data := make(map[string]*Receipts)
	push := func() {
		now := time.Now().Unix()
		for key, v := range data {
			if now-int64(v.Timestamp) < int64(waitTime.Seconds()) {
				continue
			}
			result, err := util.TryReturn(func() (result interface{}, err error) {
				resp, err := p.Opt.ChainIo.ReceiptLog(v.Tx)
				if err != nil {
					time.Sleep(time.Second)
					return nil, err
				}
				return resp, nil
			}, 10)
			util.Panic(err)
			data[key].Receipts = result.(*services.Receipts)
			log.Debug("%s push %s %d logs to queue", p.Opt.Chain, v.Tx, len(v.Logs))
			if p.RunBeforePushMiddleware(v.Tx, v.Timestamp, v.Receipts) {
				_ = p.ReceiptDistribution(v.Tx, v.Timestamp, v.Receipts)
				p.Opt.SetStartBlock(cast.ToUint64(v.Receipts.BlockNumber))
			}
			delete(data, key)
		}
	}

	for {
		select {
		case err := <-sub.Err():
			push()
			return err
		case <-ctx.Done():
			sub.Unsubscribe()
			push()
			return nil
		case vLog := <-logs:
			tx := vLog.TxHash.Hex()
			if _, ok := data[tx]; !ok {
				result, err := util.TryReturn(func() (result interface{}, err error) {
					return client.BlockByNumber(ctx, big.NewInt(int64(vLog.BlockNumber)))
				}, 10)
				if err != nil {
					continue
				}
				b := result.(*types.Block)
				data[tx] = &Receipts{
					Tx:        tx,
					Timestamp: b.Time(),
				}
			}
		case <-t.C:
			if len(data) <= 0 {
				continue
			}
			push()
		}
	}
}
