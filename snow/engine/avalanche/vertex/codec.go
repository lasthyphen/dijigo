// Copyright (C) 2019-2021, Dijets, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vertex

import (
	"github.com/lasthyphen/dijigo/codec"
	"github.com/lasthyphen/dijigo/codec/linearcodec"
	"github.com/lasthyphen/dijigo/codec/reflectcodec"
	"github.com/lasthyphen/dijigo/utils/units"
)

const (
	// maxSize is the maximum allowed vertex size. It is necessary to deter DoS
	maxSize = units.MiB

	codecVersion            uint16 = 0
	codecVersionWithStopVtx uint16 = 1
)

var c codec.Manager

func init() {
	lc := linearcodec.New(reflectcodec.DefaultTagName+"V0", maxSize)
	lc2 := linearcodec.New(reflectcodec.DefaultTagName+"V1", maxSize)

	c = codec.NewManager(maxSize)
	// for backward compatibility, still register the initial codec version
	if err := c.RegisterCodec(codecVersion, lc); err != nil {
		panic(err)
	}
	if err := c.RegisterCodec(codecVersionWithStopVtx, lc2); err != nil {
		panic(err)
	}
}
