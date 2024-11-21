package pkg

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/goccy/go-graphviz"
	"github.com/sirupsen/logrus"
)

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

type dotAttrs map[string]string

func (p dotAttrs) List() []string {
	var l []string
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

func makeDotPortLabel(label string, dotPort string) string {
	return fmt.Sprintf("<p%s> %s", dotPort, label)
}

func ItoDotPort(port uint32) string {
	// FIXME: for port == 0, we process it as `no dot node port`
	if port == 0 {
		return ""
	}
	return StoDotPort(strconv.Itoa(int(port)))
}

func StoDotPort(port string) string {
	return ":" + "p" + port
}

func toDotId(pid int32) string {
	return "n" + strconv.Itoa(int(pid))
}

func makeDotLabel(parts map[int]string, items ...string) string {
	var records = items
	for id, label := range parts {
		records = append(records, makeDotPortLabel(label, strconv.Itoa(id)))
	}

	internal := strings.Join(records, " | ")
	return internal
}

type dotGraphData struct {
	Title string
	// Attrs   dotAttrs
	// Cluster *dotCluster
	Nodes   []*dotNode
	Edges   []*dotEdge
	Options map[string]string
}

type DotRender struct {
	Render
	engine *graphviz.Graphviz
}

func NewDotRender() (Render, error) {
	g, err := graphviz.New(context.Background())
	if err != nil {
		return nil, err
	}
	return &DotRender{engine: g}, nil
}

func (r *DotRender) writeData(data *dotGraphData, output string) {
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

	logrus.Debugln(buf.String())
	graph, err := graphviz.ParseBytes(buf.Bytes())
	if err != nil {
		panic(err)
	}
	// FIXME: use parsed graph, not r.graph
	if !strings.HasSuffix(output, ".dot") {
		output = output + ".dot"
	}

	// output dot file
	os.WriteFile(output, buf.Bytes(), os.ModePerm)

	// if err := r.engine.RenderFilename(graph, graphviz.Format(graphviz.DOT), output); err != nil {
	// 	//fd, _ := os.Open(output)
	// 	//fd.WriteString(err.Error())
	// 	logrus.Errorln(err)
	// 	logrus.Errorln("parse graph error, but try to output the file")
	// 	return
	// }

	if err := r.engine.RenderFilename(context.Background(), graph, graphviz.PNG, output+".png"); err != nil {
		// fd, _ := os.Open(output)
		// fd.WriteString(err.Error())
		logrus.Errorln(err)
		logrus.Errorln("parse graph error, but try to output the file")
		return
	}
}

func (r *DotRender) toData(topo *PSTopo) (*dotGraphData, error) {
	// create node
	var nodes []*dotNode
	for _, n := range topo.PidSet {
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
		{
			// FIXME: add default port for 0
			// parts[int(0)] = ":" + strconv.Itoa(int(0))

			set, ok := topo.Snapshot.PidPort[n.Pid]
			if ok {
				for port := range set.Iter() {
					// if contains(related, port) {
					parts[int(port)] = ":" + strconv.Itoa(int(port))
					// }
				}
			}
		}

		// put listen bellow to avoid overwrite
		{
			set, ok := topo.Snapshot.PidListenPort[n.Pid]
			if ok {
				for port := range set.Iter() {
					// if contains(related, port) {
					parts[int(port)] = "Listen " + ":" + strconv.Itoa(int(port))
					// }
				}
			}
		}

		paths := strings.Split(n.Exec, string(os.PathSeparator))
		pidLabel := makeDotPortLabel(strconv.Itoa(int(n.Pid)), "p")
		label := makeDotLabel(parts, paths[len(paths)-1], pidLabel)
		node.Label = label

		nodes = append(nodes, node)
	}

	// generate edge data
	var edges []*dotEdge
	for _, e := range topo.PidChildSet {
		edge := newDotEdge()
		edge.From = toDotId(e.From) + StoDotPort("p")
		edge.To = toDotId(e.To) + StoDotPort("p")
		edge.Attrs["label"] = ""
		edge.Attrs["color"] = "red"
		edges = append(edges, edge)
	}
	for _, e := range topo.PidConnSet {
		edge := newDotEdge()
		edge.From = toDotId(e.From) + ItoDotPort(e.Connection.Laddr.Port)
		edge.To = toDotId(e.To) + ItoDotPort(e.Connection.Raddr.Port)
		edge.Attrs["label"] = ""
		edge.Attrs["color"] = "darkgreen"
		edge.Attrs["dir"] = "both"
		edges = append(edges, edge)
	}
	for _, e := range topo.IPConnSet {
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
		edge.From = toDotId(e.From) + ItoDotPort(e.Connection.Laddr.Port)
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

func (r *DotRender) Write(topo *PSTopo, output string) error {
	data, err := r.toData(topo)
	if err != nil {
		return err
	}
	r.writeData(data, output)
	return nil
}
