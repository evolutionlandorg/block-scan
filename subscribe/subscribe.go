package subscribe

import (
	"block-scan/scan"
	"block-scan/services"
	"block-scan/util"
	"context"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"block-scan/util/ethclient"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/garyburd/redigo/redis"
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

func (p *Subscribe) Init(c services.ChainIo, cache services.GetCacheFunc, chain string,
	contractsName map[services.ContractsAddress]services.ContractsName, sleepTime time.Duration, getCallbackFunc services.GetCallbackFunc) error {
	if p.Polling == nil {
		p.Polling = new(scan.Polling)
	}
	if err := p.Polling.Init(c, cache, chain, contractsName, sleepTime, getCallbackFunc); err != nil {
		return err
	}
	wss := os.Getenv(fmt.Sprintf("%s_WSS_RPC", strings.ToUpper(p.Chain)))
	if !strings.HasPrefix(wss, "ws") || wss == "" {
		return fmt.Errorf("check if %s_WSS_RPC is a valid websocket connection", strings.ToUpper(p.Chain))
	}
	p.wss = wss
	return nil
}

func (p *Subscribe) filterLogs(ctx context.Context, startBlock uint64, client *ethclient.Client) uint64 {
	query := new(ethereum.FilterQuery)
	for k := range p.ContractsName {
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
			continue
		}

		var (
			data = make(map[string]*Receipts)
		)

		for _, v := range rawLogs {
			tx := v.TxHash.Hex()
			if _, ok := data[tx]; !ok {
				result, err := util.TryReturn(func() (result interface{}, err error) {
					return client.BlockByNumber(ctx, big.NewInt(int64(v.BlockNumber)))
				}, 10)
				if err != nil {
					continue
				}
				data[tx] = &Receipts{
					Receipts: &services.Receipts{
						Status:      "0x01",
						ChainSource: p.Chain,
						Solidity:    true,
						BlockHash:   v.BlockHash.Hex(),
						BlockNumber: cast.ToString(v.BlockNumber),
					},
					Tx:          v.TxHash.Hex(),
					Timestamp:   result.(*types.Block).Time(),
					BlockNumber: v.BlockNumber,
				}
			}

			l := &services.Log{Address: v.Address.Hex(), Data: util.BytesToHex(v.Data)}
			for _, t := range v.Topics {
				l.Topics = append(l.Topics, t.Hex())
			}
			data[tx].Receipts.Logs = append(data[tx].Receipts.Logs, *l)
		}
		for _, v := range data {
			_ = p.ReceiptDistribution(v.Tx, v.Timestamp, v.Receipts)
		}
		startBlock = endBlock
		p.setCurrentBlockHeightToCache(ctx, p.Chain, startBlock)
	}
}

func (p *Subscribe) WipeBlock(ctx context.Context, initBlock uint64) error {
	var filterContracts []string
	query := new(ethereum.FilterQuery)
	for k := range p.ContractsName {
		contractsAddress := strings.ToLower(k.String())
		filterContracts = append(filterContracts, contractsAddress)
		query.Addresses = append(query.Addresses, common.HexToAddress(contractsAddress))
	}

	client, err := ethclient.DialContext(ctx, p.wss)
	if err != nil {
		return err
	}

	// 先筛选
	currentBlockNum, _ := p.getCurrentBlockHeightFromCache(ctx, p.Chain)
	if currentBlockNum == 0 {
		currentBlockNum = initBlock
	}

	query.FromBlock = big.NewInt(int64(p.filterLogs(ctx, currentBlockNum, client)))
	logs := make(chan types.Log)

	sub, err := client.SubscribeFilterLogs(ctx, *query, logs)
	if err != nil {
		return err
	}

	sleepTime := time.Second * 2
	waitTime := time.Second * 5
	t := time.NewTicker(sleepTime)
	defer t.Stop()

	data := make(map[string]*Receipts)
	push := func() {
		now := time.Now().Unix()
		for key, v := range data {
			if now-int64(v.Timestamp) < int64(waitTime.Seconds()) {
				continue
			}
			delete(data, key)
			p.setCurrentBlockHeightToCache(ctx, p.Chain, cast.ToUint64(v.Receipts.BlockNumber))
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
					Receipts: &services.Receipts{
						Status:      "0x01",
						ChainSource: p.Chain,
						Solidity:    true,
						BlockHash:   vLog.BlockHash.Hex(),
						BlockNumber: cast.ToString(vLog.BlockNumber),
					},
					Tx:        tx,
					Timestamp: b.Time(),
				}
			}
			l := &services.Log{Address: vLog.Address.Hex(), Data: util.BytesToHex(vLog.Data)}
			for _, t := range vLog.Topics {
				l.Topics = append(l.Topics, t.Hex())
			}
			data[tx].Receipts.Logs = append(data[tx].Receipts.Logs, *l)
		case <-t.C:
			if len(data) <= 0 {
				continue
			}
			push()
		}
	}
}

func (s Subscribe) getCurrentBlockHeightFromCache(ctx context.Context, cacheKey string) (uint64, error) {
	return redis.Uint64(s.GetCache(ctx)("HGET", "WipeBlock", cacheKey))
}

func (s Subscribe) setCurrentBlockHeightToCache(ctx context.Context, cacheKey string, value uint64) {
	_, _ = s.GetCache(ctx)("HSET", "WipeBlock", cacheKey, value)
}
