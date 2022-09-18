package main

import (
	"encoding/json"
	"io/ioutil"
	"net"
	"path"
	"strconv"
	"strings"

	mapset "github.com/deckarep/golang-set/v2"
	jsoniter "github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/FFengIll/pstopo/pkg"
)

var reloadCmd = &cobra.Command{
	Use: "reload",
	Run: func(cmd *cobra.Command, args []string) {
		var json = jsoniter.ConfigCompatibleWithStandardLibrary

		outputDir := args[0]
		if outputDir == "" {
			panic("no given name")
		}

		snapshotPath := path.Join(outputDir, "snapshot.json")
		configPath := path.Join(outputDir, "config.json")
		outputPath := path.Join(outputDir, "output.dot")

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
		// except args[0]
		for _, arg := range args[1:] {
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
		render.Write(topo, outputPath)
		if update {
			logrus.Infoln("overwrite snapshot")
			snapshot.DumpFile(snapshotPath)

			logrus.Infoln("overwrite config")
			dumpConfigFile(config, configPath)
		}
	},
}

func dumpConfigFile(config *pkg.Config, configPath string) {
	cmdSet := mapset.NewSet[string]()
	for _, c := range config.Cmd {
		cmdSet.Add(c)
	}
	config.Cmd = cmdSet.ToSlice()

	// dump config
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return
	}
	ioutil.WriteFile(configPath, data, 0660)

	logrus.Infof("config to: %s\n", configPath)
}

func init() {
	flags := reloadCmd.PersistentFlags()
	flags.BoolVarP(&update, "update", "w", false, "update and rewrite related file if possible")
}
