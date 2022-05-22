package pkg

import (
	"gonum.org/v1/gonum/graph"
)

type PSRelationship struct {
	graph.Edge

	From PairID
	To   PairID
}

func (r PSRelationship) IsNetwork() bool {
	if r.From.Port != 0 || r.To.Port != 0 {
		return true
	}
	return false
}

type PairID struct {
	Pid  int32
	Port uint32
}

type PSTopo struct {
	graph.Graph

	Nodes        []*Process
	ProcessEdges []*ProcessEdge
	NetworkEdges []*ProcessEdge
	Caches       map[int32]bool
}

func (this *PSTopo) LinkProcess(pid, pid2 int32) {
	this.NetworkEdges = append(this.NetworkEdges,
		&ProcessEdge{
			From:   pid,
			To:     pid2,
			where:  0,
			where2: 0,
		},
	)
}

func (this *PSTopo) LinkNetwork(pid int32, port uint32, pid2 int32, port2 uint32) {
	this.NetworkEdges = append(this.NetworkEdges,
		&ProcessEdge{
			From:   pid,
			To:     pid2,
			where:  port,
			where2: port2,
		},
	)
}

func (this *PSTopo) AddProcess(process *Process) {
	if this.Caches[process.Pid] == true {
		return
	}
	this.Nodes = append(this.Nodes, process)
	this.Caches[process.Pid] = true
}

func makeTopoID(pid int32, port uint32) int64 {
	var res int64 = int64(pid)
	res |= int64(port) << 32
	return res
}

func NewTopo() *PSTopo {
	return &PSTopo{
		Nodes:        []*Process{},
		NetworkEdges: []*ProcessEdge{},
		ProcessEdges: []*ProcessEdge{},
		Caches:       map[int32]bool{},
	}
}

func AddNode() {

}

func LinkNode() {

}
