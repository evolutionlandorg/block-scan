package block_scan

import (
	"context"
	"github.com/evolutionlandorg/block-scan/metrics"
	"time"

	"github.com/evolutionlandorg/block-scan/scan"
	"github.com/evolutionlandorg/block-scan/services"
	"github.com/evolutionlandorg/block-scan/subscribe"
	"github.com/evolutionlandorg/block-scan/util"
	"github.com/evolutionlandorg/block-scan/util/log"
)

type ScanType string

var (
	SUBSCRIBE ScanType = "subscribe"
	POLLING   ScanType = "polling"
)

func StartScanChainEvents(ctx context.Context, scanType ScanType, opt services.ScanEventsOptions) error {
	var instance services.Scan
	switch scanType {
	case SUBSCRIBE:
		instance = new(subscribe.Subscribe)
	case POLLING:
		instance = new(scan.Polling)
	default:
		log.Panic("not implement '%s' type", scanType)
	}
	instance.SetMetrics(metrics.NewMetrics())
	if err := opt.Check(); err != nil {
		return err
	}

	if err := instance.Init(opt); err != nil {
		return err
	}

	if opt.RunForever {
		defer func() {
			if err := recover(); err != nil {
				log.Error("run %s WipeBlock error: %v. restarting...", opt.Chain, err)
				time.Sleep(time.Second * 1)
				_ = StartScanChainEvents(ctx, scanType, opt)
			}
		}()
		util.Panic(instance.WipeBlock(ctx))
		return nil
	}
	return instance.WipeBlock(ctx)
}
