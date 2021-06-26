// stream_check project m3u8.go
package m3u8

import (
	"bufio"
	//	"delay_stream/src/alarm"
	"bytes"
	"delay_stream/src/download"
	"delay_stream/src/file"
	"delay_stream/src/hls"
	"delay_stream/src/public"
	"io"
	"os"
	"strings"
	"time"

	log4plus "github.com/alecthomas/log4go"
)

type M3u8 struct {
	m3u8Url    string            //M3U8的下载地址
	localPath  string            //m3u8本地保存路径
	fileMgrPtr *file.FileManager //文件管理
	hlsPtr     *hls.Hls          //HLS的管理类
	tsArray    []*public.TsInfo  //TS的slice
}

func (m3u8Ptr *M3u8) checkError(err error, info string) bool {

	if err != nil {

		log4plus.Error("checkError Error %s=%s", info, err.Error())
		return false
	}
	return true
}

func (m3u8Ptr *M3u8) analysis(localFilePath string) bool {

	//读取文件
	file, err := os.Open(localFilePath)
	if err != nil {
		panic(err)
		return false
	}
	defer file.Close()

	//按行读取数据
	rd := bufio.NewReader(file)

	b := bytes.Buffer{}
	prefix := ""
	for {
		line, err := rd.ReadString('\n') //以'\n'为结束符读入一行
		if err != nil || io.EOF == err {
			break
		}
		line = strings.Replace(line, "\n", "", -1)
		line = strings.Replace(line, "\r", "", -1)

		//判断行;
		timerInterval, exist := public.GetTsTimerInterval(line)
		if exist {

			line, err = rd.ReadString('\n') //以'\n'为结束符读入一行
			if err != nil || io.EOF == err {
				break
			}

			line = strings.Replace(line, "\n", "", -1)
			line = strings.Replace(line, "\r", "", -1)

			var newM3u8Info *public.TsInfo = new(public.TsInfo)
			newM3u8Info.TimerInterval = timerInterval
			newM3u8Info.RemoteName = line
			newM3u8Info.LocalTsPathName = m3u8Ptr.m3u8Url + "/" + line
			m3u8Ptr.tsArray = append(m3u8Ptr.tsArray, newM3u8Info)
			b.WriteString(prefix)
			b.WriteString(line)
			prefix = "|"
		}
	}

	log4plus.Warn("analysis ts=%s", b.String())

	return true
}

//下载M3U8文件
func (m3u8Ptr *M3u8) DownloadM3u8() (bool, []*public.TsInfo, string) {

	startTimer := time.Now()

	//得到m3u8的名字
	m3u8Name, result := public.GetRemoteM3u8Name(m3u8Ptr.m3u8Url)
	if result {

		localM3u8Name := public.GetCurrentM3u8Name(m3u8Name)
		localM3u8Path := m3u8Ptr.localPath + "/m3u8/" + localM3u8Name
		downloadResult, m3u8Context, respCode := download.DownloadToContext(m3u8Ptr.m3u8Url)
		if downloadResult {

			endTimer := time.Now()
			if respCode == public.RESPCODE_SUCCESS {

				//判断内容是否相同
				if !m3u8Ptr.fileMgrPtr.FileContextEqual(m3u8Context) {

					if !public.FileExist(localM3u8Path) {

						file, err := os.Create(localM3u8Path)
						if err == nil {

							defer file.Close()
							io.WriteString(file, m3u8Context)

							//分解M3u8内容
							var startDownloadTimer string = startTimer.Format("2006-01-02 15:04:05")
							newresult, newm3u8Info := m3u8Ptr.hlsPtr.SplitM3u8(m3u8Context)
							if newresult {

								if m3u8Ptr.fileMgrPtr.LastM3u8Context != "" {

									var downloadConsuming int64 = (endTimer.UnixNano() / 1e6) - (startTimer.UnixNano() / 1e6)
									log4plus.Debug("Download M3u8 Start Timer %s New Sequence %d Size %d Timer %d Millisecond File %s ",
										startDownloadTimer,
										newm3u8Info.Sequence,
										len(m3u8Context),
										downloadConsuming,
										localM3u8Path)
								}

							}

							m3u8Ptr.tsArray = make([]*public.TsInfo, 0)
							result = m3u8Ptr.analysis(localM3u8Path)
							return true, m3u8Ptr.tsArray, localM3u8Name
						}
					} else {

						var startDownloadTimer string = startTimer.Format("2006-01-02 15:04:05")
						result, newm3u8Info := m3u8Ptr.hlsPtr.SplitM3u8(m3u8Context)
						if result {

							if m3u8Ptr.fileMgrPtr.LastM3u8Context != "" {

								var downloadConsuming int64 = (endTimer.UnixNano() / 1e6) - (startTimer.UnixNano() / 1e6)

								log4plus.Error("File Exists Download M3u8 Start Timer %s Old Sequence %d New Sequence %d Size %d Timer %d Millisecond File %s ",
									startDownloadTimer,
									newm3u8Info.Sequence,
									len(m3u8Context),
									downloadConsuming,
									localM3u8Path)
							}
						}
					}
				} else {
					//					log4plus.Warn("DownloadM3u8 file context equal! url=%s", m3u8Ptr.m3u8Url)
					/*var startDownloadTimer string = startTimer.Format("2006-01-02 15:04:05")
					result, newm3u8Info := m3u8Ptr.hlsPtr.SplitM3u8(m3u8Context)
					if result {

						if m3u8Ptr.fileMgrPtr.LastM3u8Context != "" {

							var downloadConsuming int64 = (endTimer.UnixNano() / 1e6) - (startTimer.UnixNano() / 1e6)
							_, oldm3u8Info := m3u8Ptr.hlsPtr.SplitM3u8(m3u8Ptr.fileMgrPtr.LastM3u8Context)

							log4plus.Error("not FileContextEqual Download M3u8 Start Timer %s Old Sequence %d New Sequence %d Size %d Timer %d Millisecond File %s ",
								startDownloadTimer,
								oldm3u8Info.Sequence,
								newm3u8Info.Sequence,
								len(m3u8Context),
								downloadConsuming,
								localM3u8Path)
						}
					}*/
				}
			} else {
				log4plus.Error("streamPtr.tsPtr.DownloadM3u8 respCode=%d", respCode)
			}
		}
	} else {
		//		alarm.DoAlarm(alarm.PoorLevel, alarm.AlarmTypeM3u8DownFail, m3u8Ptr.m3u8Url, "m3u8 download fail!")
	}
	return false, nil, ""
}

//创建M3U8
func CreateM3u8(url, localM3u8 string) *M3u8 {

	m3u8Ptr := new(M3u8)
	m3u8Ptr.m3u8Url = url
	m3u8Ptr.localPath = localM3u8
	m3u8Ptr.hlsPtr = hls.CreateHls()
	m3u8Ptr.tsArray = make([]*public.TsInfo, 0, 1)
	m3u8Ptr.fileMgrPtr = file.CreateFileManager(m3u8Ptr.localPath)

	return m3u8Ptr
}
