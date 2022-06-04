package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/FFengIll/pstopo/pkg"
	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:  "root",
	Args: cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		filterOption := pkg.NewFilterOption()
		//log := logrus.StandardLogger()
		var snapshot *pkg.Snapshot
		if snapshotPath == "" {
			// if no given snapshot, then generate a new one
			snapshot, _ = pkg.TakeSnapshot()
			snapshot.DumpFile(snapshotPath)
		} else {
			file, err := os.Open(snapshotPath)
			defer file.Close()
			if errors.Is(err, os.ErrNotExist) {
				snapshot, _ = pkg.TakeSnapshot()
				snapshot.DumpFile(snapshotPath)
			} else {
				var json = jsoniter.ConfigCompatibleWithStandardLibrary
				data, _ := ioutil.ReadFile(snapshotPath)
				err := json.Unmarshal(data, &snapshot)
				if err != nil {
					panic(err)
				}
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
			if len(args) <= 0 {
				filterOption.All = true
			} else {
				filterOption.All = false
			}
		}

		// add filter options from cli
		for _, arg := range args {
			// :xx as port
			// yy as cmdline
			if strings.HasPrefix(arg, ":") {
				port, _ := strconv.Atoi(arg[1:])
				filterOption.Port = append(filterOption.Port, uint32(port))
			} else {
				filterOption.Cmd = append(filterOption.Cmd, arg)
			}
		}

		topo = pkg.AnalyseSnapshot(snapshot, filterOption)
		render, _ := pkg.NewRender(snapshot, topo)
		render.Write(outputPath)
	},
}

var snapshotPath = ""
var topoConfig = ""
var outputPath = ""

func init() {
	rootCmd.AddCommand(snapshotCmd)

	flags := rootCmd.PersistentFlags()
	flags.StringVarP(&snapshotPath, "snapshot", "s", "", "local cached snapshot file path")
	flags.StringVarP(&topoConfig, "topo", "t", "", "local topo config file path")
	flags.StringVarP(&outputPath, "output", "o", "output.dot", "output file path")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
