// stream_check project public.go
package public

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	log4plus "github.com/alecthomas/log4go"
)

const (
	RESPCODE_SUCCESS  = 200 //web返回正常
	RESPCODE_NOTFOUND = 404 //找不到网页
)

type TsInfo struct {
	RemoteName      string //远端M3U8的中的文件名
	TimerInterval   string //TS文件的时长
	LocalTsPathName string
}

//得到文件名(不包含路径 /20171226/700/20171226T143444.ts -> 20171226T143444.ts)
func GetRemoteFileName(url string) (string, bool) {

	index := strings.LastIndex(url, "/")
	if index == -1 {
		return "", false
	}
	return string(url[index:len(url)]), true
}

//得到文件路径(不包含文件名 /20171226/700/20171226T143444.ts -> /20171226/700/)
func GetRemoteFilePath(url string) (string, bool) {

	index := strings.LastIndex(url, "/")
	if index == -1 {
		return "", false
	}
	return string(url[0:index]), true
}

//得到远端文件名(700/20180102/20180102T182312.ts -> http://guoguang.live.otvcloud.com/otv/xjgg/live/channel17/700/20180102/20180102T182312.ts)
func GetRemoteTsPath(tsName, m3u8Url string) (string, bool) {

	m3u8RemtePath, result := GetRemoteM3u8Path(m3u8Url)
	if result {

		return m3u8RemtePath + "/" + tsName, true
	}
	return "", false
}

//拆分函数
func split(s rune) bool {
	if s == '/' {
		return true
	}
	return false
}

//字符串拆分(http://guoguang.live.otvcloud.com/otv/xjgg/live/channel17/700.m3u8)
func FieldsFunc(m3u8Url string) []string {

	return strings.FieldsFunc(m3u8Url, split)
}

//得到本地文件名(700/20180102/20180102T182312.ts -> localpath/700/20180102/20180102T182312.ts)
func GetLocalTsPath(tsName, m3u8Url, tsLocalPath string) (string, bool) {

	return tsLocalPath + tsName, true
}

//得到文件路径(不包含文件名 /20171226/700/700.m3u8 -> 700.m3u8)
func GetRemoteM3u8Name(url string) (string, bool) {

	index := strings.LastIndex(url, "/")
	if index == -1 {
		return "", false
	}
	return string(url[index+1 : len(url)]), true
}

//得到文件路径(不包含文件名 /20171226/700/700.m3u8 -> 700.m3u8)
func GetRemoteM3u8Path(url string) (string, bool) {

	index := strings.LastIndex(url, "/")
	if index == -1 {
		return "", false
	}
	return string(url[0:index]), true
}

//得到当前时间 20171226T143444
func GetCurrentTimerISO() string {

	timeNow := time.Now()
	var currentDay string = timeNow.Format("20060102")
	currentTimer := timeNow.Format("150405")
	timeNowString := currentDay + "T" + currentTimer
	return timeNowString
}

//得到当前时间 20171226T143444
func GetStdTime(intime string) int64 {
	timeLayout := "2006-01-02 15:04:05"
	//标准时间格式  20180130T142339
	beginTimeYear := intime[:4]
	beginTimeMonth := intime[4:6]
	beginTimeDay := intime[6:8]
	beginTimeTime := intime[9:11]
	beginTimeMin := intime[11:13]
	beginTimeSed := intime[13:]
	Temptime := beginTimeYear + "-" + beginTimeMonth + "-" + beginTimeDay + " " + beginTimeTime + ":" + beginTimeMin + ":" + beginTimeSed
	loc, _ := time.LoadLocation("Local")
	tmp, _ := time.ParseInLocation(timeLayout, Temptime, loc)
	beginTimestamp := tmp.Unix() //转化为时间戳 类型是int64
	return beginTimestamp
}

//产生二级M3u8的名称(700.m3u8 -> 20171226T143444_700.m3u8)
func GetCurrentM3u8Name(m3u8Name string) string {

	isoTimer := GetCurrentTimerISO()
	return isoTimer + "_" + m3u8Name
}

//产生二级M3u8的名称(/20171226/700.m3u8 -> localPath/20171226/700/20171226T143444_700.m3u8)
func GetCurrentM3u8Path(localM3u8Name string) string {

	isoTimer := GetCurrentTimerISO()
	return isoTimer + "_" + localM3u8Name
}

//得到目录(localPath/20171226/700/20171226T143444_700.m3u8 -> localPath/20171226/700)
func GetCurrentPath(localFileName string) string {

	index := strings.LastIndex(localFileName, "/")
	if index == -1 {
		return ""
	}
	return string(localFileName[0:index])
}

//得到TS的下载地址(/20171226/1234.ts , /20171226/700.m3u8  -> /20171226/20171226/1234.ts)
func GetTsDownloadUrl(tsName, m3u8Url string) string {

	urlPath, _ := GetRemoteFilePath(m3u8Url)
	return urlPath + tsName
}

//得到当前时间目录
func GetCurrentTimerDir() string {

	timeNow := time.Now().Format("20060102")
	return timeNow
}

//得到TS的时间间隔(#EXTINF:2.000000, ->2.000000)
func GetTsTimerInterval(line string) (string, bool) {

	//是否是时间;
	flagIndex := strings.LastIndex(line, "#EXTINF:")
	if flagIndex >= 0 {

		commaIndex := strings.LastIndex(line, ",")
		return line[flagIndex+len("#EXTINF:") : commaIndex], true
	}
	return "", false
}

//读取文件内容
func ReadFileContext(filePath string) string {

	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	rd := bufio.NewReader(file)
	chunks := make([]byte, 1024, 1024)
	buffer := make([]byte, 1024)
	for {
		n, err := rd.Read(buffer)
		if err != nil && err != io.EOF {
			panic(err)
		}
		if 0 == n {
			break
		}
		chunks = append(chunks, buffer[:n]...)
	}
	return string(chunks)
}

//判断文件是否存在
func FileExist(filePath string) bool {

	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			return false
		} else {
			return false
		}
	}
	return true
}

//获取当前路径
func GetCurrentDirectory() string {

	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return ""
	}
	return strings.Replace(dir, "\\", "/", -1)
}

func CheckError(err error, info string) bool {
	if err != nil {
		log4plus.Error("checkError Error %s=%s", info, err.Error())
		return false
	}
	return true
}
