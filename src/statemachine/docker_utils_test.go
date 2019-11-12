package statemachine

import (
	"testing"
	"x/src/common"
	"time"
)

func TestDockerInit(t *testing.T) {
	common.InitConf("/Users/daijia/go/src/x/deploy/daily/x1.ini")
	DockerInit("test.yaml", 8080)
	time.Sleep(1000 * time.Minute)
}
