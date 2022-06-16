// Copyright 2022 Guan Jianchang. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/yxlib/yx"
)

var (
	ErrSrvNetClose       = errors.New("NetClose")
	ErrSrvProcNil        = errors.New("processor is nil")
	ErrSrvWrongFormatRet = errors.New("return value format wrong")
)

type RpcHandler = func(req interface{}, resp interface{}, srcPeerType uint32, srcPeerNo uint32) error

type Interceptor interface {
	// Unmarshal the request payload to interface{}.
	// @param funcName, the func name.
	// @param payload, the request payload
	// @return interface{}, an object unmarshal from payload.
	// @return error, error.
	OnPreHandle(funcName string, payload []byte) (interface{}, interface{}, error)

	// Marshal the response to []byte.
	// @param funcName, the func name.
	// @param v, the response object
	// @return []byte, bytes after marshal.
	// @return error, error.
	OnHandleCompletion(funcName string, req interface{}, resp interface{}) ([]byte, error)
}

type Server interface {
	SetMark(mark string)
	GetRpcNet() Net
	AddReflectProcessor(processor reflect.Value, funcNo uint16, funcName string) error
	Start()
	Stop()
}

type BaseServer struct {
	mark              string
	bDebugMode        bool
	mapFuncNo2Name    map[uint16]string
	mapFuncNo2Handler map[uint16]reflect.Value
	inter             Interceptor
	net               Net
	ec                *yx.ErrCatcher
	logger            *yx.Logger
}

func NewBaseServer(net Net) *BaseServer {
	s := &BaseServer{
		mark:              "",
		bDebugMode:        false,
		mapFuncNo2Name:    make(map[uint16]string),
		mapFuncNo2Handler: make(map[uint16]reflect.Value),
		inter:             nil,
		net:               net,
		ec:                yx.NewErrCatcher("rpc.Server"),
		logger:            yx.NewLogger("rpc.Server"),
	}

	return s
}

func (s *BaseServer) SetInterceptor(inter Interceptor) {
	s.inter = inter
}

func (s *BaseServer) SetMark(mark string) {
	s.mark = mark
	s.net.SetReadMark(s.mark, true, 0, 0)
}

func (s *BaseServer) GetMark() string {
	return s.mark
}

func (s *BaseServer) GetRpcNet() Net {
	return s.net
}

func (s *BaseServer) SetDebugMode(bDebugMode bool) {
	s.bDebugMode = bDebugMode
}

func (s *BaseServer) IsDebugMode() bool {
	return s.bDebugMode
}

func (s *BaseServer) AddReflectProcessor(processor reflect.Value, funcNo uint16, funcName string) error {
	var err error = nil
	defer s.ec.DeferThrow("AddReflectProcessor", &err)

	if processor.String() == "<invalid Value>" {
		err = ErrSrvProcNil
		return err
	}

	_, ok := s.mapFuncNo2Handler[funcNo]
	if !ok {
		s.mapFuncNo2Handler[funcNo] = processor
		s.mapFuncNo2Name[funcNo] = funcName
	}

	return nil
}

func (s *BaseServer) Start() {
	s.readPackLoop()
}

func (s *BaseServer) Stop() {
	s.net.Close()
}

func (s *BaseServer) WritePack(payload []ByteArray, dstPeerType uint32, dstPeerNo uint32) error {
	err := s.net.WriteRpcPack(payload, dstPeerType, dstPeerNo)
	return s.ec.Throw("WritePack", err)
}

func (s *BaseServer) OnFetchFuncList(req interface{}, resp interface{}, srcPeerType uint32, srcPeerNo uint32) error {
	respData := resp.(*FetchFuncListResp)
	respData.MapFuncName2No = s.getFuncList()
	return nil
}

func (s *BaseServer) getFuncList() map[string]uint16 {
	mapFuncName2No := make(map[string]uint16)
	for funcNo, funcName := range s.mapFuncNo2Name {
		mapFuncName2No[funcName] = funcNo
	}

	return mapFuncName2No
}

func (s *BaseServer) readPackLoop() {
	for {
		data, err := s.net.ReadRpcPack()
		if err != nil {
			break
		}

		h := NewPackHeader([]byte(s.mark), 0, 0)
		// req := NewRequest(s.mark, 0, 0, nil)
		err = h.Unmarshal(data.Payload)
		if err != nil {
			s.ec.Catch("Start", &err)
			continue
		}

		headerLen := h.GetHeaderLen()
		req := NewSingleFrameRequest(h, data.Payload[headerLen:])
		yx.RunDangerCode(func() {
			s.handleRequest(req, data.PeerType, data.PeerNo)
		}, s.bDebugMode)
	}
}

func (s *BaseServer) handleRequest(req *Request, peerType uint32, peerNo uint32) {
	var err error = nil
	defer s.ec.Catch("handleRequest", &err)

	// pre handle
	handler, ok := s.getHandler(req.Header.FuncNo)
	if !ok {
		err = fmt.Errorf("no handler for funcNo %d", req.Header.FuncNo)
		return
	}

	funcName := s.mapFuncNo2Name[req.Header.FuncNo]
	fullFuncName := GetFullFuncName(s.mark, funcName)
	reqData := interface{}(req.Payload[0])
	respData := interface{}(nil)
	if s.inter != nil {
		reqData, respData, err = s.inter.OnPreHandle(fullFuncName, req.Payload[0])
		if err != nil {
			return
		}
	}

	// handle
	err = handler(reqData, respData, peerType, peerNo)
	if err != nil {
		return
	}

	// handle completion
	var returnData []byte = nil
	if s.inter != nil {
		returnData, err = s.inter.OnHandleCompletion(fullFuncName, reqData, respData)
		if err != nil {
			return
		}
	} else if respData != nil {
		returnData, ok = respData.([]byte)
		if !ok {
			err = ErrSrvWrongFormatRet
			return
		}
	}

	if returnData == nil {
		return
	}

	// response
	headerData, err := req.Header.Marshal()
	if err != nil {
		return
	}

	payload := make([]ByteArray, 0, 2)
	payload = append(payload, headerData, returnData)
	err = s.WritePack(payload, peerType, peerNo)
}

func (s *BaseServer) getHandler(funcNo uint16) (RpcHandler, bool) {
	var m RpcHandler = nil
	v, ok := s.mapFuncNo2Handler[funcNo]
	if ok {
		m = v.Interface().(RpcHandler)
	}

	return m, ok
}
