package cli

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"testing"
)

//go:embed dist
var testFS embed.FS

func TestFileServer(t *testing.T) {
	mux := http.NewServeMux()
	fsys, _ := fs.Sub(testFS, "dist")
	fileServer := http.FileServer(http.FS(fsys))
	mux.Handle("/", fileServer)
	mux.Handle("/index", http.StripPrefix("/index", fileServer))
	mux.Handle("/minerInfo/", http.StripPrefix("/minerInfo/", fileServer))
	mux.Handle("/blockDetail", http.StripPrefix("/blockDetail", fileServer))

	http.ListenAndServe(":10000", mux)
}

func TestFileServer1(t *testing.T) {
	var h = &handler{}
	http.ListenAndServe(":10000", h)
}

type handler struct {
}

func (srv *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		fmt.Printf("url:%s\n", r.URL.Path)
		if r.URL.Path == "/" {
			b, _ := testFS.ReadFile("dist/index.html")
			w.Write(b)
		} else {
			b, _ := testFS.ReadFile("dist" + r.URL.Path)
			w.Write(b)
		}
		return
	}
}
