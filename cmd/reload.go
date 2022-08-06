package main

import (
	"io/ioutil"
	"net"
	"strconv"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/FFengIll/pstopo/pkg"
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

		snapshot := &pkg.Snapshot{}
		data, _ := ioutil.ReadFile(snapshotPath)
		err := json.Unmarshal(data, snapshot)
		if err != nil {
			panic(err)
		}

		config := pkg.NewConfig()
		if existFile(configPath) {
			data, _ = ioutil.ReadFile(configPath)
			err = json.Unmarshal(data, &config)
			if err != nil {
				panic(err)
			}
		} else {
			logrus.Warningln("no such config, use empty")
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

		var topo *pkg.PSTopo
		topo = pkg.NewTopo(snapshot)
		topo = topo.Analyse(config)
		render, _ := pkg.NewDotRender()
		render.Write(topo, outputName)
		if update {
			snapshot.DumpFile(snapshotPath)
		}
	},
}

var update = false

func init() {
	flags := reloadCmd.PersistentFlags()
	flags.BoolVarP(&update, "update", "u", false, "update related file if possible")
}
