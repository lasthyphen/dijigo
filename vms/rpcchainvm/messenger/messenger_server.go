// Copyright (C) 2019-2021, Dijets, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package messenger

import (
	"context"
	"errors"

	"github.com/lasthyphen/dijigo/api/proto/messengerproto"
	"github.com/lasthyphen/dijigo/snow/engine/common"
)

var (
	errFullQueue = errors.New("full message queue")

	_ messengerproto.MessengerServer = &Server{}
)

// Server is a messenger that is managed over RPC.
type Server struct {
	messengerproto.UnimplementedMessengerServer
	messenger chan<- common.Message
}

// NewServer returns a messenger connected to a remote channel
func NewServer(messenger chan<- common.Message) *Server {
	return &Server{messenger: messenger}
}

func (s *Server) Notify(_ context.Context, req *messengerproto.NotifyRequest) (*messengerproto.NotifyResponse, error) {
	msg := common.Message(req.Message)
	select {
	case s.messenger <- msg:
		return &messengerproto.NotifyResponse{}, nil
	default:
		return nil, errFullQueue
	}
}
