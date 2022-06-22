<div id="top"></div>


## Getting Started

### Prerequisites
* golang version >= 1.17

### Installation
```shell
go get github.com/evolutionlandorg/block-scan
```

## Usage

```go
type ChainIo struct{
}

func (c *ChainIo)ReceiptLog(tx string) (*services.Receipts, error){
	// get receipt log from chain
}

func (c *ChainIo)BlockNumber() uint64{
	// get last block number from chain
}

func (c *ChainIo)FilterTrans(blockNum uint64, filter []string) (txn []string, contracts []string, timestamp uint64, transactionTo []string){
    // Filter the required log from the specified block height
}

func (c *ChainIo)BlockHeader(blockNum uint64) *services.BlockHeader{
	// get BlockHeader from chain
}

func (c *ChainIo)GetTransactionStatus(tx string) string{
	// get tx status from chain
	// status in (0x01, 0x00)
}
func main(){
	err := block_scan.StartScanChainEvents(ctx, block_scan.POLLING, &block_scan.StartScanChainEventsOptions{
        CallbackMethodPrefix: []string{"Transfer"}, // GetCallbackFunc must have TransferCallback method
        ChainIo:              new(ChainIo),
        Cache: func(ctx context.Context) services.CacheFunc {
            // return redis do func
            // must accept ‘HSET’,'HGET' params
            return database.WithContextRedis(ctx)
          },
        Chain:         chain,
        ContractsName: map[services.ContractsAddress]services.ContractsName{
            "0x7cD44a3C9696185BAC374F0Cd3018F4b24986cb0":"objectOwnership",
            "0x6c74a72444048A8588dEBeb749Ee60DB842aD90f":"apostle"
        },
        GetCallbackFunc: func(tx string, blockTimestamp uint64, receipt *services.Receipts) interface{} {
            // return object must have TransferCallback method
            return models.EthTransactionCallback{
                Tx:             tx,
                Receipt:        receipt,
                BlockTimestamp: int64(blockTimestamp),
            }
          },
        // filter events start InitBlock block height
        // if name=WipeBlock key={chain} in redis and value != 0, use redis value
        InitBlock:  0,
        // If true ignore errors and run recursively
        RunForever: true,
    })	
}
```