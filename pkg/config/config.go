package config

import (
	"fmt"
	"github.com/go-yaml/yaml"
	"io/ioutil"
	"log"
	"multi_cluster_informer/pkg"
)

// TODO: 配置文件

var SysConfig *Config

type Config struct {
	MaxReQueueTime int          `json:"maxrequeuetime", yaml:"maxrequeuetime"`
	Clusters       []pkg.Cluster `json:"clusters", yaml"clusters"`
}

func NewConfig() *Config {
	return &Config{}
}

func loadConfigFile(path string) []byte {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		log.Println(err)
		return nil
	}
	return b
}

func LoadConfig(path string) (*Config, error) {
	config := NewConfig()
	if b := loadConfigFile(path); b != nil {

		err := yaml.Unmarshal(b, config)
		if err != nil {
			return nil, err
		}
		return config, err
	} else {
		return nil, fmt.Errorf("load config file error...")
	}

}
