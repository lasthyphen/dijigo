// Copyright (C) 2019-2021, Dijets, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package network

import (
	"crypto/tls"
	"sync"
	"testing"

	"github.com/lasthyphen/dijigo/ids"
	"github.com/lasthyphen/dijigo/network/peer"
	"github.com/lasthyphen/dijigo/staking"
)

var (
	certLock   sync.Mutex
	tlsCerts   []*tls.Certificate
	tlsConfigs []*tls.Config
)

func getTLS(t *testing.T, index int) (ids.ShortID, *tls.Certificate, *tls.Config) {
	certLock.Lock()
	defer certLock.Unlock()

	for len(tlsCerts) <= index {
		cert, err := staking.NewTLSCert()
		if err != nil {
			t.Fatal(err)
		}
		tlsConfig := peer.TLSConfig(*cert)

		tlsCerts = append(tlsCerts, cert)
		tlsConfigs = append(tlsConfigs, tlsConfig)
	}

	cert := tlsCerts[index]
	return peer.CertToID(cert.Leaf), cert, tlsConfigs[index]
}
