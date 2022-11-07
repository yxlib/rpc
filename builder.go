// Copyright 2022 Guan Jianchang. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import (
	"github.com/yxlib/yx"
)

type builder struct {
	logger *yx.Logger
}

var Builder = &builder{
	logger: yx.NewLogger("rpc.Builder"),
}

// func (b *builder) BuildSrv(srvCfg *SrvConf) {
// 	objFactory := Server.GetObjectFactory()

// 	for _, cfg := range srvCfg.Services {
// 		obj, err := objFactory.CreateObject(cfg.Service)
// 		if err != nil {
// 			b.logger.W("service object not register: ", cfg.Service)
// 			continue
// 		}

// 		srv, ok := obj.(Service)
// 		if !ok {
// 			b.logger.W("not Service object: ", cfg.Service)
// 			continue
// 		}

// 		srv.SetName(cfg.Name)
// 		v := reflect.ValueOf(srv)

// 		funcNo := uint16(RPC_FUNC_NO_FUNC_LIST)
// 		for funcName, funcCfg := range cfg.MapFuncName2Info {
// 			if funcName == RPC_FUNC_NAME_FUNC_LIST {
// 				funcCfg.FuncNo = RPC_FUNC_NO_FUNC_LIST
// 			} else {
// 				funcNo++
// 				funcCfg.FuncNo = funcNo
// 			}

// 			// proto
// 			fullFuncName := GetFullFuncName(cfg.Name, funcName)
// 			err := ProtoBinder.BindProto(fullFuncName, funcCfg.Request, funcCfg.Response)
// 			if err != nil {
// 				b.logger.W("not support func ", fullFuncName)
// 				continue
// 			}

// 			// handler
// 			m := v.MethodByName(funcCfg.Handler)
// 			err = srv.AddReflectProcessor(m, funcCfg.FuncNo, funcName)
// 			if err != nil {
// 				b.logger.E("AddReflectProcessor err: ", err)
// 				b.logger.W("not support func ", fullFuncName)
// 				continue
// 			}
// 		}

// 		Server.AddService(srv)
// 	}
// }

// build rpc client.
// @param cli, dest rpc client.
// @param cfg, the client config.
// func (b *builder) BuildCli(cli Client, cfg *CliConf) {
// 	for mark, peerCfg := range cfg.MapMark2Peer {
// 		cli.AddPeerType(peerCfg.PeerType, mark)
// 		b.buildPeerProto(cli, mark, peerCfg)
// 	}
// }

// func (b *builder) buildPeerProto(cli Client, mark string, peerCfg *PeerConf) {
// 	v := reflect.ValueOf(cli)

// 	for funcName, cfg := range peerCfg.MapFuncName2Info {
// 		// proto
// 		fullFuncName := GetFullFuncName(mark, funcName)
// 		err := ProtoBinder.BindProto(fullFuncName, cfg.Request, cfg.Response)
// 		if err != nil {
// 			b.logger.W("not support func ", funcName)
// 			continue
// 		}

// 		// marshaler
// 		marshaler := v.MethodByName(cfg.Marshaler)
// 		unmarshaler := v.MethodByName(cfg.Unmarshaler)
// 		err = cli.AddReflectProcessor(marshaler, unmarshaler, mark, funcName)
// 		if err != nil {
// 			b.logger.E("AddReflectProcessor err: ", err)
// 			b.logger.W("not support mark ", mark)
// 			continue
// 		}
// 	}
// }
