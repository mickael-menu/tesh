package paths

import "path/filepath"

func Canonical(path string) (string, error) {
	path = filepath.Clean(path)
	path, err := filepath.Abs(path)
	if err != nil {
		return path, err
	}
	return filepath.EvalSymlinks(path)
}
