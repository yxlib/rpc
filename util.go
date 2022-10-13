// Copyright 2022 Guan Jianchang. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

func GetPeerId(peerType uint32, peerNo uint32) uint32 {
	return peerType<<16 | peerNo
}

func GetFullFuncName(serviceName string, funcName string) string {
	return serviceName + "." + funcName
}

func CheckRpcMark(mark []byte, buff []byte) bool {
	if len(buff) < len(mark) {
		return false
	}

	for i, ch := range mark {
		if ch != buff[i] {
			return false
		}
	}

	return true
}
