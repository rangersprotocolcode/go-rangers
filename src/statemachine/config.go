package statemachine

import (
	"fmt"
	"strings"
	"path/filepath"
	"os"
	"log"
	"io/ioutil"
	"gopkg.in/yaml.v2"
)

//PortInt: 端口号类型
type PortInt uint16

//PortInt.String() : 将端口号转换为字符串
//  返回值: 字符串格式的端口号
func (pi PortInt) String() string {
	return fmt.Sprintf("%d", pi)
}

// NetworkConfig: To create network
// Name: 网络的名称
type NetworkConfig struct {
	Name string
}

// NetworkConfig.Name: 返回创建网络的名称
// 返回值:网络的名称
func (n *NetworkConfig) String() string {
	return n.Name
}

//Port:端口映射信息数据
//Port.Host:宿主机端口
//Port.Target: 容器内部端口
type Port struct {
	Host   PortInt `yaml:"host"`
	Target PortInt `yaml:"target"`
}
type Ports []Port

//Port.String: 输出端口映射配列
func (p *Port) String() string {
	return fmt.Sprintf("%d:%d", p.Host, p.Target)
}

func (p Ports) Len() int {
	return len(p)
}
func (p Ports) Less(i, j int) bool {
	return p[i].Host > p[j].Host
}
func (p Ports) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

//Vol: 设置卷映射
//Vol.Host: 宿主机文件夹
//Vol.Target: 目标容器文件夹
type Vol struct {
	Host   string `yaml:"host"`
	Target string `yaml:"target"`
}

//Vol.String: 输出端口映射配列
func (v *Vol) String() string {
	return fmt.Sprintf("%s:%s", v.Host, v.Target)
}

//Vols: 储存多卷映射序列
type Vols []Vol

//ReplacePWD: 替换卷映射过程中的" pwd" 为当前工作目录
func (vs *Vols) ReplacePWD() {
	curDir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	for i, v := range *vs {
		if strings.ToLower(v.Host[:3]) == "pwd" {
			(*vs)[i].Host = strings.Replace(v.Host, v.Host[:3], curDir, -1)
		}
	}
}

//YAMLConfig: 储存从 yaml 读取的配置信息
//Title: 配置名称
//Service: 服务(对应于容器)
type YAMLConfig struct {
	Title    string           `yaml:"title"`
	Services []ContainerConfig `yaml:"services"`
}

// Init toml from *.yaml
//filename: 文件名信息
func (t *YAMLConfig) InitFromFile(filename string) error {
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}

	err = yaml.UnmarshalStrict(yamlFile, t)
	if err != nil {
		log.Fatal(err)
	}

	return err
}

//    Priority      设定启动顺序
//    Game          游戏id
//    Name 			设定容器名
//    Image			string,设定镜像名
//    Detached 		bool,设定是否后台运行(不输出初始化日志记录),
//                  true 表示不输出初始化记录
//                  false 表示
//    WorkDir		设定容器工作目录
//    CMD           设定容器运行的命令
//    Net           配置容器的网络信息
//    Ports 		配置容器端口供外部访问
//                  支持挂载多端口
//    Volumes		配置挂载卷信息
//    AutoRemove    设定容器运行完毕后是否删除该容器
//                  true 表示自动删除
//                  false 表示不删除
//    Type          公链类型
type ContainerConfig struct {
	Priority   uint   `yaml:"priority"`
	Game       string `yaml:"game"`
	Name       string `yaml:"name"`
	Image      string `yaml:"image"`
	Detached   bool   `yaml:"detached"`
	WorkDir    string `yaml:"work_dir"`
	CMD        string `yaml:"cmd"`
	Net        string `yaml:"net"`
	Ports      Ports  `yaml:"ports"`
	Volumes    Vols   `yaml:"volumes"`
	AutoRemove bool   `yaml:"auto_remove"`
	Import     string `yaml:"import"`
	Type       string `yaml:"type"`
}