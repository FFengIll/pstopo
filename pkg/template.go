package pkg

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

const tmplGraph = `digraph gocallvis {
    label="{{.Title}}";
    labeljust="l";
    fontname="Arial";
    fontsize="14";
    // rankdir="{{.Options.rankdir}}";
	rankdir="LR";
    bgcolor="lightgray";
    style="solid";
    penwidth="0.5";
    pad="0.0";
    // nodesep="{{.Options.nodesep}}";
    // node [shape="{{.Options.nodeshape}}" style="{{.Options.nodestyle}}" fillcolor="honeydew" fontname="Verdana" penwidth="1.0" margin="0.05,0.0"];
    // edge [minlen="{{.Options.minlen}}"]
	{{range .Nodes}}
	{{template "node" .}}
	{{- end}}
    {{- range .Edges}}
    {{template "edge" .}}
    {{- end}}
}`
