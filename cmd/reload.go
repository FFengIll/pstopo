package main

import (
	"github.com/FFengIll/pstopo/pkg"
	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/cobra"
	"io/ioutil"
	"net"
	"strconv"
	"strings"
)

var reloadCmd = &cobra.Command{
	Use: "reload",
	Run: func(cmd *cobra.Command, args []string) {
		var json = jsoniter.ConfigCompatibleWithStandardLibrary

		reloadName := args[0]
		if reloadName == "" {
			panic("no given name")
		}

		snapshotPath = reloadName + ".snapshot.json"
		configPath = reloadName + ".topo.json"
		outputName = reloadName

		config := pkg.NewConfig()

		var snapshot *pkg.Snapshot
		data, _ := ioutil.ReadFile(snapshotPath)
		err := json.Unmarshal(data, &snapshot)
		if err != nil {
			panic(err)
		}

		var topo *pkg.PSTopo
		data, _ = ioutil.ReadFile(configPath)
		err = json.Unmarshal(data, &config)
		if err != nil {
			panic(err)
		}

		// add filter options from cli
		for _, arg := range args {
			// :xx as port
			// yy as cmdline
			if strings.HasPrefix(arg, ":") {
				port, _ := strconv.Atoi(arg[1:])
				config.Port = append(config.Port, uint32(port))
				continue
			}

			ip := net.ParseIP(arg)
			if ip != nil {

			}

			config.Cmd = append(config.Cmd, arg)
		}

		topo = pkg.AnalyseSnapshot(snapshot, config)
		render, _ := pkg.NewDotRender(snapshot, topo)
		render.WriteTo(outputName)
	},
}

var update = false

func init() {
	flags := reloadCmd.PersistentFlags()
	flags.BoolVarP(&update, "update", "u", false, "update related file if possible")
}
