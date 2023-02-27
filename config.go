package raft

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type RConfig struct {
	Port    int      `yaml:"port"`
	Cluster []string `yaml:"cluster"`
}

var Config RConfig

func init() {
	err := loadConfig()
	if err != nil {
		panic(err)
	}
}

func loadConfig() error {
	byt, err := ioutil.ReadFile("app.yml")
	if err != nil {
		return fmt.Errorf("read configration from app.yml error:%v\n", err)
	}
	err = yaml.Unmarshal(byt, &Config)
	if err != nil {
		return err
	}

	fmt.Println(Config)

	return nil
}
