// Copyright (C) 2019-2021, Dijets, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package getter

import (
	"time"

	"github.com/lasthyphen/dijigo/ids"
	"github.com/lasthyphen/dijigo/snow/choices"
	"github.com/lasthyphen/dijigo/snow/consensus/avalanche"
	"github.com/lasthyphen/dijigo/snow/engine/avalanche/vertex"
	"github.com/lasthyphen/dijigo/snow/engine/common"
	"github.com/lasthyphen/dijigo/utils/constants"
	"github.com/lasthyphen/dijigo/utils/logging"
	"github.com/lasthyphen/dijigo/utils/metric"
	"github.com/lasthyphen/dijigo/utils/wrappers"
)

// Get requests are always served, regardless node state (bootstrapping or normal operations).
var _ common.AllGetsServer = &getter{}

func New(manager vertex.Manager, commonCfg common.Config) (common.AllGetsServer, error) {
	gh := &getter{
		manager: manager,
		sender:  commonCfg.Sender,
		cfg:     commonCfg,
		log:     commonCfg.Ctx.Log,
	}

	var err error
	gh.getAncestorsVtxs, err = metric.NewAverager(
		"bs",
		"get_ancestors_vtxs",
		"vertices fetched in a call to GetAncestors",
		commonCfg.Ctx.Registerer,
	)
	return gh, err
}

type getter struct {
	manager vertex.Manager
	sender  common.Sender
	cfg     common.Config

	log              logging.Logger
	getAncestorsVtxs metric.Averager
}

func (gh *getter) GetAcceptedFrontier(validatorID ids.ShortID, requestID uint32) error {
	acceptedFrontier := gh.manager.Edge()
	gh.sender.SendAcceptedFrontier(validatorID, requestID, acceptedFrontier)
	return nil
}

func (gh *getter) GetAccepted(validatorID ids.ShortID, requestID uint32, containerIDs []ids.ID) error {
	acceptedVtxIDs := make([]ids.ID, 0, len(containerIDs))
	for _, vtxID := range containerIDs {
		if vtx, err := gh.manager.GetVtx(vtxID); err == nil && vtx.Status() == choices.Accepted {
			acceptedVtxIDs = append(acceptedVtxIDs, vtxID)
		}
	}
	gh.sender.SendAccepted(validatorID, requestID, acceptedVtxIDs)
	return nil
}

func (gh *getter) GetAncestors(validatorID ids.ShortID, requestID uint32, vtxID ids.ID) error {
	startTime := time.Now()
	gh.log.Verbo("GetAncestors(%s, %d, %s) called", validatorID, requestID, vtxID)
	vertex, err := gh.manager.GetVtx(vtxID)
	if err != nil || vertex.Status() == choices.Unknown {
		gh.log.Verbo("dropping getAncestors")
		return nil // Don't have the requested vertex. Drop message.
	}

	queue := make([]avalanche.Vertex, 1, gh.cfg.AncestorsMaxContainersSent) // for BFS
	queue[0] = vertex
	ancestorsBytesLen := 0                                                 // length, in bytes, of vertex and its ancestors
	ancestorsBytes := make([][]byte, 0, gh.cfg.AncestorsMaxContainersSent) // vertex and its ancestors in BFS order
	visited := ids.Set{}                                                   // IDs of vertices that have been in queue before
	visited.Add(vertex.ID())

	for len(ancestorsBytes) < gh.cfg.AncestorsMaxContainersSent && len(queue) > 0 && time.Since(startTime) < gh.cfg.MaxTimeGetAncestors {
		var vtx avalanche.Vertex
		vtx, queue = queue[0], queue[1:] // pop
		vtxBytes := vtx.Bytes()
		// Ensure response size isn't too large. Include wrappers.IntLen because the size of the message
		// is included with each container, and the size is repr. by an int.
		if newLen := wrappers.IntLen + ancestorsBytesLen + len(vtxBytes); newLen < constants.MaxContainersLen {
			ancestorsBytes = append(ancestorsBytes, vtxBytes)
			ancestorsBytesLen = newLen
		} else { // reached maximum response size
			break
		}
		parents, err := vtx.Parents()
		if err != nil {
			return err
		}
		for _, parent := range parents {
			if parent.Status() == choices.Unknown { // Don't have this vertex;ignore
				continue
			}
			if parentID := parent.ID(); !visited.Contains(parentID) { // If already visited, ignore
				queue = append(queue, parent)
				visited.Add(parentID)
			}
		}
	}

	gh.getAncestorsVtxs.Observe(float64(len(ancestorsBytes)))
	gh.sender.SendAncestors(validatorID, requestID, ancestorsBytes)
	return nil
}

func (gh *getter) Get(validatorID ids.ShortID, requestID uint32, vtxID ids.ID) error {
	// If this engine has access to the requested vertex, provide it
	if vtx, err := gh.manager.GetVtx(vtxID); err == nil {
		gh.sender.SendPut(validatorID, requestID, vtxID, vtx.Bytes())
	}
	return nil
}
