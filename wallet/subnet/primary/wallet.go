// Copyright (C) 2019-2021, Dijets, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package primary

import (
	"context"

	"github.com/lasthyphen/dijigo/api/info"
	"github.com/lasthyphen/dijigo/codec"
	"github.com/lasthyphen/dijigo/ids"
	"github.com/lasthyphen/dijigo/utils/constants"
	"github.com/lasthyphen/dijigo/vms/avm"
	"github.com/lasthyphen/dijigo/vms/platformvm"
	"github.com/lasthyphen/dijigo/vms/secp256k1fx"
	"github.com/lasthyphen/dijigo/wallet/chain/p"
	"github.com/lasthyphen/dijigo/wallet/chain/x"
	"github.com/lasthyphen/dijigo/wallet/subnet/primary/common"
)

var _ Wallet = &wallet{}

// Wallet provides chain wallets for the primary network.
type Wallet interface {
	P() p.Wallet
	X() x.Wallet
}

// NewWallet returns a wallet that supports issuing transactions to the chains
// living in the primary network.
//
// On creation, the wallet attaches to the provided [uri] and fetches all UTXOs
// that reference any of the keys contained in [kc]. If the UTXOs are modified
// through an external issuance process, such as another instance of the wallet,
// the UTXOs may become out of sync.
//
// The wallet manages all UTXOs locally, and performs all tx signing locally.
func NewWallet(ctx context.Context, uri string, kc *secp256k1fx.Keychain) (Wallet, error) {
	infoClient := info.NewClient(uri)
	xClient := avm.NewClient(uri, "X")

	pCTX, err := p.NewContextFromClients(ctx, infoClient, xClient)
	if err != nil {
		return nil, err
	}
	pAddrs, err := FormatAddresses("P", pCTX.HRP(), kc.Addrs)
	if err != nil {
		return nil, err
	}

	xCTX, err := x.NewContextFromClients(ctx, infoClient, xClient)
	if err != nil {
		return nil, err
	}
	xAddrs, err := FormatAddresses("X", xCTX.HRP(), kc.Addrs)
	if err != nil {
		return nil, err
	}

	pClient := platformvm.NewClient(uri)
	xChainID := xCTX.BlockchainID()
	utxos := NewUTXOs()

	chains := []struct {
		id     ids.ID
		client UTXOClient
		codec  codec.Manager
		addrs  []string
	}{
		{
			id:     constants.PlatformChainID,
			client: pClient,
			codec:  platformvm.Codec,
			addrs:  pAddrs,
		},
		{
			id:     xChainID,
			client: xClient,
			codec:  x.Codec,
			addrs:  xAddrs,
		},
	}
	for _, destinationChain := range chains {
		for _, sourceChain := range chains {
			err = AddAllUTXOs(
				ctx,
				utxos,
				destinationChain.client,
				destinationChain.codec,
				sourceChain.id,
				destinationChain.id,
				destinationChain.addrs,
			)
			if err != nil {
				return nil, err
			}
		}
	}

	pUTXOs := NewChainUTXOs(constants.PlatformChainID, utxos)
	pTXs := make(map[ids.ID]*platformvm.Tx)
	pBackend := p.NewBackend(pCTX, pUTXOs, pTXs)
	pBuilder := p.NewBuilder(kc.Addrs, pBackend)
	pSigner := p.NewSigner(kc, pBackend)

	xUTXOs := NewChainUTXOs(xChainID, utxos)
	xBackend := x.NewBackend(xCTX, xChainID, xUTXOs)
	xBuilder := x.NewBuilder(kc.Addrs, xBackend)
	xSigner := x.NewSigner(kc, xBackend)

	return &wallet{
		p: p.NewWallet(pBuilder, pSigner, pClient, pBackend),
		x: x.NewWallet(xBuilder, xSigner, xClient, xBackend),
	}, nil
}

func NewWalletWithOptions(w Wallet, options ...common.Option) Wallet {
	return &wallet{
		p: p.NewWalletWithOptions(w.P(), options...),
		x: x.NewWalletWithOptions(w.X(), options...),
	}
}

type wallet struct {
	p p.Wallet
	x x.Wallet
}

func (w *wallet) P() p.Wallet { return w.p }
func (w *wallet) X() x.Wallet { return w.x }
