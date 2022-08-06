package pkg

import (
	"strings"

	"github.com/sirupsen/logrus"
	_ "gopkg.in/yaml.v3"
)

func AnalyseSnapshot(snapshot *Snapshot, options *Config) *PSTopo {
	topo := NewTopo(snapshot)

	if options.All {
		logrus.Warningf("will generate with all data, it maybe hard")
		GenerateAll(snapshot, topo)
	} else {
		pids := map[int32]bool{}
		ports := map[uint32]bool{}
		for _, pid := range options.Pid {
			pids[pid] = true
		}

		// match name
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

		// match port
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
		}

		// add all ports for the pid
		for _, set := range snapshot.PidPort {
			for port := range set.Iter() {
				ports[port] = true
			}
		}
		for _, set := range snapshot.PidListenPort {
			for port := range set.Iter() {
				ports[port] = true
			}
		}

		// process Port
		topo.processPort(ports)
	}

	return topo
}

func GenerateAll(snapshot *Snapshot, topo *PSTopo) {
	for pid, process := range snapshot.PidProcess {
		topo.AddProcess(process)
		topo.AddPidNeighbor(pid)
	}
	for pid, ports := range snapshot.PidPort {
		for port := range ports.Iter() {
			conn := snapshot.GetConnection(port)
			otherPid := snapshot.PortPid[conn.Raddr.Port]
			topo.LinkNetwork(pid, otherPid, conn)
		}
	}
}
