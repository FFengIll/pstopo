package pkg

import (
	"io/ioutil"
	"testing"

	jsoniter "github.com/json-iterator/go"
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
	filterOption := &Config{
		Cmd: []string{"Elec"},
	}
	topo := AnalyseSnapshot(snapshot, filterOption)
	println(topo)
}
