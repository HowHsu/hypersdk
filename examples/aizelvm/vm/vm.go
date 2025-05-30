// Copyright (C) 2024, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"errors"

	"github.com/ava-labs/hypersdk/auth"
	"github.com/ava-labs/hypersdk/chain"
	"github.com/ava-labs/hypersdk/codec"
	"github.com/ava-labs/hypersdk/examples/aizelvm/actions"
	"github.com/ava-labs/hypersdk/examples/aizelvm/storage"
	"github.com/ava-labs/hypersdk/genesis"
	"github.com/ava-labs/hypersdk/state/metadata"
	"github.com/ava-labs/hypersdk/vm"
	"github.com/ava-labs/hypersdk/vm/defaultvm"
)

var (
	ActionParser *codec.TypeParser[chain.Action]
	AuthParser   *codec.TypeParser[chain.Auth]
	OutputParser *codec.TypeParser[codec.Typed]

	AuthProvider *auth.AuthProvider

	Parser *chain.TxTypeParser
)

// Setup types
func init() {
	ActionParser = codec.NewTypeParser[chain.Action]()
	AuthParser = codec.NewTypeParser[chain.Auth]()
	OutputParser = codec.NewTypeParser[codec.Typed]()
	AuthProvider = auth.NewAuthProvider()

	if err := auth.WithDefaultPrivateKeyFactories(AuthProvider); err != nil {
		panic(err)
	}

	if err := errors.Join(
		// When registering new actions, ALWAYS make sure to append at the end.
		// Pass nil as second argument if manual marshalling isn't needed (if in doubt, you probably don't)
		ActionParser.Register(&actions.Transfer{}, actions.UnmarshalTransfer),

		ActionParser.Register(&actions.StoreReport{}, nil),

		// When registering new auth, ALWAYS make sure to append at the end.
		AuthParser.Register(&auth.ED25519{}, auth.UnmarshalED25519),
		AuthParser.Register(&auth.SECP256R1{}, auth.UnmarshalSECP256R1),
		AuthParser.Register(&auth.BLS{}, auth.UnmarshalBLS),

		OutputParser.Register(&actions.TransferResult{}, actions.UnmarshalTransferResult),
		OutputParser.Register(&actions.StoreReportOutput{}, nil),
	); err != nil {
		panic(err)
	}

	Parser = chain.NewTxTypeParser(ActionParser, AuthParser)
}

// New returns a VM with the specified options
func New(options ...vm.Option) (*vm.VM, error) {
	factory := NewFactory()
	return factory.New(options...)
}

func NewFactory() *vm.Factory {
	options := append(defaultvm.NewDefaultOptions(), With())
	return vm.NewFactory(
		genesis.DefaultGenesisFactory{},
		&storage.BalanceHandler{},
		metadata.NewDefaultManager(),
		ActionParser,
		AuthParser,
		OutputParser,
		auth.DefaultEngines(),
		options...,
	)
}
