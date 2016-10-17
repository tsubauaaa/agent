package agent

import (
	"errors"
	"strconv"
	"time"

	"github.com/tsubauaaa/agent/logging"
	"gopkg.in/jmcvetta/napping.v3"
)

const (
	// AgentVersion はエージェントバージョン
	AgentVersion = "0.1.0"
)

// RegistrationRequest はAgentからServerに送信するメッセージの構造体
type RegistrationRequest struct {
	AgentVersion       string
	HostName           string
	AssignedHostname   string
	ProviderServerID   string
	ProviderServerType string
	Platform           string
	PrivateIPAddress   string
	PrivateDNSName     string
	PublicIPAddress    string
	PublicDNSName      string
	Region             string
	StartTime          int64
}

// RegistrationInfo はAgent登録成功後にServerからAgentへ返却するメッセージの構造体
type RegistrationInfo struct {
	AgentID             string
	CreateTime          int64
	UpdateTime          int64
	ActionQueueEndpoint string // 例：https://sqs.us-east-1.amazonaws.com
	AWSAccessKey        string
	AWSSecretAccessKey  string
	AWSSecurityToken    string
}

// startTime はAgentの開始時刻
var startTime = time.Now().Unix() * 1000

// getAgentRegistrationRequest は取得したメターデータからサーバ登録情報を構成するファンクション
func getAgentRegistrationRequest(data HostMetaData) RegistrationRequest {
	return RegistrationRequest{
		AgentVersion:       AgentVersion,
		HostName:           data.HostName,
		AssignedHostname:   data.AssignedHostname,
		ProviderServerID:   data.ProviderID,
		ProviderServerType: data.ProviderType,
		Platform:           data.Platform,
		PrivateIPAddress:   data.PrivateIPAddress,
		PublicIPAddress:    data.PublicIPAddress,
		PrivateDNSName:     data.PrivateDNSName,
		PublicDNSName:      data.PublicDNSName,
		Region:             data.Region,
		StartTime:          startTime,
	}
}

// RegisterAgent はServerにHTTP POSTしてAgentを登録して返却メッセージを得るファンクション
func RegisterAgent(data HostMetaData, configObj *ServerConfig) (*RegistrationInfo, error) {
	request := getAgentRegistrationRequest(data)
	response := RegistrationInfo{}
	logging.Info("Registering the agent.", logging.Fields{"request": request})

	// napping.Post(url, payload, result, errMsg)
	resp, err := napping.Post(joinURL(configObj.EndPoint, "register", configObj.APIKey), &request, &response, nil)
	if err != nil {
		logging.Error("Could not post to server.", logging.Fields{"error": err, "response": resp})
		return &response, err
	}

	if 200 <= resp.Status() && resp.Status() <= 299 {
		logging.Info("Successfully registered the agent.", logging.Fields{"agentId": response.AgentID})
		return &response, nil
	}
	logging.Warn("Unexpected status from server.", logging.Fields{"status": resp.Status()})
	return &response, errors.New("Server returned unexpected status: " + strconv.Itoa(resp.Status()))

}
