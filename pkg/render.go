package pkg

type Render interface {
	Write(topo *PSTopo, output string) error
}
