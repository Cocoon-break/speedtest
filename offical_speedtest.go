package speedtest

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

//command: speedtest -f json,output struct
type SpeedtestCliResult struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Ping      struct {
		Jitter  float64 `json:"jitter"`
		Latency float64 `json:"latency"`
	} `json:"ping"`
	Download struct {
		Bandwidth int `json:"bandwidth"`
		Bytes     int `json:"bytes"`
		Elapsed   int `json:"elapsed"`
	} `json:"download"`
	Upload struct {
		Bandwidth int `json:"bandwidth"`
		Bytes     int `json:"bytes"`
		Elapsed   int `json:"elapsed"`
	} `json:"upload"`
	PacketLoss int    `json:"packetLoss"`
	Isp        string `json:"isp"`
	Interface  struct {
		InternalIP string `json:"internalIp"`
		Name       string `json:"name"`
		MacAddr    string `json:"macAddr"`
		IsVpn      bool   `json:"isVpn"`
		ExternalIP string `json:"externalIp"`
	} `json:"interface"`
	Server struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		Location string `json:"location"`
		Country  string `json:"country"`
		Host     string `json:"host"`
		Port     int    `json:"port"`
		IP       string `json:"ip"`
	} `json:"server"`
	Result struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	} `json:"result"`
}

// you must install speedtest cli
func BySpeedtestCli(interfaceOps []string, cmdTimoutSecond int) (BatchReport, error) {
	var batchReport BatchReport
	if len(interfaceOps) == 0 {
		return batchReport, errors.New("interfaceOps less 1")
	}
	var failedNet []string
	var reports []*SpeedReport
	for _, interfaceOp := range interfaceOps {
		_, stdout, err := ExecCmd("speedtest", cmdTimoutSecond, "--accept-license", "-I", interfaceOp, "-f", "json")
		if err != nil {
			failedNet = append(failedNet, interfaceOp)
			continue
		}
		var result SpeedtestCliResult
		err = json.Unmarshal([]byte(stdout), &result)
		if err != nil {
			failedNet = append(failedNet, interfaceOp)
			continue
		}
		reports = append(reports, transformToReport(result, interfaceOp))
	}
	batchReport.FailedNet = failedNet
	batchReport.SuccessNet = reports
	return batchReport, nil
}

func transformToReport(cliResult SpeedtestCliResult, interfaceOp string) *SpeedReport {
	report := &SpeedReport{
		UploadSpeed:   float64(cliResult.Upload.Bandwidth * 8 / 1000 / 1000),
		DownloadSpeed: float64(cliResult.Download.Bandwidth * 8 / 1000 / 1000),
	}
	report.SpeedtestServer.Country = cliResult.Server.Country
	report.SpeedtestServer.Name = cliResult.Server.Name
	report.SpeedtestServer.Latency = fmt.Sprintf("%+v", cliResult.Ping.Latency)
	report.NetInterface.Name = interfaceOp
	report.NetInterface.InternalIp = cliResult.Interface.InternalIP
	return report
}
