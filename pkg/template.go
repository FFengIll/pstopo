package pkg

const tmpLegend = `{{define "legend" -}}
 subgraph cluster_legend { 
    label = "Legend";
	graph [shape=box, fontsize=25]
	node [shape="box"]
    process->socket [color=green, label="socket connection"]
    process->ip_port [color=blue, label="connection to ip"]
    process->child_pid [color=red, label="process hierarchy"]
    process [label="executable | pid, e.g. 23333 | colon port, e.g. :8080", shape=record]
  }
{{- end}}`

const tmplCluster = `{{define "cluster" -}}
    {{printf "subgraph %q {" .}}
        {{printf "%s" .Attrs.Lines}}
        {{range .Nodes}}
        {{template "node" .}}
        {{- end}}
        {{range .Clusters}}
        {{template "cluster" .}}
        {{- end}}
    {{println "}" }}
{{- end}}`

const tmplEdge = `{{define "edge" -}}
    {{printf "%s" .}}
{{- end}}`

const tmplNode = `{{define "node" -}}
    {{printf "%s" .}}
{{- end}}`

const tmplGraph = `digraph pstopo {
    label="{{.Title}}";
    labeljust="t";
	labelloc=t;
    fontname="Arial";
    fontsize="25";
    // rankdir="{{.Options.rankdir}}";
	rankdir="LR";
    bgcolor="lightgray";
    style="solid";
    penwidth="0.5";
    pad="0.0";
    // nodesep="{{.Options.nodesep}}";
    // node [shape="{{.Options.nodeshape}}" style="{{.Options.nodestyle}}" fillcolor="honeydew" fontname="Verdana" penwidth="1.0" margin="0.05,0.0"];
    // edge [minlen="{{.Options.minlen}}"]
	
	{{template "legend" .}}

	{{range .Nodes}}
	{{template "node" .}}
	{{- end}}

    {{- range .Edges}}
    {{template "edge" .}}
    {{- end}}
}`
