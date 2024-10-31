// Author: yuweizzz <yuwei764969238@gmail.com>.
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
	"bytes"
	"errors"
	"fmt"
	"math"
	"os"
	"path"

	"github.com/cilium/ebpf"
	manager "github.com/gojue/ebpfmanager"
	"github.com/gojue/ecapture/user/config"
	"github.com/gojue/ecapture/user/event"
	"golang.org/x/sys/unix"
)

// gnutls_mac_algorithm_t: https://github.com/gnutls/gnutls/blob/master/lib/includes/gnutls/gnutls.h.in#L365
// gnutls_protocol_t: https://github.com/gnutls/gnutls/blob/master/lib/includes/gnutls/gnutls.h.in#L822

const (
	_                             = iota
	GNUTLS_SSL3, GNUTLS_DTLS1_0   = iota, iota + 200
	GNUTLS_TLS1_0, GNUTLS_DTLS1_2 = iota, iota + 200
	GNUTLS_TLS1_1                 = iota
	GNUTLS_TLS1_2
	GNUTLS_TLS1_3
	GNUTLS_MAC_SHA256
	GNUTLS_MAC_SHA384
)

var GnutlsVersionToString = map[int32]string{
	GNUTLS_SSL3:    "GNUTLS_SSL3",
	GNUTLS_TLS1_0:  "GNUTLS_TLS1_0",
	GNUTLS_TLS1_1:  "GNUTLS_TLS1_1",
	GNUTLS_TLS1_2:  "GNUTLS_TLS1_2",
	GNUTLS_TLS1_3:  "GNUTLS_TLS1_3",
	GNUTLS_DTLS1_0: "GNUTLS_DTLS1_0",
	GNUTLS_DTLS1_2: "GNUTLS_DTLS1_2",
}

func (g *MGnutlsProbe) setupManagersKeylog() error {
	var binaryPath string
	switch g.conf.(*config.GnutlsConfig).ElfType {
	case config.ElfTypeSo:
		binaryPath = g.conf.(*config.GnutlsConfig).Gnutls
	default:
		//如果没找到  "/lib/x86_64-linux-gnu/libgnutls.so.30"
		binaryPath = path.Join(defaultSoPath, "libgnutls.so.30")
	}
	_, err := os.Stat(binaryPath)
	if err != nil {
		return err
	}

	g.logger.Info().Str("binaryPath", binaryPath).Uint8("elfType", g.conf.(*config.GnutlsConfig).ElfType).Msg("gnutls binary path")
	g.bpfManager = &manager.Manager{
		Probes: []*manager.Probe{
			{
				Section:          "uprobe/gnutls_handshake",
				EbpfFuncName:     "uprobe_gnutls_master_key",
				AttachToFuncName: "gnutls_handshake",
				BinaryPath:       binaryPath,
			},
			{
				Section:          "uretprobe/gnutls_handshake",
				EbpfFuncName:     "uretprobe_gnutls_master_key",
				AttachToFuncName: "gnutls_handshake",
				BinaryPath:       binaryPath,
			},
		},

		Maps: []*manager.Map{
			{
				Name: "mastersecret_gnutls_events",
			},
		},
	}

	g.bpfManagerOptions = manager.Options{
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

	if g.conf.EnableGlobalVar() {
		// 填充 RewriteContants 对应map
		g.bpfManagerOptions.ConstantEditors = g.constantEditor()
	}
	return nil
}

func (m *MGnutlsProbe) initDecodeFunKeylog() error {
	MasterkeyEventsMap, found, err := m.bpfManager.GetMap("mastersecret_gnutls_events")
	if err != nil {
		return err
	}
	if !found {
		return errors.New("cant found map: mastersecret_gnutls_events")
	}
	m.eventMaps = append(m.eventMaps, MasterkeyEventsMap)

	var masterkeyEvent event.IEventStruct

	masterkeyEvent = &event.MasterSecretGnutlsEvent{}

	m.eventFuncMaps[MasterkeyEventsMap] = masterkeyEvent
	return nil
}

func (g *MGnutlsProbe) saveMasterSecret(secretEvent *event.MasterSecretGnutlsEvent) {
	clientRandomHex := fmt.Sprintf("%02x", secretEvent.ClientRandom[0:event.GnutlsRandomSize])
	k := fmt.Sprintf("%s-%s", "CLIENT_RANDOM", clientRandomHex)

	_, f := g.masterKeys[k]
	if f {
		// 已存在该随机数的masterSecret，不需要重复写入
		return
	}

	g.masterKeys[k] = true
	buf := bytes.NewBuffer(nil)
	switch secretEvent.Version {
	// tls1.3
	case GNUTLS_TLS1_3:
		var length int
		switch secretEvent.CipherId {
		case GNUTLS_MAC_SHA384:
			length = 48
		case GNUTLS_MAC_SHA256:
			fallthrough
		default:
			// default MAC output length: 32 -- SHA256
			length = 32
		}
		chSecret := secretEvent.ClientHandshakeSecret[0:length]
		buf.WriteString(fmt.Sprintf("%s %s %02x\n", "CLIENT_HANDSHAKE_TRAFFIC_SECRET", clientRandomHex, chSecret))
		shSecret := secretEvent.ServerHandshakeSecret[0:length]
		buf.WriteString(fmt.Sprintf("%s %s %02x\n", "SERVER_HANDSHAKE_TRAFFIC_SECRET", clientRandomHex, shSecret))
		emSecret := secretEvent.ExporterMasterSecret[0:length]
		buf.WriteString(fmt.Sprintf("%s %s %02x\n", "EXPORTER_SECRET", clientRandomHex, emSecret))
		ctSecret := secretEvent.ClientTrafficSecret[0:length]
		buf.WriteString(fmt.Sprintf("%s %s %02x\n", "CLIENT_TRAFFIC_SECRET_0", clientRandomHex, ctSecret))
		stSecret := secretEvent.ServerTrafficSecret[0:length]
		buf.WriteString(fmt.Sprintf("%s %s %02x\n", "SERVER_TRAFFIC_SECRET_0", clientRandomHex, stSecret))
	// tls1.2
	case GNUTLS_TLS1_2:
		fallthrough
	// tls1.1, tls1.0, ssl3.0, dtls 1.0 and dtls 1.2
	default:
		masterSecret := secretEvent.MasterSecret[0:event.GnutlsMasterSize]
		buf.WriteString(fmt.Sprintf("%s %s %02x\n", "CLIENT_RANDOM", clientRandomHex, masterSecret))
	}

	var e error
	switch g.eBPFProgramType {
	case TlsCaptureModelTypeKeylog:
		_, e = g.keylogger.WriteString(buf.String())
		if e != nil {
			g.logger.Warn().Err(e).Str("ClientRandom", k).Msg("save masterSecrets to keylog error")
			return
		}
		g.logger.Info().Str("TlsVersion", GnutlsVersionToString[secretEvent.Version]).Str("ClientRandom", clientRandomHex).Msg("CLIENT_RANDOM save success")
	case TlsCaptureModelTypePcap:
		e = g.savePcapngSslKeyLog(buf.Bytes())
		if e != nil {
			g.logger.Warn().Err(e).Str("ClientRandom", k).Msg("save masterSecrets to pcapNG error")
			return
		}
		g.logger.Info().Str("TlsVersion", GnutlsVersionToString[secretEvent.Version]).Str("ClientRandom", clientRandomHex).Str("eBPFProgramType", g.eBPFProgramType.String()).Msg("CLIENT_RANDOM save success")
	default:
		g.logger.Warn().Uint8("eBPFProgramType", uint8(g.eBPFProgramType)).Str("ClientRandom", clientRandomHex).Msg("unhandled default case with eBPF Program type")
	}
}
