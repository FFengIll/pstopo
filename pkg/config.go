package pkg

import (
	"io/ioutil"

	jsoniter "github.com/json-iterator/go"
)

type Config struct {
	All  bool     `json:"all" default:"false"`
	Cmd  []string `json:"cmd"`
	Port []uint32 `json:"port"`
	Pid  []int32  `json:"pid"`
}

func NewConfig() *Config {
	return &Config{
		All:  false,
		Cmd:  []string{},
		Port: []uint32{},
		Pid:  []int32{},
	}
}

func (c *Config) WriteTo(path string) {
	var json = jsoniter.ConfigCompatibleWithStandardLibrary

	data, err := json.Marshal(c)
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(path, data, 0555)
	if err != nil {
		panic(err)
	}
}
