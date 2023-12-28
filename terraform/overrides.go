package terraform

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func copyOverrides(src, dst string) error {
	srcPath, err := filepath.Abs(src)
	if err != nil {
		return err
	}
	if _, err = os.Stat(srcPath); err != nil {
		return nil
	}
	err = filepath.WalkDir(srcPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// mounted k8s secrets creates symlinks in dirs prefixed with '..'
		// these can be ignored
		if strings.Contains(path, "/..") {
			return nil
		}

		if d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			err = createDir(filepath.Join(dst, strings.TrimPrefix(path, src)), info.Mode())
			if err != nil {
				return err
			}
			return nil
		}

		// https://developer.hashicorp.com/terraform/language/files/override
		err = copyFile(path, filepath.Join(dst, strings.TrimPrefix(path, src)))
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func createDir(dir string, mode os.FileMode) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.MkdirAll(dir, mode)
		return err
	}
	return nil
}

func copyFile(src, dst string) error {
	var err error
	var srcfd *os.File
	var dstfd *os.File
	var srcinfo os.FileInfo

	if srcfd, err = os.Open(src); err != nil {
		return err
	}
	defer srcfd.Close()

	if dstfd, err = os.Create(dst); err != nil {
		return err
	}
	defer dstfd.Close()

	if _, err = io.Copy(dstfd, srcfd); err != nil {
		return err
	}
	if srcinfo, err = os.Stat(src); err != nil {
		return err
	}
	return os.Chmod(dst, srcinfo.Mode())
}
