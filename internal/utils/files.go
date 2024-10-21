package utils

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func RemoveGlob(path string) (err error) {
	contents, err := filepath.Glob(path)
	if err != nil {
		return
	}
	for _, item := range contents {
		err = os.RemoveAll(item)
		if err != nil {
			return
		}
	}
	return
}

func GetPidFromFile(name string) (int, error) {
	pidData, err := os.ReadFile(name)
	if err != nil {
		return 0, errors.New(fmt.Sprintf("error: '%v' read pid file: '%v' ", err, name))
	}

	var pid int
	n, err := fmt.Sscanf(string(pidData), "%d\n", &pid)
	if err != nil {
		return 0, errors.New(fmt.Sprintf("error: '%v' parse pid file: '%v' ", err, name))
	}
	if n != 1 {
		return 0, errors.New(fmt.Sprintf("error: '%v' parse pid file, incorrect parsed arguments numer: %d", err, n))
	}
	return pid, nil
}
