// This file is a modified version of write.go from
// https://github.com/bmaupin/go-epub which has the following license:
//
// The MIT License (MIT)
//
// Copyright (c) 2016 Bryan Maupin
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const mimetypeFilename = "mimetype"

// Write the EPUB file itself by zipping up everything from a temp directory
func writeEpub(tempDir string, destFilePath string) error {
	f, err := os.Create(destFilePath)
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Close(); err != nil {
			Log("closing epub: %v", err)
		}
	}()

	z := zip.NewWriter(f)
	defer func() {
		if err := z.Close(); err != nil {
			Log("closing epub writer: %v", err)
		}
	}()

	skipMimetypeFile := false

	var addFileToZip = func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get the path of the file relative to the folder we're zipping
		relativePath, err := filepath.Rel(tempDir, path)
		relativePath = filepath.ToSlash(relativePath)
		if err != nil {
			// tempDir and path are both internal, so we shouldn't get here
			return fmt.Errorf("error closing EPUB file: %s", err)
		}

		// Only include regular files, not directories
		if !info.Mode().IsRegular() {
			return nil
		}

		var w io.Writer
		if path == filepath.Join(tempDir, mimetypeFilename) {
			// Skip the mimetype file if it's already been written
			if skipMimetypeFile == true {
				return nil
			}
			// The mimetype file must be uncompressed according to the EPUB spec
			w, err = z.CreateHeader(&zip.FileHeader{
				Name:   relativePath,
				Method: zip.Store,
			})
		} else {
			w, err = z.Create(relativePath)
		}
		if err != nil {
			return fmt.Errorf("error creating zip writer: %s", err)
		}

		r, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("error opening file being added to EPUB: %s", err)
		}
		defer func() {
			if err := r.Close(); err != nil {
				Log("closing file: %v", err)
			}
		}()

		_, err = io.Copy(w, r)
		if err != nil {
			return fmt.Errorf("error copying contents of file being added EPUB: %s", err)
		}

		return nil
	}

	// Add the mimetype file first
	mimetypeFilePath := filepath.Join(tempDir, mimetypeFilename)
	mimetypeInfo, err := os.Lstat(mimetypeFilePath)
	if err != nil {
		return fmt.Errorf("unable to get FileInfo for mimetype file: %s", err)
	}
	err = addFileToZip(mimetypeFilePath, mimetypeInfo, nil)
	if err != nil {
		return fmt.Errorf("unable to add mimetype file to EPUB: %s", err)
	}

	skipMimetypeFile = true

	err = filepath.Walk(tempDir, addFileToZip)
	if err != nil {
		return fmt.Errorf("unable to add file to EPUB: %s", err)
	}

	return nil
}
