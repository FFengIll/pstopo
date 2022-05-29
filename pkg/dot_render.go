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
	Label *dotLabel
	Attrs dotAttrs
}

func (n *dotNode) AddPart(id int, part string) {
	n.Label.parts[id] = part
}

type dotEdge struct {
	From  string
	To    string
	Attrs dotAttrs
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

func (this *Render) Data() *dotGraphData {
	topo := this.topo

	var nodes []*dotNode
	for _, n := range topo.Nodes {
		node := &dotNode{
			ID:    "n" + strconv.Itoa(int(n.Pid)),
			Label: NewDotLabel(),
			Attrs: dotAttrs{
				"shape": "record",
			},
		}
		//node.AddPart(0, strconv.Itoa(int(n.Pid)))
		//cmdline := strings.Split(n.Cmdline, " ")[0]
		paths := strings.Split(n.Exec, string(os.PathSeparator))
		node.AddPart(1, paths[len(paths)-1])

		for _, edge := range topo.NetworkEdges {
			if edge.Connnetcion.Pid == n.Pid {
				port := edge.Connnetcion.Laddr.Port
				node.AddPart(int(port), ":"+strconv.Itoa(int(port)))
			}
		}

		nodes = append(nodes, node)
	}

	var edges []*dotEdge
	for _, e := range topo.ProcessEdges {
		edge := &dotEdge{
			From: "n" + strconv.Itoa(int(e.From)),
			To:   "n" + strconv.Itoa(int(e.To)),
		}
		edges = append(edges, edge)
	}
	for _, e := range topo.NetworkEdges {
		edge := &dotEdge{
			From: "n" + strconv.Itoa(int(e.From)),
			To:   "n" + strconv.Itoa(int(e.To)),
		}
		edges = append(edges, edge)
	}
	for _, e := range topo.PublicNetworkEdges {
		edge := &dotEdge{
			From: "n" + strconv.Itoa(int(e.From)),
			To:   "n" + strconv.Itoa(int(e.To)),
		}
		edges = append(edges, edge)
	}

	return &dotGraphData{
		Title: "test",
		Nodes: nodes,
		Edges: edges,
	}
}

func NewDotLabel() *dotLabel {
	return &dotLabel{
		parts: map[int]string{},
	}
}
