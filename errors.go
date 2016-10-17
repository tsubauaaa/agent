package agent

// ErrorsChannel はAgentにエラーが発生したら、このチャネルにエラーをプッシュ
var ErrorsChannel = make(chan string, 10)

// AgentError はAgentエラーに必要な情報の構造体
type AgentError struct {
	ErrorMessage string
	AgentID      string
	FullLogs     bool
	Hostname     string
	Status       string
}

// ReportError はAgentにエラーが発生した場合にチャネルにエラーをプッシュ
func ReportError(err string) {
	ErrorsChannel <- err
}
