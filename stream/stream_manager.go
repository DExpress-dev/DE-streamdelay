// stream_check project stream.go
package stream

import (
	"delay_stream/src/alarm"
	"delay_stream/src/config"
	"delay_stream/src/download"
	"delay_stream/src/hls"
	"delay_stream/src/public"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	log4plus "github.com/alecthomas/log4go"
	"github.com/widuu/goini"
)

type StreamInfo struct {
	url       string //下载的HLS地址
	localPath string //本地保存路径
	delayTime int
	streams   []*Stream
}

//下载一级m3u8
func (info *StreamInfo) Run() {
	startTimer := time.Now()

	//得到indexM3u8名字
	indexM3u8, result := public.GetRemoteM3u8Name(info.url)
	if result {
		indexM3u8Path := info.localPath + indexM3u8
		downloadResult, m3u8Context, respCode := download.DownloadToContext(info.url)
		if downloadResult {
			endTimer := time.Now()
			if respCode == public.RESPCODE_SUCCESS {
				fIndexM3u8, err := os.Create(indexM3u8Path)
				if err != nil {
					log4plus.Error("Open index m3u8 File Failed err=%s path=%s", err.Error(), indexM3u8Path)
					alarm.DoAlarm(alarm.CriticalSeriousLevel, alarm.AlarmTypeM3u8CreateFail, indexM3u8Path, "Index m3u8 create fail!")
					return
				}
				defer fIndexM3u8.Close()
				io.WriteString(fIndexM3u8, m3u8Context)

				//解析M3u8内容
				var startDownloadTimer string = startTimer.Format("2006-01-02 15:04:05")
				hlsPtr := new(hls.Hls)
				res, indexM3u8Info := hlsPtr.SplitM3u8(m3u8Context)
				if res {
					//判断是1级m3u8还是2级m3u8
					if len(indexM3u8Info.M3u8Array) > 0 {
						//1级
						remotePath, result := public.GetRemoteFilePath(info.url)
						if result {
							for _, stream := range indexM3u8Info.M3u8Array {
								streamPath := remotePath + "/" + stream
								streamPtr := CreateStream(streamPath,
									info.delayTime,
									info.localPath)
								info.streams = append(info.streams, streamPtr)
							}
						}
					} else {
						//2级
						streamPtr := CreateStream(info.url,
							info.delayTime,
							info.localPath)
						info.streams = append(info.streams, streamPtr)
					}

					var downloadConsuming int64 = (endTimer.UnixNano() / 1e6) - (startTimer.UnixNano() / 1e6)
					log4plus.Debug("Download M3u8 Start Timer %s New Sequence %d Size %d Timer %d Millisecond File %s ",
						startDownloadTimer,
						indexM3u8Info.Sequence,
						len(m3u8Context),
						downloadConsuming,
						indexM3u8Path)

				}
			}
		} else {
			alarm.DoAlarm(alarm.CriticalSeriousLevel, alarm.AlarmTypeIndexM3u8DownFail, info.url, "Index m3u8 download fail!")
		}
	} else {
		alarm.DoAlarm(alarm.CriticalSeriousLevel, alarm.AlarmTypeIndexM3u8DownFail, info.url, "Index m3u8 download fail!")
	}
}

type StreamManager struct {
	streamLock sync.Mutex             //临界区
	streamMap  map[string]*StreamInfo //下载的对象
}

//添加下载管理
func (streamMgrPtr *StreamManager) streamAdd(streamInfoPtr *StreamInfo) {
	streamMgrPtr.streamLock.Lock()
	defer streamMgrPtr.streamLock.Unlock()
	streamMgrPtr.streamMap[streamInfoPtr.url] = streamInfoPtr
}

func (streamMgrPtr *StreamManager) Run() {

	//start stream
	{
		streamMgrPtr.streamLock.Lock()
		for _, stream := range streamMgrPtr.streamMap {
			go stream.Run()
		}
		streamMgrPtr.streamLock.Unlock()
	}

	//start Web
	go streamMgrPtr.StartWeb()

	//main loop
	for {
		time.Sleep(1 * time.Minute)
	}
}

func (streamMgrPtr *StreamManager) Initialize() bool {
	streamMgrPtr.loadConfig()
	return true
}

//读取配置
func (streamMgrPtr *StreamManager) loadConfig() bool {

	//得到流下载地址
	currentPath := public.GetCurrentDirectory()
	urlPath := currentPath + "/" + config.GetInstance().UrlFile

	//判断文件是否存在
	if !public.FileExist(urlPath) {

		log4plus.Error("Url File Not Found %s", urlPath)
		return false
	}

	//读取url的文件配置
	urlIni := goini.Init(config.GetInstance().UrlFile)
	sessions := urlIni.ReadSessions()

	for i := 0; i < len(sessions); i++ {
		log4plus.Info("url.ini Sessions  ", sessions[i])

		sourceUrl := urlIni.ReadString(sessions[i], "sourceUrl", "")
		sourceUrl = strings.Replace(sourceUrl, "\r", "", -1)
		sourceUrl = strings.Replace(sourceUrl, "\n", "", -1)

		localPath := urlIni.ReadString(sessions[i], "localPath", "")

		if "" != sourceUrl && "" != localPath {
			streamInfo := new(StreamInfo)
			streamInfo.url = sourceUrl
			streamInfo.localPath = localPath
			streamInfo.delayTime = urlIni.ReadInt(sessions[i], "delayTime", 30)
			streamInfo.streams = make([]*Stream, 0)
			streamMgrPtr.streamAdd(streamInfo)
			os.MkdirAll(streamInfo.localPath, 0777)
			os.MkdirAll(streamInfo.localPath+"/m3u8", 0777)
		} else {
			log4plus.Info("sourceUrl or localPath is null it is wrong ")
		}
	}
	return true
}

//检测跨域
func (streamMgrPtr *StreamManager) checkCrossDomain(w http.ResponseWriter, req *http.Request) {

	if origin := req.Header.Get("Origin"); origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	}
}

//http请求类型
func (streamMgrPtr *StreamManager) webCmd(w http.ResponseWriter, req *http.Request) {

	log4plus.Info("webCmd is running...")
	streamMgrPtr.checkCrossDomain(w, req)
}

func (streamMgrPtr *StreamManager) StartWeb() {

	//注册WEB处理的接口
	http.HandleFunc("/otvcloud/otv/rtmp/live", streamMgrPtr.webCmd)
	log4plus.Info("Regedit WEBCmd Function")

	//服务器要监听的主机地址和端口号
	listenString := config.GetInstance().WebIp + ":" + config.GetInstance().WebPort
	err := http.ListenAndServe(listenString, nil)

	if err != nil {
		log4plus.Info("ListenAndServe error: %s", err.Error())
	}
}

//初始化
func CreateStreamManager() *StreamManager {
	streamMgrPtr := new(StreamManager)
	streamMgrPtr.streamMap = make(map[string]*StreamInfo)
	return streamMgrPtr
}
