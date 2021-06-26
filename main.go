// stream_check project main.go
package main

import (
	"common/compile"
	"delay_stream/src/stream"
	"flag"
	"fmt"

	log4plus "github.com/alecthomas/log4go"
)

//测流版本号
var ver string = "1.0.7"

func main() {

	if err := log4plus.SetupLogWithConf("./log.json"); err != nil {
		panic(err)
	}
	defer log4plus.Close()

	//查看版本指令 ./main -V
	checkVer := flag.Bool("V", false, "is ok")
	flag.Parse()
	if *checkVer {
		verString := "Delay Stream Version: " + ver + "\r\n"
		verString += compile.BuildTime() + "\r\n"
		fmt.Println(verString)
		log4plus.Debug(verString)
		return
	}

	//启动流处理
	streamMgrPtr := stream.CreateStreamManager()
	streamMgrPtr.Initialize()
	streamMgrPtr.Run()
}
