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
	ErrPeerNotSupportFunc = errors.New("not support this func")
	ErrPeerNetNil         = errors.New("rpc net is nil")
	ErrPeerForceCallStop  = errors.New("force call stop")
)

type Peer struct {
	net            Net
	mark           string
	peerType       uint32
	peerNo         uint32
	mapFuncName2No map[string]uint16
	timeoutSec     uint32

	maxSerialNo uint16
	mapSno2Req  map[uint16]*Request
	lckRequests *sync.Mutex

	ec     *yx.ErrCatcher
	logger *yx.Logger
}

func NewPeer(net Net, mark string, peerType uint32, peerNo uint32) *Peer {
	p := &Peer{
		net:            net,
		mark:           mark,
		peerType:       peerType,
		peerNo:         peerNo,
		mapFuncName2No: make(map[string]uint16),
		timeoutSec:     0,

		maxSerialNo: 0,
		mapSno2Req:  make(map[uint16]*Request),
		lckRequests: &sync.Mutex{},

		ec:     yx.NewErrCatcher("rpc.Peer"),
		logger: yx.NewLogger("rpc.Peer"),
	}

	p.net.SetReadMark(mark, false, peerType, peerNo)
	return p
}

func (p *Peer) GetMark() string {
	return p.mark
}

func (p *Peer) SetTimeout(timeoutSec uint32) {
	p.timeoutSec = timeoutSec
}

func (p *Peer) Start() {
	p.readPackLoop()
}

func (p *Peer) Stop() {
	p.net.Close()
	// p.net.RemoveReadMark(p.mark, p.peerType, p.peerNo)
	p.stopAllRequest()
}

func (p *Peer) FetchFuncList(cb func([]byte) (*FetchFuncListResp, error)) error {
	payload, err := p.callByFuncNo(RPC_FUNC_NO_FUNC_LIST, nil, false)
	if err != nil {
		return p.ec.Throw("FetchFuncList", err)
	}

	if cb != nil {
		resp, err := cb(payload)
		if err != nil {
			return p.ec.Throw("FetchFuncList", err)
		}

		p.mapFuncName2No = resp.MapFuncName2No
	}

	return nil

	// resp := &FuncListResp{}
	// err = json.Unmarshal(payload, resp)
	// if err != nil {
	// 	return p.ec.Throw("FetchFuncList", err)
	// }

	// p.mapFuncName2No = resp.MapFuncName2No
	// return nil
}

func (p *Peer) Call(funcName string, params []byte) ([]byte, error) {
	buff, err := p.callByFuncName(funcName, params, false)
	return buff, p.ec.Throw("Call", err)
}

func (p *Peer) AsyncCall(funcName string, params []byte, cb func([]byte, error)) {
	go func() {
		result, err := p.Call(funcName, params)
		if cb != nil {
			cb(result, err)
		}
	}()
}

func (p *Peer) CallNoReturn(funcName string, params []byte) error {
	_, err := p.callByFuncName(funcName, params, true)
	return p.ec.Throw("CallNoReturn", err)
}

func (p *Peer) callByFuncName(funcName string, params []byte, bNoReturn bool) ([]byte, error) {
	funcNo, ok := p.mapFuncName2No[funcName]
	if !ok {
		return nil, p.ec.Throw("Call", ErrPeerNotSupportFunc)
	}

	payload, err := p.callByFuncNo(funcNo, params, bNoReturn)
	return payload, p.ec.Throw("Call", err)
}

func (p *Peer) callByFuncNo(funcNo uint16, params []byte, bNoReturn bool) ([]byte, error) {
	var err error = nil
	defer p.ec.DeferThrow("callByFuncNo", &err)

	if p.net == nil {
		err = ErrPeerNetNil
		return nil, err
	}

	if bNoReturn {
		err := p.callNoReturnImpl(funcNo, params)
		return nil, err
	}

	// add to list
	req, payload, err := p.addRequest(funcNo, params)
	if err != nil {
		return nil, err
	}

	defer p.stopRequest(req.Header.SerialNo)

	// send
	err = p.net.WriteRpcPack(payload, p.peerType, p.peerNo)
	if err != nil {
		return nil, err
	}

	// go c.readPack()

	// wait
	err = p.wait(req)
	if err != nil {
		return nil, err
	}

	// get response
	_, ok := p.getRequest(req.Header.SerialNo)
	if !ok {
		err = ErrPeerForceCallStop
		return nil, err
	}

	return req.respPayload, nil
}

func (p *Peer) callNoReturnImpl(funcNo uint16, params []byte) error {
	var err error = nil
	defer p.ec.DeferThrow("callNoReturnImpl", &err)

	h := NewPackHeader([]byte(p.mark), 0, funcNo)
	headerData, err := h.Marshal()
	if err != nil {
		return err
	}

	payload := make([]ByteArray, 0, 2)
	if len(params) > 0 {
		payload = append(payload, headerData, params)
	}

	err = p.net.WriteRpcPack(payload, p.peerType, p.peerNo)
	return err
}

// func (p *RpcPeer) StopCall() {
// 	p.resetCurRequest()
// }

func (p *Peer) addRequest(funcNo uint16, params []byte) (*Request, []ByteArray, error) {
	p.lckRequests.Lock()
	defer p.lckRequests.Unlock()

	sno := p.maxSerialNo + 1
	h := NewPackHeader([]byte(p.mark), sno, funcNo)
	headerData, err := h.Marshal()
	if err != nil {
		return nil, nil, p.ec.Throw("addRequest", err)
	}

	req := NewSingleFrameRequest(h, params)
	payload := make([]ByteArray, 0, 2)
	if len(params) > 0 {
		payload = append(payload, headerData, params)
	} else {
		payload = append(payload, headerData)
	}

	p.maxSerialNo++
	p.mapSno2Req[sno] = req
	return req, payload, nil
}

func (p *Peer) stopRequest(sno uint16) {
	p.lckRequests.Lock()
	defer p.lckRequests.Unlock()

	req, ok := p.mapSno2Req[sno]
	if ok {
		req.Cancel()
		delete(p.mapSno2Req, sno)
	}
}

func (p *Peer) stopAllRequest() {
	p.lckRequests.Lock()
	defer p.lckRequests.Unlock()

	for _, req := range p.mapSno2Req {
		req.Cancel()
	}

	p.mapSno2Req = make(map[uint16]*Request)
}

func (p *Peer) getRequest(sno uint16) (*Request, bool) {
	p.lckRequests.Lock()
	defer p.lckRequests.Unlock()

	req, ok := p.mapSno2Req[sno]
	return req, ok
}

func (p *Peer) wait(req *Request) error {
	var err error = nil
	if p.timeoutSec == 0 {
		err = req.Wait()
	} else {
		err = req.WaitUntilTimeout(p.timeoutSec * 1000)
	}

	if err != nil {
		p.logger.W(err.Error())
		return err
	}

	return nil
}

func (p *Peer) readPackLoop() {
	for {
		data, err := p.net.ReadRpcPack()
		if err != nil {
			break
		}

		h := NewPackHeader([]byte(p.mark), 0, 0)
		err = h.Unmarshal(data.Payload)
		if err != nil {
			p.ec.Catch("readPackLoop", &err)
			continue
		}

		headerLen := h.GetHeaderLen()
		p.handlePack(h.SerialNo, h.FuncNo, data.Payload[headerLen:])
	}
}

func (p *Peer) handlePack(serialNo uint16, funcNo uint16, payload []byte) {
	req, ok := p.getRequest(serialNo)
	if !ok {
		return
	}

	if funcNo != req.Header.FuncNo {
		return
	}

	req.respPayload = payload
	req.Signal()
}
