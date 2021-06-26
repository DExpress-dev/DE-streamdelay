// stream_check project alarm.go
package alarm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	log4plus "github.com/alecthomas/log4go"
	"github.com/widuu/goini"
)

const (
	AlarmType                   = iota
	AlarmTypeIndexM3u8DownFail  //IndexM3u8下载失败
	AlarmTypeM3u8CreateFail     //M3u8文件创建失败
	AlarmTypeM3u8DownFail       //M3u8下载失败
	AlarmTypeTSDownFail         //TS下载失败
	AlarmTypeTransferCreateFail //延时文件创建失败
)

//报警级别
const (
	CommonLevel          = 1    //一般
	PoorLevel            = 10   //较重
	SeriousLevel         = 100  //严重
	CriticalSeriousLevel = 1000 //特别严重
)

//sub struct of outAlarmBody
type AlarmJson struct {
	Alarmlevel   int    `json:"alarmlevel"`
	Alarmtype    int    `json:"alarmtype"`
	Url          string `json:"url"`
	Alarmip      string `json:"alarmip"`
	Alarmcontext string `json:"alarmcontext"`
	Alarmtimer   string `json:"alarmtimer"`
}

//post body to monitor server
type AlarmBody struct {
	Cpid   string      `json:"cpid"`
	Alarms []AlarmJson `json:"alarms"`
}

type Alarm struct {
	alarmUrl  string //报警发送的URL
	alarmAuth string //报警的认证
	alarmIp   string //报警ip
	cpid      string

	bodyLock sync.Mutex
	body     *AlarmBody
}

var _alarm *Alarm = initAlarm()

//报警信息
func (alarmPtr *Alarm) bodyReset() {
	alarmPtr.body = new(AlarmBody)
	alarmPtr.body.Cpid = alarmPtr.cpid
}

//报警信息
func (alarmPtr *Alarm) checkSend() {
	for {
		var checkBody *AlarmBody = nil
		alarmPtr.bodyLock.Lock()
		log4plus.Debug("Alarm.checkSend alarms.len=%d", len(alarmPtr.body.Alarms))
		if len(alarmPtr.body.Alarms) > 0 {
			checkBody = alarmPtr.body
			alarmPtr.bodyReset()
		}
		alarmPtr.bodyLock.Unlock()

		if checkBody != nil {
			bytes, _ := json.Marshal(*checkBody)
			//	SendAlarm(bytes)
			log4plus.Error("Alarm checkSend: %s", string(bytes))
		}
		time.Sleep(5 * time.Second)
	}
}

//报警信息
func (alarmPtr *Alarm) loadConfig() {
	configPtr := goini.Init("config.ini")
	alarmPtr.alarmUrl = configPtr.ReadString("ALARM", "URL", "")
	alarmPtr.alarmAuth = configPtr.ReadString("ALARM", "AUTH", "")
	alarmPtr.alarmIp = configPtr.ReadString("ALARM", "IP", "")
	alarmPtr.cpid = configPtr.ReadString("ALARM", "cpid", "")
}

//发送报警信息
func httpPost(bodyString string) {

	//发送数据
	resp, err := http.Post(_alarm.alarmUrl, "application/x-www-form-urlencoded", strings.NewReader(bodyString))
	if err != nil {
		log4plus.Error("httpPost error:" + err.Error())
		return
	}
	defer resp.Body.Close()

	//得到返回数据
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {

		log4plus.Error("no resp error:" + err.Error())
		return
	}
	fmt.Println(string(body))
	log4plus.Debug("Post Alarm Response %s", string(body))
}

//发送POST消息给指定
func SendAlarm(body []byte) bool {

	//生成client 参数为默认
	client := &http.Client{}

	//提交请求
	request, err := http.NewRequest("POST", _alarm.alarmUrl, bytes.NewReader(body))
	if err != nil {

		log4plus.Error("Post Alarm Message %s Error %s", body, err.Error())
		return false
	}
	log4plus.Debug("Post Alarm Message %s", body)

	//拼接包头并发送数据
	request.Proto = "HTTP/1.1"
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Add("Authorization", _alarm.alarmAuth)
	response, err := client.Do(request)
	if err == nil {

		//读取返回信息
		responseBody, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log4plus.Error("response StatusCode %v error %s", response.StatusCode, err.Error())
			return false
		}
		status := response.StatusCode
		log4plus.Debug("response StatusCode: %v body: %s", status, string(responseBody))
		return true

	} else {
		log4plus.Error("client.Do error %s", err.Error())
		return false
	}
}

func DoAlarm(aLevel int, aType int, url string, context string) {
	_alarm.bodyLock.Lock()
	defer _alarm.bodyLock.Unlock()
	_alarm.body.Alarms = append(_alarm.body.Alarms, AlarmJson{aLevel, aType, url, _alarm.alarmIp, context, time.Now().Format("2006-01-02 15:04:05")})
}

func initAlarm() *Alarm {
	alarmPtr := new(Alarm)
	alarmPtr.loadConfig()
	alarmPtr.bodyReset()
	go alarmPtr.checkSend()
	return alarmPtr
}
