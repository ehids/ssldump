//go:build !androidgki
// +build !androidgki

package config

import (
	"debug/elf"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	DEFAULT_IFNAME = "eth0"
)

func (this *OpensslConfig) checkOpenssl() error {
	soPath, e := getDynPathByElf(this.Curlpath, "libssl.so")
	if e != nil {
		//this.logger.Printf("get bash:%s dynamic library error:%v.\n", bash, e)
		_, e = os.Stat(X86_BINARY_PREFIX)
		prefix := X86_BINARY_PREFIX
		if e != nil {
			prefix = OTHERS_BINARY_PREFIX
		}

		//	ubuntu 21.04	libssl.so.1.1   default
		this.Openssl = filepath.Join(prefix, "libssl.so.1.1")
		this.ElfType = ELF_TYPE_SO
		_, e = os.Stat(this.Openssl)
		if e != nil {
			return e
		}
	} else {
		this.Openssl = soPath
		this.ElfType = ELF_TYPE_SO
	}
	return nil
}

func (this *OpensslConfig) checkConnect() error {
	var sharedObjects = []string{
		"libpthread.so.0", // ubuntu 21.04 server
		"libc.so.6",       // ubuntu 21.10 server
		"libc.so",         // Android
	}

	var funcName = ""
	var found bool
	for _, so := range sharedObjects {
		pthreadSoPath, e := getDynPathByElf(this.Curlpath, so)
		if e != nil {
			_, e = os.Stat(X86_BINARY_PREFIX)
			prefix := X86_BINARY_PREFIX
			if e != nil {
				prefix = OTHERS_BINARY_PREFIX
			}
			this.Pthread = filepath.Join(prefix, so)
			_, e = os.Stat(this.Pthread)
			if e != nil {
				// search all of sharedObjects
				//return e
				continue
			}
		} else {
			this.Pthread = pthreadSoPath
		}

		_elf, e := elf.Open(this.Pthread)
		if e != nil {
			//return e
			continue
		}

		dynamicSymbols, err := _elf.DynamicSymbols()
		if err != nil {
			//return err
			continue
		}

		//
		for _, sym := range dynamicSymbols {
			if sym.Name != "connect" {
				continue
			}
			//fmt.Printf("\tsize:%d,  name:%s,  offset:%d\n", sym.Size, sym.Name, 0)
			funcName = sym.Name
			found = true
			break
		}

		// if found
		if found && funcName != "" {
			break
		}
	}

	//如果没找到，则报错。
	if !found || funcName == "" {
		return errors.New(fmt.Sprintf("cant found 'connect' function to hook in files::%v", sharedObjects))
	}
	return nil
}

func (this *OpensslConfig) Check() error {

	var checkedOpenssl, checkedConnect bool
	// 如果readline 配置，且存在，则直接返回。
	if this.Openssl != "" || len(strings.TrimSpace(this.Openssl)) > 0 {
		_, e := os.Stat(this.Openssl)
		if e != nil {
			return e
		}
		this.ElfType = ELF_TYPE_SO
		checkedOpenssl = true
	}

	//如果配置 Curlpath的地址，判断文件是否存在，不存在则直接返回
	if this.Curlpath != "" || len(strings.TrimSpace(this.Curlpath)) > 0 {
		_, e := os.Stat(this.Curlpath)
		if e != nil {
			return e
		}
	} else {
		//如果没配置，则直接指定。
		this.Curlpath = "/usr/bin/curl"
	}

	if this.Pthread != "" || len(strings.TrimSpace(this.Pthread)) > 0 {
		_, e := os.Stat(this.Pthread)
		if e != nil {
			return e
		}
		checkedConnect = true
	}

	if this.Ifname == "" || len(strings.TrimSpace(this.Ifname)) == 0 {
		this.Ifname = DEFAULT_IFNAME
	}

	if checkedConnect && checkedOpenssl {
		return nil
	}

	if this.NoSearch {
		return errors.New("NoSearch requires specifying lib path")
	}

	if !checkedOpenssl {
		e := this.checkOpenssl()
		if e != nil {
			return e
		}
	}

	if !checkedConnect {
		return this.checkConnect()
	}
	return nil
}
