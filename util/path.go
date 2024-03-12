package util

import (
	"os"
	"path/filepath"
)

var (
	_ExeDir  string
	_ExeName string
)

// WorkDir 程序执行目录
func WorkDir() string {
	if _ExeDir != "" {
		return _ExeDir
	}
	p, err := os.Executable()
	if err != nil {
		panic(err)
	}
	_ExeDir = filepath.ToSlash(filepath.Dir(p))
	return _ExeDir
}

// SetExeDir 手动指定执行目录
func SetExeDir(dir string) {
	_ExeDir = dir
}

func ExeName() string {
	if _ExeName != "" {
		return _ExeName
	}
	executable, _ := os.Executable()
	ext := filepath.Ext(executable)
	_ExeName = filepath.Base(executable[:len(executable)-len(ext)])
	return _ExeName
}
