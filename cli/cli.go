package main

import (
	"fmt"
	"github.com/FFengIll/pstopo/pkg"
	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
)

var rootCmd = &cobra.Command{
	Use: "root",
	Run: func(cmd *cobra.Command, args []string) {
		filterOption := pkg.NewGroup()
		//log := logrus.StandardLogger()
		var snapshot *pkg.Snapshot
		if cachedSnapshot == "" {
			snapshot, _ = pkg.TakeSnapshot()
		} else {
			var json = jsoniter.ConfigCompatibleWithStandardLibrary
			data, _ := ioutil.ReadFile(cachedSnapshot)
			err := json.Unmarshal(data, &snapshot)
			if err != nil {
				panic(err)
			}
		}
		var topo *pkg.PSTopo
		if topoConfig != "" {
			var json = jsoniter.ConfigCompatibleWithStandardLibrary
			data, _ := ioutil.ReadFile(topoConfig)
			err := json.Unmarshal(data, &filterOption)
			if err != nil {
				panic(err)
			}
		} else {
			filterOption.All = true
		}
		topo = pkg.AnalyseSnapshot(snapshot, filterOption)
		render, _ := pkg.NewRender(topo, snapshot)
		render.Write()
	},
}

var cachedSnapshot = ""
var topoConfig = ""

func init() {
	rootCmd.AddCommand(snapshotCmd)

	flags := rootCmd.PersistentFlags()
	flags.StringVarP(&cachedSnapshot, "snapshot", "s", "snapshot.json", "local cached snapshot file path")
	flags.StringVarP(&topoConfig, "topo", "t", "topo.json", "local topo config file path")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
