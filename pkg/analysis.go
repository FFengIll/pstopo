package pkg

import (
	_ "gopkg.in/yaml.v3"
	"net"
	"strings"
)

type FilterOption struct {
	All  bool
	Cmd  []string `json:"cmd"`
	Port []uint32 `json:"port"`
	Pid  []int32  `json:"pid"`
}

func NewGroup() *FilterOption {
	return &FilterOption{
		Cmd:  []string{},
		Port: []uint32{},
		Pid:  []int32{},
	}
}

func AnalyseSnapshot(snapshot *Snapshot, options *FilterOption) *PSTopo {
	topo := NewTopo()
	topo.Snapshot = snapshot

	if options.All {
		TopoAll(snapshot, topo)
	} else {
		pids := []int32{}
		ports := []uint32{}
		for _, pid := range options.Pid {
			pids = append(pids, pid)
		}
		for _, name := range options.Cmd {
			for _, p := range snapshot.Processes() {
				if strings.Contains(p.Cmdline, name) {
					pids = append(pids, p.Pid)
				}
			}
		}
		for _, port := range options.Port {
			for listenPort, pid := range snapshot.ListenPortPid {
				if port == listenPort {
					pids = append(pids, pid)
				}
			}
			ports = append(ports, port)
		}

		// analyse with pids and ports
		// process Pid
		for _, pid := range pids {
			topo.AddPid(pid)
			topo.AddPidNeighbor(pid)

			// add their ports
			for _, port := range snapshot.PidListenPort[pid] {
				ports = append(ports, port)
			}
			for _, port := range snapshot.PidPort[pid] {
				ports = append(ports, port)
			}

		}
		// process Port
		for _, port := range ports {
			// listen Port
			listenPort := port
			listenPid, ok := snapshot.ListenPortPid[listenPort]
			if ok {
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
				continue
			}

			// establish Port
			connPort := port
			connPid, ok := snapshot.PortPid[connPort]
			if ok {
				conn := snapshot.GetConnections(connPort)
				if conn.Laddr.Port == connPort { //redundant
					listenIP, listenPort := conn.Raddr.IP, conn.Raddr.Port

					if !isPrivateIP(net.IP(listenIP)) {
						// remote is external ip
						topo.LinkPublicNetwork(connPid, conn)
					} else {
						// remote is process
						listenPid, ok := snapshot.ListenPortPid[listenPort]
						if ok {
							topo.LinkNetwork(connPid, listenPid, conn)
						}

					}
				}
			}
		}
	}

	return topo
}

func TopoAll(snapshot *Snapshot, topo *PSTopo) {
	for pid, process := range snapshot.PidProcess {
		topo.AddProcess(process)
		topo.AddPidNeighbor(pid)
	}
	for pid, ports := range snapshot.PidPort {
		for _, port := range ports {
			conn := snapshot.GetConnections(port)
			otherPid := snapshot.PortPid[conn.Raddr.Port]
			topo.LinkNetwork(pid, otherPid, conn)
		}
	}
}
