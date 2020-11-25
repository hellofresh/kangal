// +build tools

package main

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) != 4 {
		panic("Expected 4 arguments.")
	}

	zipFile := os.Args[1]
	targetFile := os.Args[2]
	filePathInZip := os.Args[3]

	r, err := zip.OpenReader(zipFile)
	if err != nil {
		panic(err)
	}
	defer r.Close()

	for _, f := range r.File {
		if filepath.Clean(f.Name) != filePathInZip {
			continue
		}

		outFile, err := os.OpenFile(targetFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			panic(err)
		}
		defer outFile.Close()

		rc, err := f.Open()
		if err != nil {
			panic(err)
		}
		defer rc.Close()

		_, err = io.Copy(outFile, rc)
		if err != nil {
			panic(err)
		}

		return
	}
}
