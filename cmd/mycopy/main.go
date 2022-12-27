package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/julieqiu/copy/internal/mycopy"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: copy [new-repo] [new-dir] [old-repo] [old-dir]\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Copy a package inside a Go repo to x/metrics.\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  new-repo: name of the repo to copy from, for example, pkgsite\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  new-dir: name of the directory inside the repo to copy from, for example, internal/fetch\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  old-repo: name of the repo to copy from, for example, pkgsite\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  old-: name of the directory inside the repo to copy from, for example, internal/fetch\n")
		flag.PrintDefaults()
	}

	flag.Parse()
	if flag.NArg() != 4 {
		flag.Usage()
		os.Exit(1)
	}
	newRepo := flag.Args()[0]
	newDir := flag.Args()[1]
	oldRepo := flag.Args()[2]
	oldDir := flag.Args()[3]

	if err := mycopy.Run(newRepo, newDir, oldRepo, oldDir); err != nil {
		log.Fatal(err)
	}
}
