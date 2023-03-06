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

package module

import (
	"ecapture/user/config"
	"ecapture/user/event"
	"errors"
	"fmt"
	"github.com/cilium/ebpf"
	manager "github.com/gojue/ebpfmanager"
	"golang.org/x/sys/unix"
	"math"
	"net"
	"strings"
)

type NetEventMetadata struct {
	TimeStamp   uint64   `json:"timeStamp"`
	HostTid     uint32   `json:"hostTid"`
	ProcessName [16]byte `json:"processName"`
}

func (this *MOpenSSLProbe) setupManagersTC() error {
	var ifname, binaryPath, sslVersion string

	ifname = this.conf.(*config.OpensslConfig).Ifname
	this.ifName = ifname
	interf, err := net.InterfaceByName(this.ifName)
	if err != nil {
		return err
	}

	// loopback devices are special, some tc probes should be skipped
	isNetIfaceLo := interf.Flags&net.FlagLoopback == net.FlagLoopback
	skipLoopback := true // TODO: detect loopback devices via aquasecrity/tracee/pkg/ebpf/probes/probe.go line 322
	if isNetIfaceLo && skipLoopback {
		return fmt.Errorf("%s\t%s is a loopback interface, skip it", this.Name(), this.ifName)
	}
	this.ifIdex = interf.Index

	sslVersion = this.conf.(*config.OpensslConfig).SslVersion
	sslVersion = strings.ToLower(sslVersion)
	switch this.conf.(*config.OpensslConfig).ElfType {
	case config.ElfTypeBin:
		binaryPath = this.conf.(*config.OpensslConfig).Curlpath
	case config.ElfTypeSo:
		binaryPath = this.conf.(*config.OpensslConfig).Openssl
		err := this.getSslBpfFile(binaryPath, sslVersion)
		if err != nil {
			return err
		}
	default:
		//如果没找到
		binaryPath = "/lib/x86_64-linux-gnu/libssl.so.1.1"
		err := this.getSslBpfFile(binaryPath, sslVersion)
		if err != nil {
			return err
		}
	}

	this.logger.Printf("%s\tHOOK type:%d, binrayPath:%s\n", this.Name(), this.conf.(*config.OpensslConfig).ElfType, binaryPath)
	this.logger.Printf("%s\tIfname:%s, Ifindex:%d,  Port:%d, Pcapng filepath:%s\n", this.Name(), this.ifName, this.ifIdex, this.conf.(*config.OpensslConfig).Port, this.pcapngFilename)
	this.logger.Printf("%s\tHook masterKey function:%s\n", this.Name(), this.masterHookFunc)

	// create pcapng writer
	netIfs, err := net.Interfaces()
	if err != nil {
		return err
	}

	err = this.createPcapng(netIfs)
	if err != nil {
		return err
	}

	this.bpfManager = &manager.Manager{
		Probes: []*manager.Probe{
			// customize deleteed TC filter
			// tc filter del dev eth0 ingress
			// tc filter del dev eth0 egress
			// loopback devices are special, some tc probes should be skipped
			// TODO: detect loopback devices via aquasecrity/tracee/pkg/ebpf/probes/probe.go line 322
			// isNetIfaceLo := netIface.Flags&net.FlagLoopback == net.FlagLoopback
			//	if isNetIfaceLo && p.skipLoopback {
			//		return nil
			//	}
			{
				Section:          "classifier/egress",
				EbpfFuncName:     "egress_cls_func",
				Ifname:           this.ifName,
				NetworkDirection: manager.Egress,
			},
			{
				Section:          "classifier/ingress",
				EbpfFuncName:     "ingress_cls_func",
				Ifname:           this.ifName,
				NetworkDirection: manager.Ingress,
			},
			// --------------------------------------------------

			// openssl masterkey
			{
				Section:          "uprobe/SSL_write_key",
				EbpfFuncName:     "probe_ssl_master_key",
				AttachToFuncName: this.masterHookFunc, // SSL_do_handshake or SSL_write
				BinaryPath:       binaryPath,
				UID:              "uprobe_ssl_master_key",
			},
		},

		Maps: []*manager.Map{
			{
				Name: "mastersecret_events",
			},
			{
				Name: "skb_events",
			},
		},
	}

	this.bpfManagerOptions = manager.Options{
		DefaultKProbeMaxActive: 512,

		VerifierOptions: ebpf.CollectionOptions{
			Programs: ebpf.ProgramOptions{
				LogSize: 2097152,
			},
		},

		RLimit: &unix.Rlimit{
			Cur: math.MaxUint64,
			Max: math.MaxUint64,
		},
	}

	if this.conf.EnableGlobalVar() {
		// 填充 RewriteContants 对应map
		this.bpfManagerOptions.ConstantEditors = this.constantEditor()
	}
	return nil
}

func (this *MOpenSSLProbe) initDecodeFunTC() error {
	//SkbEventsMap 与解码函数映射
	SkbEventsMap, found, err := this.bpfManager.GetMap("skb_events")
	if err != nil {
		return err
	}
	if !found {
		return errors.New("cant found map:skb_events")
	}
	this.eventMaps = append(this.eventMaps, SkbEventsMap)
	sslEvent := &event.TcSkbEvent{}
	//sslEvent.SetModule(this)
	this.eventFuncMaps[SkbEventsMap] = sslEvent

	MasterkeyEventsMap, found, err := this.bpfManager.GetMap("mastersecret_events")
	if err != nil {
		return err
	}
	if !found {
		return errors.New("cant found map:mastersecret_events")
	}
	this.eventMaps = append(this.eventMaps, MasterkeyEventsMap)

	var masterkeyEvent event.IEventStruct

	if this.isBoringSSL {
		masterkeyEvent = &event.MasterSecretBSSLEvent{}
	} else {
		masterkeyEvent = &event.MasterSecretEvent{}
	}

	//masterkeyEvent.SetModule(this)
	this.eventFuncMaps[MasterkeyEventsMap] = masterkeyEvent
	return nil
}
