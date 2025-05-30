// Copyright (C) 2024, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package e2e

import (
	"context"
	"fmt"

	"github.com/ava-labs/hypersdk/api/ws"
	"github.com/ava-labs/hypersdk/chain"
)

type pacer struct {
	ws *ws.WebSocketClient

	inflight chan struct{}
	done     chan struct{}

	err error
}

func newPacer(ws *ws.WebSocketClient, maxPending int) *pacer {
	return &pacer{
		ws:       ws,
		inflight: make(chan struct{}, maxPending),
		done:     make(chan struct{}),
	}
}

func (p *pacer) run(ctx context.Context) {
	defer close(p.done)

	for range p.inflight {
		txID, result, err := p.ws.ListenTx(ctx)
		if err != nil {
			p.err = fmt.Errorf("error listening to tx %s: %w", txID, err)
			return
		}
		if result == nil {
			p.err = fmt.Errorf("tx %s expired", txID)
			return
		}
		if !result.Success {
			// Should never happen
			p.err = fmt.Errorf("tx failure %w: %s", ErrTxFailed, result.Error)
			return
		}
	}
}

func (p *pacer) add(tx *chain.Transaction) error {
	// If Run failed, return the first error immediately, otherwise register the next tx
	select {
	case <-p.done:
		return p.err
	default:
		if err := p.ws.RegisterTx(tx); err != nil {
			return err
		}
	}

	select {
	case p.inflight <- struct{}{}:
		return nil
	case <-p.done:
		return p.err
	}
}

func (p *pacer) wait() error {
	close(p.inflight)
	// Wait for Run to complete
	<-p.done
	return p.err
}
