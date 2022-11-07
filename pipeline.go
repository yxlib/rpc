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
	ErrPipelineNotSupportFunc = errors.New("not support this func")
	ErrPipelineInterNil       = errors.New("interceptor is nil")
	ErrPipelineNetNil         = errors.New("rpc net is nil")
	ErrPipelineForceCallStop  = errors.New("force call stop")
)

// type PipelineInterceptor interface {
// 	OnMarshalRequest(funcName string, reqObj interface{}) ([]byte, error)
// 	OnUnmarshalResponse(funcName string, respData []byte, respObj interface{}) error
// }

type Pipeline struct {
	net            Net
	service        string
	peerType       uint32
	peerNo         uint32
	mapFuncName2No map[string]uint16
	timeoutSec     uint32
	inter          Interceptor

	maxSerialNo uint16
	mapSno2Req  map[uint16]*Request
	lckRequests *sync.Mutex

	ec     *yx.ErrCatcher
	logger *yx.Logger
}

func NewPipeline(net Net, peerType uint32, peerNo uint32, service string) *Pipeline {
	p := &Pipeline{
		net:            net,
		service:        service,
		peerType:       peerType,
		peerNo:         peerNo,
		mapFuncName2No: make(map[string]uint16),
		timeoutSec:     0,
		inter:          nil,

		maxSerialNo: 0,
		mapSno2Req:  make(map[uint16]*Request),
		lckRequests: &sync.Mutex{},

		ec:     yx.NewErrCatcher("rpc.Pipeline"),
		logger: yx.NewLogger("rpc.Pipeline"),
	}

	p.net.SetService(p.service, false, peerType, peerNo)
	return p
}

func (p *Pipeline) GetService() string {
	return p.service
}

func (p *Pipeline) SetInterceptor(inter Interceptor) {
	p.inter = inter
}

func (p *Pipeline) SetTimeout(timeoutSec uint32) {
	p.timeoutSec = timeoutSec
}

func (p *Pipeline) GetFuncList() []string {
	funcList := make([]string, 0, len(p.mapFuncName2No))
	for name := range p.mapFuncName2No {
		funcList = append(funcList, name)
	}

	return funcList
}

func (p *Pipeline) Start() {
	p.readPackLoop()
}

func (p *Pipeline) Stop() {
	p.net.Close()
	// p.net.RemoveReadMark(p.mark, p.peerType, p.peerNo)
	p.stopAllRequest()
}

func (p *Pipeline) FetchFuncList() error {
	if p.inter == nil {
		return p.ec.Throw("FetchFuncList", ErrPipelineInterNil)
	}

	code, payload, err := p.CallByFuncNo(RPC_FUNC_NO_FUNC_LIST, false)
	if err != nil {
		return p.ec.Throw("FetchFuncList", err)
	}

	if code != RES_CODE_SUCC {
		err = errors.New(string(payload))
		return p.ec.Throw("FetchFuncList", err)
	}

	resp := &FetchFuncListResp{}
	fullFuncName := GetFullFuncName(p.service, RPC_FUNC_NAME_FUNC_LIST)
	err = p.inter.OnUnmarshal(fullFuncName, payload, resp)
	if err != nil {
		return p.ec.Throw("FetchFuncList", err)
	}

	p.mapFuncName2No = resp.MapFuncName2No
	return nil

	// resp := &FuncListResp{}
	// err = json.Unmarshal(payload, resp)
	// if err != nil {
	// 	return p.ec.Throw("FetchFuncList", err)
	// }

	// p.mapFuncName2No = resp.MapFuncName2No
	// return nil
}

func (p *Pipeline) AsyncFetchFuncList(cb func(err error)) {
	if p.inter == nil {
		if cb != nil {
			cb(ErrPipelineInterNil)
		}

		return
	}

	go func() {
		err := p.FetchFuncList()
		if cb != nil {
			cb(err)
		}
	}()
}

func (p *Pipeline) Call(funcName string, reqObj interface{}, respObj interface{}) (int32, error) {
	code := RES_CODE_SYS_ERR

	if p.inter == nil {
		return code, p.ec.Throw("Call", ErrPipelineInterNil)
	}

	fullFuncName := GetFullFuncName(p.service, funcName)
	params, err := p.inter.OnMarshal(fullFuncName, reqObj)
	if err != nil {
		return code, p.ec.Throw("Call", err)
	}

	code, buff, err := p.CallByFuncName(funcName, false, params)
	if err != nil {
		return code, p.ec.Throw("Call", err)
	}

	if respObj != nil {
		err = p.inter.OnUnmarshal(fullFuncName, buff, respObj)
		if err != nil {
			return code, p.ec.Throw("Call", err)
		}
	}

	return code, nil
}

func (p *Pipeline) AsyncCall(cb func(code int32, resp interface{}, err error), funcName string, reqObj interface{}, respObj interface{}) {
	if p.inter == nil {
		if cb != nil {
			cb(RES_CODE_SYS_ERR, respObj, ErrPipelineInterNil)
		}

		return
	}

	go func() {
		code, err := p.Call(funcName, reqObj, respObj)
		if cb != nil {
			cb(code, respObj, err)
		}
	}()
}

func (p *Pipeline) CallNoReturn(funcName string, reqObj interface{}) error {
	if p.inter == nil {
		return p.ec.Throw("CallNoReturn", ErrPipelineInterNil)
	}

	fullFuncName := GetFullFuncName(p.service, funcName)
	params, err := p.inter.OnMarshal(fullFuncName, reqObj)
	if err != nil {
		return p.ec.Throw("CallNoReturn", err)
	}

	_, _, err = p.CallByFuncName(funcName, true, params)
	return p.ec.Throw("CallNoReturn", err)
}

func (p *Pipeline) CallByFuncName(funcName string, bNoReturn bool, params ...[]byte) (int32, []byte, error) {
	funcNo, ok := p.mapFuncName2No[funcName]
	if !ok {
		return RES_CODE_SYS_ERR, nil, p.ec.Throw("CallByFuncName", ErrPipelineNotSupportFunc)
	}

	code, payload, err := p.CallByFuncNo(funcNo, bNoReturn, params...)
	if err != nil {
		return code, nil, p.ec.Throw("CallByFuncName", err)
	}

	if code != RES_CODE_SUCC {
		err = errors.New(string(payload))
		return code, nil, p.ec.Throw("CallByFuncName", err)
	}

	return code, payload, nil
}

func (p *Pipeline) CallByFuncNo(funcNo uint16, bNoReturn bool, params ...[]byte) (int32, []byte, error) {
	var err error = nil
	defer p.ec.DeferThrow("callByFuncNo", &err)

	code := RES_CODE_SYS_ERR
	if p.net == nil {
		err = ErrPipelineNetNil
		return code, nil, err
	}

	if bNoReturn {
		err := p.callNoReturnImpl(funcNo, params...)
		if err == nil {
			code = RES_CODE_SUCC
		}

		return code, nil, err
	}

	// add to list
	req, payload, err := p.addRequest(funcNo, params...)
	if err != nil {
		return code, nil, err
	}

	defer p.stopRequest(req.Header.SerialNo)

	// send
	err = p.net.WriteRpcPack(p.peerType, p.peerNo, payload...)
	if err != nil {
		return code, nil, err
	}

	// go c.readPack()

	// wait
	err = p.wait(req)
	if err != nil {
		return code, nil, err
	}

	// get response
	_, ok := p.getRequest(req.Header.SerialNo)
	if !ok {
		err = ErrPipelineForceCallStop
		return code, nil, err
	}

	code, respPayload := req.GetResponse()
	return code, respPayload, nil
}

func (p *Pipeline) callNoReturnImpl(funcNo uint16, params ...[]byte) error {
	var err error = nil
	defer p.ec.DeferThrow("callNoReturnImpl", &err)

	h := NewPackHeader(p.service, 0, funcNo)
	headerData, err := h.Marshal()
	if err != nil {
		return err
	}

	payload := make([]ByteArray, 0)
	payload = append(payload, headerData)
	if len(params) > 0 {
		payload = append(payload, params...)
	}

	err = p.net.WriteRpcPack(p.peerType, p.peerNo, payload...)
	return err
}

// func (p *RpcPeer) StopCall() {
// 	p.resetCurRequest()
// }

func (p *Pipeline) addRequest(funcNo uint16, params ...[]byte) (*Request, []ByteArray, error) {
	p.lckRequests.Lock()
	defer p.lckRequests.Unlock()

	sno := p.maxSerialNo + 1
	h := NewPackHeader(p.service, sno, funcNo)
	headerData, err := h.Marshal()
	if err != nil {
		return nil, nil, p.ec.Throw("addRequest", err)
	}

	req := NewRequest(h)
	if len(params) > 0 {
		req.AddFrames(params)
	}

	payload := make([]ByteArray, 0)
	payload = append(payload, headerData)
	if len(params) > 0 {
		payload = append(payload, params...)
	}

	p.maxSerialNo++
	p.mapSno2Req[sno] = req
	return req, payload, nil
}

func (p *Pipeline) stopRequest(sno uint16) {
	p.lckRequests.Lock()
	defer p.lckRequests.Unlock()

	req, ok := p.mapSno2Req[sno]
	if ok {
		req.Cancel()
		delete(p.mapSno2Req, sno)
	}
}

func (p *Pipeline) stopAllRequest() {
	p.lckRequests.Lock()
	defer p.lckRequests.Unlock()

	for _, req := range p.mapSno2Req {
		req.Cancel()
	}

	p.mapSno2Req = make(map[uint16]*Request)
}

func (p *Pipeline) getRequest(sno uint16) (*Request, bool) {
	p.lckRequests.Lock()
	defer p.lckRequests.Unlock()

	req, ok := p.mapSno2Req[sno]
	return req, ok
}

func (p *Pipeline) wait(req *Request) error {
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

func (p *Pipeline) readPackLoop() {
	for {
		data, err := p.net.ReadRpcPack()
		if err != nil {
			break
		}

		h := NewPackHeader(p.service, 0, 0)
		err = h.Unmarshal(data.Payload)
		if err != nil {
			p.ec.Catch("readPackLoop", &err)
			continue
		}

		headerLen := h.GetHeaderLen()
		p.handlePack(h.SerialNo, h.FuncNo, h.Code, data.Payload[headerLen:])
	}
}

func (p *Pipeline) handlePack(serialNo uint16, funcNo uint16, code int32, payload []byte) {
	req, ok := p.getRequest(serialNo)
	if !ok {
		return
	}

	if funcNo != req.Header.FuncNo {
		return
	}

	// req.respPayload = payload
	req.SetResponse(code, payload)
	req.Signal()
}
