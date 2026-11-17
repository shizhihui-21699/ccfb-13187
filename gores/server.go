package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"gores/internal/gores"
)

func cmdServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	addr := fs.String("addr", ":8080", "listen address")
	web := fs.String("web", "", "directory with built Nuxt static files (optional)")
	fs.Parse(args)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/analyze", handleAnalyze)
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ok":true}`))
	})
	if *web != "" {
		dir := http.Dir(*web)
		mux.Handle("GET /", http.FileServer(dir))
	}
	fmt.Printf("gores server listening on %s\n", *addr)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		fatal(err)
	}
}

func handleAnalyze(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 2<<30) // 2 GiB cap
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		http.Error(w, "bad upload: "+err.Error(), http.StatusBadRequest)
		return
	}
	f, hdr, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing file field: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer f.Close()

	// Spool to a temp file so the parser gets an io.ReaderAt.
	tmp, err := os.CreateTemp("", "gores-upload-*")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()
	if _, err := io.Copy(tmp, f); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	name := filepath.Base(hdr.Filename)
	res, err := gores.Analyze(tmp, name, false)
	if err != nil {
		// Still return partial result with the error noted.
		res = &gores.Result{FileName: name}
		if res.Build == nil {
			res.Build = &gores.BuildInfo{}
		}
		w.Header().Set("X-Gores-Warning", err.Error())
	}
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	if err := enc.Encode(res); err != nil {
		fmt.Fprintln(os.Stderr, "encode:", err)
	}
}
