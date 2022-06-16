// Copyright 2022 Guan Jianchang. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import (
	"errors"
	"reflect"

	"github.com/yxlib/yx"
)

var (
	ErrProtoBindProtoExist    = errors.New("proto has exist")
	ErrProtoBindProtoNotExist = errors.New("proto not exist")
	ErrProtoBindReuseIsNil    = errors.New("reuse object is nil")
)

const (
	INIT_REUSE_COUNT = 10
	MAX_REUSE_COUNT  = 100
)

type protoBinder struct {
	mapFuncName2ReqName  map[string]string
	mapFuncName2RespName map[string]string
	factory              *yx.ObjectFactory
	ec                   *yx.ErrCatcher
}

var ProtoBinder = &protoBinder{
	mapFuncName2ReqName:  make(map[string]string),
	mapFuncName2RespName: make(map[string]string),
	factory:              yx.NewObjectFactory(),
	ec:                   yx.NewErrCatcher("protoBinder"),
}

// Register proto type.
// @param proto, the proto.
func (b *protoBinder) RegisterProto(proto interface{}) error {
	_, err := b.factory.RegisterObject(proto, nil, MAX_REUSE_COUNT)
	return b.ec.Throw("RegisterProto", err)
}

// Get the proto type by type name.
// @param name, the proto type name.
// @return reflect.Type, the reflect type of the proto.
// @return bool, true mean success, false mean failed.
func (b *protoBinder) GetProtoType(name string) (reflect.Type, bool) {
	return b.factory.GetReflectType(name)
}

// Bind protos.
// @param mod, the module of the service.
// @param cmd, the command of the service.
// @param reqProtoName, the request proto name.
// @param respProtoName, the response proto name.
// @return error, error.
func (b *protoBinder) BindProto(funcName string, reqProtoName string, respProtoName string) error {
	if reqProtoName != "" {
		_, ok := b.mapFuncName2ReqName[funcName]
		if ok {
			return b.ec.Throw("BindProto", ErrProtoBindProtoExist)
		}

		_, ok = b.factory.GetReflectType(reqProtoName)
		if !ok {
			return b.ec.Throw("BindProto", ErrProtoBindProtoNotExist)
		}
	}

	if respProtoName != "" {
		_, ok := b.mapFuncName2RespName[funcName]
		if ok {
			return b.ec.Throw("BindProto", ErrProtoBindProtoExist)
		}

		_, ok = b.factory.GetReflectType(respProtoName)
		if !ok {
			return b.ec.Throw("BindProto", ErrProtoBindProtoNotExist)
		}
	}

	if reqProtoName != "" {
		b.mapFuncName2ReqName[funcName] = reqProtoName
	}

	if respProtoName != "" {
		b.mapFuncName2RespName[funcName] = respProtoName
	}

	return nil
}

// Get the request reflect type.
// @param mod, the module of the service.
// @param cmd, the command of the service.
// @return reflect.Type, reflect type of the request.
// @return error, error.
// func (b *protoBinder) GetRequestType(funcName string) (reflect.Type, error) {
// 	name, ok := b.mapFuncName2ReqName[funcName]
// 	if !ok {
// 		return nil, ErrProtoBindProtoNotExist
// 	}

// 	return b.getReflectTypeByName(name)
// }

// // Get the response reflect type.
// // @param mod, the module of the service.
// // @param cmd, the command of the service.
// // @return reflect.Type, reflect type of the response.
// // @return error, error.
// func (b *protoBinder) GetResponseType(funcName string) (reflect.Type, error) {
// 	name, ok := b.mapFuncName2RespName[funcName]
// 	if !ok {
// 		return nil, ErrProtoBindProtoNotExist
// 	}

// 	return b.getReflectTypeByName(name)
// }

// Get an request object.
// @param mod, the module of the service.
// @param cmd, the command of the service.
// @return interface{}, the request object.
// @return error, error.
func (b *protoBinder) GetRequest(funcName string) (interface{}, error) {
	name, ok := b.mapFuncName2ReqName[funcName]
	if !ok {
		return nil, b.ec.Throw("GetRequest", ErrProtoBindProtoNotExist)
	}

	obj, err := b.factory.CreateObject(name)
	return obj, b.ec.Throw("GetRequest", err)
}

// Reuse an request object.
// @param v, the reuse request.
// @param mod, the module of the service.
// @param cmd, the command of the service.
// @return error, error.
func (b *protoBinder) ReuseRequest(v interface{}, funcName string) error {
	if v == nil {
		return b.ec.Throw("ReuseRequest", ErrProtoBindReuseIsNil)
	}

	name, ok := b.mapFuncName2ReqName[funcName]
	if !ok {
		return b.ec.Throw("ReuseRequest", ErrProtoBindProtoNotExist)
	}

	err := b.factory.ReuseObject(v, name)
	return b.ec.Throw("ReuseRequest", err)
}

// Get an response object.
// @param mod, the module of the service.
// @param cmd, the command of the service.
// @return interface{}, the response object.
// @return error, error.
func (b *protoBinder) GetResponse(funcName string) (interface{}, error) {
	name, ok := b.mapFuncName2RespName[funcName]
	if !ok {
		return nil, b.ec.Throw("GetResponse", ErrProtoBindProtoNotExist)
	}

	obj, err := b.factory.CreateObject(name)
	return obj, b.ec.Throw("GetResponse", err)
}

// Reuse an response object.
// @param v, the reuse response.
// @param mod, the module of the service.
// @param cmd, the command of the service.
// @return error, error.
func (b *protoBinder) ReuseResponse(v interface{}, funcName string) error {
	if v == nil {
		return b.ec.Throw("ReuseResponse", ErrProtoBindReuseIsNil)
	}

	name, ok := b.mapFuncName2RespName[funcName]
	if !ok {
		return b.ec.Throw("ReuseResponse", ErrProtoBindProtoNotExist)
	}

	err := b.factory.ReuseObject(v, name)
	return b.ec.Throw("ReuseResponse", err)
}

// func (b *protoBinder) getReflectTypeByName(name string) (reflect.Type, error) {
// 	objType, ok := b.factory.GetReflectType(name)
// 	if !ok {
// 		return nil, ErrProtoBindProtoNotExist
// 	}

// 	return objType, nil
// }
