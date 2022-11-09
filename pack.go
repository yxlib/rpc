// Copyright 2022 Guan Jianchang. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import (
	"bytes"
	"encoding/binary"
	"errors"

	"github.com/yxlib/yx"
)

var (
	ErrPackFrameIsNil      = errors.New("frame is nil")
	ErrPackMarkCheckFailed = errors.New("rpc mark check failed")
	ErrPackTooSmall        = errors.New("pack header data not enough")
)

const (
	RPC_SERIAL_NO_LEN = 2
	RPC_FUNC_NO_LEN   = 2
	RPC_CODE_LEN      = 4
)

const (
	RES_CODE_SUCC    int32 = 0
	RES_CODE_SYS_ERR int32 = 1
)

type PackHeader struct {
	Mark     string
	SerialNo uint16
	FuncNo   uint16
	Code     int32

	ec *yx.ErrCatcher
}

func NewPackHeader(mark string, serialNo uint16, funcNo uint16) *PackHeader {
	return &PackHeader{
		Mark:     mark,
		SerialNo: serialNo,
		FuncNo:   funcNo,
		Code:     RES_CODE_SUCC,
		ec:       yx.NewErrCatcher("rpc.PackHeader"),
	}
}

func (p *PackHeader) GetHeaderLen() int {
	return len([]byte(p.Mark)) + RPC_SERIAL_NO_LEN + RPC_FUNC_NO_LEN + RPC_CODE_LEN
}

func (p *PackHeader) Marshal() ([]byte, error) {
	var err error = nil
	defer p.ec.DeferThrow("Marshal", &err)

	headerLen := p.GetHeaderLen()
	buff := make([]byte, 0, headerLen)
	buffWrap := bytes.NewBuffer(buff)

	// ====== head
	// mark
	err = binary.Write(buffWrap, binary.BigEndian, []byte(p.Mark))
	if err != nil {
		return nil, err
	}

	// copy(buff, p.Mark)

	// tmpBuff := make([]byte, 0, RPC_SERIAL_NO_LEN+RPC_FUNC_NO_LEN)
	// buffWrap := bytes.NewBuffer(tmpBuff)

	// serial No.
	err = binary.Write(buffWrap, binary.BigEndian, p.SerialNo)
	if err != nil {
		return nil, err
	}

	// func No.
	err = binary.Write(buffWrap, binary.BigEndian, p.FuncNo)
	if err != nil {
		return nil, err
	}

	// code
	err = binary.Write(buffWrap, binary.BigEndian, p.Code)
	if err != nil {
		return nil, err
	}

	return buffWrap.Bytes(), nil

	// offset := len(p.Mark)
	// copy(buff[offset:], buffWrap.Bytes())

	// // ====== payload
	// offset += RPC_SERIAL_NO_LEN + RPC_FUNC_NO_LEN
	// if len(p.Payload) > 0 {
	// 	copy(buff[offset:], p.Payload)
	// }

	// return buff, nil
}

func (p *PackHeader) Unmarshal(buff []byte) error {
	var err error = nil
	defer p.ec.DeferThrow("Unmarshal", &err)

	// ====== head
	// mark
	if !CheckRpcMark([]byte(p.Mark), buff) {
		err = ErrPackMarkCheckFailed
		return err
	}

	// markLen := len([]byte(p.Mark))
	// if len(buff) < markLen+RPC_SERIAL_NO_LEN+RPC_FUNC_NO_LEN {
	if len(buff) < p.GetHeaderLen() {
		err = ErrPackTooSmall
		return err
	}

	offset := len([]byte(p.Mark))
	buffWrap := bytes.NewBuffer(buff[offset:])

	// serial No.
	err = binary.Read(buffWrap, binary.BigEndian, &p.SerialNo)
	if err != nil {
		return err
	}

	// func No.
	err = binary.Read(buffWrap, binary.BigEndian, &p.FuncNo)
	if err != nil {
		return err
	}

	// code
	err = binary.Read(buffWrap, binary.BigEndian, &p.Code)
	if err != nil {
		return err
	}

	// ====== payload
	// offset += RPC_SERIAL_NO_LEN + RPC_FUNC_NO_LEN
	// if len(buff) > offset {
	// 	p.Payload = buff[offset:]
	// }

	return nil
}

//========================
//       Pack
//========================
type PackFrame = []byte

type Pack struct {
	Header  *PackHeader
	Payload []PackFrame

	ec *yx.ErrCatcher
}

func NewPack(h *PackHeader) *Pack {
	return &Pack{
		Header:  h,
		Payload: make([]PackFrame, 0),

		ec: yx.NewErrCatcher("rpc.Pack"),
	}
}

func NewSingleFramePack(h *PackHeader, payload []byte) *Pack {
	p := NewPack(h)
	p.AddFrame(payload)

	return p
}

func (p *Pack) AddFrame(frame []byte) error {
	if nil == frame {
		return p.ec.Throw("AddFrame", ErrPackFrameIsNil)
	}

	p.Payload = append(p.Payload, frame)
	return nil
}

func (p *Pack) AddFrames(frames ...[]byte) error {
	if nil == frames {
		return p.ec.Throw("AddFrames", ErrPackFrameIsNil)
	}

	p.Payload = append(p.Payload, frames...)
	return nil
}

//========================
//       Request
//========================
type Request struct {
	*Pack
	respCode    int32
	respPayload []byte
	evt         *yx.Event
}

func NewRequest(h *PackHeader) *Request {
	return &Request{
		Pack:        NewPack(h),
		respCode:    RES_CODE_SUCC,
		respPayload: nil,
		evt:         yx.NewEvent(),
	}
}

func NewSingleFrameRequest(h *PackHeader, payload []byte) *Request {
	r := NewRequest(h)
	r.AddFrame(payload)

	return r
}

func (r *Request) Wait() error {
	return r.evt.Wait()
}

func (r *Request) WaitUntilTimeout(timeoutSec uint32) error {
	return r.evt.WaitUntilTimeout(timeoutSec)
}

func (r *Request) Signal() error {
	return r.evt.Send()
}

func (r *Request) SetResponse(code int32, payload []byte) {
	r.respCode = code
	r.respPayload = payload
}

func (r *Request) GetResponse() (int32, []byte) {
	return r.respCode, r.respPayload
}

func (r *Request) GetResponseCode() int32 {
	return r.respCode
}

func (r *Request) GetResponseData() []byte {
	return r.respPayload
}

func (r *Request) Cancel() {
	// close(r.evt.C)
	r.evt.Close()
}

//========================
//       Response
//========================
type Response = Pack

func NewResponse(h *PackHeader) *Response {
	return NewPack(h)
}

// type Response struct {
// 	*Pack
// }

// func NewResponse(h PackHeader) *Response {
// 	return &Response{
// 		Pack: NewPack(h),
// 	}
// }

// func NewResponseByReq(req *Request) *Response {
// 	return &Response{
// 		Pack: NewPack(req.Mark, req.SerialNo, req.FuncNo, nil),
// 	}
// }

//========================
//    fetch func list
//========================
const RPC_FUNC_NO_FUNC_LIST = uint16(1)
const RPC_FUNC_NAME_FUNC_LIST = "FetchFuncList"

type FetchFuncListResp struct {
	MapFuncName2No map[string]uint16 `json:"func_mapper"`
}
