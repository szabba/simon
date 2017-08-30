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

	"github.com/pkg/errors"

	"github.com/szabba/msc-thesis/assert"
)

func main() {
	var simon Simon
	simon.Main(os.Args)
}

type Simon struct {
	flag.FlagSet
	simondir Simondir
}

func (cmd *Simon) Main(args []string) {
	cmd.simondir.Set(".simon")
	cmd.Var(&cmd.simondir, "simondir", "")
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
	job, err := cmd.defineFromReader(os.Stdin)
	if err != nil {
		log.Printf("%s", err)
		return
	}
	fmt.Print(cmd.simondir.NormalizeJobPath(job))
}

func (cmd *Simon) build(args []string) {
	var (
		job LocatedJobSpec
		err error
	)
	defer fmt.Printf("%s", cmd.simondir.NormalizeJobPath(job))
	job, err = cmd.builtJob(args)
	assert.That(err == nil, log.Fatalf, "%s", err)
}

func (cmd *Simon) run(args []string) {
	var (
		fset  flag.FlagSet
		build bool
	)
	fset.BoolVar(&build, "build", true, "whether to build the job before running it")
	fset.Parse(args)

	var (
		job LocatedJobSpec
		err error
	)
	defer fmt.Printf("%s", cmd.simondir.NormalizeJobPath(job))

	if build {
		job, err = cmd.builtJob(fset.Args())
		assert.That(err == nil, log.Fatalf, "%s", err)
	} else {
		log.Printf("skipping build...")
		job, err = cmd.loadedJob(fset.Args())
		assert.That(err == nil, log.Fatalf, "%s", err)
	}

	err = job.Init(context.Background(), cmd.simondir)
	assert.That(err == nil, log.Fatalf, "%s", err)

	err = job.Run(context.Background(), cmd.simondir)
	assert.That(err == nil, log.Fatalf, "%s", err)
}

func (cmd *Simon) printUsage() {
	cmd.PrintDefaults()
}

func (cmd *Simon) builtJob(args []string) (LocatedJobSpec, error) {
	job, err := cmd.definedJob(args)
	if err == nil {
		err = job.Build(context.Background(), cmd.simondir)
	}
	return job, errors.Wrap(err, "build failed")
}

func (cmd *Simon) definedJob(args []string) (LocatedJobSpec, error) {
	assert.That(len(args) < 2, log.Fatalf, "too many arguments (%d) -- command supports upto 1 job path", len(args))

	var (
		job LocatedJobSpec
		err error
	)
	if len(args) == 1 {
		job, err = cmd.loadedJob(args)
	} else {
		job, err = cmd.defineFromReader(os.Stdin)
	}
	return job, err
}

func (cmd *Simon) loadedJob(args []string) (LocatedJobSpec, error) {
	assert.That(len(args) == 1, log.Fatalf, "wrong number of arguments (%d) -- command requires 1 job path", len(args))
	var job LocatedJobSpec
	err := job.Load(cmd.simondir, args[0])
	return job, err
}

func (cmd *Simon) defineFromReader(r io.Reader) (LocatedJobSpec, error) {
	var spec LocatedJobSpec

	spec.ReadCommands(os.Stdin)
	err := spec.PopulateVersion()

	if err == nil {
		spec = spec.Locate(cmd.simondir.FreshJobPath())
	}

	if err == nil {
		err = spec.EnsureLocationExists()
	}

	if err == nil {
		err = spec.StoreDefinition(cmd.simondir)
	}

	if err == nil {
		log.Printf("created job defintion at %q", spec.Dir)
	}

	return spec, errors.Wrapf(err, "can't define job")
}

func (job *LocatedJobSpec) Load(simondir Simondir, name string) error {
	job.Dir = name

	specPath := simondir.InJob(*job, specFileName)
	specFile, err := os.Open(specPath)
	err = errors.Wrap(err, "can't open job spec")

	if err == nil {
		defer specFile.Close()
		dec := json.NewDecoder(specFile)
		err = dec.Decode(&job.JobSpec)
		err = errors.Wrapf(err, "can't decode spec file %q", specPath)
	}

	return errors.Wrapf(err, "can't load job %q", name)
}
