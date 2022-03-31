package speedtest

import (
	"errors"
	"sync"
)

// speedtest by distance
func ByDistance(interfaceOp string, httpTimeout int) (*SpeedReport, error) {
	st, err := initStClient(interfaceOp, httpTimeout)
	if err != nil {
		return nil, err
	}
	servers, err := st.FetchServerList()
	if err != nil {
		return nil, err
	}
	servers, err = st.ServerListByDistance(servers)
	if err != nil {
		return nil, err
	}
	if len(servers) == 0 {
		return nil, errors.New("not found speedtest server")
	}
	nearest := servers[0]
	return nearest.Report(interfaceOp, httpTimeout)
}

// speedtest by latency
func ByLatency(interfaceOp string, httpTimeout int) (*SpeedReport, error) {
	st, err := initStClient(interfaceOp, httpTimeout)
	if err != nil {
		return nil, err
	}
	servers, err := st.FetchServerList()
	if err != nil {
		return nil, err
	}
	servers, err = st.ServerListByLatency(servers)
	if err != nil {
		return nil, err
	}
	if len(servers) == 0 {
		return nil, errors.New("not found speedtest server")
	}
	fastServer := servers[0]

	return fastServer.Report(interfaceOp, httpTimeout)
}

type BatchReport struct {
	SuccessNet []*SpeedReport `json:"success_net"`
	FailedNet  []string       `json:"failed_net"`
}

// useless for test ,limit by speedtest server
func Concurrent(interfaceOps []string, httpTimeout int, isLatency bool) (*BatchReport, error) {
	if len(interfaceOps) == 0 {
		return nil, errors.New("interfaceOps less 1")
	}
	st, err := initStClient(interfaceOps[0], httpTimeout)
	if err != nil {
		return nil, err
	}
	servers, err := st.FetchServerList()
	if err != nil {
		return nil, err
	}
	if isLatency {
		servers, err = st.ServerListByLatency(servers)
	} else {
		servers, err = st.ServerListByDistance(servers)
	}
	if err != nil {
		return nil, err
	}
	if len(servers) == 0 {
		return nil, errors.New("not found speedtest server")
	}
	targetServer := servers[0]
	var mu sync.Mutex
	var wg sync.WaitGroup
	var reports []*SpeedReport
	var failedNet []string
	for i := range interfaceOps {
		wg.Add(1)
		go func(s *serverItem, interfaceOp string, httpTimeout int) {
			report, err := s.Report(interfaceOp, httpTimeout)
			mu.Lock()
			if err != nil {
				failedNet = append(failedNet, interfaceOp)
			} else {
				reports = append(reports, report)
			}
			mu.Unlock()
			wg.Done()
		}(targetServer, interfaceOps[i], httpTimeout)
	}
	wg.Wait()
	batchReport := &BatchReport{
		SuccessNet: reports,
		FailedNet:  failedNet,
	}
	return batchReport, nil
}

// speedtest one by one with config eth name
func OnebyOne(interfaceOps []string, httpTimeout int, isLatency bool, testNum int) (*BatchReport, error) {
	if len(interfaceOps) == 0 {
		return nil, errors.New("interfaceOps less 1")
	}
	st, err := initStClient(interfaceOps[0], httpTimeout)
	if err != nil {
		return nil, err
	}
	servers, err := st.FetchServerList()
	if err != nil {
		return nil, err
	}
	if isLatency {
		servers, err = st.ServerListByLatency(servers)
	} else {
		servers, err = st.ServerListByDistance(servers)
	}
	if err != nil {
		return nil, err
	}
	if len(servers) == 0 {
		return nil, errors.New("not found speedtest server")
	}
	if testNum < 0 || len(servers) < testNum {
		testNum = len(servers)
	}
	testServers := make([]*serverItem, testNum)
	for i := 0; i < testNum; i++ { // fast server
		testServers[i] = servers[i]
	}
	var reports []*SpeedReport
	var failedNet []string
	for i := range interfaceOps {
		var maxSpeedReport *SpeedReport
		for j := range testServers {
			fastServer := testServers[j]
			report, err := fastServer.Report(interfaceOps[i], httpTimeout)
			if err != nil {
				continue
			}
			if maxSpeedReport.UploadSpeed < report.UploadSpeed {
				maxSpeedReport = report
			}
		}
		if maxSpeedReport.UploadSpeed < 1 { //小于1M 则直接认为是失败
			failedNet = append(failedNet, interfaceOps[i])
		} else {
			reports = append(reports, maxSpeedReport)
		}
	}
	batchReport := &BatchReport{
		SuccessNet: reports,
		FailedNet:  failedNet,
	}
	return batchReport, nil
}
