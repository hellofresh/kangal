// +build tools

package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) < 3 {
		panic("Expected at least 3 arguments.")
	}

	gzipFile := os.Args[1]
	targetPath := os.Args[2]

	var filePathInGZip string
	if len(os.Args) > 3 {
		filePathInGZip = os.Args[3]
	}

	r, err := os.Open(gzipFile)
	if err != nil {
		panic(err)
	}
	defer r.Close()

	uncompressedStream, err := gzip.NewReader(r)
	if err != nil {
		panic(err)
	}

	tarReader := tar.NewReader(uncompressedStream)

	for true {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			panic(err)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			// if we want to extract only one file - we extract it into dest root
			if filePathInGZip != "" {
				continue
			}

			err := os.MkdirAll(filepath.Join(targetPath, header.Name), 0766)
			if err != nil {
				panic(err)
			}

		case tar.TypeReg:
			if filePathInGZip != "" && filepath.Clean(header.Name) != filePathInGZip {
				continue
			}

			if err := writeFile(tarReader, filepath.Join(targetPath, header.Name)); err != nil {
				panic(err)
			}

		case tar.TypeXGlobalHeader:
			continue

		default:
			panic(fmt.Errorf("uknown type: %s in %s", string(header.Typeflag), header.Name))
		}
	}
}

func writeFile(tarReader io.Reader, dest string) error {
	outFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, tarReader)
	return err
}
