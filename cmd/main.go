package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/FFengIll/pstopo/pkg"
)

var fs afero.Fs

func existFile(path string) bool {
	// exist
	if ok, _ := afero.Exists(fs, path); !ok {
		return false
	}

	// id dir
	if ok, _ := afero.DirExists(fs, path); ok {
		return false
	}

	return true
}

var rootCmd = &cobra.Command{
	Use:  "root",
	Args: cobra.ArbitraryArgs,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if verbose {
			logrus.SetLevel(logrus.DebugLevel)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		config := pkg.NewConfig()

		if outputName != "" && snapshotPath == "" {
			snapshotPath = fixSnapshotPath(outputName)
			logrus.WithField("snapshot", snapshotPath).Infoln("set default snapshot")
		}

		var snapshot *pkg.Snapshot
		if !existFile(snapshotPath) {
			logrus.WithField("snapshot", snapshotPath).Infoln("no snapshot existed, take one")
			// if no given snapshot, then generate a new one
			snapshot, _ = pkg.TakeSnapshot(connectionKind)
			snapshot.DumpFile(snapshotPath)
		} else {
			var json = jsoniter.ConfigCompatibleWithStandardLibrary
			data, _ := ioutil.ReadFile(snapshotPath)
			err := json.Unmarshal(data, &snapshot)
			if err != nil {
				panic(err)
			}
		}

		var topo *pkg.PSTopo
		if !existFile(configPath) {
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

		topo = pkg.NewTopo(snapshot)
		topo = topo.Analyse(config)
		render, _ := pkg.NewDotRender()
		render.Write(topo, outputName)
	},
}

func fixSnapshotPath(name string) string {
	if !strings.HasSuffix(name, ".snapshot.json") {
		res := name + ".snapshot.json"
		return res
	} else {
		return name
	}
}

func init() {
	fs = afero.NewOsFs()

	rootCmd.AddCommand(snapshotCmd)
	rootCmd.AddCommand(reloadCmd)

	flags := rootCmd.PersistentFlags()
	flags.StringVarP(&snapshotPath, "snapshot", "s", "", "local cached snapshot file path")
	flags.StringVarP(&configPath, "config", "c", "", "local topo config file path")
	flags.StringVarP(&outputName, "output", "o", "output.dot", "output file name")
	flags.StringVarP(&connectionKind, "kind", "k", "all", "connection kind")
	flags.BoolVarP(&verbose, "verbose", "v", false, "verbose with debug info")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
