//   This Source Code Form is subject to the terms of the Mozilla Public
//   License, v. 2.0. If a copy of the MPL was not distributed with this
//   file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"os"

	"github.com/szabba/msc-thesis/assert"
)

func main() {
	var simon Simon
	simon.Main(os.Args)
}

type Simon struct {
	flag.FlagSet
	simondir string
}

func (cmd *Simon) Main(args []string) {
	cmd.StringVar(&cmd.simondir, "simondir", ".simon", "")
	cmd.Parse(args[1:])

	switch cmd.Arg(0) {
	case "def":
		cmd.define(cmd.restArgs())
	case "run":
		cmd.run(cmd.restArgs())
	default:
		cmd.printUsage()
	}
}

func (cmd *Simon) restArgs() []string { return cmd.Args()[1:] }

func (cmd *Simon) define(args []string) {
	cmd.defineFromReader(os.Stdin)
}

func (cmd *Simon) run(args []string) {
	var job LocatedJobSpec
	if len(args) == 1 {
		job = cmd.loadDef(args[0])
	} else {
		job = cmd.defineFromReader(os.Stdin)
	}
	job.Build()
	job.Init()
	job.Run()
}

func (cmd *Simon) printUsage() {
	cmd.PrintDefaults()
}

func (cmd *Simon) defineFromReader(r io.Reader) LocatedJobSpec {
	var spec JobSpec

	spec.ReadCommands(os.Stdin)
	spec.PopulateVersion()

	located := spec.Locate(cmd.freshPath())

	located.EnsureLocationExists()
	located.StoreDefinition()

	log.Printf("created job defintion at %q", located.Dir)

	return located
}

func (cmd *Simon) loadDef(dir string) LocatedJobSpec {
	spec := LocatedJobSpec{Dir: dir}

	specFile := spec.open(specFileName)
	defer specFile.Close()

	dec := json.NewDecoder(specFile)
	err := dec.Decode(&spec.JobSpec)
	assert.That(err == nil, log.Fatalf, "can't decode spec file %q: %s", spec.specFilePath(), err)

	return spec
}

func (cmd *Simon) freshPath() string { return freshPath(cmd.simondir) }
