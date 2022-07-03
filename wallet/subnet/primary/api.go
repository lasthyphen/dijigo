// Copyright (C) 2019-2021, Dijets, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package primary

import (
	"context"

	"github.com/lasthyphen/dijigo/api"
	"github.com/lasthyphen/dijigo/codec"
	"github.com/lasthyphen/dijigo/ids"
	"github.com/lasthyphen/dijigo/utils/formatting"
	"github.com/lasthyphen/dijigo/vms/avm"
	"github.com/lasthyphen/dijigo/vms/components/djtx"
	"github.com/lasthyphen/dijigo/vms/platformvm"
)

const (
	MainnetAPIURI = "https://archiver.switzerlandnorth.cloudapp.azure.com:443"
	FujiAPIURI    = "http://api.djtx-test.network"
	LocalAPIURI   = "http://localhost:9650"

	fetchLimit = 1024
)

// TODO: refactor UTXOClient definition to allow the client implementations to
//       perform their own assertions.
var (
	_ UTXOClient = platformvm.Client(nil)
	_ UTXOClient = avm.Client(nil)
)

type UTXOClient interface {
	GetAtomicUTXOs(
		ctx context.Context,
		addrs []string,
		sourceChain string,
		limit uint32,
		startAddress,
		startUTXOID string,
	) ([][]byte, api.Index, error)
}

// FormatAddresses returns the string format of the provided address set for the
// requested chain and hrp. This is useful to use with the API clients to
// support address queries.
func FormatAddresses(chain, hrp string, addrSet ids.ShortSet) ([]string, error) {
	addrs := make([]string, 0, addrSet.Len())
	for addr := range addrSet {
		addrStr, err := formatting.FormatAddress(chain, hrp, addr[:])
		if err != nil {
			return nil, err
		}
		addrs = append(addrs, addrStr)
	}
	return addrs, nil
}

// AddAllUTXOs fetches all the UTXOs referenced by [addresses] that were sent
// from [sourceChainID] to [destinationChainID] from the [client]. It then uses
// [codec] to parse the returned UTXOs and it adds them into [utxos]. If [ctx]
// expires, then the returned error will be immediately reported.
func AddAllUTXOs(
	ctx context.Context,
	utxos UTXOs,
	client UTXOClient,
	codec codec.Manager,
	sourceChainID ids.ID,
	destinationChainID ids.ID,
	addrs []string,
) error {
	var (
		sourceChainIDStr = sourceChainID.String()
		startAddr        string
		startUTXO        string
	)
	for {
		utxosBytes, index, err := client.GetAtomicUTXOs(
			ctx,
			addrs,
			sourceChainIDStr,
			fetchLimit,
			startAddr,
			startUTXO,
		)
		if err != nil {
			return err
		}

		for _, utxoBytes := range utxosBytes {
			var utxo djtx.UTXO
			_, err := codec.Unmarshal(utxoBytes, &utxo)
			if err != nil {
				return err
			}

			if err := utxos.AddUTXO(ctx, sourceChainID, destinationChainID, &utxo); err != nil {
				return err
			}
		}

		if len(utxosBytes) < fetchLimit {
			break
		}

		startAddr = index.Address
		startUTXO = index.UTXO
	}
	return nil
}
