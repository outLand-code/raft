package raft

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"regexp"
	"time"
)

type RConfig struct {
	Id                 int      `yaml:"id"`
	Port               int      `yaml:"port"`
	Cluster            []string `yaml:"cluster"`
	Times              int
	electionTimeout    int
	heartBeatInterval  time.Duration
	rpcClientCheckTime time.Duration
}

var Config *RConfig

const (
	ElectionBaseTimeOut = 150
	HeartBeatInterval   = 20
	RPCClientCheckTime  = 20
)

func init() {
	//err := loadConfig()
	//if err != nil {
	//	log.Fatalf("load config error %v\n", err)
	//}
}

func NewConfig(c *RConfig) *RConfig {
	dataCheck(c, idCheck(), clusterCheck(), addlCheck())
	return c
}

type checkOption func(c *RConfig) error

func dataCheck(c *RConfig, fn ...checkOption) {
	for _, f := range fn {
		if err := f(c); err != nil {
			log.Fatalf("data check error :%v", err)
		}
	}
}
func clusterCheck() checkOption {
	return func(c *RConfig) error {
		if len(c.Cluster)%2 != 0 {
			return fmt.Errorf("the number of node is not enough ,it must be an even number\n")
		}
		for _, address := range c.Cluster {
			if b, err := regexp.MatchString("^(((25[0-5]|2[0-4]d|((1\\d{2})|([1-9]?\\d)))\\.)"+
				"{3}(25[0-5]|2[0-4]\\d|((1\\d{2})|([1-9]?\\d))))"+
				"\\:([0-9]|[1-9]\\d{1,3}|[1-5]\\d{4}|6[0-4]\\d{4}|65[0-4]"+
				"\\d{2}|655[0-2]\\d|6553[0-5])$", address); !b || err != nil {
				return fmt.Errorf("this address %s is error\n", address)
			}
		}
		return nil
	}
}

func idCheck() checkOption {
	return func(c *RConfig) error {
		if c.Id == 0 {
			return fmt.Errorf("id must be greater then zero\n")
		}
		return nil
	}
}

func addlCheck() checkOption {
	return func(c *RConfig) error {
		if c.Times == 0 {
			c.Times = 1
		}
		c.electionTimeout = ElectionBaseTimeOut * c.Times
		c.heartBeatInterval = time.Duration(HeartBeatInterval*c.Times) * time.Millisecond
		c.rpcClientCheckTime = time.Duration(RPCClientCheckTime*c.Times) * time.Millisecond
		return nil
	}
}

func loadConfig() error {

	byt, err := ioutil.ReadFile("app.yml")
	if err != nil {
		return fmt.Errorf("read configration from app.yml error:%v\n", err)
	}
	var c RConfig
	err = yaml.Unmarshal(byt, &c)
	if err != nil {
		return err
	}
	fmt.Println(c)
	Config = NewConfig(&c)
	return nil
}
