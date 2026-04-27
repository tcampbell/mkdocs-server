package main

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"os"
)

//go:embed all:site
var site embed.FS

func main() {
	sub, err := fs.Sub(site, "site")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	addr := ":4000"
	if p := os.Getenv("PORT"); p != "" {
		addr = ":" + p
	}
	fmt.Printf("docs: http://localhost%s\n", addr)
	if err := http.ListenAndServe(addr, http.FileServer(http.FS(sub))); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
