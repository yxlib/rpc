// Copyright 2022 Guan Jianchang. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import (
	"sync"

	"github.com/yxlib/yx"
)

// var (
// 	ErrClientPeerNotFound    = errors.New("peer not found")
// 	ErrClientPeerCfgNotFound = errors.New("config for this peer type is not found")
// 	ErrClientFuncCfgNotFound = errors.New("config for this func is not found")
// 	ErrClientProcNil         = errors.New("processor is nil")
// 	ErrClientProcExist       = errors.New("this func name already has processor")
// 	ErrClientMarshalFailed   = errors.New("marshal failed")
// 	ErrClientUnmarshalFailed = errors.New("unmarshal failed")
// )

//==========================================
//               Client
//==========================================
// type RpcCallback = func([]interface{}, error)
// type RpcMarshaler = func(funcName string, params ...interface{}) ([]byte, error)
// type RpcUnmarshaler = func(funcName string, b []byte) ([]interface{}, error)

// type Client interface {
// 	AddPeerType(peerType uint16, mark string)
// 	AddReflectProcessor(marshaler reflect.Value, unmarshaler reflect.Value, mark string, funcName string) error
// }

type Client struct {
	// cfg              *CliConf
	// mapPeerType2Mark map[uint16]string
	mapPeerId2Peer map[uint32]*Peer
	lckPeer        *sync.Mutex

	// mapFuncName2Marshaler   map[string]reflect.Value
	// mapFuncName2Unmarshaler map[string]reflect.Value
	ec *yx.ErrCatcher
}

func NewClient() *Client {
	return &Client{
		// cfg:                     cfg,
		// mapPeerType2Mark: make(map[uint16]string),
		mapPeerId2Peer: make(map[uint32]*Peer),
		lckPeer:        &sync.Mutex{},
		// mapFuncName2Marshaler:   make(map[string]reflect.Value),
		// mapFuncName2Unmarshaler: make(map[string]reflect.Value),
		ec: yx.NewErrCatcher("rpc.Client"),
	}
}

// func (c *Client) InitCfg() {
// 	for mark, peer := range CliCfgInst.MapMark2Peer {
// 		c.mapPeerType2Mark[peer.PeerType] = mark
// 	}
// }

// func (c *Client) AddPeerType(peerType uint16, mark string) {
// 	c.mapPeerType2Mark[peerType] = mark
// }

// func (c *Client) AddReflectProcessor(marshaler reflect.Value, unmarshaler reflect.Value, mark string, funcName string) error {
// 	var err error = nil
// 	defer c.ec.DeferThrow("AddReflectProcessor", &err)

// 	if marshaler.String() == "<invalid Value>" || unmarshaler.String() == "<invalid Value>" {
// 		err = ErrClientProcNil
// 		return err
// 	}

// 	fullFuncName := GetFullFuncName(mark, funcName)
// 	_, ok := c.mapFuncName2Marshaler[fullFuncName]
// 	if !ok {
// 		c.mapFuncName2Marshaler[fullFuncName] = marshaler
// 	}

// 	_, ok = c.mapFuncName2Unmarshaler[fullFuncName]
// 	if !ok {
// 		c.mapFuncName2Unmarshaler[fullFuncName] = unmarshaler
// 	}

// 	return nil
// }

func (c *Client) AddPeer(net Net, mark string, peerType uint16, peerNo uint16, timeoutSec uint32) error {
	// mark, ok := c.mapPeerType2Mark[peerType]
	// if !ok {
	// 	return c.ec.Throw("AddPeer", ErrClientPeerCfgNotFound)
	// }

	oldPeer, newPeer := c.addPeer(net, mark, peerType, peerNo)
	if oldPeer != nil {
		oldPeer.Stop()
	}

	// init new peer
	// cfg := c.cfg.MapMark2Peer[mark]
	newPeer.SetTimeout(timeoutSec)
	newPeer.Start()
	return nil
}

func (c *Client) GetPeer(peerType uint16, peerNo uint16) (*Peer, bool) {
	return c.getPeer(peerType, peerNo)
}

func (c *Client) RemovePeer(peerType uint16, peerNo uint16) {
	peer, ok := c.removePeer(peerType, peerNo)
	if ok {
		peer.Stop()
	}
}

// func (c *Client) SetTimeout(peerType uint16, peerNo uint16, timeoutSec uint32) {
// 	peer, ok := c.getPeer(peerType, peerNo)
// 	if ok {
// 		peer.SetTimeout(timeoutSec)
// 	}
// }

// func (c *Client) FetchFuncList(peerType uint16, peerNo uint16, cb func([]byte) (*FetchFuncListResp, error)) error {
// 	peer, ok := c.getPeer(peerType, peerNo)
// 	if !ok {
// 		return c.ec.Throw("FetchFuncList", ErrClientPeerNotFound)
// 	}

// 	err := peer.FetchFuncList(cb)
// 	return c.ec.Throw("FetchFuncList", err)
// }

// func (c *Client) Call(peerType uint16, peerNo uint16, funcName string, params ...interface{}) ([]interface{}, error) {
// 	return c.callImpl(false, peerType, peerNo, funcName, params...)
// }

// func (c *Client) CallNoReturn(peerType uint16, peerNo uint16, funcName string, params ...interface{}) error {
// 	_, err := c.callImpl(true, peerType, peerNo, funcName, params...)
// 	return err
// }

// func (c *Client) AsyncCall(cb func([]interface{}, error), peerType uint16, peerNo uint16, funcName string, params ...interface{}) {
// 	go func() {
// 		result, err := c.callImpl(false, peerType, peerNo, funcName, params...)
// 		if cb != nil {
// 			cb(result, err)
// 		}
// 	}()
// }

// func (c *Client) callWithData(funcName string, params []byte, peerType uint8, peerNo uint16) ([]byte, error) {
// 	peer, ok := c.getPeer(peerType, peerNo)
// 	if !ok {
// 		return nil, c.ec.Throw("callWithData", ErrClientPeerNotFound)
// 	}

// 	payload, err := peer.Call(funcName, params)
// 	return payload, c.ec.Throw("callWithData", err)
// }

// func (c *Client) getPeerId(peerType uint16, peerNo uint16) uint32 {
// 	return uint32(peerType)<<16 | uint32(peerNo)
// }

func (c *Client) addPeer(net Net, mark string, peerType uint16, peerNo uint16) (oldPeer *Peer, newPeer *Peer) {
	c.lckPeer.Lock()
	defer c.lckPeer.Unlock()

	peerId := GetPeerId(peerType, peerNo)
	oldPeer = c.mapPeerId2Peer[peerId]

	peer := NewPeer(net, mark, peerType, peerNo)
	c.mapPeerId2Peer[peerId] = peer
	return oldPeer, peer
}

func (c *Client) getPeer(peerType uint16, peerNo uint16) (*Peer, bool) {
	c.lckPeer.Lock()
	defer c.lckPeer.Unlock()

	peerId := GetPeerId(peerType, peerNo)
	peer, ok := c.mapPeerId2Peer[peerId]
	return peer, ok
}

func (c *Client) removePeer(peerType uint16, peerNo uint16) (*Peer, bool) {
	c.lckPeer.Lock()
	defer c.lckPeer.Unlock()

	peerId := GetPeerId(peerType, peerNo)
	peer, ok := c.mapPeerId2Peer[peerId]
	if ok {
		delete(c.mapPeerId2Peer, peerId)
	}

	return peer, ok
}

// func (c *Client) getFullFuncName(mark string, funcName string) string {
// 	return mark + "." + funcName
// }

// func (c *Client) getMarshaler(fullFuncName string) (RpcMarshaler, bool) {
// 	var m RpcMarshaler = nil
// 	v, ok := c.mapFuncName2Marshaler[fullFuncName]
// 	if ok {
// 		m = v.Interface().(RpcMarshaler)
// 	}

// 	return m, ok
// }

// func (c *Client) getUnmarshaler(fullFuncName string) (RpcUnmarshaler, bool) {
// 	var m RpcUnmarshaler = nil
// 	v, ok := c.mapFuncName2Unmarshaler[fullFuncName]
// 	if ok {
// 		m = v.Interface().(RpcUnmarshaler)
// 	}

// 	return m, ok
// }

// func (c *Client) callImpl(bNoReturn bool, peerType uint16, peerNo uint16, funcName string, params ...interface{}) ([]interface{}, error) {
// 	peer, ok := c.getPeer(peerType, peerNo)
// 	if !ok {
// 		return nil, c.ec.Throw("Call", ErrClientPeerNotFound)
// 	}

// 	mark := peer.GetMark()
// 	fullFuncName := GetFullFuncName(mark, funcName)

// 	// marshal
// 	marshaler, ok := c.getMarshaler(fullFuncName)
// 	if !ok {
// 		return nil, c.ec.Throw("Call", ErrClientFuncCfgNotFound)
// 	}

// 	reqData, err := marshaler(fullFuncName, params...)
// 	if err != nil {
// 		return nil, c.ec.Throw("Call", ErrClientMarshalFailed)
// 	}

// 	// call
// 	if bNoReturn {
// 		err = peer.CallNoReturn(funcName, reqData)
// 		return nil, err
// 	}

// 	// respData, err := c.callWithData(funcName, reqData, peerType, peerNo)
// 	respData, err := peer.Call(funcName, reqData)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// unmarshal
// 	unmarshaler, ok := c.getUnmarshaler(fullFuncName)
// 	if !ok {
// 		return nil, c.ec.Throw("Call", ErrClientFuncCfgNotFound)
// 	}

// 	v, err := unmarshaler(fullFuncName, respData)
// 	if err != nil {
// 		return nil, c.ec.Throw("Call", ErrClientUnmarshalFailed)
// 	}

// 	return v, nil
// }
