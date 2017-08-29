//   This Source Code Form is subject to the terms of the Mozilla Public
//   License, v. 2.0. If a copy of the MPL was not distributed with this
//   file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

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

	case "build", "bld":
		cmd.build(cmd.restArgs())

	case "run":
		cmd.run(cmd.restArgs())

	case "serve", "srv":
		cmd.serve(cmd.restArgs())

	default:
		cmd.printUsage()
	}
}

func (cmd *Simon) restArgs() []string { return cmd.Args()[1:] }

func (cmd *Simon) define(args []string) {
	job := cmd.defineFromReader(os.Stdin)
	fmt.Print(job.Dir)
}

func (cmd *Simon) build(args []string) {
	var job LocatedJobSpec
	defer fmt.Print(job.Dir)
	job = cmd.builtJob(args)
}

func (cmd *Simon) run(args []string) {
	var (
		fset  flag.FlagSet
		build bool
	)
	fset.BoolVar(&build, "build", true, "whether to build the job before running it")
	fset.Parse(args)

	var job LocatedJobSpec
	defer fmt.Print(job.Dir)

	if build {
		job = cmd.builtJob(fset.Args())
	} else {
		log.Printf("skipping build...")
		job = cmd.loadedJob(fset.Args())
	}

	job.Init()
	job.Run(context.Background())
}

func (cmd *Simon) printUsage() {
	cmd.PrintDefaults()
}

func (cmd *Simon) builtJob(args []string) LocatedJobSpec {
	job := cmd.definedJob(args)
	// TODO: Maybe make it so we don't have to rebuild the job each time?
	job.Build()
	return job
}

func (cmd *Simon) definedJob(args []string) LocatedJobSpec {
	assert.That(len(args) < 2, log.Fatalf, "too many arguments (%d) -- command supports upto 1 job path", len(args))

	var job LocatedJobSpec
	if len(args) == 1 {
		job = cmd.loadedJob(args)
	} else {
		job = cmd.defineFromReader(os.Stdin)
	}
	return job
}

func (cmd *Simon) loadedJob(args []string) LocatedJobSpec {
	assert.That(len(args) == 1, log.Fatalf, "wrong number of arguments (%d) -- command requires 1 job path", len(args))
	return cmd.loadDef(args[0])
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
	assert.That(err == nil, log.Fatalf, "can't decode spec file %q: %s", spec.prefixPath(specFileName), err)

	return spec
}

func (cmd *Simon) freshPath() string {
	now := time.Now().UTC()
	day := now.Format("2006-01-02")
	hour := now.Format("15:04:05.000000000")
	return filepath.Join(cmd.simondir, day, hour)
}
