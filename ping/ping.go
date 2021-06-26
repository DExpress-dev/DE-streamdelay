package ping

import (
	"fmt"
	"sync"
	"time"
)

//ping的对象
type PingObject struct {
	url           string //ping的地址
	beginTimer    string //启动检测时间
	sucessCount   int    //成功次数
	failCount     int    //失败次数
	lastFailTimer string //最后一次失败时间
}

//ping 管理类
type PingManager struct {
	urlLock sync.Mutex             //成功临界区
	urlMap  map[string]*PingObject //需要PING的IP数组
}

//ping检测处理
func (pingObjPtr *PingObject) Ping(host string) {

	/*var count int
	var size int
	var timeout int64
	var neverstop bool
	count = args["n"].(int)
	size = args["l"].(int)
	timeout = args["w"].(int64)
	neverstop = args["t"].(bool)

	cname, _ := net.LookupCNAME(host)
	starttime := time.Now()
	conn, err := net.DialTimeout("ip4:icmp", host, time.Duration(timeout*1000*1000))
	ip := conn.RemoteAddr()
	fmt.Println("正在 Ping " + cname + " [" + ip.String() + "] 具有 32 字节的数据:")

	var seq int16 = 1
	id0, id1 := genidentifier(host)
	const ECHO_REQUEST_HEAD_LEN = 8

	sendN := 0
	recvN := 0
	lostN := 0
	shortT := -1
	longT := -1
	sumT := 0

	for count > 0 || neverstop {
		sendN++
		var msg []byte = make([]byte, size+ECHO_REQUEST_HEAD_LEN)
		msg[0] = 8                        // echo
		msg[1] = 0                        // code 0
		msg[2] = 0                        // checksum
		msg[3] = 0                        // checksum
		msg[4], msg[5] = id0, id1         //identifier[0] identifier[1]
		msg[6], msg[7] = gensequence(seq) //sequence[0], sequence[1]

		length := size + ECHO_REQUEST_HEAD_LEN

		check := checkSum(msg[0:length])
		msg[2] = byte(check >> 8)
		msg[3] = byte(check & 255)

		conn, err = net.DialTimeout("ip:icmp", host, time.Duration(timeout*1000*1000))

		checkError(err)

		starttime = time.Now()
		conn.SetDeadline(starttime.Add(time.Duration(timeout * 1000 * 1000)))
		_, err = conn.Write(msg[0:length])

		const ECHO_REPLY_HEAD_LEN = 20

		var receive []byte = make([]byte, ECHO_REPLY_HEAD_LEN+length)
		n, err := conn.Read(receive)
		_ = n

		var endduration int = int(int64(time.Since(starttime)) / (1000 * 1000))

		sumT += endduration

		time.Sleep(1000 * 1000 * 1000)

		if err != nil || receive[ECHO_REPLY_HEAD_LEN+4] != msg[4] || receive[ECHO_REPLY_HEAD_LEN+5] != msg[5] || receive[ECHO_REPLY_HEAD_LEN+6] != msg[6] || receive[ECHO_REPLY_HEAD_LEN+7] != msg[7] || endduration >= int(timeout) || receive[ECHO_REPLY_HEAD_LEN] == 11 {
			lostN++
			fmt.Println("对 " + cname + "[" + ip.String() + "]" + " 的请求超时。")
		} else {
			if shortT == -1 {
				shortT = endduration
			} else if shortT > endduration {
				shortT = endduration
			}
			if longT == -1 {
				longT = endduration
			} else if longT < endduration {
				longT = endduration
			}
			recvN++
			ttl := int(receive[8])
			//			fmt.Println(ttl)
			fmt.Println("来自 " + cname + "[" + ip.String() + "]" + " 的回复: 字节=32 时间=" + strconv.Itoa(endduration) + "ms TTL=" + strconv.Itoa(ttl))
		}

		seq++
		count--
	}
	stat(ip.String(), sendN, lostN, recvN, shortT, longT, sumT)
	c <- 1*/
}

func (pingObjPtr *PingObject) checkSum(msg []byte) uint16 {
	sum := 0

	length := len(msg)
	for i := 0; i < length-1; i += 2 {
		sum += int(msg[i])*256 + int(msg[i+1])
	}
	if length%2 == 1 {
		sum += int(msg[length-1]) * 256 // notice here, why *256?
	}

	sum = (sum >> 16) + (sum & 0xffff)
	sum += (sum >> 16)
	var answer uint16 = uint16(^sum)
	return answer
}

func (pingObjPtr *PingObject) checkError(err error) {
	/*if err != nil {
		os.Exit(1)
	}*/
}

func (pingObjPtr *PingObject) gensequence(v int16) (byte, byte) {
	ret1 := byte(v >> 8)
	ret2 := byte(v & 255)
	return ret1, ret2
}

func (pingObjPtr *PingObject) genidentifier(host string) (byte, byte) {
	return host[0], host[1]
}

func (pingObjPtr *PingObject) stat(ip string, sendN int, lostN int, recvN int, shortT int, longT int, sumT int) {
	fmt.Println()
	fmt.Println(ip, " 的 Ping 统计信息:")
	fmt.Printf("    数据包: 已发送 = %d，已接收 = %d，丢失 = %d (%d%% 丢失)，\n", sendN, recvN, lostN, int(lostN*100/sendN))
	fmt.Println("往返行程的估计时间(以毫秒为单位):")
	if recvN != 0 {
		fmt.Printf("    最短 = %dms，最长 = %dms，平均 = %dms\n", shortT, longT, sumT/sendN)
	}
}

//得到当前时间 20171226T143444
func (pingManagerPtr *PingManager) getCurrentTimerISO() string {

	timeNow := time.Now()
	var currentDay string = timeNow.Format("20060102")
	currentTimer := timeNow.Format("150405")
	timeNowString := currentDay + "T" + currentTimer
	return timeNowString
}

//添加
func (pingManagerPtr *PingManager) AddPing(url string) {

	pingManagerPtr.urlLock.Lock()
	defer pingManagerPtr.urlLock.Unlock()

	var pingObjPtr *PingObject = new(PingObject)
	pingObjPtr.url = url
	pingObjPtr.beginTimer = pingManagerPtr.getCurrentTimerISO()
	pingObjPtr.sucessCount = 0
	pingObjPtr.failCount = 0
	pingObjPtr.lastFailTimer = ""

	pingManagerPtr.urlMap[url] = pingObjPtr
	go pingObjPtr.Ping(url)
}

//删除
func (pingManagerPtr *PingManager) deletePing(url string) {

	pingManagerPtr.urlLock.Lock()
	defer pingManagerPtr.urlLock.Unlock()

	delete(pingManagerPtr.urlMap, url)
}

//查询
func (pingManagerPtr *PingManager) findPing(url string) *PingObject {

	pingManagerPtr.urlLock.Lock()
	defer pingManagerPtr.urlLock.Unlock()

	pingObjPtr, _ := pingManagerPtr.urlMap[url]
	return pingObjPtr

}

//是否存在
func (pingManagerPtr *PingManager) existPing(url string) bool {

	pingManagerPtr.urlLock.Lock()
	defer pingManagerPtr.urlLock.Unlock()

	_, result := pingManagerPtr.urlMap[url]
	return result
}

//创建类
func CreatePingManager() *PingManager {

	pingManagerPtr := new(PingManager)
	pingManagerPtr.urlMap = make(map[string]*PingObject)
	return pingManagerPtr
}
