// Copyright (C) 2024, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package dsmr

import (
	"context"
	"errors"
	"fmt"

	"github.com/ava-labs/avalanchego/codec/linearcodec"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/units"
	"github.com/ava-labs/avalanchego/utils/wrappers"
	"github.com/ava-labs/avalanchego/vms/platformvm/warp"

	"github.com/ava-labs/hypersdk/codec"

	acodec "github.com/ava-labs/avalanchego/codec"
)

const (
	CodecVersion = 0

	MaxMessageSize = units.KiB
)

var Codec acodec.Manager

func init() {
	Codec = acodec.NewManager(MaxMessageSize)
	lc := linearcodec.NewDefault()

	err := errors.Join(
		Codec.RegisterCodec(CodecVersion, lc),
	)
	if err != nil {
		panic(err)
	}
}

type ChunkReference struct {
	ChunkID  ids.ID     `serialize:"true"`
	Producer ids.NodeID `serialize:"true"`
	Expiry   int64      `serialize:"true"`
}

type emapChunkCertificate struct {
	ChunkCertificate
}

func (e emapChunkCertificate) GetID() ids.ID { return e.ChunkID }

func (e emapChunkCertificate) GetExpiry() int64 { return e.Expiry }

type ChunkCertificate struct {
	ChunkReference `serialize:"true"`
	Signature      *warp.BitSetSignature `serialize:"true"`
}

func (c *ChunkCertificate) GetChunkID() ids.ID { return c.ChunkID }

func (c *ChunkCertificate) GetSlot() int64 { return c.Expiry }

func (c *ChunkCertificate) Bytes() []byte {
	bytes, err := Codec.Marshal(CodecVersion, c)
	if err != nil {
		panic(err)
	}
	return bytes
}

func (c *ChunkCertificate) Verify(
	ctx context.Context,
	chainState ChainState,
) error {
	packer := wrappers.Packer{MaxSize: MaxMessageSize}
	if err := codec.LinearCodec.MarshalInto(c.ChunkReference, &packer); err != nil {
		return fmt.Errorf("failed to marshal chunk reference: %w", err)
	}
	networkID := chainState.GetNetworkID()
	msg, err := warp.NewUnsignedMessage(networkID, chainState.GetChainID(), packer.Bytes)
	if err != nil {
		return fmt.Errorf("failed to initialize unsigned warp message: %w", err)
	}
	canonicalValidatorSet, err := chainState.GetCanonicalValidatorSet(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve validators set: %w", err)
	}
	if err := c.Signature.Verify(
		msg,
		networkID,
		canonicalValidatorSet,
		chainState.GetQuorumNum(),
		chainState.GetQuorumDen(),
	); err != nil {
		return fmt.Errorf("failed verification: %w", err)
	}

	return nil
}
