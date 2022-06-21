package main

import (
	"github.com/FFengIll/pstopo/pkg"
	"github.com/spf13/cobra"
)

var snapshotCmd = &cobra.Command{
	Use: "snapshot",
	Run: func(cmd *cobra.Command, args []string) {
		executeSnapshot()
	},
}

func executeSnapshot() {
	snapshot, err := pkg.TakeSnapshot(connectionKind)
	if err != nil {
		panic(err)
	}
	snapshot.DumpFile(snapshotFilepath)
}

var snapshotFilepath = ""

func init() {
	flags := snapshotCmd.PersistentFlags()
	flags.StringVarP(&snapshotFilepath, "output", "o", "", "cache snapshot to file")
	flags.StringVarP(&connectionKind, "kind", "k", "all", "connection kind")
}
