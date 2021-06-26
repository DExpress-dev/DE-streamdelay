// stream_check project download.go
package ts

import (
	"delay_stream/src/alarm"
	"delay_stream/src/download"
	"delay_stream/src/public"
	//	"math/rand"
	"sort"
	"sync"
	"time"

	log4plus "github.com/alecthomas/log4go"
)

const MAP_SIZE = 20   //MAP存放的数量;
const REDUCE_SIZE = 5 //一次性删除的数量;

//成功下载的TS结构
type sucessInfo struct {
	UrlTSName string
	//Timer       string
	LocalTSName string
}

//下载失败的TS结构
type failInfo struct {
	UrlTSName string
	tsUrl     string //远端M3U8中写的文件名（有可能包含路径）
	localFile string //本地文件名（包含路径）
	HttpCode  string
	Count     int
}

//正在下载的TS结构
type downloadingInfo struct {
	tsUrl       string //远端M3U8中写的文件名（有可能包含路径）
	fileName    string //远端文件名（不包含路径）
	localFile   string //本地文件名（包含路径）
	m3u8Name    string //存放此TS的M3U8文件（第一次出现的M3U8文件）
	tsTimer     string //时长
	downloading bool   //是否正在下载;
}

type Ts struct {
	localPath      string                      //本地保存路径
	sucessLock     sync.Mutex                  //成功临界区
	SucessMap      map[string]*sucessInfo      //下载成功ts的Map
	failLock       sync.Mutex                  //失败临界区
	FailMap        map[string]*failInfo        //下载失败TS的MAP
	downloadLock   sync.Mutex                  //正在下载临界区
	downloadingMap map[string]*downloadingInfo //正在下载TS的map
}

func (tsPtr *Ts) checkError(err error, info string) bool {

	if err != nil {

		log4plus.Error("checkError Error %s=%s", info, err.Error())
		return false
	}
	return true
}

//成功map管理
func (tsPtr *Ts) addSucess(fileName string, sucessPtr *sucessInfo) {

	tsPtr.sucessLock.Lock()
	defer tsPtr.sucessLock.Unlock()

	tsPtr.SucessMap[fileName] = sucessPtr
}

func (tsPtr *Ts) existSucess(fileName string) bool {

	//tsPtr.sucessLock.Lock()
	//defer tsPtr.sucessLock.Unlock()

	if len(tsPtr.SucessMap) > 0 {
		_, exist := tsPtr.SucessMap[fileName]
		return exist
	} else {
		return false
	}
}

func (tsPtr *Ts) findSucess(fileName string) *sucessInfo {

	tsPtr.sucessLock.Lock()
	defer tsPtr.sucessLock.Unlock()

	sucessPtr, _ := tsPtr.SucessMap[fileName]
	return sucessPtr
}

func (tsPtr *Ts) deleteSucess(fileName string) {

	tsPtr.sucessLock.Lock()
	defer tsPtr.sucessLock.Unlock()

	delete(tsPtr.SucessMap, fileName)
}

func (tsPtr *Ts) sortSucess() []string {

	tsPtr.sucessLock.Lock()
	defer tsPtr.sucessLock.Unlock()

	var keys []string
	for k := range tsPtr.SucessMap {
		keys = append(keys, k)
	}
	return keys
}

func (tsPtr *Ts) clearSucess() {

	tsPtr.sucessLock.Lock()
	defer tsPtr.sucessLock.Unlock()

	var keys []string
	for k := range tsPtr.SucessMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	postion := 0
	for _, key := range keys {

		delete(tsPtr.SucessMap, key)
		postion++
		if postion >= REDUCE_SIZE {
			return
		}
	}
}

//失败map管理
func (tsPtr *Ts) addFail(fileName string, failPtr *failInfo) {

	tsPtr.failLock.Lock()
	defer tsPtr.failLock.Unlock()

	tsPtr.FailMap[fileName] = failPtr
}

func (tsPtr *Ts) existFail(fileName string) bool {

	//tsPtr.failLock.Lock()
	//defer tsPtr.failLock.Unlock()

	if len(tsPtr.FailMap) > 0 {
		_, exist := tsPtr.FailMap[fileName]
		return exist
	} else {
		return false
	}

}

func (tsPtr *Ts) findFail(fileName string) *failInfo {

	tsPtr.failLock.Lock()
	defer tsPtr.failLock.Unlock()

	failPtr, _ := tsPtr.FailMap[fileName]
	return failPtr
}

func (tsPtr *Ts) deleteFail(fileName string) {

	tsPtr.failLock.Lock()
	defer tsPtr.failLock.Unlock()

	delete(tsPtr.FailMap, fileName)
}

func (tsPtr *Ts) getFail() (bool, *failInfo) {

	tsPtr.failLock.Lock()
	defer tsPtr.failLock.Unlock()

	var keys []string
	for k := range tsPtr.FailMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	if len(keys) > 0 {

		for _, key := range keys {

			failPtr := tsPtr.findFail(key)
			delete(tsPtr.FailMap, key)
			return true, failPtr
		}
	}
	return false, nil
}

func (tsPtr *Ts) clearFail() {

	tsPtr.failLock.Lock()
	defer tsPtr.failLock.Unlock()

	var keys []string
	for k := range tsPtr.FailMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	postion := 0
	for _, key := range keys {

		delete(tsPtr.FailMap, key)
		postion++
		if postion >= REDUCE_SIZE {
			return
		}
	}
}

//正在下载map管理
func (tsPtr *Ts) addDownloading(fileName string, downloadingPtr *downloadingInfo) {

	tsPtr.downloadLock.Lock()
	defer tsPtr.downloadLock.Unlock()

	tsPtr.downloadingMap[fileName] = downloadingPtr
}

func (tsPtr *Ts) existDownloading(fileName string) bool {

	//tsPtr.downloadLock.Lock()
	//defer tsPtr.downloadLock.Unlock()

	if len(tsPtr.downloadingMap) > 0 {
		_, exist := tsPtr.downloadingMap[fileName]
		return exist
	} else {
		return false
	}
}

func (tsPtr *Ts) findDownloading(fileName string) *downloadingInfo {

	tsPtr.downloadLock.Lock()
	defer tsPtr.downloadLock.Unlock()

	downloadingPtr, _ := tsPtr.downloadingMap[fileName]
	return downloadingPtr
}

func (tsPtr *Ts) deleteDownloading(fileName string) {

	tsPtr.downloadLock.Lock()
	defer tsPtr.downloadLock.Unlock()

	delete(tsPtr.downloadingMap, fileName)
}

//下载TS文件
func (tsPtr *Ts) downloadTS(downloadingPtr *downloadingInfo) {

	var currentTimer string = public.GetCurrentTimerISO()
	log4plus.Debug("Timer %s Download TS File %s", currentTimer, downloadingPtr.tsUrl)

	//下载TS文件
	downloadPtr := download.CreateDownload()
	localPath := public.GetCurrentPath(downloadingPtr.localFile)

	//	//for test
	//	prob1 := 5
	//	prob2 := 2
	//	var result bool
	//	var respCode int

	//	//模拟概率下载缓慢
	//	if rand.Intn(100) < prob1 {
	//		log4plus.Warn("downloadTS Sleep for test url=%s", downloadingPtr.tsUrl)
	//		time.Sleep(500 * time.Millisecond)
	//	}

	//	//模拟概率TS下载失败
	//	if rand.Intn(100) > prob2 {
	//		result, respCode = downloadPtr.DownloadSaveFile(downloadingPtr.tsUrl, downloadingPtr.localFile, localPath)
	//	} else {
	//		log4plus.Warn("downloadTS Fail for test url=%s", downloadingPtr.tsUrl)
	//		result, respCode = false, 701
	//	}

	result, respCode := downloadPtr.DownloadSaveFile(downloadingPtr.tsUrl, downloadingPtr.localFile, localPath)
	if result {

		if respCode == public.RESPCODE_SUCCESS {

			//从正在下载的中删除
			tsPtr.deleteDownloading(downloadingPtr.tsUrl)

			//判断成功map的大小
			if len(tsPtr.SucessMap) >= MAP_SIZE {
				tsPtr.clearSucess()
			}

			//添加到成功map中
			var sucessPtr *sucessInfo = new(sucessInfo)
			sucessPtr.LocalTSName = localPath
			sucessPtr.UrlTSName = downloadingPtr.tsUrl
			tsPtr.addSucess(sucessPtr.UrlTSName, sucessPtr)
			log4plus.Debug("Timer %s Download TS File Success %s", currentTimer, sucessPtr.UrlTSName)

		} else if respCode == public.RESPCODE_NOTFOUND {

			log4plus.Error("Timer %s Download TS File Failed %s HttpCode %d", currentTimer, downloadingPtr.tsUrl, respCode)
			alarm.DoAlarm(alarm.CommonLevel, alarm.AlarmTypeTSDownFail, downloadingPtr.tsUrl, "TS download fail!")

			//从正在下载的中删除
			tsPtr.deleteDownloading(downloadingPtr.tsUrl)

			//判断失败map的大小
			if len(tsPtr.FailMap) >= MAP_SIZE {

				tsPtr.clearFail()
			}

			//添加到失败map中
			var failPtr *failInfo = new(failInfo)
			failPtr.UrlTSName = downloadingPtr.tsUrl
			failPtr.HttpCode = "404"
			failPtr.tsUrl = downloadingPtr.tsUrl
			failPtr.localFile = downloadingPtr.localFile
			failPtr.Count = 1
			tsPtr.addFail(failPtr.UrlTSName, failPtr)
		}

	} else {

		log4plus.Error("Timer %s Download TS File Failed %s HttpCode %d", currentTimer, downloadingPtr.tsUrl, respCode)
		alarm.DoAlarm(alarm.CommonLevel, alarm.AlarmTypeTSDownFail, downloadingPtr.tsUrl, "TS download fail!")

		//从正在下载的中删除
		tsPtr.deleteDownloading(downloadingPtr.tsUrl)

		//判断失败map的大小
		if len(tsPtr.FailMap) >= MAP_SIZE {

			tsPtr.clearFail()
		}

		//添加到失败map中
		var failPtr *failInfo = new(failInfo)
		failPtr.UrlTSName = downloadingPtr.tsUrl
		failPtr.HttpCode = "0"
		failPtr.tsUrl = downloadingPtr.tsUrl
		failPtr.localFile = downloadingPtr.localFile
		failPtr.Count = 1
		tsPtr.addFail(failPtr.UrlTSName, failPtr)
	}
}

//下载
func (tsPtr *Ts) DownloadUrl(m3u8Url string, tsArray []*public.TsInfo) {

	for index := range tsArray {

		tsInfoPtr := tsArray[index]
		urlName, result := public.GetRemoteTsPath(tsInfoPtr.RemoteName, m3u8Url)
		if !result {

			log4plus.Debug("DownloadUrl failed form :RemoteName = %s,  m3u8Url = %s", tsInfoPtr.RemoteName, m3u8Url)
			continue
		}

		sucessed := tsPtr.existSucess(urlName)
		failed := tsPtr.existFail(urlName)
		downloading := tsPtr.existDownloading(urlName)

		if !sucessed && !failed && !downloading {

			var downloadingPtr *downloadingInfo = new(downloadingInfo)
			downloadingPtr.tsUrl = urlName
			downloadingPtr.fileName, _ = public.GetRemoteFileName(downloadingPtr.tsUrl)
			downloadingPtr.localFile, _ = public.GetLocalTsPath(tsInfoPtr.RemoteName, m3u8Url, tsPtr.localPath)
			downloadingPtr.downloading = true
			tsPtr.addDownloading(downloadingPtr.tsUrl, downloadingPtr)

			go tsPtr.downloadTS(downloadingPtr)
		}

	}
}

//下载TS文件
func (tsPtr *Ts) downloadFail(failPtr *failInfo) {

	log4plus.Debug("downloadFail ReDownload TS = %s", failPtr.tsUrl)

	//下载TS文件
	downloadPtr := download.CreateDownload()
	localPath := public.GetCurrentPath(failPtr.localFile)
	result, respCode := downloadPtr.DownloadSaveFile(failPtr.tsUrl, failPtr.localFile, localPath)
	if result {

		if respCode == public.RESPCODE_SUCCESS {

			//从正在下载的中删除
			tsPtr.deleteDownloading(failPtr.tsUrl)

			//添加到成功map中
			var sucessPtr *sucessInfo = new(sucessInfo)
			sucessPtr.LocalTSName = localPath
			sucessPtr.UrlTSName = failPtr.tsUrl
			tsPtr.addSucess(sucessPtr.UrlTSName, sucessPtr)

		} else if respCode == public.RESPCODE_NOTFOUND {

			//从正在下载的中删除
			tsPtr.deleteDownloading(failPtr.tsUrl)

			//添加到失败map中
			var newFailPtr *failInfo = new(failInfo)
			newFailPtr.tsUrl = failPtr.tsUrl
			newFailPtr.localFile = failPtr.localFile
			newFailPtr.HttpCode = "404"
			newFailPtr.Count = failPtr.Count + 1
			newFailPtr.UrlTSName = failPtr.UrlTSName
			tsPtr.addFail(newFailPtr.UrlTSName, newFailPtr)
		}

	} else {

		//从正在下载的中删除
		tsPtr.deleteDownloading(failPtr.tsUrl)

		//添加到失败map中
		var newFailPtr *failInfo = new(failInfo)
		newFailPtr.tsUrl = failPtr.tsUrl
		newFailPtr.localFile = failPtr.localFile
		newFailPtr.HttpCode = "0"
		newFailPtr.Count = failPtr.Count + 1
		newFailPtr.UrlTSName = failPtr.UrlTSName
		tsPtr.addFail(newFailPtr.UrlTSName, newFailPtr)
	}
}

func (tsPtr *Ts) CheckFail() {

	for {

		result, failPtr := tsPtr.getFail()
		if result {

			if failPtr.Count < 5 {

				go tsPtr.downloadFail(failPtr)
			}
		}
		time.Sleep(2 * time.Second)
	}
}

//TS管理
func CreateTs(tsLocalPath string) *Ts {

	tsPtr := new(Ts)
	tsPtr.localPath = tsLocalPath
	tsPtr.SucessMap = make(map[string]*sucessInfo)
	tsPtr.FailMap = make(map[string]*failInfo)
	tsPtr.downloadingMap = make(map[string]*downloadingInfo)

	go tsPtr.CheckFail()

	return tsPtr
}
