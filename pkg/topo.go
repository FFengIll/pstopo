package pkg

import (
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
	Caches             map[int32]bool
}

type TopoNode Process

type TopoEdge struct {
	From        int32
	To          int32
	Connnetcion net.ConnectionStat
}

func (this *PSTopo) LinkProcess(pid, pid2 int32) {
	this.ProcessEdges = append(this.ProcessEdges,
		&TopoEdge{
			From: pid,
			To:   pid2,
		},
	)
}

func (this *PSTopo) LinkNetwork(pid int32, pid2 int32, conn net.ConnectionStat) {
	this.NetworkEdges = append(this.NetworkEdges,
		&TopoEdge{
			From:        pid,
			To:          pid2,
			Connnetcion: conn,
		},
	)
}

func (this *PSTopo) AddProcess(process *Process) {

	if _, ok := this.Caches[process.Pid]; ok {
		return
	}
	this.Nodes = append(this.Nodes, process)
	this.Caches[process.Pid] = true
}

func (this *PSTopo) AddPid(pid int32) {
	process, ok := this.Snapshot.PidProcess[pid]
	if ok {
		this.AddProcess(process)
	}
}

func (this *PSTopo) LinkPublicNetwork(pid int32, conn net.ConnectionStat) {
	this.NetworkEdges = append(this.NetworkEdges,
		&TopoEdge{
			From:        pid,
			Connnetcion: conn,
		},
	)
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

func makeTopoID(pid int32, port uint32) int64 {
	var res int64 = int64(pid)
	res |= int64(port) << 32
	return res
}

func NewTopo() *PSTopo {
	return &PSTopo{
		Nodes:              []*Process{},
		NetworkEdges:       []*TopoEdge{},
		PublicNetworkEdges: []*TopoEdge{},
		ProcessEdges:       []*TopoEdge{},
		Caches:             map[int32]bool{},
	}
}

func AddNode() {

}

func LinkNode() {

}
