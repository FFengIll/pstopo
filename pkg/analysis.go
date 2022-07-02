package pkg

import (
	_ "gopkg.in/yaml.v3"
	"strings"
)

func AnalyseSnapshot(snapshot *Snapshot, options *Config) *PSTopo {
	topo := NewTopo(snapshot)

	if options.All {
		TopoAll(snapshot, topo)
	} else {
		pids := map[int32]bool{}
		ports := map[uint32]bool{}
		for _, pid := range options.Pid {
			pids[pid] = true
		}
		for _, name := range options.Cmd {
			for _, p := range snapshot.Processes() {
				if strings.Contains(p.Cmdline, name) {
					pids[p.Pid] = true
					for _, c := range p.Children {
						pids[c] = true
					}
				}
			}
		}
		for _, port := range options.Port {
			for listenPort, pid := range snapshot.ListenPortPid {
				if port == listenPort {
					pids[pid] = true
				}
			}
			ports[port] = true
		}

		// analyse with pids and ports
		// process Pid
		for pid, _ := range pids {
			topo.AddPid(pid)
			topo.AddPidNeighbor(pid)

			// add their ports
			for _, port := range snapshot.PidListenPort[pid] {
				ports[port] = true
			}
			for _, port := range snapshot.PidPort[pid] {
				ports[port] = true
			}
		}

		// process Port
		topo.processPort(ports)
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
