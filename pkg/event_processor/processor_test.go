package event_processor

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"testing"
	"time"
)

var (
	testFile = "testdata/all.json"
)

type SSLDataEventTmp struct {
	//Event_type   uint8    `json:"Event_type"`
	DataType  int64      `json:"DataType"`
	Timestamp uint64     `json:"Timestamp"`
	Pid       uint32     `json:"Pid"`
	Tid       uint32     `json:"Tid"`
	DataLen   int32      `json:"DataLen"`
	Comm      [16]byte   `json:"Comm"`
	Fd        uint32     `json:"Fd"`
	Version   int32      `json:"Version"`
	Data      [4096]byte `json:"Data"`
}

func TestEventProcessor_Serve(t *testing.T) {

	logger := log.Default()
	//var buf bytes.Buffer
	//logger.SetOutput(&buf)
	var output = "./output.log"
	f, e := os.Create(output)
	if e != nil {
		t.Fatal(e)
	}
	logger.SetOutput(f)
	ep := NewEventProcessor(f, true)
	go func() {
		var err error
		err = ep.Serve()
		if err != nil {
			log.Fatalf(err.Error())
		}
	}()
	content, err := os.ReadFile(testFile)
	if err != nil {
		//Do something
		log.Fatalf("open file error: %s, file:%s", err.Error(), testFile)
	}
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}
		var eventSSL SSLDataEventTmp
		err := json.Unmarshal([]byte(line), &eventSSL)
		if err != nil {
			t.Fatalf("json unmarshal error: %s, body:%v", err.Error(), line)
		}
		payloadFile := fmt.Sprintf("testdata/%d.bin", eventSSL.Timestamp)
		b, e := os.ReadFile(payloadFile)
		if e != nil {
			t.Fatalf("read payload file error: %s, file:%s", e.Error(), payloadFile)
		}
		copy(eventSSL.Data[:], b)
		ep.Write(&BaseEvent{DataLen: eventSSL.DataLen, Data: eventSSL.Data, DataType: eventSSL.DataType, Timestamp: eventSSL.Timestamp, Pid: eventSSL.Pid, Tid: eventSSL.Tid, Comm: eventSSL.Comm, Fd: eventSSL.Fd, Version: eventSSL.Version})
	}

	tick := time.NewTicker(time.Second * 10)
	<-tick.C

	err = ep.Close()
	logger.SetOutput(io.Discard)
	bufString, e := os.ReadFile(output)
	if e != nil {
		t.Fatal(e)
	}

	lines = strings.Split(string(bufString), "\n")
	ok := true
	h2Req := 0
	h2Resq := 0
	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), "dump") {
			t.Log(line)
			ok = false
		}
		// http2 parse error log
		if strings.Contains(line, "[http2 re") {
			t.Log(line)
			ok = false
		}
		// http2 parse count
		if strings.Contains(line, "HTTP2Response") {
			h2Resq += 1
		}
		if strings.Contains(line, "HTTP2Request") {
			h2Req += 1
		}
	}
	if err != nil {
		t.Fatalf("close error: %s", err.Error())
	}
	if h2Resq != 3 {
		t.Fatalf("some errors occurred: HTTP2Response lack")
	}
	if h2Req != 2 {
		t.Fatalf("some errors occurred: HTTP2Request lack")
	}
	if !ok {
		t.Fatalf("some errors occurred")
	}
	//t.Log(string(bufString))
	t.Log("done")
}
