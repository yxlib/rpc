// Copyright 2022 Guan Jianchang. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import (
	"os"
	"strings"

	"github.com/yxlib/yx"
)

func GenSrvRegisterFile(cfgPath string, regFilePath string, regPackName string) {
	yx.LoadJsonConf(SrvConfInst, cfgPath, nil)
	GenSrvRegisterFileByCfg(SrvConfInst, regFilePath, regPackName)
}

// Generate the service register file.
// @param cfgPath, the config path.
// @param regFilePath, the output register file.
// @param regPackName, the package name of the file.
func GenSrvRegisterFileByCfg(srvCfg *SrvConf, regFilePath string, regPackName string) {
	f, err := os.OpenFile(regFilePath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return
	}

	defer f.Close()

	writePackage(f, regPackName)

	funcConfs := make([]*FuncConf, 0, len(srvCfg.MapFuncName2Info))
	for funcName, cfg := range srvCfg.MapFuncName2Info {
		cfg.FuncName = funcName
		funcConfs = append(funcConfs, cfg.FuncConf)
	}
	writeImport(funcConfs, regPackName, f)
	writeRegFunc(funcConfs, f)
}

func GenCliRegisterFile(cfgPath string, regFilePath string, regPackName string) {
	var cliCfg *CliConf = &CliConf{}
	yx.LoadJsonConf(cliCfg, cfgPath, nil)
	GenCliRegisterFileByCfg(cliCfg, regFilePath, regPackName)
}

// Generate the service register file.
// @param cfgPath, the config path.
// @param regFilePath, the output register file.
// @param regPackName, the package name of the file.
func GenCliRegisterFileByCfg(cliCfg *CliConf, regFilePath string, regPackName string) {
	f, err := os.OpenFile(regFilePath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return
	}

	defer f.Close()

	writePackage(f, regPackName)

	funcConfs := make([]*FuncConf, 0, len(cliCfg.MapMark2Peer))
	for _, peerCfg := range cliCfg.MapMark2Peer {
		for funcName, cfg := range peerCfg.MapFuncName2Info {
			cfg.FuncName = funcName
			funcConfs = append(funcConfs, cfg.FuncConf)
		}
	}
	writeImport(funcConfs, regPackName, f)
	writeRegFunc(funcConfs, f)
}

func writePackage(f *os.File, regPackName string) {
	f.WriteString("// This File auto generate by tool.\n")
	f.WriteString("// Please do not modify.\n")
	f.WriteString("// See rpc.GenSrvRegisterFileByCfg().\n\n")
	f.WriteString("package " + regPackName + "\n\n")
}

func writeImport(funcConfs []*FuncConf, regPackName string, f *os.File) {
	f.WriteString("import (\n")

	packSet := yx.NewSet(yx.SET_TYPE_OBJ)
	packSet.Add("github.com/yxlib/rpc")

	for _, cfg := range funcConfs {
		addProtoPackage(cfg.Request, regPackName, packSet)
		addProtoPackage(cfg.Response, regPackName, packSet)
	}

	elements := packSet.GetElements()
	for _, packName := range elements {
		f.WriteString("    \"" + packName.(string) + "\"\n")
	}

	f.WriteString(")\n\n")
}

func addProtoPackage(protoCfg string, regPackName string, packSet *yx.Set) {
	if protoCfg == "" {
		return
	}

	idx := strings.LastIndex(protoCfg, ".")
	packName := protoCfg[:idx]

	idx = strings.LastIndex(packName, "/")
	if idx < 0 {
		if packName != regPackName {
			packSet.Add(packName)
		}

		return
	}

	if packName[idx+1:] != regPackName {
		packSet.Add(packName)
	}
}

func writeRegFunc(funcConfs []*FuncConf, f *os.File) {
	f.WriteString("// Auto generate by tool.\n")
	f.WriteString("func RegisterRpcFuncs() {\n")

	for _, cfg := range funcConfs {
		f.WriteString("    // " + cfg.FuncName + "\n")

		// request
		if cfg.Request == "" {
			println("[Error]    Request can not be empty!!!    ========> ", cfg.FuncName)
			break
		}

		reqStr := cfg.Request
		idx := strings.LastIndex(reqStr, "/")
		if idx >= 0 {
			reqStr = reqStr[idx+1:]
		}
		f.WriteString("    rpc.ProtoBinder.RegisterProto(&" + reqStr + "{})\n")

		// response
		if cfg.Response != "" {
			respStr := cfg.Response
			idx := strings.LastIndex(respStr, "/")
			if idx >= 0 {
				respStr = respStr[idx+1:]
			}

			f.WriteString("    rpc.ProtoBinder.RegisterProto(&" + respStr + "{})\n")
		}

		f.WriteString("\n")
	}

	f.WriteString("}")
}
