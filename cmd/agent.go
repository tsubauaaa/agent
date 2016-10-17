package cmd

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/tsubauaaa/agent"
	"github.com/tsubauaaa/agent/logging"
)

var (
	endPoint         string
	apiKey           string
	configFilePath   string
	registrationInfo *agent.RegistrationInfo
)

func init() {
	flag.StringVar(&endPoint, "endpoint", "", "API endpoint at which the agent should register.")
	flag.StringVar(&apiKey, "api_key", "", "API key for your account. Get this from app.")
	flag.StringVar(&configFilePath, "config", "", "path to the agent config file.")
}

// ServerConfigを検証するファンクション
func validateConfig(configObj agent.ServerConfig) error {
	if len(configObj.APIKey) == 0 {
		return errors.New("Server API key is missing.")
	}

	if len(configObj.EndPoint) == 0 {
		return errors.New("Server endpoint is missing.")
	}
	return nil
}

// MainLoop はAgentのメイン処理のファンクション
func MainLoop(errorChannel chan error, exitChannel chan struct{}) error {
	// コマンドライン引数のパース
	flag.Parse()

	// configFilePathが未指定の場合の処理
	if len(configFilePath) == 0 {
		dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err == nil {
			configFilePath = filepath.Join(dir, agent.DefaultConfigFileName)
		} else {
			errorChannel <- err
			agent.ReportError(fmt.Sprintf("Could not get the absolute path of the installer. Error: %v", err))
			//ServerUpdate処理
		}
	}

	// コマンドライン引数からServerConfig構造体を構成
	cmdlineConfig := agent.ServerConfig{EndPoint: endPoint, APIKey: apiKey}

	// ServerConfigとAgentConfigパラメータを取得
	serverConfig, agentConfig, err := agent.GetConfig(configFilePath, cmdlineConfig, errorChannel)
	if err != nil {
		fmt.Printf("Invalid config file. Error: %v\n", err)
		//ServerUpdate処理
		os.Exit(1)
	}

	if err := validateConfig(serverConfig); err != nil {
		errorChannel <- err
		agent.ReportError(fmt.Sprintf("Invalid config values. Error: %v", err))
		fmt.Printf("Invalid config values. Error: %v\n", err)
		//ServerUpdate処理
		os.Exit(1)
	} else {
		//ServerUpdate処理
	}

	logFilePath := agentConfig.LogFile
	//ログファイルパスが絶対パスとして正しいかチェック
	if !filepath.IsAbs(agentConfig.LogFile) {
		dir := filepath.Dir(configFilePath)
		logFilePath = filepath.Join(dir, logFilePath)
	}
	err = logging.SetupLogger(logFilePath, agentConfig.DebugMode, agent.ErrorsChannel)
	if err != nil {
		errorChannel <- err
		agent.ReportError(fmt.Sprintf("Could not setup logger. Error: %v", err))
	}

	logging.Info("Starting Server agent....", logging.Fields{"version": agent.AgentVersion})
	logging.Debug("Final config.", logging.Fields{"config": serverConfig})

	// Agentのメタデータを取得
	/*
		metaData, err := agent.GetHostMetaData(&agentConfig)
		if err != nil {
			logging.Error("Cloud not get metadata from host.", logging.Fields{"error": err})
			os.Exit(1)
		}

		// AgentRegistration処理。失敗すると最大5分間隔でリトライする
		// AgentRegistrationが完了するとRegistrationInfo(AgentID、AWS認証情報、SQSエンドポイントなど)を取得する
			i := 0
			for {
				registrationInfo, err := agent.RegisterAgent(metaData, &serverConfig)
				i++
				if err != nil {
					sleepDelay := math.Min(float64(i*30), 300)
					logging.Error("Cloud not register the agent. Retrying..", logging.Fields{"error": err, "delay": sleepDelay})
					time.Sleep(time.Second * time.Duration(sleepDelay))
				} else {
					break
				}
			}
	*/

	registrationInfo := &agent.RegistrationInfo{
		AgentID:             "123456789",
		CreateTime:          time.Now().UnixNano() / int64(time.Millisecond),
		UpdateTime:          time.Now().UnixNano() / int64(time.Millisecond),
		ActionQueueEndpoint: "https://sqs.ap-northeast-1.amazonaws.com/905774158693/agent",
		AWSAccessKey:        "AKIAI4EYMCG3RPOHWR6A",
		AWSSecretAccessKey:  "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",
		AWSSecurityToken:    "",
	}

	if len(registrationInfo.AgentID) > 0 {
		//Agentステータス更新処理
	}

	// AgentRegistration情報送信
	// Agentホストのイベント情報初期化

	//TimeTickerの定義

	// AgentRegistration情報およびメタデータをログ送信開始

	regInfoUpdatesCh := make(chan string, 5)
	triggerReregistrationCh := make(chan time.Time, 5)

	//定期的にServerへAgentRegistration、ハートビート、ログ送信を行うgo routine処理

	events := make(chan *agent.Event, 10)

	// SQSメッセージポーリングの無限ループを行うgo routine処理
	go func() {
		agent.RunLoop(registrationInfo, regInfoUpdatesCh, events, triggerReregistrationCh)
	}()

	// SQSメッセージに則ってRunbookを実行するgo routine処理

	// Runbook実行結果をServerに送信するgo routine処理

	<-exitChannel

	return nil

}
