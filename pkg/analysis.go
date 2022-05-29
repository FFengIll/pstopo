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
			for _, conn := range snapshot.PidListen[pid] {
				ports = append(ports, conn.Laddr.Port)
			}
			for _, conn := range snapshot.PidConnection[pid] {
				ports = append(ports, conn.Laddr.Port)
			}

		}
		// process Port
		for _, port := range ports {
			// listen Port
			listenPort := port
			listenPid, ok := snapshot.ListenPortPid[listenPort]
			if ok {
				connections := snapshot.RemotePortConnection[listenPort]
				for _, conn := range connections {
					connPort := conn.Laddr.Port
					connPid, ok := snapshot.LocalPortPid[connPort]
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
			connPid, ok := snapshot.LocalPortPid[connPort]
			if ok {
				connections := snapshot.PidConnection[connPid]
				for _, conn := range connections {
					if conn.Laddr.Port == connPort {
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
	}

	return topo
}

func TopoAll(snapshot *Snapshot, topo *PSTopo) {
	for pid, process := range snapshot.PidProcess {
		topo.AddProcess(process)
		topo.AddPidNeighbor(pid)
	}
	for pid, connections := range snapshot.PidConnection {
		for _, conn := range connections {
			otherPid := snapshot.LocalPortPid[conn.Raddr.Port]
			topo.LinkNetwork(pid, otherPid, conn)
		}
	}
}

func dummy() {

	//res := NewGroup()
	//
	//// get All Pid
	//pids, err := Cmd.PidsWithContext(context.Background())
	//if err != nil {
	//	return nil
	//}
	//for _, Pid := range pids {
	//	for _, target := range group.Pid {
	//		if Pid == target {
	//			res.Pid = append(res.Pid, target)
	//		}
	//	}
	//}
	//
	//// get each Cmd
	//ps := make([]*Cmd.Process, 10)
	//for _, Pid := range pids {
	//	p, _ := Cmd.NewProcessWithContext(context.Background(), Pid)
	//	cmdline, _ := p.Cmdline()
	//	for _, target := range group.Cmd {
	//		if strings.Contains(cmdline, target) {
	//			ps = append(ps, p)
	//			res.Cmd = append(res.Cmd, target)
	//		}
	//	}
	//}
	//
	//// get each by Port
	//for _, Pid := range pids {
	//	connections, err := net.ConnectionsPid("ESTABLISHED", Pid)
	//	if err != nil {
	//		return nil
	//	}
	//	for _, conn := range connections {
	//		fmt.Printf("PID=%d, LISTEN on :%d, Cmdline=%s\n", conn.Pid, conn.Laddr.Port, conn.Laddr)
	//		break
	//	}
	//}
}
