package pkg

import (
	"fmt"
	"github.com/shirou/gopsutil/v3/net"
	"gonum.org/v1/gonum/graph"
	gonet "net"
	"strconv"
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
	PidCaches          map[int32]bool
	ConnectionCaches   map[string]bool
	HierarchyCaches    map[string]bool
}

type TopoNode Process

type TopoEdge struct {
	From       int32
	To         int32
	Connection net.ConnectionStat
}

func NewTopo() *PSTopo {
	return &PSTopo{
		Nodes:              []*Process{},
		NetworkEdges:       []*TopoEdge{},
		PublicNetworkEdges: []*TopoEdge{},
		ProcessEdges:       []*TopoEdge{},

		PidCaches:        map[int32]bool{},
		ConnectionCaches: map[string]bool{},
		HierarchyCaches:  map[string]bool{},
	}
}

func (t *TopoEdge) String() string {
	return strconv.Itoa(int(t.From)) + "->" + strconv.Itoa(int(t.To))
}

func (this *PSTopo) LinkProcess(pid, pid2 int32) {
	if pid == 0 || pid2 == 0 {
		return
	}
	if pid == pid2 {
		return
	}

	key := fmt.Sprintf("%d,%d", pid, pid2)
	_, ok := this.HierarchyCaches[key]
	if ok {
		return
	}
	this.ProcessEdges = append(this.ProcessEdges,
		&TopoEdge{
			From: pid,
			To:   pid2,
		},
	)
	this.HierarchyCaches[key] = true
}

func (this *PSTopo) LinkNetwork(pid int32, pid2 int32, conn net.ConnectionStat) {
	if pid == 0 || pid2 == 0 {
		return
	}
	if pid == pid2 {
		return
	}
	if !isPrivateIP(gonet.ParseIP(conn.Raddr.IP)) {
		this.LinkPublicNetwork(pid, conn)
	} else {
		_, ok := this.ConnectionCaches[conn.String()]
		if ok {
			return
		}
		this.NetworkEdges = append(this.NetworkEdges,
			&TopoEdge{
				From:       pid,
				To:         pid2,
				Connection: conn,
			},
		)
		this.ConnectionCaches[conn.String()] = true
	}
}

func (this *PSTopo) AddProcess(process *Process) {
	if process.Pid == 0 {
		return
	}

	if _, ok := this.PidCaches[process.Pid]; ok {
		return
	}
	this.Nodes = append(this.Nodes, process)
	this.PidCaches[process.Pid] = true
}

func (this *PSTopo) AddPid(pid int32) {
	process, ok := this.Snapshot.PidProcess[pid]
	if ok {
		this.AddProcess(process)
	}
}

func (this *PSTopo) LinkPublicNetwork(pid int32, conn net.ConnectionStat) {

	if pid == 0 {
		return
	}
	_, ok := this.ConnectionCaches[conn.String()]
	if ok {
		return
	}
	this.PublicNetworkEdges = append(this.PublicNetworkEdges,
		&TopoEdge{
			From:       pid,
			Connection: conn,
		},
	)
	this.ConnectionCaches[conn.String()] = true

}

func (this *PSTopo) AddPidNeighbor(pid int32) {
	snapshot := this.Snapshot
	process := snapshot.PidProcess[pid]
	for _, child := range process.Children {
		if childProcess, ok := snapshot.PidProcess[child]; ok {
			this.LinkProcess(pid, child)
			this.AddProcess(childProcess)
		}
	}
	if parentProcess, ok := snapshot.PidProcess[process.Parent]; ok {
		this.LinkProcess(process.Parent, pid)
		this.AddProcess(parentProcess)
	}
}

func (this *PSTopo) AddProcessNeighbor(process *Process) {
	snapshot := this.Snapshot
	pid := process.Pid
	for _, child := range process.Children {
		if childProcess, ok := snapshot.PidProcess[child]; ok {
			this.LinkProcess(pid, child)
			this.AddProcess(childProcess)
		}
	}
	if parentProcess, ok := snapshot.PidProcess[process.Parent]; ok {
		this.LinkProcess(process.Parent, pid)
		this.AddProcess(parentProcess)
	}
}

func (topo *PSTopo) processPort(ports []uint32) {
	snapshot := topo.Snapshot
	listenPorts := []uint32{}
	establishPorts := []uint32{}
	for _, port := range ports {
		_, ok := topo.Snapshot.ListenPortPid[port]
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
				topo.AddPid(listenPid)
				topo.AddPid(connPid)
				topo.LinkNetwork(connPid, listenPid, conn)
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
					topo.LinkPublicNetwork(connPid, conn)
				} else {
					// remote is process
					remotePid, ok := snapshot.PortPid[remotePort]
					if ok {
						topo.LinkNetwork(connPid, remotePid, conn)
					}

				}
			}
		}
	}
}

func AddNode() {

}

func LinkNode() {

}
