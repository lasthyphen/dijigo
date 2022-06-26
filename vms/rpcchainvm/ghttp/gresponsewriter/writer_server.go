// Copyright (C) 2019-2021, Dijets, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package gresponsewriter

import (
	"context"
	"errors"
	"net/http"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/hashicorp/go-plugin"

	"github.com/lasthyphen/dijigo/api/proto/gconnproto"
	"github.com/lasthyphen/dijigo/api/proto/greaderproto"
	"github.com/lasthyphen/dijigo/api/proto/gresponsewriterproto"
	"github.com/lasthyphen/dijigo/api/proto/gwriterproto"
	"github.com/lasthyphen/dijigo/utils/math"
	"github.com/lasthyphen/dijigo/vms/rpcchainvm/ghttp/gconn"
	"github.com/lasthyphen/dijigo/vms/rpcchainvm/ghttp/greader"
	"github.com/lasthyphen/dijigo/vms/rpcchainvm/ghttp/gwriter"
	"github.com/lasthyphen/dijigo/vms/rpcchainvm/grpcutils"
)

var (
	errUnsupportedFlushing  = errors.New("response writer doesn't support flushing")
	errUnsupportedHijacking = errors.New("response writer doesn't support hijacking")

	_ gresponsewriterproto.WriterServer = &Server{}
)

// Server is an http.ResponseWriter that is managed over RPC.
type Server struct {
	gresponsewriterproto.UnimplementedWriterServer
	writer http.ResponseWriter
	broker *plugin.GRPCBroker
}

// NewServer returns an http.ResponseWriter instance managed remotely
func NewServer(writer http.ResponseWriter, broker *plugin.GRPCBroker) *Server {
	return &Server{
		writer: writer,
		broker: broker,
	}
}

func (s *Server) Write(ctx context.Context, req *gresponsewriterproto.WriteRequest) (*gresponsewriterproto.WriteResponse, error) {
	headers := s.writer.Header()
	for key := range headers {
		delete(headers, key)
	}
	for _, header := range req.Headers {
		headers[header.Key] = header.Values
	}

	n, err := s.writer.Write(req.Payload)
	if err != nil {
		return nil, err
	}
	return &gresponsewriterproto.WriteResponse{
		Written: int32(n),
	}, nil
}

func (s *Server) WriteHeader(ctx context.Context, req *gresponsewriterproto.WriteHeaderRequest) (*emptypb.Empty, error) {
	headers := s.writer.Header()
	for key := range headers {
		delete(headers, key)
	}
	for _, header := range req.Headers {
		headers[header.Key] = header.Values
	}
	s.writer.WriteHeader(int(req.StatusCode))
	return &emptypb.Empty{}, nil
}

func (s *Server) Flush(ctx context.Context, req *emptypb.Empty) (*emptypb.Empty, error) {
	flusher, ok := s.writer.(http.Flusher)
	if !ok {
		return nil, errUnsupportedFlushing
	}
	flusher.Flush()
	return &emptypb.Empty{}, nil
}

func (s *Server) Hijack(ctx context.Context, req *emptypb.Empty) (*gresponsewriterproto.HijackResponse, error) {
	hijacker, ok := s.writer.(http.Hijacker)
	if !ok {
		return nil, errUnsupportedHijacking
	}
	conn, readWriter, err := hijacker.Hijack()
	if err != nil {
		return nil, err
	}

	connReadWriterID := s.broker.NextId()
	closer := grpcutils.ServerCloser{}

	go s.broker.AcceptAndServe(connReadWriterID, func(opts []grpc.ServerOption) *grpc.Server {
		opts = append(opts,
			grpc.MaxRecvMsgSize(math.MaxInt),
			grpc.MaxSendMsgSize(math.MaxInt),
		)
		server := grpc.NewServer(opts...)
		closer.Add(server)
		gconnproto.RegisterConnServer(server, gconn.NewServer(conn, &closer))
		greaderproto.RegisterReaderServer(server, greader.NewServer(readWriter))
		gwriterproto.RegisterWriterServer(server, gwriter.NewServer(readWriter))
		return server
	})

	local := conn.LocalAddr()
	remote := conn.RemoteAddr()

	return &gresponsewriterproto.HijackResponse{
		LocalNetwork:         local.Network(),
		LocalString:          local.String(),
		RemoteNetwork:        remote.Network(),
		RemoteString:         remote.String(),
		ConnReadWriterServer: connReadWriterID,
	}, nil
}
