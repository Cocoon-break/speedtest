### speedtest 

install to your project

```shell
go get github.com/Cocoon-break/speedtest
```

usage

Latency

```go
func byLatency() {
	report, err := speedtest.ByLatency("eth0", 60)
	if err != nil {
		fmt.Printf("failed:%s", err.Error())
		return
	}
	fmt.Printf("%+v", report)
}
```

Distance

```go
func byDistance() {
	report, err := speedtest.ByDistance("eth0", 60)
	if err != nil {
		fmt.Printf("failed:%s", err.Error())
		return
	}
	fmt.Printf("%+v", report)
}
```

BySpeedtestCli，the Preconditions is you installed speediest-cli

```go
func bySpeedtestclit() {
  report, err := speedtest.BySpeedtestCli([]string{"eth0"}, 120)
	if err != nil {
		fmt.Printf("failed:%s", err.Error())
		return
	}
	fmt.Printf("%+v", report)
}
```

note: the result of speed unit is MB.
