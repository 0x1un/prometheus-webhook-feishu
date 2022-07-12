package main

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	logging "github.com/ipfs/go-log/v2"
	"github.com/magicst0ne/alertmanager-webhook-feishu/feishu"
	"github.com/magicst0ne/alertmanager-webhook-feishu/model"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	"io/ioutil"
)

var (
	configFile = kingpin.Flag(
		"config.file",
		"configuration file path.",
	).Short('c').Default("config.yml").String()

	serverPort = kingpin.Flag(
		"web.listen-address",
		"Address to listen on",
	).Short('p').Default(":8086").String()

	sc = &SafeConfig{
		C: &Config{},
	}

	L = logging.Logger("<Feishu-AlertManager")
)

type BatchGetID struct {
	Code int `json:"code"`
	Data struct {
		UserList []struct {
			Mobile string `json:"mobile"`
			UserID string `json:"user_id,omitempty"`
		} `json:"user_list"`
	} `json:"data"`
	Msg string `json:"msg"`
}

func main() {
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	// load config  first time
	if err := sc.ReloadConfig(*configFile); err != nil {
		L.Fatalf("failed to load config file, %s", err)
	}

	r := gin.Default()
	r.POST("/webhook", func(c *gin.Context) {

		bodyRaw, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			L.Errorf("read request body error, %s", err)
		}

		var alertMsg model.AlertMessage
		err = json.Unmarshal([]byte(bodyRaw), &alertMsg)
		if err != nil {
			L.Errorf("failed to parse WebHookMessage, %s", err)
			c.JSON(200, gin.H{"ret": "-1", "msg": "invalid data"})
		} else {

			accessToken := c.Query("access_token")
			receiver := c.Query("receiver")

			receiverConfig, err := sc.GetConfigByName(receiver)
			if err != nil {
				c.JSON(200, gin.H{"ret": "-1", "msg": "receiver not exists"})
				L.Errorf("receiver(%s) does not exists", receiver)
				return
			}

			if accessToken != receiverConfig.AccessToken {
				c.JSON(200, gin.H{"ret": "-1", "msg": "invalid access_token"})
				L.Errorf("invalid access_token(%s)", accessToken)
				return
			}

			webhook, _ := feishu.NewFeishu(receiverConfig.Fsurl)

			webhookMessage := model.WebhookMessage{AlertMessage: alertMsg}
			webhookMessage.AlertHosts = make(map[string]string)
			token := sc.GetTenantAccessToken()
			mentions := token.GetUserIDByMobilesOrEmails(receiverConfig.Mentions.Mobiles, receiverConfig.Mentions.Emails)
			webhookMessage.OpenIDs = mentions
			err = webhook.Send(&webhookMessage)

			if err != nil {
				c.JSON(200, gin.H{"ret": "-1", "msg": "unknown error " + err.Error()})
				L.Errorf("unknown error, %s", err)
				return
			}
			c.JSON(200, gin.H{"ret": "0", "msg": "ok"})
		}
	})

	L.Fatal(r.Run(*serverPort))
}
