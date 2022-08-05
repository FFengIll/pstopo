package pkg

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/goccy/go-graphviz"
	"github.com/goccy/go-graphviz/cgraph"
	"github.com/sirupsen/logrus"
)

type Render struct {
	engine *graphviz.Graphviz
	graph  *cgraph.Graph
}

type dotNode struct {
	ID    string
	Label string
	Attrs dotAttrs
}

type dotEdge struct {
	From  string
	To    string
	Label string
	Attrs dotAttrs
}

func newDotEdge() *dotEdge {
	return &dotEdge{
		Label: "",
		Attrs: dotAttrs{},
	}
}

func (e dotEdge) String() string {
	return fmt.Sprintf("%s -> %s [ label=\"%s\", %s ]", e.From, e.To, e.Label, e.Attrs)
}

func (n dotNode) String() string {
	return fmt.Sprintf("%s [ label=\"%s\", %s ]", n.ID, n.Label, n.Attrs)
}

type dotGraphData struct {
	Title string
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

func NewDotRender() (*Render, error) {
	g := graphviz.New()
	graph, _ := g.Graph()

	return &Render{g, graph}, nil
}

func (r *Render) WriteTo(data *dotGraphData, output string) {
	t := template.New("dot")
	for _, s := range []string{tmplLegend, tmplCluster, tmplNode, tmplEdge, tmplGraph} {
		if _, err := t.Parse(s); err != nil {
			panic(err)
		}
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		panic(err)
	}

	fmt.Println(buf.String())
	graph, err := graphviz.ParseBytes(buf.Bytes())
	if err != nil {
		panic(err)
	}
	// FIXME: use parsed graph, not r.graph
	if !strings.HasSuffix(output, ".dot") {
		output = output + ".dot"
	}
	if err := r.engine.RenderFilename(graph, graphviz.Format(graphviz.DOT), output); err != nil {
		//fd, _ := os.Open(output)
		//fd.WriteString(err.Error())
		logrus.Errorln(err)
		logrus.Errorln("parse graph error, but try to output the file")
		return
	}
}

func makeDotPortLabel(label string, dotPort string) string {
	return fmt.Sprintf("<p%s> %s", dotPort, label)
}

func (r *Render) RenderToData(snapshot *Snapshot, topo *PSTopo) (*dotGraphData, error) {
	graph := r.graph
	index := map[int32]*cgraph.Node{}
	for _, node := range topo.Nodes {
		process := snapshot.PidProcess[node.Pid]
		name := "n" + strconv.Itoa(int(process.Pid))
		n, _ := graph.CreateNode(name)
		index[node.Pid] = n
	}

	for _, edge := range topo.NetworkEdges {
		fromNode := index[edge.From]
		toNode := index[edge.To]
		if fromNode == nil || toNode == nil {
			continue
		}
		e, err := graph.CreateEdge("", fromNode, toNode)
		if err != nil {
			return nil, errors.New("failed")
		}
		e.SetColor("red")
	}

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

		// TODO: may only include related port (but it may not good)
		//related := relatedPidPorts[n.Pid]
		for _, port := range topo.Snapshot.PidPort[n.Pid] {
			//if contains(related, port) {
			parts[int(port)] = ":" + strconv.Itoa(int(port))
			//}
		}
		// put listen bellow to avoid overwrite
		for _, port := range topo.Snapshot.PidListenPort[n.Pid] {
			//if contains(related, port) {
			parts[int(port)] = "Listen " + ":" + strconv.Itoa(int(port))
			//}
		}

		paths := strings.Split(n.Exec, string(os.PathSeparator))
		pidLabel := makeDotPortLabel(strconv.Itoa(int(n.Pid)), "0")
		label := makeDotLabel(parts, paths[len(paths)-1], pidLabel)
		node.Label = label

		nodes = append(nodes, node)
	}

	var edges []*dotEdge

	for _, e := range topo.ProcessEdges {
		edge := newDotEdge()
		edge.From = toDotId(e.From) + toDotPort(0)
		edge.To = toDotId(e.To) + toDotPort(0)
		edge.Attrs["label"] = ""
		edge.Attrs["color"] = "red"
		edges = append(edges, edge)
	}
	for _, e := range topo.NetworkEdges {
		edge := newDotEdge()
		edge.From = toDotId(e.From) + toDotPort(e.Connection.Laddr.Port)
		edge.To = toDotId(e.To) + toDotPort(e.Connection.Raddr.Port)
		edge.Attrs["label"] = ""
		edge.Attrs["color"] = "green"
		edge.Attrs["dir"] = "both"
		edges = append(edges, edge)
	}
	for _, e := range topo.PublicNetworkEdges {
		ip := e.Connection.Raddr.IP
		id := "ip" + replaceIPChar(ip)
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
		edge.Attrs["dir"] = "both"
		edge.From = toDotId(e.From) + toDotPort(e.Connection.Laddr.Port)
		edge.To = id
		edges = append(edges, edge)
	}

	now := time.Now()
	return &dotGraphData{
		Title: fmt.Sprintf("%s (%s)", "PSTopo", now.Format(time.RFC3339)),
		Nodes: nodes,
		Edges: edges,
	}, nil
}

func toDotPort(port uint32) string {
	// for port == 0, we process it as `no dot node port`
	if port == 0 {
		return ""
	}
	return ":" + "p" + strconv.Itoa(int(port))
}

func toDotId(pid int32) string {
	return "n" + strconv.Itoa(int(pid))
}

func makeDotLabel(parts map[int]string, items ...string) string {
	var records []string = items
	for id, label := range parts {
		records = append(records, makeDotPortLabel(label, strconv.Itoa(id)))
	}

	internal := strings.Join(records, " | ")
	return fmt.Sprintf("%s", internal)
}
