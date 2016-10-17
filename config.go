// Package agent 内のconfig.go はAgent設定ファイルとコマンドライン引数のパラメータを解析し
// 最終調整するプログラム
package agent

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

// Config はAgent設定ファイルのパラメータの構造体
type Config struct {
	Server ServerConfig
	Agent  AgentConfig
}

// ServerConfig はAgent設定ファイルのうちServer部のパラメータの構造体
type ServerConfig struct {
	APIKey   string
	EndPoint string
}

// AgentConfig はAgent設定ファイルのうちAgent部のパラメータの構造体
type AgentConfig struct {
	AssignedHostname string
	LogFile          string
	DebugMode        bool
}

const (
	// DefaultConfigFileName デフォルト設定ファイル
	DefaultConfigFileName = "agent.json"
	// DefaultBaseURL デフォルトAPIエンドポイント
	DefaultBaseURL     = "tsubauaaa.com"
	defaultLogFileName = "agent.log"
)

// parseConfigは設定ファイルをConfig構造体にパースするファンクション
func parseConfig(configFilePath string) (Config, error) {
	file, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		fmt.Printf("Could not read the config file. Error: %v\n", err)
		return Config{}, err
	}

	var obj Config
	err = json.Unmarshal(file, &obj)
	if err != nil {
		fmt.Printf("Could not parse the config JSON. Error: %v\n", err)
		return Config{}, err
	}
	return obj, nil
}

// getDefaultConfigはデフォルト値からConfig構造体にパースするファンクション
func getDefaultConfig() Config {
	return Config{
		ServerConfig{EndPoint: DefaultBaseURL},
		AgentConfig{LogFile: defaultLogFileName, DebugMode: false},
	}
}

// mergeConfigsはコマンドライン引数とAgent設定ファイルとデフォルト値からConfig構造体を構成するファンクション
// ServerConfigの設定値はコマンドライン引数の値が優先される
func mergeConfigs(cmdConfig ServerConfig, configObj Config) (ServerConfig, AgentConfig, error) {
	var apiKey string
	if len(cmdConfig.APIKey) > 0 {
		apiKey = cmdConfig.APIKey
	} else {
		apiKey = configObj.Server.APIKey
	}

	var endPoint string
	if len(cmdConfig.EndPoint) > 0 {
		endPoint = cmdConfig.EndPoint
	} else {
		endPoint = configObj.Server.EndPoint
	}

	return ServerConfig{
			APIKey:   apiKey,
			EndPoint: endPoint,
		},
		configObj.Agent, nil
}

// GetConfig はServerConfigとAgentConfigを返却する
func GetConfig(configFilePath string, cmdlineConfig ServerConfig, errorChannel chan error) (ServerConfig, AgentConfig, error) {
	// 設定ファイル項目を設定ファイルやデフォルト値から構成する
	var configObject Config
	var err error

	if len(configFilePath) > 0 {
		//設定ファイルをConfig構造体にパース
		configObject, err = parseConfig(configFilePath)
		if err != nil {
			errorChannel <- err
			return ServerConfig{}, AgentConfig{}, err
		}
	} else {
		//デフォルト値からConfig構造体にパース
		configObject = getDefaultConfig()
	}
	//コマンドライン引数と設定ファイルとデフォルト値からConfig構造体を構成する
	return mergeConfigs(cmdlineConfig, configObject)
}
