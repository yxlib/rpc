// Copyright 2022 Guan Jianchang. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import (
	"sync"

	"github.com/yxlib/yx"
)

//==========================================
//               Client
//==========================================
type Client struct {
	mapPeerId2Peer map[uint32]*Peer
	lckPeer        *sync.Mutex
	ec             *yx.ErrCatcher
}

func NewClient() *Client {
	return &Client{
		mapPeerId2Peer: make(map[uint32]*Peer),
		lckPeer:        &sync.Mutex{},
		ec:             yx.NewErrCatcher("rpc.Client"),
	}
}

func (c *Client) AddPeer(net Net, mark string, peerType uint32, peerNo uint32, timeoutSec uint32) (*Peer, error) {
	oldPeer, newPeer := c.addPeer(net, mark, peerType, peerNo)
	if oldPeer != nil {
		oldPeer.Stop()
	}

	newPeer.SetTimeout(timeoutSec)
	go newPeer.Start()
	return newPeer, nil
}

func (c *Client) GetPeer(peerType uint32, peerNo uint32) (*Peer, bool) {
	return c.getPeer(peerType, peerNo)
}

func (c *Client) RemovePeer(peerType uint32, peerNo uint32) {
	peer, ok := c.removePeer(peerType, peerNo)
	if ok {
		peer.Stop()
	}
}

func (c *Client) RemoveAllPeers() {
	c.removeAllPeers()
}

func (c *Client) addPeer(net Net, mark string, peerType uint32, peerNo uint32) (oldPeer *Peer, newPeer *Peer) {
	c.lckPeer.Lock()
	defer c.lckPeer.Unlock()

	peerId := GetPeerId(peerType, peerNo)
	oldPeer = c.mapPeerId2Peer[peerId]

	peer := NewPeer(net, mark, peerType, peerNo)
	c.mapPeerId2Peer[peerId] = peer
	return oldPeer, peer
}

func (c *Client) getPeer(peerType uint32, peerNo uint32) (*Peer, bool) {
	c.lckPeer.Lock()
	defer c.lckPeer.Unlock()

	peerId := GetPeerId(peerType, peerNo)
	peer, ok := c.mapPeerId2Peer[peerId]
	return peer, ok
}

func (c *Client) removePeer(peerType uint32, peerNo uint32) (*Peer, bool) {
	c.lckPeer.Lock()
	defer c.lckPeer.Unlock()

	peerId := GetPeerId(peerType, peerNo)
	peer, ok := c.mapPeerId2Peer[peerId]
	if ok {
		delete(c.mapPeerId2Peer, peerId)
	}

	return peer, ok
}

func (c *Client) removeAllPeers() {
	c.lckPeer.Lock()
	defer c.lckPeer.Unlock()

	for _, peer := range c.mapPeerId2Peer {
		peer.Stop()
	}

	c.mapPeerId2Peer = make(map[uint32]*Peer)
}
