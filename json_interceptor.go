// Copyright 2022 Guan Jianchang. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import (
	"encoding/json"
)

type JsonInterceptor struct {
}

func (i *JsonInterceptor) OnPreHandle(funcName string, payload []byte) (interface{}, interface{}, error) {
	reqData, err := ProtoBinder.GetRequest(funcName)
	if err == nil {
		err = json.Unmarshal(payload, reqData)
		if err != nil {
			return nil, nil, err
		}
	}

	respData, _ := ProtoBinder.GetResponse(funcName)
	return reqData, respData, nil
}

func (i *JsonInterceptor) OnHandleCompletion(funcName string, req interface{}, resp interface{}) ([]byte, error) {
	if req != nil {
		ProtoBinder.ReuseRequest(req, funcName)
	}

	payload := []byte(nil)
	err := error(nil)
	if resp != nil {
		payload, err = json.Marshal(resp)
	}

	if resp != nil {
		ProtoBinder.ReuseResponse(resp, funcName)
	}

	return payload, err
}
