package pkg

import (
	"os"
	"testing"

	jsoniter "github.com/json-iterator/go"
)

func generateSnapshot() *Snapshot {
	cachedSnapshot := "./latest.json"
	snapshot := NewSnapshot()
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	data, _ := os.ReadFile(cachedSnapshot)
	err := json.Unmarshal(data, snapshot)
	if err != nil {
		panic(err)
	}
	return snapshot
}

func TestAnalyseSnapshot(t *testing.T) {
	snapshot := generateSnapshot()
	cfg := &Config{
		Cmd: []string{"Elec"},
	}
	topo := NewTopo(snapshot)
	topo = topo.Analyse(cfg)
	println(topo)
}
