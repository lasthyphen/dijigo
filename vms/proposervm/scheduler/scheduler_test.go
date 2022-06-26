// Copyright (C) 2019-2021, Dijets, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package scheduler

import (
	"testing"
	"time"

	"github.com/lasthyphen/dijigo/snow/engine/common"
	"github.com/lasthyphen/dijigo/utils/logging"
)

func TestDelayFromNew(t *testing.T) {
	toEngine := make(chan common.Message, 10)
	startTime := time.Now().Add(50 * time.Millisecond)

	s, fromVM := New(logging.NoLog{}, toEngine)
	defer s.Close()
	go s.Dispatch(startTime)

	fromVM <- common.PendingTxs

	<-toEngine
	if time.Until(startTime) > 0 {
		t.Fatalf("passed message too soon")
	}
}

func TestDelayFromSetTime(t *testing.T) {
	toEngine := make(chan common.Message, 10)
	now := time.Now()
	startTime := now.Add(50 * time.Millisecond)

	s, fromVM := New(logging.NoLog{}, toEngine)
	defer s.Close()
	go s.Dispatch(now)

	s.SetBuildBlockTime(startTime)

	fromVM <- common.PendingTxs

	<-toEngine
	if time.Until(startTime) > 0 {
		t.Fatalf("passed message too soon")
	}
}

func TestReceipt(t *testing.T) {
	toEngine := make(chan common.Message, 10)
	now := time.Now()
	startTime := now.Add(50 * time.Millisecond)

	s, fromVM := New(logging.NoLog{}, toEngine)
	defer s.Close()
	go s.Dispatch(now)

	fromVM <- common.PendingTxs

	s.SetBuildBlockTime(startTime)

	<-toEngine
}
