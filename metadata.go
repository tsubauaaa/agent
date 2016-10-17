package agent

import (
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/tsubauaaa/agent/logging"
)

// HostMetaData は登録するAgent情報の構造体
type HostMetaData struct {
	HostName         string
	AssignedHostname string
	ProviderID       string
	ProviderType     string
	Platform         string
	PrivateIPAddress string
	PrivateDNSName   string
	PublicIPAddress  string
	PublicDNSName    string
	Region           string
}

// queryDataはurlに接続して応答を取得するファンクション
func queryData(url string) (string, error) {
	timeout := time.Duration(5 * time.Second)
	client := http.Client{
		Timeout: timeout,
	}

	logging.Debug("Querying the url for cloud metadata.", logging.Fields{"url": url})
	resp, err := client.Get(url)
	if err != nil {
		logging.Warn("Could not query cloud metadata.", logging.Fields{
			"url":      url,
			"error":    err,
			"response": resp,
		})
		return "", err
	}

	if 200 <= resp.StatusCode && resp.StatusCode <= 299 {
		defer resp.Body.Close()
		contents, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			logging.Warn("Could not read response.", logging.Fields{"url": url, "error": err})
			return "", errors.New("Cloud not read response.")
		}
		return string(contents), nil
	}
	return "", errors.New("Server returned unexpected status: " + strconv.Itoa(resp.StatusCode))
}

// getLocalIP はAgentホストのローカルIPアドレスを取得するファンクション
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

// GetHostMetaData はAgentが起動するホストのメタデータを取得するファンクション
func GetHostMetaData(agentConfig *AgentConfig) (HostMetaData, error) {
	logging.Debug("Getting host metadata.", nil)

	hostname, err := os.Hostname()
	if err != nil {
		logging.Error("Could not get host name.", logging.Fields{"error": err})
		os.Exit(1)
	}
	privateIP := getLocalIP()
	publicIP, err := queryData("http://ip.42.pl/raw")
	platform := string(runtime.GOOS) + " " + string(runtime.GOARCH)

	var privateDNS string
	if addr, e := net.LookupAddr(privateIP); err == nil && len(addr) > 0 {
		privateDNS = addr[0]
	} else {
		logging.Warn("Cloud not get private DNS name.", logging.Fields{"error": e})
	}

	var publicDNS string
	if addr, e := net.LookupAddr(publicIP); err == nil && len(addr) > 0 {
		publicDNS = addr[0]
	} else {
		logging.Warn("Cloud not get public DNS name.", logging.Fields{"error": e})
	}

	var providerServerID string
	var providerType string
	var region string
	providerServerID, err = queryData("http://169.254.169.254/latest/meta-data/instance-id")
	if err == nil && len(providerServerID) > 0 {
		providerType = "AWS"
		regionValue, err := queryData("http://169.254.169.254/latest/meta-data/placement/availability-zone")
		if err != nil && len(regionValue) > 0 {
			region = regionValue
		}
	} else {
		providerType = "NON_AWS"
	}

	data := HostMetaData{
		HostName:         hostname,
		AssignedHostname: agentConfig.AssignedHostname,
		PrivateIPAddress: privateIP,
		PublicIPAddress:  publicIP,
		Platform:         platform,
		PrivateDNSName:   privateDNS,
		PublicDNSName:    publicDNS,
		ProviderID:       providerServerID,
		ProviderType:     providerType,
		Region:           region,
	}
	return data, nil
}
