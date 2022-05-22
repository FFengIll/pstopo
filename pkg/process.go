package pkg

type Process struct {
	Pid      int32   `json:"pid"`
	Cmdline  string  `json:"cmdline"`
	Parent   int32   `json:"parent"`
	Children []int32 `json:"children"`
}
