package speedtest

import (
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"
)

// 从speedtest获取配置使用
const stConfigUrl = "https://www.speedtest.net/speedtest-config.php"

// 从speedtest获取可以测速的站点列表
const stServersUrl = "https://www.speedtest.net/speedtest-servers-static.php"

type config struct {
	IP  string
	Lat float64
	Lon float64
	Isp string
}
type netInterface struct {
	Name       string `json:"name"`
	InternalIp string `json:"internal_ip"`
}

type STClient struct {
	Config       *config
	NetInterface *netInterface
	Timeout      int
}

// speedtest response xml
type remoteConfig struct {
	Clients []struct {
		IP  string `xml:"ip,attr"`
		Lat string `xml:"lat,attr"`
		Lon string `xml:"lon,attr"`
		Isp string `xml:"isp,attr"`
	} `xml:"client"`
}

func initStClient(interfaceOp string, timeout int) (*STClient, error) {
	url := fmt.Sprintf("%s?x=%+v", stConfigUrl, time.Now().Unix())
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Cache-Control", "no-cache")
	httpUtil, err := getHttpUtil(interfaceOp, timeout)
	if err != nil {
		return nil, err
	}
	resp, err := httpUtil.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	decoder := xml.NewDecoder(resp.Body)
	var remoteConfig remoteConfig
	if err := decoder.Decode(&remoteConfig); err != nil {
		return nil, err
	}

	if len(remoteConfig.Clients) == 0 {
		return nil, errors.New("failed to fetch speedtest config clients information")
	}
	client := remoteConfig.Clients[0]
	c := &config{
		IP:  client.IP,
		Lat: stringToFloat(client.Lat),
		Lon: stringToFloat(client.Lon),
		Isp: client.Isp,
	}
	n := &netInterface{
		Name:       httpUtil.Interface.Name,
		InternalIp: httpUtil.Interface.InternalIp,
	}
	return &STClient{
		Config:       c,
		NetInterface: n,
		Timeout:      timeout,
	}, nil
}

// speedtest response xml
type server struct {
	URL     string `xml:"url,attr" json:"url"`
	Lat     string `xml:"lat,attr" json:"lat"`
	Lon     string `xml:"lon,attr" json:"lon"`
	Name    string `xml:"name,attr" json:"name"`
	Country string `xml:"country,attr" json:"country"`
	Sponsor string `xml:"sponsor,attr" json:"sponsor"`
	ID      string `xml:"id,attr" json:"id"`
	URL2    string `xml:"url2,attr" json:"url_2"`
	Host    string `xml:"host,attr" json:"host"`
}

// speedtest response xml
type serverList struct {
	Servers []server `xml:"servers>server"`
}

func (st *STClient) ServerListByLatency(servers []*serverItem) ([]*serverItem, error) {
	wg := sync.WaitGroup{}
	for i := range servers {
		wg.Add(1)
		go func(s *serverItem, in string) {
			latency, err := s.LatencyTest(in, 15)
			if err != nil {
				s.Latency = time.Duration(1 * time.Minute)
			} else {
				s.Latency = latency
			}
			wg.Done()
		}(servers[i], st.NetInterface.Name)
	}
	wg.Wait()
	sort.Sort(latency(servers))
	return servers, nil
}

func (st *STClient) ServerListByDistance(servers []*serverItem) ([]*serverItem, error) {
	for i := range servers {
		dis := calDistance(servers[i].Lat, servers[i].Lon, st.Config.Lat, st.Config.Lon)
		servers[i].Distance = dis
	}
	sort.Sort(distance(servers))
	return servers, nil
}

// fetch server list from speedtest api
func (st *STClient) FetchServerList() ([]*serverItem, error) {
	if st.Config == nil {
		return nil, errors.New("didn't init config")
	}
	url := fmt.Sprintf("%s?x=%+v", stServersUrl, time.Now().Unix())
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Cache-Control", "no-cache")
	httpUtil, err := getHttpUtil(st.NetInterface.Name, st.Timeout)
	if err != nil {
		return nil, err
	}
	resp, err := httpUtil.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	decoder := xml.NewDecoder(resp.Body)

	var list serverList
	if err := decoder.Decode(&list); err != nil {
		return nil, err
	}
	serverItems := make([]*serverItem, 0, len(list.Servers))
	for i := range list.Servers {
		speedtestServer := list.Servers[i]
		sItem := &serverItem{}
		sItem.URL = speedtestServer.URL
		sItem.Lat = stringToFloat(speedtestServer.Lat)
		sItem.Lon = stringToFloat(speedtestServer.Lon)
		sItem.Name = speedtestServer.Name
		sItem.Country = speedtestServer.Country
		sItem.Sponsor = speedtestServer.Sponsor
		sItem.ID = speedtestServer.ID
		serverItems = append(serverItems, sItem)
	}
	return serverItems, nil
}
