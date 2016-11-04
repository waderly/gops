// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Program gops is a tool to list currently running Go processes.
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"

	"hello/gops/agent/signal"
	"hello/gops/internal/objfile"

	ps "github.com/keybase/go-ps"
)

const helpText = `Usage: gops is a tool to list and diagnose Go processes.

  gops  Lists all Go processes currently running.

Advanced tools, requires gops agent:
       gops stack -p pid    Prints the stack trace of the process.
       gops gc -p pid       Runs the garbage collector.
       gops gcstats -p pid  Prints the GC stats of the process.
       gops help            Prints this help message.
`

var (
	pid     = flag.Int("p", -1, "")
	stack   = flag.Bool("stack", false, "")
	gc      = flag.Bool("gc", false, "")
	gcstats = flag.Bool("gcstats", false, "")
	help    = flag.Bool("help", false, "")
)

func main() {
	flag.Usage = usage
	flag.Parse()

	if len(os.Args) < 2 {
		list()
		return
	}

	if *pid == -1 {
		usage()
	}
	if *help {
		usage()
	}

	if *stack {
		out, err := cmd(signal.Stack)
		exit(err)
		fmt.Println(out)
	}

	if *gc {
		_, err := cmd(signal.GC)
		exit(err)
	}

	if *gcstats {
		out, err := cmd(signal.GCStats)
		exit(err)
		fmt.Printf(out)
	}

}

func cmd(c byte) (string, error) {
	sock := fmt.Sprintf("/tmp/gops%d.sock", *pid)
	conn, err := net.Dial("unix", sock)
	if err != nil {
		return "", err
	}
	if _, err := conn.Write([]byte{c}); err != nil {
		return "", err
	}
	all, err := ioutil.ReadAll(conn)
	if err != nil {
		return "", err
	}
	return string(all), nil
}

func list() {
	pss, err := ps.Processes()
	if err != nil {
		log.Fatal(err)
	}
	var undetermined int
	for _, pr := range pss {
		name, err := pr.Path()
		if err != nil {
			undetermined++
			continue
		}
		ok, err := isGo(name)
		if err != nil {
			// TODO(jbd): worth to report the number?
			continue
		}
		if ok {
			fmt.Printf("%d\t%v\t(%v)\n", pr.Pid(), pr.Executable(), name)
		}
	}
	if undetermined > 0 {
		fmt.Printf("\n%d processes left undetermined\n", undetermined)
	}
}

func isGo(filename string) (ok bool, err error) {
	obj, err := objfile.Open(filename)
	if err != nil {
		return false, err
	}
	defer obj.Close()

	symbols, err := obj.Symbols()
	if err != nil {
		return false, err
	}

	// TODO(jbd): find a faster way to determine Go programs
	// looping over the symbols is a joke.
	for _, s := range symbols {
		if s.Name == "runtime.buildVersion" {
			return true, nil
		}
	}
	return false, nil
}

func usage() {
	fmt.Fprintf(os.Stderr, "%v\n", helpText)
	os.Exit(1)
}

func exit(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}