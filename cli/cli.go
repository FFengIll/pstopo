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
		snapshotFiltered := pkg.FilterSnapshot(filterOption, snapshot)
		topo := pkg.AnalyseSnapshot(snapshotFiltered)
		render, _ := pkg.NewRender(topo, snapshotFiltered)
		render.Write()
	},
}

var cachedSnapshot = ""

func init() {
	rootCmd.AddCommand(snapshotCmd)

	flags := rootCmd.PersistentFlags()
	flags.StringVarP(&cachedSnapshot, "snapshot", "s", "snapshot.json", "local cached snapshot file path")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
