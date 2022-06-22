package block_scan

import (
	"context"
	"errors"
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

type StartScanChainEventsOptions struct {
	ChainIo              services.ChainIo
	Cache                services.GetCacheFunc
	Chain                string
	ContractsName        map[services.ContractsAddress]services.ContractsName
	SleepTime            time.Duration
	GetCallbackFunc      services.GetCallbackFunc
	CallbackMethodPrefix []string
	InitBlock            uint64
	RunForever           bool
}

func (s *StartScanChainEventsOptions) Check() error {
	if s.ChainIo == nil {
		return errors.New("chainIo must be not nil")
	}
	if s.Cache == nil {
		return errors.New("cache must be not nil")
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

func StartScanChainEvents(ctx context.Context, scanType ScanType, opt *StartScanChainEventsOptions) error {
	var instance services.Scan
	switch scanType {
	case SUBSCRIBE:
		instance = new(subscribe.Subscribe)
	case POLLING:
		instance = new(scan.Polling)
	default:
		log.Panic("not implement '%s' type", scanType)
	}
	if err := opt.Check(); err != nil {
		return err
	}
	if err := instance.Init(opt.ChainIo, opt.Cache, opt.Chain, opt.ContractsName, opt.SleepTime, opt.GetCallbackFunc, opt.CallbackMethodPrefix); err != nil {
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
		util.Panic(instance.WipeBlock(ctx, opt.InitBlock))
		return nil
	}
	return instance.WipeBlock(ctx, opt.InitBlock)
}
