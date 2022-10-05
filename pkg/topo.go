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
	Snapshot            *Snapshot
	PidSet              map[int32]*Process
	ConnectionSet       map[string]*TopoEdge
	PublicConnectionSet map[string]*TopoEdge
	PidChildSet         map[string]*TopoEdge
}

type TopoEdge struct {
	From       int32
	To         int32
	Connection net.ConnectionStat
}

func NewTopo(snapshot *Snapshot) *PSTopo {
	return &PSTopo{
		Snapshot:            snapshot,
		PidSet:              map[int32]*Process{},
		ConnectionSet:       map[string]*TopoEdge{},
		PublicConnectionSet: map[string]*TopoEdge{},
		PidChildSet:         map[string]*TopoEdge{},
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

	key := fmt.Sprintf("%d,%d", pid, pid2)
	_, ok := tp.PidChildSet[key]
	if ok {
		return
	}
	tp.PidChildSet[key] = &TopoEdge{
		From: pid,
		To:   pid2,
	}
}

func (tp *PSTopo) linkNetwork(pid int32, pid2 int32, conn net.ConnectionStat) {
	if pid == 0 || pid2 == 0 {
		return
	}
	if pid == pid2 {
		return
	}
	_, ok := tp.ConnectionSet[conn.String()]
	if ok {
		return
	}
	tp.ConnectionSet[conn.String()] = &TopoEdge{
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

func (tp *PSTopo) linkPublicNetwork(pid int32, conn net.ConnectionStat) {
	if pid == 0 {
		return
	}
	if _, ok := tp.PublicConnectionSet[conn.String()]; ok {
		return
	}
	tp.PublicConnectionSet[conn.String()] = &TopoEdge{
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
				tp.linkNetwork(connPid, listenPid, conn)

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
						tp.linkNetwork(connPid, remotePid, conn)
					}
				} else {
					// remote is external ip
					tp.addPid(conn.Pid)
					tp.linkPublicNetwork(connPid, conn)
				}
			}
		}
	}
}

func (tp *PSTopo) Analyse(cfg *Config) *PSTopo {
	if cfg.All {
		logrus.Warningf("will generate with all data, it maybe hard")
		tp.addAll()
	} else {
		tp.addMatched(cfg)
	}

	return tp
}

func (tp *PSTopo) addAll() {
	var snapshot = tp.Snapshot
	for pid, process := range snapshot.PidProcess {
		tp.addProcess(process)
		tp.addPidNeighbor(pid)
	}
	for pid, ports := range snapshot.PidPort {
		for port := range ports.Iter() {
			conn := snapshot.GetConnection(port)
			otherPid := snapshot.PortPid[conn.Raddr.Port]
			tp.linkNetwork(pid, otherPid, conn)
		}
	}
}

func (tp *PSTopo) match(cfg *Config) (map[int32]bool, map[uint32]bool) {
	var snapshot = tp.Snapshot

	pids := map[int32]bool{}
	ports := map[uint32]bool{}
	for _, pid := range cfg.Pid {
		pids[pid] = true
	}

	// match name
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

	// match port
	for _, port := range cfg.Port {
		for listenPort, pid := range snapshot.ListenPortPid {
			if port == listenPort {
				pids[pid] = true
			}
		}
		ports[port] = true
	}

	return pids, ports
}

func (tp *PSTopo) addMatched(cfg *Config) {
	var snapshot = tp.Snapshot

	pids, ports := tp.match(cfg)

	// analyse with pids and ports
	// process Pid
	for pid, _ := range pids {
		tp.addPid(pid)
		tp.addPidNeighbor(pid)
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

	// process Port
	tp.processPort(ports)
}
