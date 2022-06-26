// Copyright (C) 2019-2021, Dijets, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package peer

import (
	"crypto"
	"time"

	"github.com/lasthyphen/dijigo/ids"
	"github.com/lasthyphen/dijigo/message"
	"github.com/lasthyphen/dijigo/utils"
	"github.com/lasthyphen/dijigo/version"
)

var _ Network = &testNetwork{}

type testNetwork struct {
	mc message.Creator

	networkID uint32
	ip        utils.IPDesc
	version   version.Application
	signer    crypto.Signer
	subnets   ids.Set

	uptime uint8
}

func (n *testNetwork) Connected(ids.ShortID) {}

func (n *testNetwork) AllowConnection(ids.ShortID) bool { return true }

func (n *testNetwork) Track(utils.IPCertDesc) {}

func (n *testNetwork) Disconnected(ids.ShortID) {}

func (n *testNetwork) Version() (message.OutboundMessage, error) {
	now := uint64(time.Now().Unix())
	unsignedIP := UnsignedIP{
		IP:        n.ip,
		Timestamp: now,
	}
	signedIP, err := unsignedIP.Sign(n.signer)
	if err != nil {
		return nil, err
	}
	return n.mc.Version(
		n.networkID,
		now,
		n.ip,
		n.version.String(),
		now,
		signedIP.Signature,
		n.subnets.List(),
	)
}

func (n *testNetwork) Peers() (message.OutboundMessage, error) {
	return n.mc.PeerList(nil, true)
}

func (n *testNetwork) Pong(ids.ShortID) (message.OutboundMessage, error) {
	return n.mc.Pong(n.uptime)
}
