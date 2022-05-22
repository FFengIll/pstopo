package pkg

import (
	_ "gopkg.in/yaml.v3"
	"strings"
)

type FilterOption struct {
	all     bool
	process []string
	port    []uint32
	pid     []int32
}

func NewGroup() *FilterOption {
	return &FilterOption{
		true,
		make([]string, 3),
		make([]uint32, 3),
		make([]int32, 3),
	}
}

func AnalyseSnapshot(snapshot *Snapshot) *PSTopo {
	topo := NewTopo()

	for pid, process := range snapshot.PidProcess {
		topo.AddProcess(process)
		for _, child := range process.Children {
			childProcess := snapshot.PidProcess[child]
			topo.LinkProcess(pid, child)
			topo.AddProcess(childProcess)
		}
		parentProcess := snapshot.PidProcess[process.Parent]
		topo.LinkProcess(process.Parent, pid)
		topo.AddProcess(parentProcess)
	}
	for pid, connections := range snapshot.PidConnection {
		for _, c := range connections {
			otherPid := snapshot.ConnectionPortPid[c.Raddr.Port]
			topo.LinkNetwork(pid, c.Laddr.Port, otherPid, c.Raddr.Port)
		}
	}
	return topo
}

func FilterSnapshot(options *FilterOption, snapshot *Snapshot) *Snapshot {
	if options.all {
		res := snapshot
		return res
	}
	pids := []int32{}
	for _, pid := range options.pid {
		pids = append(pids, pid)
	}
	for _, name := range options.process {
		for _, p := range snapshot.Processes() {
			if strings.Contains(p.Cmdline, name) {
				pids = append(pids, p.Pid)
			}
		}
	}
	for _, port := range options.port {
		for listenPort, pid := range snapshot.ListenPortPid {
			if port == listenPort {
				pids = append(pids, pid)
			}
		}
	}

	return snapshot
}

func dummy() {

	//res := NewGroup()
	//
	//// get all Pid
	//pids, err := process.PidsWithContext(context.Background())
	//if err != nil {
	//	return nil
	//}
	//for _, pid := range pids {
	//	for _, target := range group.pid {
	//		if pid == target {
	//			res.pid = append(res.pid, target)
	//		}
	//	}
	//}
	//
	//// get each process
	//ps := make([]*process.Process, 10)
	//for _, pid := range pids {
	//	p, _ := process.NewProcessWithContext(context.Background(), pid)
	//	cmdline, _ := p.Cmdline()
	//	for _, target := range group.process {
	//		if strings.Contains(cmdline, target) {
	//			ps = append(ps, p)
	//			res.process = append(res.process, target)
	//		}
	//	}
	//}
	//
	//// get each by Port
	//for _, pid := range pids {
	//	connections, err := net.ConnectionsPid("ESTABLISHED", pid)
	//	if err != nil {
	//		return nil
	//	}
	//	for _, conn := range connections {
	//		fmt.Printf("PID=%d, LISTEN on :%d, Cmdline=%s\n", conn.Pid, conn.Laddr.Port, conn.Laddr)
	//		break
	//	}
	//}
}
