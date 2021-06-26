// stream_check project transfer.go
package transfer

import (
	"bufio"
	"delay_stream/src/alarm"
	"delay_stream/src/hls"
	"delay_stream/src/public"
	"io"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	log4plus "github.com/alecthomas/log4go"
)

const (
//	M3U8NAME = "a.m3u8"
)

const (
	ITEM_COUNT     = 5
	DEL_ITEM_COUNT = 200
)

//
type tsFileInfo struct {
	tsName    string
	tsTimeStr string
	tsTime    int
}

type Transfer struct {
	m3u8Name      string
	localM3u8Path string //本地保存路径
	delayTime     int    //延迟时间
	bufTime       int    //已经缓存时间
	disconnected  bool   //是否已经断开
	//****HLS信息****
	m3u8DestFile    string   //新的M3U8的文件名
	exist           bool     //信息是否已经存在
	hlsPtr          *hls.Hls //HLS的管理类
	version         string   //版本
	targetduration  string   //最大
	currentSequence int      //当前sequence
	endlist         bool     //是否存在endlist
	//****TS文件信息保存****
	tsLock  sync.Mutex             //临界区
	tsArray []string               //ts的数组
	tsMap   map[string]*tsFileInfo //ts的Map
	//****删除TS信息****
	usedDel    bool     //是否使用删除
	delTsArray []string //删除的TS数组
	//****删除M3U8信息****
	delM3u8Lock  sync.Mutex //临界区
	delM3u8Array []string   //删除的M3u8数组

}

func (transferPtr *Transfer) addDel(tsName string) {

	transferPtr.delTsArray = append(transferPtr.delTsArray, tsName)
	if len(transferPtr.delTsArray) < DEL_ITEM_COUNT {
		return
	}

	//删除文件
	var tsDelName string = transferPtr.delTsArray[0]
	var tsFilePath string = transferPtr.getDestTsFile(tsDelName)
	err := os.Remove(tsFilePath)
	if err != nil {
		log4plus.Error("Delete Ts File %s Error = %s", tsFilePath, err.Error())
	}
	transferPtr.delTsArray = append(transferPtr.delTsArray[:0], transferPtr.delTsArray[0+1:]...)
}

func (transferPtr *Transfer) addDelM3u8(localM3u8Name string) {

	transferPtr.delM3u8Lock.Lock()
	defer transferPtr.delM3u8Lock.Unlock()

	transferPtr.delM3u8Array = append(transferPtr.delM3u8Array, localM3u8Name)
	if len(transferPtr.delM3u8Array) < DEL_ITEM_COUNT {
		return
	}

	//删除文件
	var m3u8DelFile string = transferPtr.delM3u8Array[0]
	err := os.Remove(m3u8DelFile)
	if err != nil {
		log4plus.Error("Delete M3u8 File %s Error = %s", m3u8DelFile, err.Error())
	}
	transferPtr.delM3u8Array = append(transferPtr.delM3u8Array[:0], transferPtr.delM3u8Array[0+1:]...)
}

func (transferPtr *Transfer) addTs(tsName, tsTimer string) {

	result, _ := transferPtr.findTs(tsName)
	if result {
		return
	}

	transferPtr.tsLock.Lock()
	defer transferPtr.tsLock.Unlock()

	var tsFile *tsFileInfo = new(tsFileInfo)
	tsFile.tsName = tsName
	tsFile.tsTimeStr = tsTimer
	t, _ := strconv.ParseFloat(tsFile.tsTimeStr, 32)
	tsFile.tsTime = int(t)
	transferPtr.tsArray = append(transferPtr.tsArray, tsName)
	transferPtr.tsMap[tsName] = tsFile
	transferPtr.bufTime += tsFile.tsTime
	log4plus.Warn("addTs tsName=%s time=%d", tsName, tsFile.tsTime)
}

func (transferPtr *Transfer) sortTs() {

	transferPtr.tsLock.Lock()
	defer transferPtr.tsLock.Unlock()

	sort.Strings(transferPtr.tsArray)
}

func (transferPtr *Transfer) findTs(tsName string) (bool, *tsFileInfo) {

	transferPtr.tsLock.Lock()
	defer transferPtr.tsLock.Unlock()

	tsFile, exist := transferPtr.tsMap[tsName]
	return exist, tsFile
}

func (transferPtr *Transfer) getTs(index int) (bool, *tsFileInfo) {

	if len(transferPtr.tsArray) <= index {
		return false, nil
	} else {

		var tsName string = transferPtr.tsArray[index]
		tsFile, exists := transferPtr.tsMap[tsName]
		if exists {
			return true, tsFile
		}
		return false, nil
	}
}

func (transferPtr *Transfer) getM3u8Context() (bool, string, int) {
	wait := 2 //next sleep wait

	if len(transferPtr.tsArray) <= 0 {
		return false, "", wait
	}

	transferPtr.currentSequence = transferPtr.currentSequence + 1

	//设置头
	var m3u8Context string
	m3u8Context += "#EXTM3U \n"
	m3u8Context += "#EXT-X-VERSION:" + transferPtr.version + "\n"
	m3u8Context += "#EXT-X-TARGETDURATION:" + transferPtr.targetduration + "\n"
	m3u8Context += "#EXT-X-MEDIA-SEQUENCE:" + strconv.Itoa(transferPtr.currentSequence) + "\n"

	transferPtr.tsLock.Lock()
	defer transferPtr.tsLock.Unlock()

	//设置内容
	var itemCount = 0
	var firstTs *tsFileInfo
	for i := 0; i < ITEM_COUNT; i++ {
		result, tsFile := transferPtr.getTs(i)
		if result {
			if 0 == i {
				firstTs = tsFile
			}

			var tsNamePath string = transferPtr.localM3u8Path + tsFile.tsName
			if public.FileExist(tsNamePath) {

				m3u8Context += "#EXTINF:" + tsFile.tsTimeStr + ",\n"
				m3u8Context += tsFile.tsName + "\n"
				itemCount++
				if 1 == itemCount {
					wait = tsFile.tsTime
				}
			} else {

				log4plus.Error("Check TS Not Found %s", tsNamePath)
			}
		}
	}

	var tsName string = transferPtr.tsArray[0]
	transferPtr.tsArray = append(transferPtr.tsArray[:0], transferPtr.tsArray[0+1:]...)
	delete(transferPtr.tsMap, tsName)
	transferPtr.bufTime -= firstTs.tsTime
	log4plus.Warn("getM3u8Context remove ts tsName=%s time=%d", tsName, firstTs.tsTime)

	if transferPtr.usedDel {
		transferPtr.addDel(tsName)
	}

	if itemCount > 0 {

		return true, m3u8Context, wait
	} else {
		return false, "", wait
	}
}

func (transferPtr *Transfer) getDestM3u8File() string {

	return transferPtr.localM3u8Path + transferPtr.m3u8Name
}

func (transferPtr *Transfer) getDestTsFile(tsName string) string {

	return transferPtr.localM3u8Path + tsName
}

func (transferPtr *Transfer) clearM3u8Context(m3u8File string) {

	if public.FileExist(m3u8File) {

		file, err := os.Open(m3u8File)
		if err != nil {

			log4plus.Error("Clear M3u8 Context File = %s Error = %s", m3u8File, err.Error())
		}
		defer file.Close()

		file.Truncate(0)
	}
}

//将文件拷贝到指定的目录中
func (transferPtr *Transfer) createM3u8File() bool {

	//得到最终拷贝的文件路径
	m3u8Handle, err := os.OpenFile(transferPtr.m3u8DestFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log4plus.Error("Create M3u8 File Failed err=%s m3u8DestFile=%s", err.Error(), transferPtr.m3u8DestFile)
		return false
	}
	defer m3u8Handle.Close()
	return true
}

//检测延时
func (transferPtr *Transfer) checkDelay() {

	var m3u8Handle *os.File
	var err error
	m3u8Handle, err = os.OpenFile(transferPtr.m3u8DestFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log4plus.Error("Open M3u8 File Failed %s", err.Error())
		alarm.DoAlarm(alarm.PoorLevel, alarm.AlarmTypeTransferCreateFail, transferPtr.m3u8DestFile, "Transfer create fail!")
	}
	defer m3u8Handle.Close()

	for {
		tWait := 2

		log4plus.Warn("TS Files Pool Count %d bufTime=%d", len(transferPtr.tsArray), transferPtr.bufTime)

		if transferPtr.disconnected {

			//			//已经断开,需要检测存在的m3u8的数量
			//			delayCount := transferPtr.delayTime / transferPtr.sleepStep

			//			if len(transferPtr.tsArray) >= delayCount {
			if transferPtr.bufTime >= transferPtr.delayTime {

				//缓存中已经存在足够的m3u8文件可以进行恢复
				transferPtr.disconnected = false
			} else {

				//缓存中没有足够的m3u8只能继续等待
			}
		} else {

			//判断是否还有m3u8的信息
			if len(transferPtr.tsArray) <= 0 {
				transferPtr.bufTime = 0
				transferPtr.disconnected = true
			} else {
				createRet, srcFileContext, wait := transferPtr.getM3u8Context()
				tWait = wait
				if createRet {

					m3u8Handle.Seek(0, os.SEEK_SET)
					_, err = m3u8Handle.WriteString(srcFileContext)
					if err != nil {

						log4plus.Error("Write M3u8 File Error = %s", err.Error())
					} else {

						//切断后续内容
						os.Truncate(transferPtr.m3u8DestFile, int64(len(srcFileContext)))

						log4plus.Warn("Write New M3u8 Size =%d Create New M3u8 File \n%s", len(srcFileContext), srcFileContext)
					}
				} else {
					log4plus.Error("Create M3u8 Fail getM3u8Context Function Error")
				}
			}
		}

		//等待
		time.Sleep(time.Duration(tWait) * time.Second)
	}

}

//加入m3u8
func (transferPtr *Transfer) AddFIFOM3u8(localM3u8Name string) {

	srcFile, err := os.Open(localM3u8Name)
	if err != nil {

		log4plus.Error("Open Src M3u8 File = %s Error = %s", localM3u8Name, err.Error())
	}
	defer srcFile.Close()

	//读取原始文件内容
	rd := bufio.NewReader(srcFile)
	srcContext := make([]byte, 0)
	buffer := make([]byte, 1024)
	for {
		n, err := rd.Read(buffer)
		if err != nil && err != io.EOF {

			log4plus.Error("Read Src M3u8 File = %s  Error = %s", localM3u8Name, err.Error())
		}
		if 0 == n {
			break
		}
		srcContext = append(srcContext, buffer[:n]...)
	}
	srcFileContext := string(srcContext)

	//分解信息
	result, m3u8Ptr := transferPtr.hlsPtr.SplitM3u8(srcFileContext)
	if result {

		if !transferPtr.exist {

			transferPtr.version = m3u8Ptr.Version
			transferPtr.targetduration = m3u8Ptr.Targetduration
			transferPtr.endlist = m3u8Ptr.Endlist
			transferPtr.exist = true
		}

		for _, value := range m3u8Ptr.TsArray {

			transferPtr.addTs(value.TsPath, value.Extinf)
		}

		//添加m3u8文件
		if transferPtr.usedDel {

			transferPtr.addDelM3u8(localM3u8Name)
		}

		//重新排序
		transferPtr.sortTs()

	}
}

//m3u8管理
func CreateTransfer(m3u8Name string, localPath string, delaytime int, useddel bool) *Transfer {

	transferPtr := new(Transfer)
	transferPtr.m3u8Name = m3u8Name
	transferPtr.localM3u8Path = localPath
	transferPtr.delayTime = delaytime
	transferPtr.disconnected = true
	transferPtr.exist = false
	transferPtr.hlsPtr = hls.CreateHls()
	transferPtr.tsArray = make([]string, 0)
	transferPtr.tsMap = make(map[string]*tsFileInfo, 0)

	transferPtr.m3u8DestFile = transferPtr.getDestM3u8File()

	transferPtr.usedDel = useddel
	transferPtr.delTsArray = make([]string, 0, 0)
	transferPtr.delM3u8Array = make([]string, 0, 0)

	transferPtr.createM3u8File()

	go transferPtr.checkDelay()

	return transferPtr
}
