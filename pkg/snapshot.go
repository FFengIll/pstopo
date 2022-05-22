package pkg

import (
	"context"
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"strings"
	"time"
)

type Snapshot struct {
	matched FilterOption

	PidProcess    map[int32]*Process             `yaml:"process_map"`
	PidListen     map[int32][]net.ConnectionStat `yaml:"listen_map"`
	PidConnection map[int32][]net.ConnectionStat `yaml:"connection_map"`

	ListenPortPid     map[uint32]int32 `yaml:"listen_port_map"`
	ConnectionPortPid map[uint32]int32 `yaml:"connection_port_map"`
}

func NewSnapshot() *Snapshot {
	s := Snapshot{
		PidProcess:        map[int32]*Process{},
		PidListen:         map[int32][]net.ConnectionStat{},
		PidConnection:     map[int32][]net.ConnectionStat{},
		ListenPortPid:     map[uint32]int32{},
		ConnectionPortPid: map[uint32]int32{},
	}
	return &s
}

func TakeSnapshot() (*Snapshot, error) {
	snapshot := NewSnapshot()
	log := logrus.StandardLogger()
	log.Info("Take snapshot at {}", time.Now())
	pids, err := process.PidsWithContext(context.Background())
	if err != nil {
		return nil, err
	}
	for _, pid := range pids {
		p, _ := process.NewProcessWithContext(context.Background(), pid)
		cmdline, _ := p.Cmdline()
		children, _ := p.Children()
		snapshot.PidProcess[pid] = &Process{
			Pid:     p.Pid,
			Cmdline: cmdline,
			Parent: func() int32 {
				parent, err := p.Parent()
				if err != nil {
					return 0
				}
				return parent.Pid
			}(),
			Children: func() []int32 {
				res := []int32{}
				for _, c := range children {
					res = append(res, c.Pid)
				}
				return res
			}(),
		}
	}

	var kind = ""

	//if err := fillConnection(kind, pid, snapshot.PidListen[pid]); err != nil {
	//	log.Error("Pid={} error={}", pid, err)
	//}

	// here, `gopsutil` use pid=0 to fetch all connections
	connections, err := net.Connections(kind)
	if err != nil {
		return nil, err
	}
	for _, conn := range connections {
		if strings.EqualFold(conn.Status, "LISTEN") {
			listen := snapshot.PidListen[conn.Pid]
			listen = append(listen, conn)
			snapshot.PidListen[conn.Pid] = listen

			snapshot.ListenPortPid[conn.Laddr.Port] = conn.Pid
		} else {
			establish := snapshot.PidConnection[conn.Pid]
			establish = append(establish, conn)
			snapshot.PidConnection[conn.Pid] = establish

			snapshot.ConnectionPortPid[conn.Laddr.Port] = conn.Pid
		}
	}

	return snapshot, nil
}

func (s *Snapshot) Processes() []*Process {
	return func() []*Process {
		ps := []*Process{}
		for _, p := range s.PidProcess {
			ps = append(ps, p)
		}
		return ps
	}()
}

func (s *Snapshot) DumpFile(filepath string) {
	bytes := s.Dump()
	err := ioutil.WriteFile(filepath, bytes, 0644)
	if err != nil {
		return
	}
}

func (s *Snapshot) Dump() []byte {
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	data, _ := json.Marshal(s)
	return data
}

func (s *Snapshot) Print() []byte {
	data := s.Dump()
	fmt.Printf("%s", data)
	return data
}
