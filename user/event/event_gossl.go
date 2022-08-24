// Copyright © 2022 Hengqi Chen
package event

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type inner struct {
	TimestampNS uint64
	Pid         uint32
	Tid         uint32
	Len         int32
	Comm        [16]byte
	Data        [4096]byte
}

type GoSSLEvent struct {
	inner
}

func (e *GoSSLEvent) Decode(payload []byte) error {
	r := bytes.NewBuffer(payload)
	return binary.Read(r, binary.LittleEndian, &e.inner)
}

func (e *GoSSLEvent) String() string {
	s := fmt.Sprintf("PID: %d, Comm: %s, TID: %d, Payload: %s\n", e.Pid, string(e.Comm[:]), e.Tid, string(e.Data[:e.Len]))
	return s
}

func (e *GoSSLEvent) StringHex() string {
	perfix := COLORGREEN
	b := dumpByteSlice(e.Data[:e.Len], perfix)
	b.WriteString(COLORRESET)
	s := fmt.Sprintf("PID: %d, Comm: %s, TID: %d, Payload: %s\n", e.Pid, string(e.Comm[:]), e.Tid, b.String())
	return s
}

func (e *GoSSLEvent) Clone() IEventStruct {
	return &GoSSLEvent{}
}

func (e *GoSSLEvent) EventType() EventType {
	return EventTypeOutput
}

func (this *GoSSLEvent) GetUUID() string {
	return fmt.Sprintf("%d_%d_%s", this.Pid, this.Tid, this.Comm)
}

func (this *GoSSLEvent) Payload() []byte {
	return this.Data[:this.Len]
}

func (this *GoSSLEvent) PayloadLen() int {
	return int(this.Len)
}
