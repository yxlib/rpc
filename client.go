// Copyright 2022 Guan Jianchang. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import (
	"errors"
	"sync"

	"github.com/yxlib/yx"
)

var (
	ErrServNotExist = errors.New("service not exist")
)

// type PipelineList = []*Pipeline

//==========================================
//               client
//==========================================
type client struct {
	mapPeerId2Pipeline map[uint32]*Pipeline
	lckPipelines       *sync.Mutex
	ec                 *yx.ErrCatcher
}

var Client = &client{
	mapPeerId2Pipeline: make(map[uint32]*Pipeline),
	lckPipelines:       &sync.Mutex{},
	ec:                 yx.NewErrCatcher("rpc.Client"),
}

func (c *client) AddPipeline(net Net, peerType uint32, peerNo uint32, mark string, timeoutSec uint32) (*Pipeline, error) {
	oldPipeline, newPipeline := c.addPipeline(net, peerType, peerNo, mark)
	if oldPipeline != nil {
		oldPipeline.Stop()
	}

	newPipeline.SetTimeout(timeoutSec)
	go newPipeline.Start()
	return newPipeline, nil
}

func (c *client) GetPipeline(peerType uint32, peerNo uint32) (*Pipeline, bool) {
	return c.getPipeline(peerType, peerNo)
}

func (c *client) RemovePipeline(peerType uint32, peerNo uint32) {
	pipeline, ok := c.removePipeline(peerType, peerNo)
	if ok {
		pipeline.Stop()
	}
}

// func (c *client) RemovePipelines(peerType uint32, peerNo uint32) {
// 	pipelines, ok := c.removePipelines(peerType, peerNo)
// 	if ok {
// 		for _, pipeline := range pipelines {
// 			pipeline.Stop()
// 		}
// 	}
// }

func (c *client) RemoveAllPipelines() {
	pipelines := c.removeAllPipelines()
	for _, pipeline := range pipelines {
		pipeline.Stop()
	}
}

func (c *client) Call(peerType uint32, peerNo uint32, service string, funcName string, reqObj interface{}, respObj interface{}) (int32, error) {
	pipeline, ok := c.getPipeline(peerType, peerNo)
	if ok {
		return pipeline.Call(service, funcName, reqObj, respObj)
	}

	return RES_CODE_SYS_ERR, ErrServNotExist
}

func (c *client) AsyncCall(cb func(code int32, resp interface{}, err error), peerType uint32, peerNo uint32, service string, funcName string, reqObj interface{}, respObj interface{}) {
	pipeline, ok := c.getPipeline(peerType, peerNo)
	if ok {
		pipeline.AsyncCall(cb, service, funcName, reqObj, respObj)
		return
	}

	if cb != nil {
		cb(RES_CODE_SYS_ERR, respObj, ErrServNotExist)
	}
}

func (c *client) CallNoReturn(peerType uint32, peerNo uint32, service string, funcName string, reqObj interface{}) error {
	pipeline, ok := c.getPipeline(peerType, peerNo)
	if ok {
		err := pipeline.CallNoReturn(service, funcName, reqObj)
		return err
	}

	return ErrServNotExist
}

func (c *client) addPipeline(net Net, peerType uint32, peerNo uint32, mark string) (oldPipeline *Pipeline, newPipeline *Pipeline) {
	c.lckPipelines.Lock()
	defer c.lckPipelines.Unlock()

	oldPipeline = nil
	peerId := GetPeerId(peerType, peerNo)
	pipeline, ok := c.mapPeerId2Pipeline[peerId]
	if ok {
		oldPipeline = pipeline
		// for i, pipeline := range pipelines {
		// 	if pipeline.GetMark() == mark {
		// 		oldPipeline = pipeline
		// 		c.mapPeerId2Pipeline[peerId] = append(pipelines[:i], pipelines[i+1:]...)
		// 		break
		// 	}
		// }
	}

	newPipeline = NewPipeline(net, peerType, peerNo, mark)
	c.mapPeerId2Pipeline[peerId] = newPipeline
	// c.mapPeerId2Pipeline[peerId] = append(c.mapPeerId2Pipeline[peerId], newPipeline)
	return oldPipeline, newPipeline
}

func (c *client) getPipeline(peerType uint32, peerNo uint32) (*Pipeline, bool) {
	c.lckPipelines.Lock()
	defer c.lckPipelines.Unlock()

	peerId := GetPeerId(peerType, peerNo)
	pipeline, ok := c.mapPeerId2Pipeline[peerId]
	return pipeline, ok
	// if ok {
	// 	for _, pipeline := range pipelines {
	// 		if pipeline.GetMark() == mark {
	// 			return pipeline, true
	// 		}
	// 	}
	// }

	// return nil, false
}

func (c *client) removePipeline(peerType uint32, peerNo uint32) (*Pipeline, bool) {
	c.lckPipelines.Lock()
	defer c.lckPipelines.Unlock()

	peerId := GetPeerId(peerType, peerNo)
	pipeline, ok := c.mapPeerId2Pipeline[peerId]
	if ok {
		delete(c.mapPeerId2Pipeline, peerId)
		// for i, pipeline := range pipelines {
		// 	if pipeline.GetMark() == mark {
		// 		c.mapPeerId2Pipeline[peerId] = append(pipelines[:i], pipelines[i+1:]...)
		// 		return pipeline, true
		// 	}
		// }
	}

	return pipeline, ok
}

// func (c *client) removePipelines(peerType uint32, peerNo uint32) ([]*Pipeline, bool) {
// 	c.lckPipelines.Lock()
// 	defer c.lckPipelines.Unlock()

// 	peerId := GetPeerId(peerType, peerNo)
// 	pipelines, ok := c.mapPeerId2Pipeline[peerId]
// 	if ok {
// 		delete(c.mapPeerId2Pipeline, peerId)
// 	}

// 	return pipelines, ok
// }

func (c *client) removeAllPipelines() []*Pipeline {
	pipelines := make([]*Pipeline, 0)

	c.lckPipelines.Lock()
	defer c.lckPipelines.Unlock()

	for _, pipeline := range c.mapPeerId2Pipeline {
		pipelines = append(pipelines, pipeline)
		// pipeline.Stop()
		// for _, pipeline := range pipelines {
		// 	pipeline.Stop()
		// }
	}

	c.mapPeerId2Pipeline = make(map[uint32]*Pipeline)
	return pipelines
}
