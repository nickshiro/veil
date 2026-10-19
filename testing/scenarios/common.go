package scenarios

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"runtime"
	"testing"
	"time"

	"github.com/nickshiro/veil/app/dispatcher"
	"github.com/nickshiro/veil/app/proxyman"
	"github.com/nickshiro/veil/common"
	"github.com/nickshiro/veil/common/errors"
	"github.com/nickshiro/veil/common/log"
	"github.com/nickshiro/veil/common/net"
	"github.com/nickshiro/veil/common/serial"
	core "github.com/nickshiro/veil/core"
	_ "github.com/nickshiro/veil/main/distro/all"
)

func xor(b []byte) []byte {
	r := make([]byte, len(b))
	for i, v := range b {
		r[i] = v ^ 'c'
	}
	return r
}

func readFrom2(conn net.Conn, timeout time.Duration, length int) ([]byte, error) {
	b := make([]byte, length)
	deadline := time.Now().Add(timeout)
	conn.SetReadDeadline(deadline)
	n, err := io.ReadFull(conn, b[:length])
	if err != nil {
		return nil, err
	}
	return b[:n], nil
}

func InitializeServerConfigs(configs ...*core.Config) ([]*core.Instance, error) {
	servers := make([]*core.Instance, 0, 10)

	for _, config := range configs {
		server, err := InitializeServerConfig(config)
		if err != nil {
			CloseAllServers(servers)
			return nil, err
		}
		servers = append(servers, server)
	}

	time.Sleep(time.Second * 2)

	return servers, nil
}

func InitializeServerConfig(config *core.Config) (*core.Instance, error) {
	config = withDefaultApps(config)
	server, err := core.New(config)
	if err != nil {
		return nil, err
	}
	if err := server.Start(); err != nil {
		return nil, err
	}
	return server, nil
}

func CloseAllServers(servers []*core.Instance) {
	log.Record(&log.GeneralMessage{
		Severity: log.Severity_Info,
		Content:  "Closing all servers.",
	})
	for _, server := range servers {
		server.Close()
	}
	log.Record(&log.GeneralMessage{
		Severity: log.Severity_Info,
		Content:  "All server closed.",
	})
}

func CloseServer(server *core.Instance) {
	log.Record(&log.GeneralMessage{
		Severity: log.Severity_Info,
		Content:  "Closing server.",
	})
	server.Close()
	log.Record(&log.GeneralMessage{
		Severity: log.Severity_Info,
		Content:  "Server closed.",
	})
}

func withDefaultApps(config *core.Config) *core.Config {
	config.App = append(config.App, serial.ToTypedMessage(&dispatcher.Config{}))
	config.App = append(config.App, serial.ToTypedMessage(&proxyman.InboundConfig{}))
	config.App = append(config.App, serial.ToTypedMessage(&proxyman.OutboundConfig{}))
	return config
}

func testTCPConn(port net.Port, payloadSize int, timeout time.Duration) func() error {
	return func() error {
		conn, err := net.DialTCP("tcp", nil, &net.TCPAddr{
			IP:   []byte{127, 0, 0, 1},
			Port: int(port),
		})
		if err != nil {
			return err
		}
		defer conn.Close()

		return testTCPConn2(conn, payloadSize, timeout)()
	}
}

func testUDPConn(port net.Port, payloadSize int, timeout time.Duration) func() error {
	return func() error {
		conn, err := net.DialUDP("udp", nil, &net.UDPAddr{
			IP:   []byte{127, 0, 0, 1},
			Port: int(port),
		})
		if err != nil {
			return err
		}
		defer conn.Close()

		return testTCPConn2(conn, payloadSize, timeout)()
	}
}

func testTCPConn2(conn net.Conn, payloadSize int, timeout time.Duration) func() error {
	return func() (err1 error) {
		start := time.Now()
		defer func() {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			// For info on each, see: https://golang.org/pkg/runtime/#MemStats
			fmt.Println("testConn finishes:", time.Since(start).Milliseconds(), "ms\t",
				err1, "\tAlloc =", m.Alloc, "bytes",
				"\tTotalAlloc =", m.TotalAlloc, "bytes",
				"\tSys =", m.Sys, "bytes",
				"\tNumGC =", m.NumGC)
		}()
		singleWrite := func(length int) error {
			payload := make([]byte, length)
			common.Must2(rand.Read(payload))

			nBytes, err := conn.Write(payload)
			if err != nil {
				return err
			}
			if nBytes != len(payload) {
				return errors.New("expect ", len(payload), " written, but actually ", nBytes)
			}

			response, err := readFrom2(conn, timeout, length)
			if err != nil {
				return err
			}
			_ = response

			if r := bytes.Compare(response, xor(payload)); r != 0 {
				return errors.New(r)
			}

			return nil
		}
		for payloadSize > 0 {
			sizeToWrite := min(payloadSize, 1024)
			if err := singleWrite(sizeToWrite); err != nil {
				return err
			}
			payloadSize -= sizeToWrite
		}
		return nil
	}
}

func WaitConnAvailableWithTest(t *testing.T, testFunc func() error) bool {
	for i := 1; ; i++ {
		if i > 10 {
			t.Log("All attempts failed to test tcp conn")
			return false
		}
		time.Sleep(time.Millisecond * 10)
		if err := testFunc(); err != nil {
			t.Log("err ", err)
		} else {
			t.Log("success with", i, "attempts")
			break
		}
	}
	return true
}
