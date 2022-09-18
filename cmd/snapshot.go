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
	snapshotPath = fixSnapshotPath(snapshotPath)
	snapshot.DumpFile(snapshotPath)
}

func init() {
	flags := snapshotCmd.PersistentFlags()
	flags.StringVarP(&snapshotPath, "output", "o", "", "cache snapshot to file")
}
