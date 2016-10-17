package logging

import (
	"fmt"
	"os"

	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/Sirupsen/logrus"
)

const (
	maxLogFileSizeInMB = 10
	maxNumLogFiles     = 10
)

// Fields はlogrusフィールド用オブジェクト
type Fields map[string]interface{}

// logはlogrusオブジェクト
var log = logrus.New()

// convertToLogrusFields はlogrusフィールドに変換するファンクション
func convertToLogrusFields(fields Fields) logrus.Fields {
	result := logrus.Fields{}
	for k := range fields {
		result[k] = fields[k]
	}
	return result
}

// Debug はメッセージとlogrusフィールドをレベルDebugとして定義するファンクション
// 出力例：time="2015-03-26T01:27:38-04:00" level=debug msg="Failed to send event" url=... error=... response=...
func Debug(msg string, fields Fields) {
	if fields != nil {
		log.WithFields(convertToLogrusFields(fields)).Debug(msg)
	} else {
		log.Debug(msg)
	}
}

// Info はメッセージとlogrusフィールドをレベルInfoとして定義するファンクション
func Info(msg string, fields Fields) {
	if fields != nil {
		log.WithFields(convertToLogrusFields(fields)).Info(msg)
	} else {
		log.Info(msg)
	}
}

// Warn はメッセージとlogrusフィールドをレベルWarnとして定義するファンクション
func Warn(msg string, fields Fields) {
	if fields != nil {
		log.WithFields(convertToLogrusFields(fields)).Warn(msg)
	} else {
		log.Warn(msg)
	}
}

// Error はメッセージとlogrusフィールドをレベルErrorとして定義するファンクション
func Error(msg string, fields Fields) {
	if fields != nil {
		log.WithFields(convertToLogrusFields(fields)).Error(msg)
	} else {
		log.Error(msg)
	}
}

// SetupLogger はAgentのログフォーマットを定義するファンクション
func SetupLogger(logfile string, debugMode bool, errorsChannel chan string) error {
	f, err := os.OpenFile(logfile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return err
	}
	defer f.Close()

	//ログ出力設定をローテートlibraryのlumberjack構造体に定義
	log.Out = &lumberjack.Logger{
		Filename:   logfile,
		MaxSize:    maxLogFileSizeInMB, // megabytes
		MaxBackups: maxNumLogFiles,
		LocalTime:  true,
	}

	//Hook処理

	// ログレベルは設定ファイルのDebugModeによる、infoかdebugか
	if debugMode {
		log.Level = logrus.DebugLevel
	} else {
		log.Level = logrus.InfoLevel
	}

	log.Formatter = &logrus.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	}
	return nil
}
