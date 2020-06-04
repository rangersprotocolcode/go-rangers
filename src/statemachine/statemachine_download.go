// image/container下载用
package statemachine

import (
	"bufio"
	"github.com/docker/docker/api/types"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

func (s *StateMachine) download() bool {
	s.logger.Warnf("start download stm: %s, downloadUrl: %s, downloadProtocol: %s", s.Image, s.DownloadUrl, s.DownloadProtocol)
	result := false
	switch strings.ToLower(s.DownloadProtocol) {
	case "pull":
		result = s.downloadByPull()
		break
	case "file":
		result = s.downloadByFile(true)
		break
	case "filecontainer":
		result = s.downloadByFile(false)
		break
	case "ipfs":
		result = s.downloadByIPFS(true)
		break
	case "ipfscontainer":
		result = s.downloadByIPFS(false)
		break
	}

	s.logger.Warnf("end download stm: %s, downloadUrl: %s, downloadProtocol: %s, result: %t", s.Image, s.DownloadUrl, s.DownloadProtocol, result)
	return result
}

func (s *StateMachine) downloadByIPFS(isImage bool) bool {
	return true
}

func (s *StateMachine) downloadByFile(isImage bool) bool {
	url := s.DownloadUrl
	if 0 == len(url) {
		return false
	}

	isHttp := strings.HasPrefix(url, "http")
	if isHttp {
		// http下载
		response, err := s.httpClient.Get(url)
		if nil != err {
			s.logger.Errorf("fail to download stm by file, err: %s", err.Error())
			return false
		}
		defer response.Body.Close()

		err = s.loadOrImport(isImage, response.Body)
		if err != nil {
			s.logger.Errorf("fail to download stm by file, err: %s", err.Error())
			return false
		}
	} else {
		// 本地文件加载
		file, err := os.Open(url)
		if nil != err {
			s.logger.Errorf("fail to download stm by file, err: %s", err.Error())
			return false
		}
		r := bufio.NewReader(file)
		defer file.Close()

		err = s.loadOrImport(isImage, r)
		if err != nil {
			s.logger.Errorf("fail to download stm by file, err: %s", err.Error())
			return false
		}
	}

	return true
}

// 从reader里load/import docker image
func (s *StateMachine) loadOrImport(isImage bool, reader io.Reader) error {
	var result string
	if isImage {
		response, err := s.cli.ImageLoad(s.ctx, reader, true)
		if nil != err {
			return err
		}
		defer response.Body.Close()

		body, _ := ioutil.ReadAll(response.Body)
		result = string(body)
		s.logger.Warnf("ImageLoaded, result: %s", result)
	} else {
		response, err := s.cli.ImageImport(s.ctx, types.ImageImportSource{Source: reader, SourceName: "-"}, "", types.ImageImportOptions{})
		if nil != err {
			return err
		}
		defer response.Close()

		body, _ := ioutil.ReadAll(response)
		result = string(body)
		s.logger.Warnf("ImageImported, result: %s", result)
	}

	if !s.checkImageExisted() {
		imageId := s.getImageId(result)
		s.cli.ImageTag(s.ctx, imageId, s.Image)
		s.logger.Warnf("change image tag, imageId: %s, image: %s", imageId, s.Image)
	}

	return nil
}

// 从docker官方仓库下载，最简单的方式
// todo: 下载时的鉴权
func (s *StateMachine) downloadByPull() bool {
	_, err := s.cli.ImagePull(s.ctx, s.DownloadUrl, types.ImagePullOptions{})
	if err != nil {
		s.logger.Warnf("fail to pull image: %s, downloadUrl: %s. error: %s", s.Image, s.DownloadUrl, err.Error())
		return false
	}

	s.waitUntilImageExisted()
	return true
}

// docker load/import 之后的返回
// {"stream":"Loaded image ID: sha256:00f6ec4b97ae644112f18a51927911bc06afbd4b395bb3771719883cfa64451e\n"}
// 需要从里面提取image ID
func (machine *StateMachine) getImageId(s string) string {
	index := strings.Index(s, "sha256:")
	return s[index+7 : index+64+7]
}
