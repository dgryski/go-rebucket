package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"

	berrors "github.com/bugsnag/bugsnag-go/errors"
	"github.com/dgryski/go-rebucket"
)

func main() {

	dir := flag.String("d", ".", "directory containing stack traces")

	flag.Parse()

	files, err := filepath.Glob(*dir + "/*")
	if err != nil {
		log.Fatalf("error globbing: %v\n", err)
	}

	var errs []*berrors.Error

	var fnames []string

	for _, f := range files {
		r, err := ioutil.ReadFile(f)
		if err != nil {
			log.Printf("error reading %s: %v\n", f, err)
			continue
		}

		e, err := berrors.ParsePanic(string(r))
		if err != nil {
			log.Printf("error parsing panic %s: %v\n", f, err)
			continue
		}
		errs = append(errs, e)
		fnames = append(fnames, f)
	}

	// TODO(dgryski): I made up these numbers.  They "should" be computed
	// from a human tagged corpus via algorithm 6 in
	// http://research.microsoft.com/en-us/groups/sa/rebucket-icse2012.pdf
	groups := rebucket.ClusterErrors(errs, 1, 1, 1)

	fmt.Printf("groups=%v\n", groups)
	for i, g := range groups {
		fmt.Println("cluster", i)

		for _, idx := range g.Idx {
			fmt.Println("\t", fnames[idx])
		}
		fmt.Println()
	}
}
