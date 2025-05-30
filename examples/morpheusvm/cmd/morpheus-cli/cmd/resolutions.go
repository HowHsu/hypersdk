// Copyright (C) 2024, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/ava-labs/avalanchego/ids"

	"github.com/ava-labs/hypersdk/api/jsonrpc"
	"github.com/ava-labs/hypersdk/api/ws"
	"github.com/ava-labs/hypersdk/chain"
	"github.com/ava-labs/hypersdk/examples/morpheusvm/actions"
	"github.com/ava-labs/hypersdk/examples/morpheusvm/consts"
	"github.com/ava-labs/hypersdk/examples/morpheusvm/vm"
	"github.com/ava-labs/hypersdk/utils"
)

// sendAndWait may not be used concurrently
func sendAndWait(
	ctx context.Context, actions []chain.Action, cli *jsonrpc.JSONRPCClient,
	bcli *vm.JSONRPCClient, ws *ws.WebSocketClient, authFactory chain.AuthFactory, printStatus bool,
) (bool, ids.ID, error) {
	ruleFactory, err := bcli.GetRuleFactory(ctx)
	if err != nil {
		return false, ids.Empty, err
	}

	unitPrices, err := cli.UnitPrices(ctx, true)
	if err != nil {
		return false, ids.Empty, err
	}

	tx, err := chain.GenerateTransaction(ruleFactory, unitPrices, time.Now().UnixMilli(), actions, authFactory)
	if err != nil {
		return false, ids.Empty, err
	}
	if err := ws.RegisterTx(tx); err != nil {
		return false, ids.Empty, err
	}
	var result *chain.Result
	for {
		txID, txResult, err := ws.ListenTx(ctx)
		if err != nil {
			return false, ids.Empty, err
		}
		if txID == tx.GetID() {
			result = txResult
			break
		}
		if result == nil {
			return false, ids.Empty, fmt.Errorf("tx %s expired", txID)
		}
		utils.Outf("{{yellow}}skipping unexpected transaction:{{/}} %s\n", tx.GetID())
	}
	if printStatus {
		status := "❌"
		if result.Success {
			status = "✅"
		}
		utils.Outf("%s {{yellow}}txID:{{/}} %s\n", status, tx.GetID())
	}
	return result.Success, tx.GetID(), nil
}

func handleTx(tx *chain.Transaction, result *chain.Result) {
	actor := tx.Auth.Actor()
	if !result.Success {
		utils.Outf(
			"%s {{yellow}}%s{{/}} {{yellow}}actor:{{/}} %s {{yellow}}error:{{/}} [%s] {{yellow}}fee (max %.2f%%):{{/}} %s %s {{yellow}}consumed:{{/}} [%s]\n",
			"❌",
			tx.GetID(),
			actor,
			result.Error,
			float64(result.Fee)/float64(tx.Base.MaxFee)*100,
			utils.FormatBalance(result.Fee),
			consts.Symbol,
			result.Units,
		)
		return
	}

	for _, action := range tx.Actions {
		var summaryStr string
		switch act := action.(type) { //nolint:gocritic
		case *actions.Transfer:
			summaryStr = fmt.Sprintf("%s %s -> %s\n", utils.FormatBalance(act.Value), consts.Symbol, actor)
		}
		utils.Outf(
			"%s {{yellow}}%s{{/}} {{yellow}}actor:{{/}} %s {{yellow}}summary (%s):{{/}} [%s] {{yellow}}fee (max %.2f%%):{{/}} %s %s {{yellow}}consumed:{{/}} [%s]\n",
			"✅",
			tx.GetID(),
			actor,
			reflect.TypeOf(action),
			summaryStr,
			float64(result.Fee)/float64(tx.Base.MaxFee)*100,
			utils.FormatBalance(result.Fee),
			consts.Symbol,
			result.Units,
		)
	}
}
