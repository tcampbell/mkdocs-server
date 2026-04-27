package serve

import (
	"fmt"
	"net/http"
)

func Serve(dir string, port int) error {
	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("mkdocs-server: http://localhost%s\n", addr)
	return http.ListenAndServe(addr, http.FileServer(http.Dir(dir)))
}
