package speedtest

import (
	"errors"
	"math"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"time"
)

type httpUtil struct {
	Client    *http.Client
	Interface struct {
		Name       string
		InternalIp string
	}
}

func init() {
	rand.Seed(time.Now().Unix())
}

func getHttpUtil(interfaceOption string, timeout int) (*httpUtil, error) {
	util := &httpUtil{}
	httpTimeout := time.Duration(timeout) * time.Second

	dialer := net.Dialer{
		Timeout:   httpTimeout,
		KeepAlive: httpTimeout,
	}

	sourceIP, err := getSourceIP(interfaceOption)
	if err != nil {
		return nil, err
	}
	if sourceIP != "" {
		if sourceIP == interfaceOption {
			util.Interface.InternalIp = sourceIP
		} else {
			util.Interface.InternalIp = sourceIP
			util.Interface.Name = interfaceOption
		}
		bindAddrIP, err := net.ResolveIPAddr("ip", sourceIP)
		if err != nil {
			return nil, err
		}
		dialer.LocalAddr = &net.TCPAddr{IP: bindAddrIP.IP}
	}
	transport := &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		Dial:                dialer.Dial,
		TLSHandshakeTimeout: httpTimeout,
	}
	client := &http.Client{
		Timeout:   httpTimeout,
		Transport: transport,
	}
	util.Client = client
	return util, nil
}

func getSourceIP(interfaceOption string) (string, error) {
	if interfaceOption == "" {
		return "", nil
	}
	// does it look like an IP address?
	if net.ParseIP(interfaceOption) != nil {
		return interfaceOption, nil
	}

	// assume that it is the name of an interface
	iface, err := net.InterfaceByName(interfaceOption)
	if err != nil {
		return "", err
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		switch v := addr.(type) {
		case *net.IPNet:
			// todo: IPv6
			if v.IP.To4() != nil {
				return v.IP.String(), nil
			}
		case *net.IPAddr:
			if v.IP.To4() != nil {
				return v.IP.String(), nil
			}
		}
	}
	return "", errors.New("no address found")
}

func stringToFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func calDistance(lat1 float64, lon1 float64, lat2 float64, lon2 float64) float64 {
	radius := 6378.137

	a1 := lat1 * math.Pi / 180.0
	b1 := lon1 * math.Pi / 180.0
	a2 := lat2 * math.Pi / 180.0
	b2 := lon2 * math.Pi / 180.0

	x := math.Sin(a1)*math.Sin(a2) + math.Cos(a1)*math.Cos(a2)*math.Cos(b2-b1)
	return radius * math.Acos(x)
}

func randomNumStr(n int) string {
	b := make([]byte, n)
	for i := 0; i < n; i++ {
		b[i] = byte(rand.Int31())
	}
	return string(b)
}
