package pkg

import (
	"fmt"
	gonet "net"
	"strconv"

	"github.com/shirou/gopsutil/v3/net"
	"gonum.org/v1/gonum/graph"
)

type PairID struct {
	Pid  int32
	Port uint32
}

type PSTopo struct {
	graph.Graph
	Snapshot           *Snapshot
	Nodes              []*Process
	ProcessEdges       []*TopoEdge
	NetworkEdges       []*TopoEdge
	PublicNetworkEdges []*TopoEdge
	PidSet             map[int32]bool
	ConnectionSet      map[string]bool
	PidChildSet        map[string]bool
}

type TopoNode Process

type TopoEdge struct {
	From       int32
	To         int32
	Connection net.ConnectionStat
}

func NewTopo(snapshot *Snapshot) *PSTopo {
	return &PSTopo{
		Snapshot: snapshot,

		Nodes:              []*Process{},
		NetworkEdges:       []*TopoEdge{},
		PublicNetworkEdges: []*TopoEdge{},
		ProcessEdges:       []*TopoEdge{},

		PidSet:        map[int32]bool{},
		ConnectionSet: map[string]bool{},
		PidChildSet:   map[string]bool{},
	}
}

func (t *TopoEdge) String() string {
	return strconv.Itoa(int(t.From)) + "->" + strconv.Itoa(int(t.To))
}

func (tp *PSTopo) LinkProcess(pid, pid2 int32) {
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
	tp.ProcessEdges = append(tp.ProcessEdges,
		&TopoEdge{
			From: pid,
			To:   pid2,
		},
	)
	tp.PidChildSet[key] = true
}

func (tp *PSTopo) LinkNetwork(pid int32, pid2 int32, conn net.ConnectionStat) {
	if pid == 0 || pid2 == 0 {
		return
	}
	if pid == pid2 {
		return
	}
	if !isPrivateIP(gonet.ParseIP(conn.Raddr.IP)) {
		tp.LinkPublicNetwork(pid, conn)
	} else {
		_, ok := tp.ConnectionSet[conn.String()]
		if ok {
			return
		}
		tp.NetworkEdges = append(tp.NetworkEdges,
			&TopoEdge{
				From:       pid,
				To:         pid2,
				Connection: conn,
			},
		)
		tp.ConnectionSet[conn.String()] = true
	}
}

func (tp *PSTopo) AddProcess(process *Process) {
	if process.Pid == 0 {
		return
	}

	if _, ok := tp.PidSet[process.Pid]; ok {
		return
	}
	tp.Nodes = append(tp.Nodes, process)
	tp.PidSet[process.Pid] = true
}

func (tp *PSTopo) AddPid(pid int32) {
	process, ok := tp.Snapshot.PidProcess[pid]
	if ok {
		tp.AddProcess(process)
	}
}

func (tp *PSTopo) LinkPublicNetwork(pid int32, conn net.ConnectionStat) {

	if pid == 0 {
		return
	}
	_, ok := tp.ConnectionSet[conn.String()]
	if ok {
		return
	}
	tp.PublicNetworkEdges = append(tp.PublicNetworkEdges,
		&TopoEdge{
			From:       pid,
			Connection: conn,
		},
	)
	tp.ConnectionSet[conn.String()] = true

}

func (tp *PSTopo) AddPidNeighbor(pid int32) {
	snapshot := tp.Snapshot
	process := snapshot.PidProcess[pid]
	for _, child := range process.Children {
		if childProcess, ok := snapshot.PidProcess[child]; ok {
			tp.LinkProcess(pid, child)
			tp.AddProcess(childProcess)
		}
	}
	if parentProcess, ok := snapshot.PidProcess[process.Parent]; ok {
		tp.LinkProcess(process.Parent, pid)
		tp.AddProcess(parentProcess)
	}
}

func (tp *PSTopo) AddProcessNeighbor(process *Process) {
	snapshot := tp.Snapshot
	pid := process.Pid
	for _, child := range process.Children {
		if childProcess, ok := snapshot.PidProcess[child]; ok {
			tp.LinkProcess(pid, child)
			tp.AddProcess(childProcess)
		}
	}
	if parentProcess, ok := snapshot.PidProcess[process.Parent]; ok {
		tp.LinkProcess(process.Parent, pid)
		tp.AddProcess(parentProcess)
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
		connections := snapshot.ListenPortConnections[listenPort]
		for _, conn := range connections {
			connPort := conn.Laddr.Port
			connPid, ok := snapshot.PortPid[connPort]
			if ok {
				tp.AddPid(listenPid)
				tp.AddPid(connPid)
				tp.LinkNetwork(connPid, listenPid, conn)

				// FIXME: to avoid any potential error, force add the port to pid
			}

		}
	}

	for _, connPort := range establishPorts {
		// establish Port
		connPid, ok := snapshot.PortPid[connPort]
		if ok {
			conn := snapshot.GetConnections(connPort)
			if conn.Laddr.Port == connPort { //redundant
				remoteIP, remotePort := conn.Raddr.IP, conn.Raddr.Port
				if !isPrivateIP(gonet.ParseIP(remoteIP)) {
					// remote is external ip
					tp.LinkPublicNetwork(connPid, conn)
				} else {
					// remote is process
					remotePid, ok := snapshot.PortPid[remotePort]
					if ok {
						tp.LinkNetwork(connPid, remotePid, conn)
					}

				}
			}
		}
	}
}
