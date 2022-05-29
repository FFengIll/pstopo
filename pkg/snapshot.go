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

	PidProcess    map[int32]*Process             `yaml:"process"`
	PidListen     map[int32][]net.ConnectionStat `yaml:"listen"`
	PidConnection map[int32][]net.ConnectionStat `yaml:"connection"`

	ListenPortPid map[uint32]int32 `yaml:"listen_port_pid"`
	LocalPortPid  map[uint32]int32 `yaml:"local_port_pid"`

	RemotePortConnection map[uint32][]net.ConnectionStat `yaml:"remote_port_connection"`
}

func NewSnapshot() *Snapshot {
	s := Snapshot{
		PidProcess:    map[int32]*Process{},
		PidListen:     map[int32][]net.ConnectionStat{},
		PidConnection: map[int32][]net.ConnectionStat{},

		ListenPortPid: map[uint32]int32{},
		LocalPortPid:  map[uint32]int32{},

		RemotePortConnection: map[uint32][]net.ConnectionStat{},
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
		exec, _ := p.Exe()
		cmdline, _ := p.Cmdline()
		children, _ := p.Children()
		snapshot.PidProcess[pid] = &Process{
			Pid:     p.Pid,
			Exec:    exec,
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

	//if err := fillConnection(kind, Pid, snapshot.PidListen[Pid]); err != nil {
	//	log.Error("Pid={} error={}", Pid, err)
	//}

	// here, `gopsutil` use Pid=0 to fetch All connections
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

			snapshot.LocalPortPid[conn.Laddr.Port] = conn.Pid

			array := snapshot.RemotePortConnection[conn.Raddr.Port]
			snapshot.RemotePortConnection[conn.Raddr.Port] = append(array, conn)
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

func (s *Snapshot) Copy(snapshot *Snapshot, pid int32) {
	s.PidConnection[pid] = snapshot.PidConnection[pid]
	s.PidProcess[pid] = snapshot.PidProcess[pid]
	s.PidListen[pid] = snapshot.PidListen[pid]

	p := snapshot.PidProcess[pid]
	s.CopyLite(snapshot, p.Parent)
	for _, c := range p.Children {
		s.CopyLite(snapshot, c)
	}
}

func (s *Snapshot) CopyLite(snapshot *Snapshot, pid int32) {

	s.PidProcess[pid] = snapshot.PidProcess[pid]
	s.PidListen[pid] = snapshot.PidListen[pid]
}
