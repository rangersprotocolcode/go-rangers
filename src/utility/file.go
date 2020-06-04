package utility

import (
	"crypto/md5"
	"io/ioutil"
	"sync"
	"path/filepath"
	"os"
	"errors"
	"sort"
	"strings"
	"encoding/json"
	"archive/zip"
	"io"
)

type result struct {
	path   string
	md5Sum [md5.Size]byte
	err    error
}

// 详细分析文件夹内每个文件内容的md5
// 暂时不用
func checkFolderDetail(folder string, limit int) (map[string][md5.Size]byte, error) {
	returnValue := make(map[string][md5.Size]byte)
	var limitChannel chan struct{}
	if limit != 0 {
		limitChannel = make(chan struct{}, limit)
	}

	done := make(chan struct{})
	defer close(done)

	c := make(chan result)
	errc := make(chan error, 1)
	var wg sync.WaitGroup

	go func() {
		err := filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				return nil
			}

			if limit != 0 {
				//如果已经满了则阻塞在这里
				limitChannel <- struct{}{}
			}

			wg.Add(1)
			go func() {
				// todo: 不能直接读文件，小心内存爆炸
				data, err := ioutil.ReadFile(path)
				select {
				case c <- result{path: path, md5Sum: md5.Sum(data), err: err}:
				case <-done:
				}
				if limit != 0 {
					//读出数据，这样就有新的文件可以处理
					<-limitChannel
				}

				wg.Done()
			}()
			select {
			case <-done:
				return errors.New("canceled")
			default:
				return nil
			}
		})
		errc <- err
		go func() {
			wg.Wait()
			close(c)
		}()
	}()

	for r := range c {
		if r.err != nil {
			return nil, r.err
		}
		returnValue[r.path] = r.md5Sum
	}
	if err := <-errc; err != nil {
		return nil, err
	}
	return returnValue, nil
}

// 粗略分析文件夹内文件的状态,速度快
// 只考虑文件名与文件大小
// 不考虑修改时间，主要是各个节点的时间可能不一样
func CheckFolder(folder string) ([md5.Size]byte, sortableFileInfos) {
	var list sortableFileInfos
	err := filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
		if nil == info || !info.Mode().IsRegular() {
			return nil
		}

		var fileInfo fileInfo
		fileInfo.Name = path
		fileInfo.Size = info.Size()
		list = append(list, fileInfo)
		return nil
	})

	if nil != err {
		return [md5.Size]byte{}, nil
	}

	// 根据文件名排序
	sort.Sort(list)

	byteData, _ := json.Marshal(list)
	return md5.Sum(byteData), list
}

type fileInfo struct {
	Name string
	Size int64
}

type sortableFileInfos []fileInfo

func (s sortableFileInfos) Len() int {
	return len(s)
}
func (s sortableFileInfos) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s sortableFileInfos) Less(i, j int) bool {
	return strings.Compare(s[i].Name, s[j].Name) > 0
}

// srcFile could be a single file or a directory
func Zip(srcFile string, destZip string) error {
	zipfile, err := os.Create(destZip)
	if err != nil {
		return err
	}
	defer zipfile.Close()

	archive := zip.NewWriter(zipfile)
	defer archive.Close()

	filepath.Walk(srcFile, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		header.Name = strings.TrimPrefix(path, filepath.Dir(srcFile)+"/")
		// header.Name = path
		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		if ! info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			_, err = io.Copy(writer, file)
		}
		return err
	})

	return err
}

func Unzip(zipFile string, destDir string) error {
	zipReader, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer zipReader.Close()

	for _, f := range zipReader.File {
		fpath := filepath.Join(destDir, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
		} else {
			if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
				return err
			}

			inFile, err := f.Open()
			if err != nil {
				return err
			}
			defer inFile.Close()

			outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer outFile.Close()

			_, err = io.Copy(outFile, inFile)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
