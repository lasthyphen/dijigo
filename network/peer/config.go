// Copyright (C) 2019-2021, Dijets, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package peer

import (
	"time"

	"github.com/lasthyphen/dijigo/ids"
	"github.com/lasthyphen/dijigo/message"
	"github.com/lasthyphen/dijigo/network/throttling"
	"github.com/lasthyphen/dijigo/snow/networking/router"
	"github.com/lasthyphen/dijigo/snow/validators"
	"github.com/lasthyphen/dijigo/utils/logging"
	"github.com/lasthyphen/dijigo/utils/timer/mockable"
	"github.com/lasthyphen/dijigo/version"
)

type Config struct {
	Clock                mockable.Clock
	Metrics              *Metrics
	MessageCreator       message.Creator
	Log                  logging.Logger
	InboundMsgThrottler  throttling.InboundMsgThrottler
	OutboundMsgThrottler throttling.OutboundMsgThrottler
	Network              Network
	Router               router.InboundHandler
	VersionCompatibility version.Compatibility
	VersionParser        version.ApplicationParser
	MySubnets            ids.Set
	Beacons              validators.Set
	NetworkID            uint32
	PingFrequency        time.Duration
	PongTimeout          time.Duration
	MaxClockDifference   time.Duration

	// Unix time of the last message sent and received respectively
	// Must only be accessed atomically
	LastSent, LastReceived int64
}
