// Copyright 2022 Guan Jianchang. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import (
	"errors"
	"sync"

	"github.com/yxlib/yx"
)

var (
	ErrSrvServNil       = errors.New("service is nil")
	ErrSrvServNameEmpty = errors.New("service name is nil or empty")
)

type server struct {
	mapName2Service map[string]Service
	lckServices     *sync.Mutex
	objFactory      *yx.ObjectFactory
	ec              *yx.ErrCatcher
	logger          *yx.Logger
}

var Server = &server{
	mapName2Service: make(map[string]Service),
	lckServices:     &sync.Mutex{},
	objFactory:      yx.NewObjectFactory(),
	ec:              yx.NewErrCatcher("rpc.Server"),
	logger:          yx.NewLogger("rpc.Server"),
}

func (s *server) SetObjectFactory(objFactory *yx.ObjectFactory) {
	s.objFactory = objFactory
}

func (s *server) GetObjectFactory() *yx.ObjectFactory {
	return s.objFactory
}

func (s *server) AddService(serv Service) error {
	if serv == nil {
		return ErrSrvServNil
	}

	name := serv.GetName()
	if len(name) == 0 {
		return ErrSrvServNameEmpty
	}

	s.lckServices.Lock()
	defer s.lckServices.Unlock()

	s.mapName2Service[name] = serv
	return nil
}

func (s *server) GetService(name string) (Service, bool) {
	s.lckServices.Lock()
	defer s.lckServices.Unlock()

	serv, ok := s.mapName2Service[name]
	return serv, ok
}

func (s *server) RemoveService(name string) {
	s.lckServices.Lock()
	defer s.lckServices.Unlock()

	_, ok := s.mapName2Service[name]
	if ok {
		delete(s.mapName2Service, name)
	}
}
