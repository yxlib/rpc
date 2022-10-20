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
	ErrServNetClose       = errors.New("NetClose")
	ErrServProcNil        = errors.New("processor is nil")
	ErrServWrongFormatRet = errors.New("return value format wrong")
)

type RpcHandler = func(req interface{}, resp interface{}, srcPeerType uint32, srcPeerNo uint32) (uint16, error)

// type Interceptor interface {
// 	// Marshal the response to []byte.
// 	// @param funcName, the full func name.
// 	// @param v, the response object
// 	// @return []byte, bytes after marshal.
// 	// @return error, error.
// 	OnMarshal(funcName string, obj interface{}) ([]byte, error)

// 	// Unmarshal the request payload to interface{}.
// 	// @param funcName, the full func name.
// 	// @param payload, the request payload
// 	// @return interface{}, an object unmarshal from payload.
// 	// @return error, error.
// 	OnUnmarshal(funcName string, data []byte, obj interface{}) error
// }

type Service interface {
	SetName(name string)
	GetName() string
	GetRpcNet() Net
	SetDebugMode(bDebugMode bool)
	AddReflectProcessor(processor reflect.Value, funcNo uint16, funcName string) error
	Start()
	Stop()
}

type BaseService struct {
	name              string
	bDebugMode        bool
	mapFuncNo2Name    map[uint16]string
	mapFuncNo2Handler map[uint16]reflect.Value
	inter             Interceptor
	net               Net
	ec                *yx.ErrCatcher
	logger            *yx.Logger
}

func NewBaseService(net Net) *BaseService {
	return &BaseService{
		name:              "",
		bDebugMode:        false,
		mapFuncNo2Name:    make(map[uint16]string),
		mapFuncNo2Handler: make(map[uint16]reflect.Value),
		inter:             nil,
		net:               net,
		ec:                yx.NewErrCatcher("rpc.Service"),
		logger:            yx.NewLogger("rpc.Service"),
	}
}

func (s *BaseService) SetInterceptor(inter Interceptor) {
	s.inter = inter
}

// func (s *BaseService) SetMark(mark string) {
// 	s.name = mark
// 	s.net.SetService(s.name, true, 0, 0)
// }

func (s *BaseService) SetName(name string) {
	s.name = name
	s.net.SetService(s.name, true, 0, 0)
}

func (s *BaseService) GetName() string {
	return s.name
}

func (s *BaseService) GetRpcNet() Net {
	return s.net
}

func (s *BaseService) SetDebugMode(bDebugMode bool) {
	s.bDebugMode = bDebugMode
}

func (s *BaseService) IsDebugMode() bool {
	return s.bDebugMode
}

func (s *BaseService) AddReflectProcessor(processor reflect.Value, funcNo uint16, funcName string) error {
	var err error = nil
	defer s.ec.DeferThrow("AddReflectProcessor", &err)

	if processor.String() == "<invalid Value>" {
		err = ErrServProcNil
		return err
	}

	_, ok := s.mapFuncNo2Handler[funcNo]
	if !ok {
		s.mapFuncNo2Handler[funcNo] = processor
		s.mapFuncNo2Name[funcNo] = funcName
	}

	return nil
}

func (s *BaseService) Start() {
	s.readPackLoop()
}

func (s *BaseService) Stop() {
	s.net.Close()
}

func (s *BaseService) WritePack(dstPeerType uint32, dstPeerNo uint32, payload ...[]byte) error {
	err := s.net.WriteRpcPack(dstPeerType, dstPeerNo, payload...)
	return s.ec.Throw("WritePack", err)
}

func (s *BaseService) OnFetchFuncList(req interface{}, resp interface{}, srcPeerType uint32, srcPeerNo uint32) (uint16, error) {
	respData := resp.(*FetchFuncListResp)
	respData.MapFuncName2No = s.getFuncList()
	return RES_CODE_SUCC, nil
}

func (s *BaseService) getFuncList() map[string]uint16 {
	mapFuncName2No := make(map[string]uint16)
	for funcNo, funcName := range s.mapFuncNo2Name {
		mapFuncName2No[funcName] = funcNo
	}

	return mapFuncName2No
}

func (s *BaseService) readPackLoop() {
	for {
		data, err := s.net.ReadRpcPack()
		if err != nil {
			break
		}

		h := NewPackHeader(s.name, 0, 0)
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

func (s *BaseService) handleRequest(req *Request, peerType uint32, peerNo uint32) {
	var err error = nil
	defer s.ec.Catch("handleRequest", &err)

	code := RES_CODE_SYS_ERR
	var returnData []byte = nil

	defer func() {
		if err != nil {
			returnData = []byte(err.Error())
		}

		err = s.writeResp(req, peerType, peerNo, code, returnData)
	}()

	// pre handle
	handler, ok := s.getHandler(req.Header.FuncNo)
	if !ok {
		err = fmt.Errorf("no handler for funcNo %d", req.Header.FuncNo)
		return
	}

	funcName := s.mapFuncNo2Name[req.Header.FuncNo]
	funcName = GetFullFuncName(s.name, funcName)
	reqData, _ := ProtoBinder.GetRequest(funcName)
	respData, _ := ProtoBinder.GetResponse(funcName)

	defer func() {
		if reqData != nil {
			ProtoBinder.ReuseRequest(reqData, funcName)
		}

		if respData != nil {
			ProtoBinder.ReuseResponse(respData, funcName)
		}
	}()

	// reqData := interface{}(req.Payload[0])
	// respData := interface{}(nil)
	if reqData != nil && s.inter != nil {
		err = s.inter.OnUnmarshal(funcName, req.Payload[0], reqData)
		if err != nil {
			return
		}
	}

	// handle
	code, err = handler(reqData, respData, peerType, peerNo)
	if err != nil {
		return
	}

	// handle completion
	if s.inter != nil {
		returnData, err = s.inter.OnMarshal(funcName, respData)
	} else if respData != nil {
		returnData, ok = respData.([]byte)
		if !ok {
			err = ErrServWrongFormatRet
		}
	}
}

func (s *BaseService) writeResp(req *Request, peerType uint32, peerNo uint32, code uint16, returnData []byte) error {
	// response
	req.Header.Code = code
	headerData, err := req.Header.Marshal()
	if err != nil {
		return err
	}

	payload := make([]ByteArray, 0, 2)
	if len(returnData) == 0 {
		payload = append(payload, headerData)
	} else {
		payload = append(payload, headerData, returnData)
	}

	err = s.WritePack(peerType, peerNo, payload...)
	return err
}

func (s *BaseService) getHandler(funcNo uint16) (RpcHandler, bool) {
	var m RpcHandler = nil
	v, ok := s.mapFuncNo2Handler[funcNo]
	if ok {
		m = v.Interface().(RpcHandler)
	}

	return m, ok
}
