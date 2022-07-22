package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	yaml "gopkg.in/yaml.v2"
	"io/ioutil"
	"sync"
)

type Config struct {
	AppID     string                    `yaml:"app_id"`
	AppSecret string                    `yaml:"app_secret"`
	Receivers map[string]ReceiverConfig `yaml:"receivers"`
}

type SafeConfig struct {
	sync.RWMutex
	C *Config
}

type Mentions struct {
	Mobiles []string `yaml:"mobiles"`
	Emails  []string `yaml:"emails"`
}

type ReceiverConfig struct {
	AccessToken string    `yaml:"access_token"`
	Fsurl       string    `yaml:"fsurl"`
	Mentions    *Mentions `yaml:"mentions"`
}

func (sc *SafeConfig) ReloadConfig(configFile string) error {
	var c = &Config{}

	yamlFile, err := ioutil.ReadFile(configFile)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(yamlFile, c); err != nil {
		return err
	}

	sc.Lock()
	sc.C = c
	sc.Unlock()

	return nil
}

func (sc *SafeConfig) GetConfigByName(name string) (*ReceiverConfig, error) {
	sc.Lock()
	defer sc.Unlock()
	if receiver, ok := sc.C.Receivers[name]; ok {
		return &receiver, nil
	}
	return &ReceiverConfig{"", "", &Mentions{
		Mobiles: nil,
		Emails:  nil,
	}}, fmt.Errorf("no credentials found for receiver %s", name)
}

var Client = resty.New()

type Token string

func (t Token) GetUserIDByMobilesOrEmails(mobiles []string, emails []string) []string {
	if len(mobiles) == 0 && len(emails) == 0 {
		return nil
	}
	resp, err := Client.R().SetHeader(
		"Authorization", fmt.Sprintf("Bearer %s", t)).SetBody(map[string]interface{}{
		"mobiles": mobiles,
		"emails":  emails,
	}).
		Post("https://open.feishu.cn/open-apis/contact/v3/users/batch_get_id")
	if err != nil {
		L.Errorf("failed to request batch_get_id, %s", err)
		return nil
	}
	var info BatchGetID
	if err := json.Unmarshal(resp.Body(), &info); err != nil {
		L.Errorf("failed to unmarshal batch_get_id body, %s", err)
		return nil
	}
	out := make([]string, 0)
	if info.Code == 0 && info.Msg == "success" {
		for _, user := range info.Data.UserList {
			if user.UserID == "" {
				continue
			}
			out = append(out, user.UserID)
		}
		return out
	}
	return nil
}

func (sc *SafeConfig) GetTenantAccessToken() Token {
	sc.Lock()
	defer sc.Unlock()
	resp, err := Client.R().SetBody(map[string]string{
		"app_id":     sc.C.AppID,
		"app_secret": sc.C.AppSecret,
	}).
		Post("https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal")
	if err != nil {
		L.Errorf("failed to get tenant_access_token, %s", err)
		return ""
	}
	var temp map[string]interface{}
	if err := json.Unmarshal(resp.Body(), &temp); err != nil {
		L.Errorf("failed to unmarshal tenant body, %s", err)
		return ""
	}
	if token, ok := temp["tenant_access_token"]; ok {
		return Token(token.(string))
	}
	return ""
}
