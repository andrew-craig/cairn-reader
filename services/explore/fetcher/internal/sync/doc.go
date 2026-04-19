// Package sync provides functionality for synchronizing feed sources
// with a curated feed list. Operators can supply their own list by
// mounting a file into the container; otherwise a default list
// maintained in this repository is fetched over HTTP. New feeds are
// imported into the fetcher database while existing feed state is
// preserved.
package sync
