package speedtest

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
)

var dlSizes = [...]int{350, 500, 750, 1000, 1500, 2000, 2500, 3000, 3500, 4000}
var ulSizes = [...]int{100, 300, 500, 800, 1000, 1500, 2500, 3000, 3500, 4000}

// Server struct is a speedtest candidate server
type serverItem struct {
	URL      string
	Lat      float64
	Lon      float64
	Name     string
	Country  string
	Sponsor  string
	ID       string
	Distance float64
	Latency  time.Duration
}

type SpeedResult struct {
	NetInterfaceName string
	NetInterfaceIp   string
	SpeedUpload      float64
	SpeedDownload    float64
	Latency          time.Duration
}

type SpeedReport struct {
	DownloadSpeed float64       `json:"download_speed"`
	UploadSpeed   float64       `json:"upload_speed"`
	Latency       time.Duration `json:"latency"`

	SpeedtestServer struct {
		Lat      float64 `json:"lat"`
		Lon      float64 `json:"lon"`
		Name     string  `json:"name"`
		Country  string  `json:"country"`
		Sponsor  string  `json:"sponsor"`
		Distance float64 `json:"distance"`
		Latency  string  `json:"latency"`
	} `json:"speedtest_server"`
	NetInterface struct {
		Name       string `json:"name"`
		InternalIp string `json:"internal_ip"`
	} `json:"net_interface"`
}

func (s *serverItem) Report(interfaceOp string, timeout int) (*SpeedReport, error) {
	report := &SpeedReport{}
	report.SpeedtestServer.Lat = s.Lat
	report.SpeedtestServer.Lon = s.Lon
	report.SpeedtestServer.Name = s.Name
	report.SpeedtestServer.Country = s.Country
	report.SpeedtestServer.Sponsor = s.Sponsor
	result, err := s.StartSpeedTest(interfaceOp, timeout)
	if err != nil {
		return nil, err
	}
	report.Latency = result.Latency
	report.DownloadSpeed = result.SpeedDownload
	report.UploadSpeed = result.SpeedUpload
	report.NetInterface.Name = result.NetInterfaceName
	report.NetInterface.InternalIp = result.NetInterfaceIp
	return report, nil
}

// test with interfaceOp
func (s *serverItem) StartSpeedTest(interfaceOp string, timeout int) (*SpeedResult, error) {
	sourceIP, err := getSourceIP(interfaceOp)
	if err != nil {
		return nil, err
	}
	latency, err := s.LatencyTest(interfaceOp, timeout)
	if err != nil {
		return nil, err
	}
	uploadSpeed, err := s.uploadTest(interfaceOp, timeout, latency)
	if err != nil {
		return nil, err
	}
	downloadSpeed, err := s.downloadTest(interfaceOp, timeout, latency)
	if err != nil {
		return nil, err
	}
	result := &SpeedResult{
		NetInterfaceName: interfaceOp,
		NetInterfaceIp:   sourceIP,
		Latency:          latency,
		SpeedUpload:      uploadSpeed,
		SpeedDownload:    downloadSpeed,
	}
	return result, nil
}

func (s *serverItem) LatencyTest(interfaceOp string, timeout int) (latency time.Duration, err error) {
	pingURL := strings.Split(s.URL, "/upload.php")[0] + "/latency.txt"
	l := time.Duration(10 * time.Second)
	for i := 0; i < 3; i++ {
		sTime := time.Now()
		req, err := http.NewRequest(http.MethodGet, pingURL, nil)
		if err != nil {
			return latency, err
		}
		httpUtil, err := getHttpUtil(interfaceOp, timeout)
		if err != nil {
			return latency, err
		}
		resp, err := httpUtil.Client.Do(req)
		if err != nil {
			return latency, err
		}
		fTime := time.Now()
		if fTime.Sub(sTime) < l {
			l = fTime.Sub(sTime)
		}
		resp.Body.Close()
	}
	t := time.Duration(int64(l.Nanoseconds() / 2))
	return t, nil
}

func (s *serverItem) uploadTest(interfaceOp string, timeout int, latency time.Duration) (speedMB float64, err error) {
	warmSize := ulSizes[4]
	warmCount := 2

	sTime := time.Now()
	eg := errgroup.Group{}
	for i := 0; i < 2; i++ {
		eg.Go(func() error {
			v := url.Values{}
			v.Add("content", strings.Repeat("0123456789", warmSize*100-51))
			return upload(s.URL, interfaceOp, timeout, strings.NewReader(v.Encode()))
		})
	}
	if err := eg.Wait(); err != nil {
		return speedMB, err
	}
	fTime := time.Now()
	// contentSize := float64(warmSize*warmSize*2) / 1000 / 1000 //1.0M
	contentSize := float64(warmSize*warmSize) / 1000 / 1000 //1.0M
	warmSpeed := contentSize * 8 * float64(warmCount) / fTime.Sub(sTime.Add(latency)).Seconds()
	workload, weight := 0, 0
	switch {
	case 50.0 < warmSpeed:
		workload, weight = 40, 9
	case 10.0 < warmSpeed:
		workload, weight = 16, 9
	case 4.0 < warmSpeed:
		workload, weight = 8, 9
	case 2.5 < warmSpeed:
		workload, weight = 4, 5
	default:
		workload, weight = 1, 7
	}
	sTime = time.Now()
	for i := 0; i < workload; i++ {
		eg.Go(func() error {
			v := url.Values{}
			v.Add("content", strings.Repeat("0123456789", ulSizes[weight]*100-51))
			return upload(s.URL, interfaceOp, timeout, strings.NewReader(v.Encode()))
		})
	}
	if err := eg.Wait(); err != nil {
		return speedMB, err
	}
	fTime = time.Now()

	reqMB := float64(ulSizes[weight]) / 1000
	ulSpeed := reqMB * 8 * float64(workload) / fTime.Sub(sTime).Seconds()
	return ulSpeed, nil
}

func (s *serverItem) downloadTest(interfaceOp string, timeout int, latency time.Duration) (speedMB float64, err error) {
	dlURL := strings.Split(s.URL, "/upload.php")[0]

	sTime := time.Now()
	eg := errgroup.Group{}
	warmSize := dlSizes[2]
	warmCount := 2
	for i := 0; i < warmCount; i++ {
		eg.Go(func() error {
			size := strconv.Itoa(warmSize)
			url := fmt.Sprintf("%s%s%sx%s.jpg", dlURL, "/random", size, size)
			return download(url, interfaceOp, timeout)
		})
	}
	if err := eg.Wait(); err != nil {
		return speedMB, err
	}
	fTime := time.Now()
	contentSize := float64(warmSize*warmSize*2) / 1000 / 1000
	warmSpeed := contentSize * 8 * float64(warmCount) / fTime.Sub(sTime.Add(latency)).Seconds()
	workload, weight := 0, 0
	switch {
	case 50.0 < warmSpeed:
		workload, weight = 32, 6
	case 10.0 < warmSpeed:
		workload, weight = 16, 4
	case 4.0 < warmSpeed:
		workload, weight = 8, 4
	case 2.0 < warmSpeed:
		workload, weight = 4, 4
	default:
		workload, weight = 6, 3
	}
	for i := 0; i < workload; i++ {
		eg.Go(func() error {
			size := strconv.Itoa(dlSizes[weight])
			url := fmt.Sprintf("%s%s%sx%s.jpg", dlURL, "/random", size, size)
			return download(url, interfaceOp, timeout)
		})
	}
	if err := eg.Wait(); err != nil {
		return speedMB, err
	}
	fTime = time.Now()
	reqMB := dlSizes[weight] * dlSizes[weight] * 2 / 1000 / 1000
	dlSpeed := float64(reqMB) * 8 * float64(workload) / fTime.Sub(sTime).Seconds()
	return dlSpeed, nil
}

// ByDistance allows us to sort servers by distance
type distance []*serverItem

func (server distance) Len() int {
	return len(server)
}

func (server distance) Less(i, j int) bool {
	return server[i].Distance < server[j].Distance
}

func (server distance) Swap(i, j int) {
	server[i], server[j] = server[j], server[i]
}

// ByLatency allows us to sort servers by latency
type latency []*serverItem

func (server latency) Len() int {
	return len(server)
}

func (server latency) Less(i, j int) bool {
	return server[i].Latency < server[j].Latency
}

func (server latency) Swap(i, j int) {
	server[i], server[j] = server[j], server[i]
}

func upload(uploadUrl, interfaceOp string, timeout int, body io.Reader) error {
	req, err := http.NewRequest(http.MethodPost, uploadUrl, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpUtil, err := getHttpUtil(interfaceOp, timeout)
	if err != nil {
		return err
	}
	resp, err := httpUtil.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = io.Copy(ioutil.Discard, resp.Body)
	return err
}

func download(url, interfaceOp string, timeout int) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	httpUtil, err := getHttpUtil(interfaceOp, timeout)
	if err != nil {
		return err
	}

	resp, err := httpUtil.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = io.Copy(ioutil.Discard, resp.Body)
	return err
}
