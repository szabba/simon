# Simon

<p>
Simon is a tool for keeping simulation results organized.
A <b>sim</b>ulation <b>mon</b>itor.
</p>

> ⚠ **CAVEAT**
>
> Simon is was written in about two days with a very specific workflow in mind.
> In all likelihood it's brittle and won't do what you think it ought to.

Simon thinks of simulations as three-step jobs consisting of
* a _build_ phase -- where code is compiled
* an _init_ phase -- where initial conditions data is prepared
* a _run_ phase -- where the actual simulation is run

Each phase is specified as a command to be run in the shell.
When Simon runs a phase it captures the stdout and stderr and stores it on disk.
The output of the init phase command is fed to the run phase command.

The results are stored in the simondir.
By default this is the `.simon` subdirectory of the current working directory.
It can be changed by supplying a `-simondir` flag before the simon subcommand.

A job's name is a directory path.
The job spec `job_spec.json` and phase stdout / stderr logs are place in it.

> ⚠
>
> It might be a better idea to do what `git` does with `.git` directories.
> For now, that's too much work though.

Along the phase commands, Simon records the current git revision hash and diff.

> ⚠
>
> There could be a command to restore the working tree state of a job.
> There isn't one yet. 

## Command reference

### `simon def`

Defines a job.
Reads the phase commands from stdin, each from one line.
They are read in the order they are to be run.

It outputs the path of the job directory, possibly relative the current one.

### `simon bld`

An alias for `simon build`.

### `simon build [job]`

Runs the build phase of the specified job.
If a job name is not given, it will first accept a definition like `simon def`.

It outputs the path of the job directory, possibly relative the current one.

### `simon run [job]`

Runs all the phases of the specified job.
If a job name is not given, it will first accept a definition like `simon def`.

If the `-build` flag is set to false, it skips the build phase.

It outputs the path of the job directory, possibly relative the current one.

### `simon serve`

Runs a simple job scheduling service.
It runs up to a certain number of jobs concurrently.
When it could run more it periodically tries to get a job from a queue file.

The queue file is understood as a list of job paths, each on a separate line.
Before a job is started it's removed from the queue file.

> ⚠
>
> It's completely possible for a job to be dropped if the server crashes between
> removing it from the queue file and starting it.

If the `-build` flag is set to false, the server won't run the build phase of the
jobs.

The `-tick` flag controls how often the server will try to check for new jobs in
the queue when it could be running more jobs.
It defaults to 100ms.

The `-queue` flag controls the path to the queue file.
If it's relative, it's joined with the path to the simondir.

The `-max-jobs` flag controls how many concurrent jobs the server will try to
run.
It defaults to `runtime.GOMAXPROCS(0)`.

### `simon srv`

An alias for `simon serve`.