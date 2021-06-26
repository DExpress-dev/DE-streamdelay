package config

import (
	"github.com/widuu/goini"
)

type configInfo struct {
	Cpid      string
	Localhost string

	WebIp   string
	WebPort string

	AlarmUrl string

	UrlFile string

	DownloadTimeout int
}

var _cfg *configInfo

func GetInstance() *configInfo {
	return _cfg
}

func init() {
	_cfg = new(configInfo)

	configIni := goini.Init("config.ini")
	//TODO m3u8保存的天数 ts保存的天数

	_cfg.Cpid = configIni.ReadString("COMMON", "cpid", "otv")
	_cfg.Localhost = configIni.ReadString("COMMON", "localhost", "127.0.0.1")

	_cfg.WebIp = configIni.ReadString("WEB", "ip", "0.0.0.0")
	_cfg.WebPort = configIni.ReadString("WEB", "port", "80")

	_cfg.AlarmUrl = configIni.ReadString("ALARM", "url", "http://127.0.0.1/otvcloud/Alarm")

	_cfg.UrlFile = configIni.ReadString("STREAM", "urlFile", "urls.ini")

	_cfg.DownloadTimeout = configIni.ReadInt("Download", "timeout", 15)
}
