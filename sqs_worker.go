package agent

import (
	"encoding/json"
	"net/http"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/tsubauaaa/agent/logging"
)

// SQS設定の定数
const (
	sqsPollingFrequencySecs            = 5
	maxNumMessagesToFetch              = 10
	longPollTimeSeconds                = 20
	defaultVisibilityTimeout           = 120
	numSQSFailuresBeforeReregistration = 10
)

// queueURLRegex はSQSエンドポイントの正規表現文字列
var queueURLRegex = regexp.MustCompile(`https://sqs\.(.*)\.amazonaws.com(.*)`)

//requiredAttributes はSQSメッセージ属性用変数
var requiredAttributes []*string

func init() {
	// agentIDおよびsignatureはどのSQSメッセージにも付与される属性情報
	agentIDAttr := "agentID"
	signatureAttr := "signature"
	requiredAttributes = append(requiredAttributes, &agentIDAttr)
	requiredAttributes = append(requiredAttributes, &signatureAttr)
}

// changeMessageVisibility はSQSメッセージの可視時間を変更するファンクション
func changeMessageVisibility(svc *sqs.SQS, queue, receiptHandle string, timeout int64) error {
	params := &sqs.ChangeMessageVisibilityInput{
		QueueUrl:          &queue,
		ReceiptHandle:     &receiptHandle,
		VisibilityTimeout: &timeout,
	}
	_, err := svc.ChangeMessageVisibility(params)
	if err != nil {
		logging.Error("Cloud not change the message visibility.", logging.Fields{
			"receipt": receiptHandle,
			"error":   err,
		})
		return err
	}
	return nil
}

// DeleteMessage はSQSメッセージを削除するファンクション
func DeleteMessage(svc *sqs.SQS, queue, receiptHandle string) error {
	logging.Debug("Deleting the event from SQS.", nil)

	params := &sqs.DeleteMessageInput{
		QueueUrl:      &queue,
		ReceiptHandle: &receiptHandle,
	}
	_, err := svc.DeleteMessage(params)
	if err != nil {
		logging.Error("Cloud not delete the event.", logging.Fields{"error": err})
		return err
	}
	return nil
}

// parseQueueDetails はSQSエンドポイントの正規表現文字列とSQSエンドポイント文字列からRegionを求めるファンクション
func parseQueueDetails(queueURL string) string {
	result := queueURLRegex.FindStringSubmatch(queueURL)
	return result[1]
}

// getMessages はSQSをポーリングしてメッセージを取得するファンクション
func getMessages(svc *sqs.SQS, queue string) (*sqs.ReceiveMessageOutput, error) {
	params := &sqs.ReceiveMessageInput{
		QueueUrl:            &queue,
		MaxNumberOfMessages: aws.Int64(maxNumMessagesToFetch),
		VisibilityTimeout:   aws.Int64(defaultVisibilityTimeout),
		WaitTimeSeconds:     aws.Int64(longPollTimeSeconds),
		// SQSメッセージ属性を使用：http://docs.aws.amazon.com/ja_jp/AWSSimpleQueueService/latest/SQSDeveloperGuide/SQSMessageAttributes.html
		MessageAttributeNames: requiredAttributes,
	}
	logging.Debug("Polling SQS queue for messages.", nil)
	resp, err := svc.ReceiveMessage(params)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// getSQSClient はSQS接続クライアントをAWSアクセスキーおよびAWSシークレットアクセスキーから生成するファンクション
func getSQSClient(regInfo *RegistrationInfo) *sqs.SQS {
	creds := credentials.NewStaticCredentials(regInfo.AWSAccessKey, regInfo.AWSSecretAccessKey, regInfo.AWSSecurityToken)
	region := parseQueueDetails(regInfo.ActionQueueEndpoint)
	awsConfig := aws.NewConfig().WithCredentials(creds).
		WithRegion(region).
		WithHTTPClient(http.DefaultClient).
		WithMaxRetries(aws.UseServiceDefaultRetries).
		WithLogger(aws.NewDefaultLogger()).
		WithLogLevel(aws.LogOff).
		WithSleepDelay(time.Sleep)
	return sqs.New(session.New(awsConfig))
}

// RunLoop は現在は無限に連続してSQSメッセージを取得しに行ってしまう(getMessagesによって)
// そのためSleep処理が必要である
func RunLoop(regInfo *RegistrationInfo, regInfoUpdatesCh <-chan string, eventsChannel chan<- *Event, regChannel chan<- time.Time) {
	logging.Info("Initializing SQS client.", nil)
	svc := getSQSClient(regInfo)
	queue := regInfo.ActionQueueEndpoint

	// shouldLogErrorはgetMessagesファンクションによるSQSメッセージ取得に失敗となった場合に再AgentRegistrationするか
	// を判断する処理に遷移するかを決定するために用いる変数
	shouldLogError := true
	// numFailuresはgetMessagesファンクションによるSQSメッセージ取得にnumSQSFailuresBeforeReregistration回数失敗となった場合に
	// 再度AgentRegistrationするかを判断する処理で用いる変数
	numFailures := 0
	for {
		//shouldSleepはAgentのSQSメッセージがない場合のSleep制御のための変数。AgentのSQSメッセージが存在する場合はfalseになる
		shouldSleep := true
		select {

		// Agent登録情報が変更される、もしくはSQSクライアントが初期化される場合
		case <-regInfoUpdatesCh:
			logging.Info("Initializing SQS client.", nil)
			svc = getSQSClient(regInfo)
			queue = regInfo.ActionQueueEndpoint

		default:
			//getMessagesファンクションによるSQSメッセージ取得開始時刻のための変数
			t1 := time.Now()
			if resp, err := getMessages(svc, queue); err == nil {
				shouldLogError = true
				numFailures = 0
				//Agentステータス更新処理
				logging.Debug("Received messages.", logging.Fields{"count": len(resp.Messages)})

				for _, msg := range resp.Messages {
					bodyStr := *msg.Body // SQSメッセージVerifyで使う
					messageID := *msg.MessageId

					//SQSメッセージ属性であるagentIDがあることをチェック
					agentID, ok := msg.MessageAttributes["agentID"]
					if !ok {
						logging.Error("Received message does not have agentID attributes.", logging.Fields{"msgID": messageID})
						continue
					}
					//SQSメッセージ属性agentIDとAgent登録情報内のAgentIDとを照合
					if regInfo.AgentID == *agentID.StringValue {
						logging.Debug("Received a message for me. Checking message integrity.", nil)

						signature, ok := msg.MessageAttributes["signature"]
						if !ok {
							logging.Error("Received message does not have signature attributes.", logging.Fields{"msgID": messageID})
							continue
						}

						if valid, e := VerifyMessage(bodyStr, *signature.StringValue); valid && e == nil {
							var event Event
							err := json.Unmarshal([]byte(bodyStr), &event)
							if err != nil {
								logging.Error("Cloud not deserialize the SQS message.", logging.Fields{"error": err})
							} else {
								event.SQSMessageID = messageID
								event.ReceiptHandle = *msg.ReceiptHandle
							}
							// Agent登録情報とSQSメッセージ内のAgetnIDを照合して、合致したらActionを実行する処理
							// Agent登録情報とSQSメッセージ属性値のAgentIDが合致していてもメッセージ改ざんしているかをチェックする処理
							if regInfo.AgentID == event.AgentID {
								// SQSメッセージの可視時間にSQSメッセージ内のタイムアウト値に加えて2秒のバッファを設ける処理
								// これはアクションの処理中に競合することを回避する処理
								changeMessageVisibility(svc, queue, event.ReceiptHandle, int64(event.Timeout+2))

								logging.Debug("Pushing the message for processing.", logging.Fields{"eventID": event.EventID})
								eventsChannel <- &event
								shouldSleep = false
							} else {
								// 本来はありえない場合。通常はSQSメッセージ属性値とSQSメッセージ内のAgentIDは合致するので異常な場合の処理
								logging.Error("Something is wrong!! Agent id present in the message attributes matches but "+
									"agent id in event does not match. Deleting the message.",
									logging.Fields{"msgID": messageID})
								DeleteMessage(svc, queue, *msg.ReceiptHandle)
							}
						} else {
							logging.Error("Cloud not verify the message with signature so deleting the message.",
								logging.Fields{"error": err})
							DeleteMessage(svc, queue, *msg.ReceiptHandle)
						}

					} else {
						// SQSメッセージ属性値AgentIDがAgent登録情報と一致しなかった場合の処理(他のAgentのメッセージと判断)
						logging.Debug("Releasing a message which is not for me.", logging.Fields{"msgID": messageID})
						// 他のAgentのメッセージの可能性があるため可視時間を無制限にする
						changeMessageVisibility(svc, queue, *msg.ReceiptHandle, int64(0))
					}
				}
			} else if shouldLogError {
				// getMessagesによるSQSメッセージ取得に失敗してエラーとなった場合の処理
				logging.Error("Could not receive message from SQS.", logging.Fields{
					"error":    err,
					"response": resp,
				})
				// getMessagesによるSQSメッセージ取得に失敗してエラーとなった場合にshouldLogErrorをfalseにする
				shouldLogError = false
				numFailures++
			} else {
				// getMessagesによるSQSメッセージ取得に2回以上失敗した場合の処理
				numFailures++

				// numFailuresがnumSQSFailuresBeforeReregistration回数に達したらnumFailuresとshouldLogErrorを初期化して
				// 再度AgentRegistrationする処理
				if numFailures == numSQSFailuresBeforeReregistration {
					numFailures = 0
					shouldLogError = true
					regChannel <- time.Now()
				}
			}
			// SQSポーリングを少なくともsqsPollingFrequencySecsに定義した秒数Sleepさせる処理
			if shouldSleep {
				if duration := t1.Add(time.Second * sqsPollingFrequencySecs).Sub(time.Now()); duration > 0 {
					logging.Debug("Sleeping between two polls.", logging.Fields{"duration": duration})
					time.Sleep(duration)
				}
			}
		}
	}
}
