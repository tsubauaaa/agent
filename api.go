package agent

import "strings"

const (
	agentAPI = "/api/v1/agent/"
	protocol = "https://"
	slash    = "/"
)

// Event はServerから送信する単一のSQSメッセージの構造体
// 1つのEventは1つのRunbookに対応する
type Event struct {
	Timestamp        int64             `json:"timestamp"`
	Source           string            `json:"source"`
	HostName         string            `json:"hostname"`
	ActionType       string            `json:"action_type"`
	EventID          string            `json:"eventid"`
	AgentID          string            `json:"agentid"`
	RuleID           string            `json:"ruleid"`
	InflightActionID string            `json:"inflight_actionid"`
	RunbookName      string            `json:"runbook_name"`
	RawCommand       string            `json:"raw_command"`
	Signature        string            `json:"signature"`
	Timeout          int32             `json:"timeout"`
	GithubFilePath   string            `json:"github_filepath"`
	Enviroment       map[string]string `json:"env"`
	SQSMessageID     string            //SQSメッセージから取得
	ReceiptHandle    string            //SQSメッセージから取得
}

// joinURL はAPIリクエストURLを構成するファンクション
// URL1例:https://endpoint/api/v1/agent/arg1/arg2/arg3/...
func joinURL(endpoint string, args ...string) string {
	var trimmedArgs []string
	for _, arg := range args {
		trimmedArgs = append(trimmedArgs, strings.Trim(arg, slash))
	}

	return strings.Join([]string{protocol, strings.TrimRight(endpoint, slash), agentAPI, strings.Join(trimmedArgs[:len(trimmedArgs)], slash)}, "")
}
