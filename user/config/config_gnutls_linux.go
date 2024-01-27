//go:build !androidgki
// +build !androidgki

// Copyright 2022 CFC4N <cfc4n.cs@gmail.com>. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func (gc *GnutlsConfig) Check() error {

	var e error
	// 如果readline 配置，且存在，则直接返回。
	if gc.Gnutls != "" || len(strings.TrimSpace(gc.Gnutls)) > 0 {
		_, e = os.Stat(gc.Gnutls)
		if e != nil {
			return e
		}
		gc.ElfType = ElfTypeSo
		return nil
	}

	/*
		//如果配置 Curlpath的地址，判断文件是否存在，不存在则直接返回
		if gc.Curlpath != "" || len(strings.TrimSpace(gc.Curlpath)) > 0 {
			_, e := os.Stat(gc.Curlpath)
			if e != nil {
				return e
			}
		} else {
			//如果没配置，则直接指定。
			gc.Curlpath = "/usr/bin/wget"
		}

		soPath, e := getDynPathByElf(gc.Curlpath, "libgnutls.so")
		if e != nil {
			//gc.logger.Printf("get bash:%s dynamic library error:%v.\n", bash, e)
			_, e = os.Stat(X86BinaryPrefix)
			prefix := X86BinaryPrefix
			if e != nil {
				prefix = OthersBinaryPrefix
			}
			gc.Gnutls = filepath.Join(prefix, "libgnutls.so.30")
			gc.ElfType = ElfTypeSo
			_, e = os.Stat(gc.Gnutls)
			if e != nil {
				return e
			}
			return nil
		}
	*/
	var soLoadPaths = GetDynLibDirs()
	var sslPath string
	for _, soPath := range soLoadPaths {
		_, e = os.Stat(soPath)
		if e != nil {
			continue
		}
		//	libgnutls.so.30   default
		sslPath = filepath.Join(soPath, "libgnutls.so.30")
		_, e = os.Stat(sslPath)
		if e != nil {
			continue
		}
		gc.Gnutls = sslPath
		break
	}
	if gc.Gnutls == "" {
		return errors.New("cant found Gnutls so load path")
	}

	gc.ElfType = ElfTypeSo

	return nil
}
