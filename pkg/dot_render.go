package pkg

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/goccy/go-graphviz"
	"github.com/goccy/go-graphviz/cgraph"
	"os"
	"strconv"
	"strings"
	"text/template"
)

type Render struct {
	g     *graphviz.Graphviz
	graph *cgraph.Graph
	topo  *PSTopo
}

type dotLabel struct {
	parts map[int]string
}

// use dot `record` shape for node
func (d dotLabel) String() string {
	var records []string
	for id, label := range d.parts {
		records = append(records, fmt.Sprintf("<p%d> %s", id, label))
	}

	internal := strings.Join(records, " | ")
	return fmt.Sprintf("{%s}", internal)
}

type dotNode struct {
	ID    string
	Label string
	Attrs dotAttrs
}

type dotEdge struct {
	From  string
	To    string
	Attrs dotAttrs
}

func newDotEdge() *dotEdge {
	return &dotEdge{
		Attrs: dotAttrs{},
	}
}
func (e dotEdge) String() string {
	return fmt.Sprintf("%s -> %s [ label=\"%s\", %s ]", e.From, e.To, "", e.Attrs)
}

func (n dotNode) String() string {
	return fmt.Sprintf("%s [ label=\"%s\", %s ]", n.ID, n.Label, n.Attrs)
}

type dotGraphData struct {
	Title  string
	Minlen uint
	//Attrs   dotAttrs
	//Cluster *dotCluster
	Nodes   []*dotNode
	Edges   []*dotEdge
	Options map[string]string
}

type dotAttrs map[string]string

func (p dotAttrs) List() []string {
	l := []string{}
	for k, v := range p {
		l = append(l, fmt.Sprintf("%s=%q", k, v))
	}
	return l
}

func (p dotAttrs) String() string {
	return strings.Join(p.List(), " ")
}

func (p dotAttrs) Lines() string {
	return fmt.Sprintf("%s;", strings.Join(p.List(), ";\n"))
}

func NewRender(topo *PSTopo, snapshot *Snapshot) (*Render, error) {
	g := graphviz.New()
	graph, _ := g.Graph()

	caches := map[int32]*cgraph.Node{}
	for _, node := range topo.Nodes {
		process := snapshot.PidProcess[node.Pid]
		name := "n" + strconv.Itoa(int(process.Pid))
		n, _ := graph.CreateNode(name)
		caches[node.Pid] = n
	}

	for _, edge := range topo.NetworkEdges {
		fromNode := caches[edge.From]
		toNode := caches[edge.To]
		if fromNode == nil || toNode == nil {
			continue
		}
		e, err := graph.CreateEdge("", fromNode, toNode)
		if err != nil {
			return nil, errors.New("failed")
		}
		e.SetColor("red")
	}

	return &Render{g, graph, topo}, nil
}

func (this *Render) Write() {
	t := template.New("dot")
	for _, s := range []string{tmplCluster, tmplNode, tmplEdge, tmplGraph} {
		if _, err := t.Parse(s); err != nil {
			panic(err)
		}
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, this.Data()); err != nil {
		panic(err)
	}

	fmt.Println(buf.String())
	graph, err := graphviz.ParseBytes(buf.Bytes())
	if err != nil {
		panic(err)
	}
	// FIXME: use parsed graph, not this.graph
	if err := this.g.RenderFilename(graph, graphviz.Format(graphviz.DOT), "./test.dot"); err != nil {
		panic(err)
	}
}

func contains(lst []uint32, item uint32) bool {
	for _, dst := range lst {
		if item == dst {
			return true
		}
	}
	return false
}

func (this *Render) Data() *dotGraphData {
	topo := this.topo
	snapshot := this.topo.Snapshot

	relatedPidPorts := map[int32][]uint32{}
	pushPidPort := func(pid int32, port uint32) {
		ports := relatedPidPorts[pid]
		relatedPidPorts[pid] = append(ports, port)
	}
	for _, e := range topo.PublicNetworkEdges {
		pushPidPort(e.Connection.Pid, e.Connection.Laddr.Port)
		remotePid := snapshot.PortPid[e.Connection.Raddr.Port]
		pushPidPort(remotePid, e.Connection.Raddr.Port)
	}
	for _, e := range topo.NetworkEdges {
		pushPidPort(e.Connection.Pid, e.Connection.Laddr.Port)
		remotePid := snapshot.PortPid[e.Connection.Raddr.Port]
		pushPidPort(remotePid, e.Connection.Raddr.Port)
	}

	var nodes []*dotNode
	for _, n := range topo.Nodes {
		if n.Pid == 0 {
			continue
		}

		node := &dotNode{
			ID: "n" + strconv.Itoa(int(n.Pid)),
			Attrs: dotAttrs{
				"shape": "record",
			},
		}

		parts := map[int]string{}
		related := relatedPidPorts[n.Pid]
		for _, port := range topo.Snapshot.PidPort[n.Pid] {
			if contains(related, port) {
				parts[int(port)] = ":" + strconv.Itoa(int(port))
			}
		}
		// put listen bellow to avoid overwrite
		for _, port := range topo.Snapshot.PidListenPort[n.Pid] {
			if contains(related, port) {
				parts[int(port)] = "Listen " + ":" + strconv.Itoa(int(port))
			}
		}

		paths := strings.Split(n.Exec, string(os.PathSeparator))
		label := makeLabel(parts, paths[len(paths)-1], strconv.Itoa(int(n.Pid)))
		node.Label = label

		nodes = append(nodes, node)
	}

	var edges []*dotEdge

	for _, e := range topo.ProcessEdges {
		edge := newDotEdge()
		edge.From = makeDotId(e.From) + makeDotPort(e.Connection.Laddr.Port)
		edge.To = makeDotId(e.To) + makeDotPort(e.Connection.Raddr.Port)
		edge.Attrs["label"] = ""
		edge.Attrs["color"] = "red"
		edges = append(edges, edge)
	}
	for _, e := range topo.NetworkEdges {
		edge := newDotEdge()
		edge.From = makeDotId(e.From) + makeDotPort(e.Connection.Laddr.Port)
		edge.To = makeDotId(e.To) + makeDotPort(e.Connection.Raddr.Port)
		edge.Attrs["label"] = ""
		edge.Attrs["color"] = "green"
		edges = append(edges, edge)
	}
	for _, e := range topo.PublicNetworkEdges {
		ip := e.Connection.Raddr.IP
		id := "ip" + strings.ReplaceAll(ip, ".", "_")
		node := &dotNode{
			ID: id,
			Attrs: dotAttrs{
				"label": ip + ":" + strconv.Itoa(int(e.Connection.Raddr.Port)),
				"shape": "box3d",
			},
		}
		nodes = append(nodes, node)

		edge := newDotEdge()
		edge.Attrs["label"] = ""
		edge.Attrs["color"] = "blue"
		edge.From = makeDotId(e.From) + makeDotPort(e.Connection.Laddr.Port)
		edge.To = id
		edges = append(edges, edge)
	}

	return &dotGraphData{
		Title: "test",
		Nodes: nodes,
		Edges: edges,
	}
}

func makeDotPort(port uint32) string {
	if port == 0 {
		return ""
	}
	return ":" + "p" + strconv.Itoa(int(port))
}

func makeDotId(pid int32) string {
	return "n" + strconv.Itoa(int(pid))
}

func makeLabel(parts map[int]string, items ...string) string {
	var records []string = items
	for id, label := range parts {
		records = append(records, fmt.Sprintf("<p%d> %s", id, label))
	}

	internal := strings.Join(records, " | ")
	return fmt.Sprintf("%s", internal)
}

func NewDotLabel() *dotLabel {
	return &dotLabel{
		parts: map[int]string{},
	}
}
