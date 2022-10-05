package pkg

import (
	"fmt"
	gonet "net"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/v3/net"
	"github.com/sirupsen/logrus"
	"gonum.org/v1/gonum/graph"
)

type PSTopo struct {
	graph.Graph
	Snapshot    *Snapshot
	PidSet      map[int32]*Process
	PidConnSet  map[string]*TopoEdge
	IPConnSet   map[string]*TopoEdge
	PidChildSet map[string]*TopoEdge
}

type TopoEdge struct {
	From       int32
	To         int32
	Connection net.ConnectionStat
}

func NewTopo(snapshot *Snapshot) *PSTopo {
	return &PSTopo{
		Snapshot:    snapshot,
		PidSet:      map[int32]*Process{},
		PidConnSet:  map[string]*TopoEdge{},
		IPConnSet:   map[string]*TopoEdge{},
		PidChildSet: map[string]*TopoEdge{},
	}
}

func (t *TopoEdge) String() string {
	return strconv.Itoa(int(t.From)) + "->" + strconv.Itoa(int(t.To))
}

func (tp *PSTopo) linkProcess(pid, pid2 int32) {
	if pid == 0 || pid2 == 0 {
		return
	}
	if pid == pid2 {
		return
	}

	key := fmt.Sprintf("%d->%d", pid, pid2)
	if _, ok := tp.PidChildSet[key]; ok {
		return
	}
	tp.PidChildSet[key] = &TopoEdge{
		From: pid,
		To:   pid2,
	}
}

func (tp *PSTopo) linkPidPort(pid int32, pid2 int32, conn net.ConnectionStat) {
	if pid == 0 || pid2 == 0 {
		return
	}
	if pid == pid2 {
		return
	}
	_, ok := tp.PidConnSet[conn.String()]
	if ok {
		return
	}
	tp.PidConnSet[conn.String()] = &TopoEdge{
		From:       pid,
		To:         pid2,
		Connection: conn,
	}
}

func (tp *PSTopo) addProcess(process *Process) {
	if process.Pid == 0 {
		return
	}

	if _, ok := tp.PidSet[process.Pid]; ok {
		return
	}
	tp.PidSet[process.Pid] = process
}

func (tp *PSTopo) addPid(pid int32) {
	process, ok := tp.Snapshot.PidProcess[pid]
	if ok {
		tp.addProcess(process)
	}
}

func (tp *PSTopo) linkIPPort(pid int32, conn net.ConnectionStat) {
	if pid == 0 {
		return
	}
	if _, ok := tp.IPConnSet[conn.String()]; ok {
		return
	}
	tp.IPConnSet[conn.String()] = &TopoEdge{
		From:       pid,
		Connection: conn,
	}
}

func (tp *PSTopo) addPidParent(pid int32) int32 {
	snapshot := tp.Snapshot
	process := snapshot.PidProcess[pid]
	if parentProcess, ok := snapshot.PidProcess[process.Parent]; ok {
		tp.linkProcess(process.Parent, pid)
		tp.addProcess(parentProcess)
		return process.Parent
	}
	return 0
}

func (tp *PSTopo) addPidChildren(pid int32) {
	snapshot := tp.Snapshot
	process := snapshot.PidProcess[pid]
	for _, child := range process.Children {
		if childProcess, ok := snapshot.PidProcess[child]; ok {
			tp.linkProcess(pid, child)
			tp.addProcess(childProcess)
		}
	}
}

func (tp *PSTopo) addPidNeighbor(pid int32) {
	tp.addPidChildren(pid)

	// FIXME: recursively add process parent
	next := pid
	for {
		tmp := tp.addPidParent(next)
		if next == 0 || next == 1 || tmp == next {
			break
		}
		next = tmp
	}
}

func (tp *PSTopo) processPort(ports map[uint32]bool) {
	snapshot := tp.Snapshot
	var listenPorts []uint32
	var establishPorts []uint32
	for port := range ports {
		_, ok := tp.Snapshot.ListenPortPid[port]
		if ok {
			listenPorts = append(listenPorts, port)
		} else {
			establishPorts = append(establishPorts, port)
		}
	}

	for _, port := range listenPorts {
		// listen Port
		listenPort := port
		listenPid, _ := snapshot.ListenPortPid[listenPort]

		// filter
		if _, ok := tp.PidSet[listenPid]; !ok {
			continue
		}

		connections := snapshot.ListenPortConnections[listenPort]
		for _, conn := range connections {
			connPort := conn.Laddr.Port
			connPid, ok := snapshot.PortPid[connPort]
			if ok {
				tp.addPid(listenPid)
				tp.addPid(connPid)
				tp.linkPidPort(connPid, listenPid, conn)

				// FIXME: to avoid any potential error, force add the port to pid
			}

		}
	}

	for _, localPort := range establishPorts {
		// establish Port
		connPid, ok := snapshot.PortPid[localPort]
		if ok {
			// filter
			if _, ok := tp.PidSet[connPid]; !ok {
				continue
			}

			conn := snapshot.GetConnection(localPort)
			if conn.Laddr.Port == localPort { // redundant
				remoteIP, remotePort := conn.Raddr.IP, conn.Raddr.Port
				if isPrivateIP(gonet.ParseIP(remoteIP)) {
					// remote is process
					remotePid, ok := snapshot.PortPid[remotePort]
					if ok {
						tp.addPid(connPid)
						tp.addPid(remotePid)
						tp.linkPidPort(connPid, remotePid, conn)
					}
				} else {
					// remote is external ip
					tp.addPid(conn.Pid)
					tp.linkIPPort(connPid, conn)
				}
			}
		}
	}
}

func (tp *PSTopo) Analyse(cfg *Config) *PSTopo {
	if cfg.All {
		logrus.Warningf("will generate with all data, it maybe hard")
		var snapshot = tp.Snapshot
		for pid, process := range snapshot.PidProcess {
			tp.addProcess(process)
			tp.addPidNeighbor(pid)
		}
		for pid, ports := range snapshot.PidPort {
			for port := range ports.Iter() {
				conn := snapshot.GetConnection(port)
				otherPid := snapshot.PortPid[conn.Raddr.Port]
				tp.linkPidPort(pid, otherPid, conn)
			}
		}
	} else {
		tp.filter(cfg)
	}

	return tp
}

func (tp *PSTopo) filterPid(cfg *Config) map[int32]bool {
	var snapshot = tp.Snapshot

	pids := map[int32]bool{}

	// filter by pid
	for _, pid := range cfg.Pid {
		pids[pid] = true
	}

	// filter by name
	for _, name := range cfg.Cmd {
		for _, p := range snapshot.Processes() {
			if strings.Contains(p.Cmdline, name) {
				pids[p.Pid] = true
				for _, c := range p.Children {
					pids[c] = true
				}
			}
		}
	}

	// filter by (listen) port
	for _, port := range cfg.Port {
		for listenPort, pid := range snapshot.ListenPortPid {
			if port == listenPort {
				pids[pid] = true
			}
		}
	}

	return pids
}

func (tp *PSTopo) filterPort(cfg *Config) map[uint32]bool {
	snapshot := tp.Snapshot
	ports := map[uint32]bool{}

	// filter by (listen) port
	for _, port := range cfg.Port {
		ports[port] = true
	}

	// add all ports for the pid
	for _, set := range snapshot.PidPort {
		if set != nil {
			for port := range set.Iter() {
				ports[port] = true
			}
		}
	}
	for _, set := range snapshot.PidListenPort {
		if set != nil {
			for port := range set.Iter() {
				ports[port] = true
			}
		}
	}

	return ports

}

func (tp *PSTopo) filter(cfg *Config) {
	// process Pid at first and then the port

	// process Pid
	pids := tp.filterPid(cfg)
	for pid, _ := range pids {
		tp.addPid(pid)
		tp.addPidNeighbor(pid)
	}

	// process port
	ports := tp.filterPort(cfg)
	tp.processPort(ports)
}
