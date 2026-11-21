// Package storage writes files safely: exclusive creation (no overwrite) and
// atomic replacement.
package storage

import (
	"os"
	"path/filepath"
)

// WriteFileExclusive creates the file only if it does not exist. If writing
// fails, the partial file is removed.
func WriteFileExclusive(path string, data []byte, perm os.FileMode) error {
	file, err := os.OpenFile(path, //nolint:gosec // G304: path is chosen by the caller, not received from the network
		os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return err
	}
	if err := writeAndClose(file, data); err != nil {
		_ = os.Remove(path)
		return err
	}
	return nil
}

// WriteFileAtomic replaces the file atomically: the content goes to a
// temporary file in the same directory and only then receives the final name.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	done := false
	defer func() {
		if !done {
			_ = tmp.Close()
			_ = os.Remove(tmpPath)
		}
	}()

	if err := writeAndClose(tmp, data); err != nil {
		return err
	}
	if err := os.Chmod(tmpPath, perm); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	done = true
	return nil
}

// writeAndClose writes the data, forces it to disk and closes the file.
func writeAndClose(file *os.File, data []byte) error {
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}
