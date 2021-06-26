// stream_check project stream.go
package stream

import (
	//	"delay_stream/src/alarm"
	"delay_stream/src/m3u8"
	"delay_stream/src/public"
	"delay_stream/src/transfer"
	"delay_stream/src/ts"
	_ "encoding/json"
	"io/ioutil"
	"os"
	"strconv"
	"time"

	log4plus "github.com/alecthomas/log4go"
	"github.com/widuu/goini"
)

type tsFrameBody struct {
	Urls []string `json:"urls"`
}

type Stream struct {
	m3u8Url       string             //m3u8的下载地址
	localM3u8Path string             //本地保存m3u8的路径
	localTsPath   string             //保存TS文件的路径
	M3u8Name      string             //m3u8 name 例如700.m3u8
	usedDelay     bool               //是否使用延迟
	usedDelete    bool               //是否使用删除
	delayTime     int                //延时时长
	m3u8Ptr       *m3u8.M3u8         //M3U8的类
	transferPtr   *transfer.Transfer //延时
	tsPtr         *ts.Ts             //TS管理类
}

func (streamPtr *Stream) startDownload() {

	result, tsArray, localM3u8Name := streamPtr.m3u8Ptr.DownloadM3u8()
	if result {

		log4plus.Debug("streamPtr.tsPtr.DownloadUrl %s", streamPtr.m3u8Url)
		streamPtr.tsPtr.DownloadUrl(streamPtr.m3u8Url, tsArray)
		if streamPtr.usedDelay {

			//下载成功
			var srcM3u8Path string = streamPtr.localM3u8Path + "/m3u8/" + localM3u8Name
			log4plus.Debug("streamPtr.transferPtr.AddFIFOM3u8 %s", srcM3u8Path)
			streamPtr.transferPtr.AddFIFOM3u8(srcM3u8Path)
		}
	} else {
		//		alarm.DoAlarm(alarm.SeriousLevel, alarm.AlarmTypeM3u8DownFail, streamPtr.m3u8Url, "M3u8 download fail!")
	}
}

//启动检测
func (streamPtr *Stream) startCheckStream() {
	for {

		go streamPtr.startDownload()
		time.Sleep(999 * time.Millisecond)
	}
}

//删除指定目录
func (streamPtr *Stream) deleteDir(filePath string, saveDays string) bool {

	if public.FileExist(filePath) {

		dir, err := ioutil.ReadDir(filePath)
		if err != nil {
			log4plus.Error("readDir error:%s", err.Error())
			return false
		}

		for _, fi := range dir {

			if fi.IsDir() {

				tm, _ := time.Parse("20060102", fi.Name())
				days := (time.Now().Unix() - tm.Unix()) / 86400
				days2Save, err := strconv.ParseInt(saveDays, 10, 64)
				if days <= days2Save {
					continue
				}

				log4plus.Debug("days:%d to del dir :%s", days, filePath+"/"+fi.Name())
				err = os.RemoveAll(filePath + "/" + fi.Name())
				if err != nil {
					log4plus.Error("err :%s, when remove dir:%s", err.Error(), fi.Name())
				}
			}
		}
	}
	return true
}

func (streamPtr *Stream) deleteLog(m3u8Url, tsUrl string, days string) {

	for {

		log4plus.Debug("deleteLog localM3u8Path:%s,localTsPath:%s", m3u8Url, tsUrl)

		if !streamPtr.deleteDir(m3u8Url, days) || !streamPtr.deleteDir(tsUrl, days) {
			log4plus.Error("Delete Dir Error localM3u8Path:%s,localTsPath:%s", m3u8Url, tsUrl)
		}

		time.Sleep(60 * time.Second)
	}
}

//读取配置
func (streamPtr *Stream) loadConfig() {
	configPtr := goini.Init("config.ini")
	streamPtr.usedDelay = configPtr.ReadBool("DELAY", "used", true)
	streamPtr.usedDelete = configPtr.ReadBool("DELETE", "used", true)
}

//初始化
func CreateStream(m3u8Url string,
	delaytime int,
	localUrl string) *Stream {

	streamPtr := new(Stream)
	streamPtr.delayTime = delaytime
	streamPtr.loadConfig()

	log4plus.Info("CreateStream usedDelay=%t url=%s", streamPtr.usedDelay, m3u8Url)

	streamPtr.M3u8Name, _ = public.GetRemoteM3u8Name(m3u8Url)
	streamPtr.m3u8Ptr = m3u8.CreateM3u8(m3u8Url, localUrl)

	if streamPtr.usedDelay {
		//TODO 需要保证localUrl存在
		streamPtr.transferPtr = transfer.CreateTransfer(streamPtr.M3u8Name, localUrl, delaytime, streamPtr.usedDelete)
	}

	streamPtr.tsPtr = ts.CreateTs(localUrl)
	streamPtr.localM3u8Path = localUrl
	streamPtr.m3u8Url = m3u8Url

	go streamPtr.startCheckStream()

	return streamPtr
}
