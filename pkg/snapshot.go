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
	PidProcess    map[int32]*Process `yaml:"process"`
	PidListenPort map[int32][]uint32 `yaml:"pid_listen_port"`
	PidPort       map[int32][]uint32 `yaml:"pid_port"`

	ListenPortConnections map[uint32][]net.ConnectionStat `yaml:"listen_port_connection"`
	ListenPortPid         map[uint32]int32                `yaml:"listen_port_pid"`

	PortConnection map[uint32]net.ConnectionStat `yaml:"port_connection"`
	PortPid        map[uint32]int32              `yaml:"port_pid"`
}

func NewSnapshot() *Snapshot {
	s := Snapshot{
		PidProcess:    map[int32]*Process{},
		PidListenPort: map[int32][]uint32{},
		PidPort:       map[int32][]uint32{},

		ListenPortConnections: map[uint32][]net.ConnectionStat{},
		ListenPortPid:         map[uint32]int32{},

		PortConnection: map[uint32]net.ConnectionStat{},
		PortPid:        map[uint32]int32{},
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

	// here, `gopsutil` use Pid=0 to fetch All connections
	var kind = ""
	connections, err := net.Connections(kind)
	if err != nil {
		return nil, err
	}
	for _, conn := range connections {

		if strings.EqualFold(conn.Status, "LISTEN") {
			listenPort := conn.Laddr.Port

			snapshot.ListenPortPid[listenPort] = conn.Pid

			conns := snapshot.ListenPortConnections[listenPort]
			snapshot.ListenPortConnections[listenPort] = append(conns, conn)

			listen := snapshot.PidListenPort[conn.Pid]
			snapshot.PidListenPort[conn.Pid] = append(listen, listenPort)

		} else {
			localPort := conn.Laddr.Port

			snapshot.PortPid[localPort] = conn.Pid

			snapshot.PortConnection[localPort] = conn

			establish := snapshot.PidPort[conn.Pid]
			snapshot.PidPort[conn.Pid] = append(establish, localPort)

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
	log := logrus.New()
	if strings.Compare(filepath, "") == 0 {
		now := time.Now()
		filepath = fmt.Sprintf("snapshot-%s-%02d:%02d:%02d.json", now.Format("2006-01-02"), now.Hour(), now.Minute(), now.Second())
	}
	log.Infof("snapshot to: %s", filepath)
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
	s.PidPort[pid] = snapshot.PidPort[pid]
	s.PidProcess[pid] = snapshot.PidProcess[pid]
	s.PidListenPort[pid] = snapshot.PidListenPort[pid]

	p := snapshot.PidProcess[pid]
	s.CopyLite(snapshot, p.Parent)
	for _, c := range p.Children {
		s.CopyLite(snapshot, c)
	}
}

func (s *Snapshot) CopyLite(snapshot *Snapshot, pid int32) {

	s.PidProcess[pid] = snapshot.PidProcess[pid]
	s.PidListenPort[pid] = snapshot.PidListenPort[pid]
}

func (s *Snapshot) GetConnections(port uint32) net.ConnectionStat {
	return s.PortConnection[port]
}
