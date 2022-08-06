package pkg

import (
	"sync"

	sets "github.com/deckarep/golang-set"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type PortSet struct {
	internal sets.Set
	sync.Once
}

func NewPortSet() *PortSet {
	return &PortSet{
		internal: sets.NewSet(),
	}
}

func (set *PortSet) Iter() <-chan uint32 {
	ch := make(chan uint32)
	go func() {
		for elem := range set.internal.Iter() {
			ch <- elem.(uint32)
		}
		close(ch)
	}()

	return ch
}

func (set *PortSet) Add(port uint32) bool {
	return set.internal.Add(port)
}

func (set *PortSet) MarshalJSON() ([]byte, error) {
	var array []uint32
	for item := range set.Iter() {
		array = append(array, item)
	}
	return json.Marshal(array)
}

func (set *PortSet) UnmarshalJSON(data []byte) error {
	var array []uint32
	err := json.Unmarshal(data, &array)
	if err != nil {
		return err
	}
	if set.internal == nil {
		set.internal = sets.NewSet()
	}
	for _, item := range array {
		set.internal.Add(item)
	}
	return nil
}
