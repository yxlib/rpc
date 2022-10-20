// Copyright 2022 Guan Jianchang. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import (
	"encoding/json"
)

// type JsonServInterceptor struct {
// }

// func (i *JsonServInterceptor) OnPreHandle(funcName string, payload []byte) (interface{}, interface{}, error) {
// 	reqData, err := ProtoBinder.GetRequest(funcName)
// 	if err == nil {
// 		err = json.Unmarshal(payload, reqData)
// 		if err != nil {
// 			return nil, nil, err
// 		}
// 	}

// 	respData, _ := ProtoBinder.GetResponse(funcName)
// 	return reqData, respData, nil
// }

// func (i *JsonServInterceptor) OnHandleCompletion(code uint16, funcName string, req interface{}, resp interface{}) ([]byte, error) {
// 	if req != nil {
// 		ProtoBinder.ReuseRequest(req, funcName)
// 	}

// 	payload := []byte(nil)
// 	err := error(nil)
// 	if resp != nil {
// 		payload, err = json.Marshal(resp)
// 	}

// 	if resp != nil {
// 		ProtoBinder.ReuseResponse(resp, funcName)
// 	}

// 	return payload, err
// }

type Interceptor interface {
	// Marshal the response to []byte.
	// @param funcName, the full func name.
	// @param v, the response object
	// @return []byte, bytes after marshal.
	// @return error, error.
	OnMarshal(funcName string, obj interface{}) ([]byte, error)

	// Unmarshal the request payload to interface{}.
	// @param funcName, the full func name.
	// @param payload, the request payload
	// @return interface{}, an object unmarshal from payload.
	// @return error, error.
	OnUnmarshal(funcName string, data []byte, obj interface{}) error
}

type JsonInterceptor struct {
}

func (i *JsonInterceptor) OnMarshal(funcName string, reqObj interface{}) ([]byte, error) {
	payload, err := json.Marshal(reqObj)
	return payload, err
}

func (i *JsonInterceptor) OnUnmarshal(funcName string, respData []byte, respObj interface{}) error {
	return json.Unmarshal(respData, respObj)
}
