package dkim

import (
	"github.com/axgle/mahonia"
	"io/ioutil"
	"os"
)

var (
	FileNotFoundError = &fileError{"file not found"}
	FileReadError     = &fileError{"file read error"}
	ErrEmptyString    = &fileError{"empty hex string"}
	NoDecoderDef      = &fileError{"no this decoder"}
)

type fileError struct {
	msg string
}

func (err fileError) Error() string { return err.msg }


func ReadFileAsString(path string, code string) (string, error) {
	if len(path) == 0 {
		return "", ErrEmptyString
	}

	if len(code) == 0 {
		code = "UTF-8"
	}

	open, err := os.Open(path)
	if err != nil {
		return "", FileNotFoundError
	}

	defer open.Close()
	decoder := mahonia.NewDecoder(code)
	if decoder == nil {
		return "", NoDecoderDef
	}
	fd, err := ioutil.ReadAll(decoder.NewReader(open))
	if err != nil {
		return "", FileReadError
	}
	return string(fd), nil
}
