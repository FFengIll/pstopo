package pkg

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/goccy/go-graphviz"
	"github.com/goccy/go-graphviz/cgraph"
	"strconv"
	"text/template"
)

type Render struct {
	g     *graphviz.Graphviz
	graph *cgraph.Graph
	topo  *PSTopo
}

type ProcessNode Process

type ProcessEdge struct {
	From   int32
	To     int32
	where  uint32
	where2 uint32
}

type dotGraphData struct {
	Title  string
	Minlen uint
	//Attrs   dotAttrs
	//Cluster *dotCluster
	Nodes   []*ProcessNode
	Edges   []*ProcessEdge
	Options map[string]string
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
		return
	}
}

func (this *Render) Data() *dotGraphData {
	var nodes = []*ProcessNode{}
	topo := this.topo
	for _, n := range topo.Nodes {
		node := (*ProcessNode)(n)
		nodes = append(nodes, node)
	}

	var edges = []*ProcessEdge{}
	for _, e := range topo.NetworkEdges {
		edges = append(edges, e)
	}

	return &dotGraphData{
		Title: "test",
		Nodes: nodes,
		Edges: edges,
	}
}
