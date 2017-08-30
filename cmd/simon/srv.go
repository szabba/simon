//   This Source Code Form is subject to the terms of the Mozilla Public
//   License, v. 2.0. If a copy of the MPL was not distributed with this
//   file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"bytes"
	"context"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"time"

	"github.com/szabba/msc-thesis/assert"
)

func (cmd *Simon) serve(args []string) {
	srv := Server{Simon: cmd}
	srv.Run(args)
}

type Server struct {
	*Simon

	flag.FlagSet
	maxJobs   int
	build     bool
	tick      time.Duration
	queueFlag string
}

// MAIN PROGRAM + INIT

func (srv *Server) Run(args []string) {
	srv.BoolVar(&srv.build, "build", true, "whether to build the jobs before running them")
	srv.DurationVar(&srv.tick, "tick", 100*time.Millisecond, "how often to check the queue file")
	srv.StringVar(&srv.queueFlag, "queue", "queue", "path to queue file; if relative, resolved against the simnodir")
	srv.IntVar(&srv.maxJobs, "max-jobs", runtime.GOMAXPROCS(0), "the maximum number of concurrent jobs to process at once")

	srv.Parse(args)

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	requests := make(chan (chan<- LocatedJobSpec))

	if srv.maxJobs < 1 {
		srv.maxJobs = 1
	}

	wg.Add(srv.maxJobs + 2)

	srv.setupSignalHandler(&wg, cancel)

	go srv.watchQueue(ctx, &wg, requests)

	for i := 0; i < srv.maxJobs; i++ {
		go srv.worker(ctx, &wg, requests)
	}

	<-ctx.Done()
	wg.Wait()
}

func (srv *Server) setupSignalHandler(wg *sync.WaitGroup, cancel func()) {
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan)
	go srv.onSignal(wg, cancel, sigchan)
}

func (srv *Server) onSignal(wg *sync.WaitGroup, cancel func(), signals <-chan os.Signal) {
	defer wg.Done()
	for signal := range signals {
		switch signal {
		case os.Interrupt:
			cancel()
			return
		}
	}
}

// CONCURRENT ACTORS

func (srv *Server) watchQueue(ctx context.Context, wg *sync.WaitGroup, jobRequests <-chan (chan<- LocatedJobSpec)) {
	defer wg.Done()

	var reqs []chan<- LocatedJobSpec
	ticker := time.NewTicker(srv.tick)

	for {

		select {
		case <-ctx.Done():
			return
		default:
		}

		select {
		case req := <-jobRequests:
			reqs = append(reqs, req)

		case <-ticker.C:
			if len(reqs) == 0 {
				continue
			}

			job := srv.getJob()
			if job == nil {
				continue
			}

			reqs[0] <- *job

			copy(reqs[:len(reqs)-1], reqs[1:])
			reqs = reqs[:len(reqs)-1]

		case <-ctx.Done():
			return
		}
	}
}

func (srv *Server) worker(ctx context.Context, wg *sync.WaitGroup, requests chan<- (chan<- LocatedJobSpec)) {
	defer wg.Done()

	jobs := make(chan LocatedJobSpec)
	requests <- jobs

	for {

		select {
		case <-ctx.Done():
			return
		default:
		}

		select {
		case job := <-jobs:
			err := srv.do(ctx, job)
			assert.That(err == nil, log.Printf, "%s", err)

			requests <- jobs

		case <-ctx.Done():
			return
		}
	}
}

// SINGLE THREADED STUFF

func (srv *Server) getJob() *LocatedJobSpec {
	oldQueue, err := ioutil.ReadFile(srv.queuePath())
	if err != nil {
		log.Printf("can't read queuefile %q: %s", srv.queuePath(), err)
		err = ioutil.WriteFile(srv.queuePath(), []byte{}, 0644)
		assert.That(err == nil, log.Fatalf, "failed to create empty queuefile %q: %s", srv.queuePath(), err)
		return nil
	}

	if len(oldQueue) == 0 {
		return nil
	}

	jobs := bytes.Split(oldQueue, []byte("\n"))
	if len(jobs) == 0 {
		return nil
	}

	newQueue := bytes.Join(jobs[1:], []byte("\n"))
	err = ioutil.WriteFile(srv.queuePath(), newQueue, 0644)
	if err != nil {
		log.Printf("can't write back queuefile %q: %s", srv.queuePath(), err)
		return nil
	}

	var job LocatedJobSpec
	err = job.Load(srv.simondir, string(jobs[0]))
	if err != nil {
		log.Printf("%s", err)
		return nil
	}
	return &job
}

func (srv *Server) do(ctx context.Context, job LocatedJobSpec) error {

	if srv.build {

		log.Printf("job %q: build starts", job.Dir)
		err := job.Build(ctx, srv.simondir)
		if err != nil {
			return err
		}
		log.Printf("job %q: build done", job.Dir)

	} else {
		log.Printf("job %q: build skipped", job.Dir)
	}

	log.Printf("job %q: init starts", job.Dir)
	err := job.Init(ctx, srv.simondir)
	if err != nil {
		return err
	}
	log.Printf("job %q: init done", job.Dir)

	log.Printf("job %q: run starts", job.Dir)
	err = job.Run(ctx, srv.simondir)
	if err == nil {
		log.Printf("job %q: run done", job.Dir)
	}
	return err
}

func (srv *Server) queuePath() string { return srv.simondir.NormalizePath(srv.queueFlag) }
