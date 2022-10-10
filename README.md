# Resource Hog

I needed something to use resources to test Kubernetes scaling, so here it is.
This is a hack day project that has been hand tested, but doesn't have unit or
functional tests.

## Installation
`go get github.com/dooferlad/resourceHog`

## Run
Run the `./resourceHog` binary where you want resources to be used.

To control the hog, call `<server>:6776/` with one or more of the following query parameters

 * `time=<duration>`
 * `cpu=<number of CPUs to hog>`
 * `ram=<size>`
 * `disk_write=<size>`
 * `disk_read=<size>`
 * `response_size=<size>`

All sizes are parsed as standard computer units, so you can use MB, GiB etc.
Times are parsed using time.ParseDuration, so you can specify time with units
from nanoseconds up to hours.

The time parameter is needed for CPU and RAM hogs to tell the hog how much of those
resources to use, and for how long.

A note on RAM hogs - the code has been tested and RAM is cleaned up when we want
it to under test, but since Go doesn't allow direct memory management a change in
the runtime may change this. I wanted to keep this simple, so I haven't used an
alternative GC or called out to C code to malloc and free.