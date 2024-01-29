package nexelliastratum

import (
	"context"
	"fmt"
	"time"

	"github.com/GRinvestPOOL/nexellia-stratum-bridge/src/gostratum"
	"github.com/Nexellia-Network/nexelliad/app/appmessage"
	"github.com/Nexellia-Network/nexelliad/infrastructure/network/rpcclient"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type NexelliaApi struct {
	address       string
	blockWaitTime time.Duration
	logger        *zap.SugaredLogger
	nexelliad      *rpcclient.RPCClient
	connected     bool
}

func NewNexelliaApi(address string, blockWaitTime time.Duration, logger *zap.SugaredLogger) (*NexelliaApi, error) {
	client, err := rpcclient.NewRPCClient(address)
	if err != nil {
		return nil, err
	}

	return &NexelliaApi{
		address:       address,
		blockWaitTime: blockWaitTime,
		logger:        logger.With(zap.String("component", "nexelliaapi:"+address)),
		nexelliad:      client,
		connected:     true,
	}, nil
}

func (ks *NexelliaApi) Start(ctx context.Context, blockCb func()) {
	ks.waitForSync(true)
	go ks.startBlockTemplateListener(ctx, blockCb)
	go ks.startStatsThread(ctx)
}

func (ks *NexelliaApi) startStatsThread(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	for {
		select {
		case <-ctx.Done():
			ks.logger.Warn("context cancelled, stopping stats thread")
			return
		case <-ticker.C:
			dagResponse, err := ks.nexelliad.GetBlockDAGInfo()
			if err != nil {
				ks.logger.Warn("failed to get network hashrate from nexellia, prom stats will be out of date", zap.Error(err))
				continue
			}
			response, err := ks.nexelliad.EstimateNetworkHashesPerSecond(dagResponse.TipHashes[0], 1000)
			if err != nil {
				ks.logger.Warn("failed to get network hashrate from nexellia, prom stats will be out of date", zap.Error(err))
				continue
			}
			RecordNetworkStats(response.NetworkHashesPerSecond, dagResponse.BlockCount, dagResponse.Difficulty)
		}
	}
}

func (ks *NexelliaApi) reconnect() error {
	if ks.nexelliad != nil {
		return ks.nexelliad.Reconnect()
	}

	client, err := rpcclient.NewRPCClient(ks.address)
	if err != nil {
		return err
	}
	ks.nexelliad = client
	return nil
}

func (s *NexelliaApi) waitForSync(verbose bool) error {
	if verbose {
		s.logger.Info("checking nexelliad sync state")
	}
	for {
		clientInfo, err := s.nexelliad.GetInfo()
		if err != nil {
			return errors.Wrapf(err, "error fetching server info from nexelliad @ %s", s.address)
		}
		if clientInfo.IsSynced {
			break
		}
		s.logger.Warn("Karlsen is not synced, waiting for sync before starting bridge")
		time.Sleep(5 * time.Second)
	}
	if verbose {
		s.logger.Info("nexelliad synced, starting server")
	}
	return nil
}

func (s *NexelliaApi) startBlockTemplateListener(ctx context.Context, blockReadyCb func()) {
	blockReadyChan := make(chan bool)
	err := s.nexelliad.RegisterForNewBlockTemplateNotifications(func(_ *appmessage.NewBlockTemplateNotificationMessage) {
		blockReadyChan <- true
	})
	if err != nil {
		s.logger.Error("fatal: failed to register for block notifications from nexellia")
	}

	ticker := time.NewTicker(s.blockWaitTime)
	for {
		if err := s.waitForSync(false); err != nil {
			s.logger.Error("error checking nexelliad sync state, attempting reconnect: ", err)
			if err := s.reconnect(); err != nil {
				s.logger.Error("error reconnecting to nexelliad, waiting before retry: ", err)
				time.Sleep(5 * time.Second)
			}
		}
		select {
		case <-ctx.Done():
			s.logger.Warn("context cancelled, stopping block update listener")
			return
		case <-blockReadyChan:
			blockReadyCb()
			ticker.Reset(s.blockWaitTime)
		case <-ticker.C: // timeout, manually check for new blocks
			blockReadyCb()
		}
	}
}

func (ks *NexelliaApi) GetBlockTemplate(
	client *gostratum.StratumContext) (*appmessage.GetBlockTemplateResponseMessage, error) {
	template, err := ks.nexelliad.GetBlockTemplate(client.WalletAddr,
		fmt.Sprintf(`'%s' via nexellia-network/nexellia-stratum-bridge_%s`, client.RemoteApp, version))
	if err != nil {
		return nil, errors.Wrap(err, "failed fetching new block template from nexellia")
	}
	return template, nil
}
