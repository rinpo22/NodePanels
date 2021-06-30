package main

import (
	"encoding/json"
	"fmt"
	"github.com/kardianos/service"
	"nodepanels/config"
	"nodepanels/probe"
	"nodepanels/util"
	"nodepanels/websocket"
	"os"
	"runtime"
	"time"
)

//go:generate goversioninfo -icon=favicon.ico

func main() {

	serConfig := &service.Config{
		Name:        "Nodepanels-probe",
		DisplayName: "Nodepanels-probe",
		Description: "Nodepanels探针进程",
	}

	pro := &Program{}
	s, err := service.New(pro, serConfig)
	if err != nil {
		fmt.Println(err, "Create service failed")
	}

	if len(os.Args) > 1 {
		if os.Args[1] == "install" {
			err = s.Install()
			if err != nil {
				fmt.Println("Install failed", err)
			} else {
				fmt.Println("Install success")
			}
			return
		}

		if os.Args[1] == "uninstall" {
			err = s.Uninstall()
			if err != nil {
				fmt.Println("Uninstall err", err)
			} else {
				fmt.Println("Uninstall success")
			}
			return
		}
	}

	err = s.Run()
	if err != nil {
		fmt.Println("Run nodepanels-probe failed", err)
	}

}

type Program struct{}

func (p *Program) Start(s service.Service) error {
	fmt.Println("nodepanels-probe start")
	go p.run()
	return nil
}

func (p *Program) run() {
	StartProbe()
}

func (p *Program) Stop(s service.Service) error {
	fmt.Println("nodepanels-probe stop")
	return nil
}

func StartProbe() {
	util.LogFile, _ = os.OpenFile(util.Exepath()+"/log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	util.LogDebug("\n  _   _           _                             _     " +
		"\n | \\ | |         | |                           | |    " +
		"\n |  \\| | ___   __| | ___ _ __   __ _ _ __   ___| |___ " +
		"\n | . ` |/ _ \\ / _` |/ _ \\ '_ \\ / _` | '_ \\ / _ \\ / __|" +
		"\n | |\\  | (_) | (_| |  __/ |_) | (_| | | | |  __/ \\__ \\" +
		"\n |_| \\_|\\___/ \\__,_|\\___| .__/ \\__,_|_| |_|\\___|_|___/" +
		"\n                        | |                           " +
		"\n                        |_|                           " +
		"\n")

	runtime.GOMAXPROCS(1)

	//验证探针是否安装成功
	if ProbeCheck() {

		//发送服务器信息
		sendServerInfo()

		//与代理服务器建立websocket连接
		go websocket.CreateAgentConn()

		//循环发送服务器监控数据
		for {
			go sendUsageInfo()
			time.Sleep(60000 * time.Millisecond)
		}

	}
}

func ProbeCheck() bool {

	defer func() {
		err := recover()
		if err != nil {
			util.LogError("Check probe error : " + fmt.Sprintf("%s", err))
		}
	}()

	probe.InitConfigIp()
	if util.GetHostId() == "" {
		util.LogError("The program is not completely installed, please reinstall")
		return false
	}
	exist := util.Get("https://" + config.AgentUrl + "/server/exist/" + util.GetHostId())
	if exist == "1" {
		util.LogDebug("Program started successfully")
		return true
	} else {
		util.LogError("Invalid server ID, please reinstall")
		return false
	}
}

func sendUsageInfo() {

	defer func() {
		err := recover()
		if err != nil {
			util.LogError("Sending usage data error : " + fmt.Sprintf("%s", err))
		}
	}()

	probeUsage := probe.ProbeUsage{}
	probeUsage = probe.GetCpuUsage(probeUsage)
	probeUsage = probe.GetMemUsage(probeUsage)
	probeUsage = probe.GetSwapUsage(probeUsage)
	probeUsage = probe.GetDiskUsage(probeUsage)
	probeUsage = probe.GetNetUsage(probeUsage)
	probeUsage = probe.GetProcessNum(probeUsage)
	probeUsage = probe.GetProcessUsage(probeUsage)
	probeUsage = probe.GetLoadUsage(probeUsage)
	probeUsage.Unix = time.Now().Unix()

	msg, _ := json.Marshal(probeUsage)

	resultMap := make(map[string]string)
	resultMap["serverId"] = util.GetHostId()
	resultMap["msg"] = string(msg)
	result, _ := json.Marshal(resultMap)
	util.PostJson("https://"+config.ApiUrl+"/api/v1", result)
}

func sendServerInfo() {

	defer func() {
		err := recover()
		if err != nil {
			util.LogError("Sending server info data error : " + fmt.Sprintf("%s", err))
		}
	}()

	probeInfo := probe.ProbeInfo{}
	probeInfo.Version = config.Version
	probeInfo = probe.GetHostInfo(probeInfo)
	probeInfo = probe.GetCpuInfo(probeInfo)
	probeInfo = probe.GetMemInfo(probeInfo)
	probeInfo = probe.GetDiskInfo(probeInfo)
	probeInfo = probe.GetNetInfo(probeInfo)

	msg, _ := json.Marshal(probeInfo)

	resultMap := make(map[string]string)
	resultMap["serverId"] = util.GetHostId()
	resultMap["msg"] = string(msg)
	result, _ := json.Marshal(resultMap)

	go util.PostJson("https://"+config.AgentUrl+"/server/info", result)
}
