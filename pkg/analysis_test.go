package pkg

import (
	jsoniter "github.com/json-iterator/go"
	"io/ioutil"
	"testing"
)

func generateSnapshot() *Snapshot {
	cachedSnapshot := "./latest.json"
	snapshot := NewSnapshot()
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	data, _ := ioutil.ReadFile(cachedSnapshot)
	err := json.Unmarshal(data, snapshot)
	if err != nil {
		panic(err)
	}
	return snapshot
}

func TestAnalyseSnapshot(t *testing.T) {
	snapshot := generateSnapshot()
	filterOption := &FilterOption{
		Cmd: []string{"Elec"},
	}
	topo := AnalyseSnapshot(snapshot, filterOption)
	println(topo)
}
