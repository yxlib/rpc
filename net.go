// Copyright 2022 Guan Jianchang. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import (
	"errors"

	"github.com/yxlib/yx"
)

var (
	ErrNetReadChanClose = errors.New("read channel closed")
)

type ByteArray = []byte

//========================
//     NetDataWrap
//========================
type NetDataWrap struct {
	PeerType uint32
	PeerNo   uint32
	Payload  []byte
}

func NewNetDataWrap(peerType uint32, peerNo uint32, payload []byte) *NetDataWrap {
	return &NetDataWrap{
		Payload:  payload,
		PeerType: peerType,
		PeerNo:   peerNo,
	}
}

//========================
//     Net
//========================
type Net interface {
	SetReadMark(mark string, bSrv bool, srcPeerType uint32, srcPeerNo uint32)
	GetReadMark() string
	GetPeerTypeAndNo() (uint32, uint32)
	// RemoveReadMark(mark string, srcPeerType uint16, srcPeerNo uint16)
	// GetReadMark(srcPeerType uint16, srcPeerNo uint16) (string, bool)
	// PushReadPack(peerType uint16, peerNo uint16, payload []byte)
	ReadRpcPack() (*NetDataWrap, error)
	WriteRpcPack(payload []ByteArray, dstPeerType uint32, dstPeerNo uint32) error
	Close()
}

//========================
//        BaseNet
//========================
type BaseNet struct {
	// mapPeerId2Mark map[uint32]string
	mark        string
	bSrv        bool
	srcPeerType uint32
	srcPeerNo   uint32
	chanPacks   chan *NetDataWrap
	logger      *yx.Logger
	ec          *yx.ErrCatcher
}

func NewBaseNet(maxReadQue uint32) *BaseNet {
	return &BaseNet{
		// mapPeerId2Mark: make(map[uint32]string),
		mark:        "",
		bSrv:        false,
		srcPeerType: 0,
		srcPeerNo:   0,
		chanPacks:   make(chan *NetDataWrap, maxReadQue),
		logger:      yx.NewLogger("RpcNet"),
		ec:          yx.NewErrCatcher("RpcNet"),
	}
}

// rpc.Net
func (n *BaseNet) SetReadMark(mark string, bSrv bool, srcPeerType uint32, srcPeerNo uint32) {
	n.mark = mark
	n.bSrv = bSrv
	n.srcPeerType = srcPeerType
	n.srcPeerNo = srcPeerNo
	// peerId := GetPeerId(srcPeerType, srcPeerNo)
	// _, ok := n.mapPeerId2Mark[peerId]
	// if ok {
	// 	return
	// }

	// n.mapPeerId2Mark[peerId] = mark
}

func (n *BaseNet) GetReadMark() string {
	return n.mark
}

func (n *BaseNet) GetPeerTypeAndNo() (uint32, uint32) {
	return n.srcPeerType, n.srcPeerNo
}

func (n *BaseNet) IsSrvNet() bool {
	return n.bSrv
}

// func (n *BaseNet) RemoveReadMark(mark string, srcPeerType uint16, srcPeerNo uint16) {
// 	peerId := GetPeerId(srcPeerType, srcPeerNo)
// 	_, ok := n.mapPeerId2Mark[peerId]
// 	if ok {
// 		delete(n.mapPeerId2Mark, peerId)
// 	}
// }

// func (n *BaseNet) GetReadMark(srcPeerType uint16, srcPeerNo uint16) (string, bool) {
// 	peerId := GetPeerId(srcPeerType, srcPeerNo)
// 	mark, ok := n.mapPeerId2Mark[peerId]
// 	return mark, ok
// }

func (n *BaseNet) AddReadPack(peerType uint32, peerNo uint32, payload []byte) {
	pack := NewNetDataWrap(peerType, peerNo, payload)
	n.chanPacks <- pack
}

func (n *BaseNet) ReadRpcPack() (*NetDataWrap, error) {
	pack, ok := <-n.chanPacks
	if !ok {
		return nil, ErrNetReadChanClose
	}

	return pack, nil
}

func (n *BaseNet) WriteRpcPack(payload []ByteArray, dstPeerType uint32, dstPeerNo uint32) error {
	return nil
}

func (n *BaseNet) Close() {
	close(n.chanPacks)
}
