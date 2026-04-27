package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/tcampbell/mkdocs-server/internal/build"
	"github.com/tcampbell/mkdocs-server/internal/serve"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "build":
		runBuild(os.Args[2:])
	case "serve":
		runServe(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func runBuild(args []string) {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	configPath := fs.String("f", "mkdocs.yml", "path to mkdocs.yml")
	fs.Parse(args) //nolint:errcheck — ExitOnError handles errors

	if err := build.Build(*configPath); err != nil {
		fmt.Fprintf(os.Stderr, "build: %v\n", err)
		os.Exit(1)
	}
}

func runServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	dir := fs.String("d", "site", "directory to serve")
	port := fs.Int("p", 8000, "port to listen on")
	fs.Parse(args) //nolint:errcheck — ExitOnError handles errors

	if err := serve.Serve(*dir, *port); err != nil {
		fmt.Fprintf(os.Stderr, "serve: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `mkdocs-server — Go-native MkDocs Material SSG

Commands:
  build [-f mkdocs.yml]       build static site into site/
  serve [-d site] [-p 8000]   serve site/ over HTTP`)
}
