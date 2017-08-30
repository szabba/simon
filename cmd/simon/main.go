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

	"github.com/pkg/errors"

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
	job, err := cmd.defineFromReader(os.Stdin)
	if err != nil {
		log.Printf("%s", err)
		return
	}
	fmt.Print(job.Name(cmd.simondir))
}

func (cmd *Simon) build(args []string) {
	var (
		job LocatedJobSpec
		err error
	)
	defer fmt.Print(job.Name(cmd.simondir))
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
	defer fmt.Print(job.Name(cmd.simondir))

	if build {
		job, err = cmd.builtJob(fset.Args())
		assert.That(err == nil, log.Fatalf, "%s", err)
	} else {
		log.Printf("skipping build...")
		job, err = cmd.loadedJob(fset.Args())
		assert.That(err == nil, log.Fatalf, "%s", err)
	}

	err = job.Init(context.Background())
	assert.That(err == nil, log.Fatalf, "%s", err)

	err = job.Run(context.Background())
	assert.That(err == nil, log.Fatalf, "%s", err)
}

func (cmd *Simon) printUsage() {
	cmd.PrintDefaults()
}

func (cmd *Simon) builtJob(args []string) (LocatedJobSpec, error) {
	job, err := cmd.definedJob(args)
	if err == nil {
		err = job.Build(context.Background())
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
	return cmd.loadDef(args[0])
}

func (cmd *Simon) defineFromReader(r io.Reader) (LocatedJobSpec, error) {
	var spec LocatedJobSpec

	spec.ReadCommands(os.Stdin)
	err := spec.PopulateVersion()

	if err == nil {
		spec = spec.Locate(cmd.freshPath())
	}

	if err == nil {
		err = spec.EnsureLocationExists()
	}

	if err == nil {
		err = spec.StoreDefinition()
	}

	if err == nil {
		log.Printf("created job defintion at %q", spec.Dir)
	}

	return spec, errors.Wrapf(err, "can't define job")
}

func (cmd *Simon) loadDef(dir string) (LocatedJobSpec, error) {

	if !filepath.IsAbs(dir) {
		dir = filepath.Join(cmd.simondir, dir)
	}
	spec := LocatedJobSpec{Dir: dir}

	specPath := spec.prefixPath(specFileName)
	specFile, err := os.Open(specPath)
	err = errors.Wrapf(err, "can't open job spec")

	if err == nil {
		defer specFile.Close()
		dec := json.NewDecoder(specFile)
		err = errors.Wrapf(dec.Decode(&spec.JobSpec), "can't decode spec file %q", specPath)
	}

	return spec, errors.Wrapf(err, "can't load job %q", spec.Dir)
}

func (cmd *Simon) freshPath() string {
	now := time.Now().UTC()
	day := now.Format("2006-01-02")
	hour := now.Format("15:04:05.000000000")
	return filepath.Join(cmd.simondir, day, hour)
}
