package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/FFengIll/pstopo/pkg"
)

func existFile(path string) bool {
	file, err := os.Open(path)
	defer file.Close()
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	return true
}

var rootCmd = &cobra.Command{
	Use:  "root",
	Args: cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {

		config := pkg.NewConfig()

		var snapshot *pkg.Snapshot
		if snapshotPath == "" || !existFile(snapshotPath) {
			logrus.WithField("snapshot", snapshotPath).Infoln("no snapshot existed, take one")
			// if no given snapshot, then generate a new one
			snapshot, _ = pkg.TakeSnapshot(connectionKind)
			if outputName != "" {
				snapshot.DumpFile(outputName + ".snapshot.json")
			} else {
				snapshot.DumpFile(snapshotPath)
			}
		} else {
			var json = jsoniter.ConfigCompatibleWithStandardLibrary
			data, _ := ioutil.ReadFile(snapshotPath)
			err := json.Unmarshal(data, &snapshot)
			if err != nil {
				panic(err)
			}
		}

		var topo *pkg.PSTopo
		if configPath == "" || !existFile(configPath) {
			if len(args) <= 0 {
				config.All = true
			} else {
				config.All = false
			}
		} else {
			var json = jsoniter.ConfigCompatibleWithStandardLibrary
			data, _ := ioutil.ReadFile(configPath)
			err := json.Unmarshal(data, &config)
			if err != nil {
				panic(err)
			}
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
		render, _ := pkg.NewDotRender()
		render.Write(topo, outputName)
	},
}

var snapshotPath = ""
var configPath = ""
var outputName = ""
var connectionKind = ""

func init() {
	rootCmd.AddCommand(snapshotCmd)
	rootCmd.AddCommand(reloadCmd)

	flags := rootCmd.PersistentFlags()
	flags.StringVarP(&snapshotPath, "snapshot", "s", "", "local cached snapshot file path")
	flags.StringVarP(&configPath, "config", "c", "", "local topo config file path")
	flags.StringVarP(&outputName, "output", "o", "output.dot", "output file name")
	flags.StringVarP(&connectionKind, "connection-kind", "k", "all", "connection kind")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
