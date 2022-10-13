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

type PipelineList = []*Pipeline

//==========================================
//               client
//==========================================
type client struct {
	mapPeerId2Pipelines map[uint32]PipelineList
	lckPipelines        *sync.Mutex
	ec                  *yx.ErrCatcher
}

var Client = &client{
	mapPeerId2Pipelines: make(map[uint32]PipelineList),
	lckPipelines:        &sync.Mutex{},
	ec:                  yx.NewErrCatcher("rpc.Client"),
}

func (c *client) AddPipeline(net Net, peerType uint32, peerNo uint32, service string, timeoutSec uint32) (*Pipeline, error) {
	oldPipeline, newPipeline := c.addPipeline(net, peerType, peerNo, service)
	if oldPipeline != nil {
		oldPipeline.Stop()
	}

	newPipeline.SetTimeout(timeoutSec)
	go newPipeline.Start()
	return newPipeline, nil
}

func (c *client) GetPipeline(peerType uint32, peerNo uint32, service string) (*Pipeline, bool) {
	return c.getPipeline(peerType, peerNo, service)
}

func (c *client) RemovePipeline(peerType uint32, peerNo uint32, service string) {
	pipeline, ok := c.removePipeline(peerType, peerNo, service)
	if ok {
		pipeline.Stop()
	}
}

func (c *client) RemovePipelines(peerType uint32, peerNo uint32) {
	pipelines, ok := c.removePipelines(peerType, peerNo)
	if ok {
		for _, pipeline := range pipelines {
			pipeline.Stop()
		}
	}
}

func (c *client) RemoveAllPeers() {
	c.removeAllPeers()
}

func (c *client) Call(peerType uint32, peerNo uint32, service string, funcName string, reqObj interface{}, respObj interface{}) error {
	pipeline, ok := c.getPipeline(peerType, peerNo, service)
	if ok {
		return pipeline.Call(funcName, reqObj, respObj)
	}

	return ErrServNotExist
}

func (c *client) AsyncCall(peerType uint32, peerNo uint32, service string, cb func(interface{}, error), funcName string, reqObj interface{}, respObj interface{}) {
	pipeline, ok := c.getPipeline(peerType, peerNo, service)
	if ok {
		pipeline.AsyncCall(cb, funcName, reqObj, respObj)
		return
	}

	if cb != nil {
		cb(nil, ErrServNotExist)
	}
}

func (c *client) CallNoReturn(peerType uint32, peerNo uint32, service string, funcName string, reqObj interface{}) error {
	pipeline, ok := c.getPipeline(peerType, peerNo, service)
	if ok {
		err := pipeline.CallNoReturn(funcName, reqObj)
		return err
	}

	return ErrServNotExist
}

func (c *client) addPipeline(net Net, peerType uint32, peerNo uint32, service string) (oldPipeline *Pipeline, newPipeline *Pipeline) {
	c.lckPipelines.Lock()
	defer c.lckPipelines.Unlock()

	oldPipeline = nil
	peerId := GetPeerId(peerType, peerNo)
	pipelines, ok := c.mapPeerId2Pipelines[peerId]
	if ok {
		for i, pipeline := range pipelines {
			if pipeline.GetService() == service {
				oldPipeline = pipeline
				c.mapPeerId2Pipelines[peerId] = append(pipelines[:i], pipelines[i+1:]...)
				break
			}
		}
	}

	newPipeline = NewPipeline(net, peerType, peerNo, service)
	c.mapPeerId2Pipelines[peerId] = append(c.mapPeerId2Pipelines[peerId], newPipeline)
	return oldPipeline, newPipeline
}

func (c *client) getPipeline(peerType uint32, peerNo uint32, service string) (*Pipeline, bool) {
	c.lckPipelines.Lock()
	defer c.lckPipelines.Unlock()

	peerId := GetPeerId(peerType, peerNo)
	pipelines, ok := c.mapPeerId2Pipelines[peerId]
	if ok {
		for _, pipeline := range pipelines {
			if pipeline.GetService() == service {
				return pipeline, true
			}
		}
	}

	return nil, false
}

func (c *client) removePipeline(peerType uint32, peerNo uint32, service string) (*Pipeline, bool) {
	c.lckPipelines.Lock()
	defer c.lckPipelines.Unlock()

	peerId := GetPeerId(peerType, peerNo)
	pipelines, ok := c.mapPeerId2Pipelines[peerId]
	if ok {
		for i, pipeline := range pipelines {
			if pipeline.GetService() == service {
				c.mapPeerId2Pipelines[peerId] = append(pipelines[:i], pipelines[i+1:]...)
				return pipeline, true
			}
		}
	}

	return nil, false
}

func (c *client) removePipelines(peerType uint32, peerNo uint32) ([]*Pipeline, bool) {
	c.lckPipelines.Lock()
	defer c.lckPipelines.Unlock()

	peerId := GetPeerId(peerType, peerNo)
	pipelines, ok := c.mapPeerId2Pipelines[peerId]
	if ok {
		delete(c.mapPeerId2Pipelines, peerId)
	}

	return pipelines, ok
}

func (c *client) removeAllPeers() {
	c.lckPipelines.Lock()
	defer c.lckPipelines.Unlock()

	for _, pipelines := range c.mapPeerId2Pipelines {
		for _, pipeline := range pipelines {
			pipeline.Stop()
		}
	}

	c.mapPeerId2Pipelines = make(map[uint32]PipelineList)
}
