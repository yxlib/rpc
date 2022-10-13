// Copyright 2022 Guan Jianchang. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

type FuncConf struct {
	FuncName string
	FuncNo   uint16
	Request  string `json:"req"`
	Response string `json:"resp"`
}

type CliFuncConf struct {
	*FuncConf
	Marshaler   string `json:"marshaler"`
	Unmarshaler string `json:"unmarshaler"`
}

type PeerConf struct {
	PeerType         uint32                  `json:"type"`
	Timeout          uint32                  `json:"timeout_sec"`
	MapFuncName2Info map[string]*CliFuncConf `json:"func"`
}

type CliConf struct {
	MapMark2Peer map[string]*PeerConf `json:"srv_list"`
}

// var CliCfgInst *CliConf = &CliConf{}

//=========================
//      SrvFuncConf
//=========================
type SrvFuncConf struct {
	*FuncConf
	Handler string `json:"handler"`
}

type ServiceConf struct {
	Net              string                  `json:"net"`
	Name             string                  `json:"name"`
	MapFuncName2Info map[string]*SrvFuncConf `json:"func"`
}

type SrvConf struct {
	Services []*ServiceConf `json:"services"`
}

var SrvConfInst *SrvConf = &SrvConf{}
