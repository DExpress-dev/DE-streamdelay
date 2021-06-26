// stream_check project download.go
package download

import (
	"bytes"
	"delay_stream/src/public"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	log4plus "github.com/alecthomas/log4go"
)

type Download struct {
}

//下载文件
func (downloadPtr *Download) DownloadSaveFile(url string, localFile, localPath string) (bool, int) {

	//开始下载文件
	webClient := &http.Client{

		Transport: &http.Transport{
			Dial: func(netw, addr string) (net.Conn, error) {
				conn, err := net.DialTimeout(netw, addr, time.Second*5)
				if err != nil {
					return nil, err
				}
				conn.SetDeadline(time.Now().Add(time.Second * 5))
				return conn, nil
			},
			ResponseHeaderTimeout: time.Second * 5,
		},
	}

	//下载文件
	response, err := webClient.Get(url)
	if err != nil {
		public.CheckError(err, "DownloadSaveFile Fail")
		return false, 0
	}
	defer response.Body.Close()

	if 200 == response.StatusCode {
		err = os.MkdirAll(localPath, 0777)
		if err != nil {
			log4plus.Error("os.MkdirAll(localPath, 0777) Fail path=%s", localPath)
			return false, response.StatusCode
		}

		if !public.FileExist(localFile) {

			file, err := os.Create(localFile)
			if err != nil {
				log4plus.Error("os.Create(localFile) Fail localFile=%s", localFile)
				return false, response.StatusCode
			}
			defer file.Close()

			io.Copy(file, response.Body)
			return true, response.StatusCode
		} else {
			log4plus.Error("public.FileExist(localFile) localFile=%s", localFile)
			return false, response.StatusCode
		}
	} else {
		return false, response.StatusCode
	}
}

//下载文件，并读取内容
func DownloadToContext(url string) (bool, string, int) {

	webClient := &http.Client{
		Transport: &http.Transport{
			Dial: func(netw, addr string) (net.Conn, error) {
				conn, err := net.DialTimeout(netw, addr, time.Second*5)
				if err != nil {
					return nil, err
				}
				conn.SetDeadline(time.Now().Add(time.Second * 5))
				return conn, nil
			},
			ResponseHeaderTimeout: time.Second * 5,
		},
	}

	response, err := webClient.Get(url)
	if err != nil {
		public.CheckError(err, "DownloadToContext Fail")
		return false, "", 0
	}
	defer response.Body.Close()

	if 200 == response.StatusCode {

		buf := new(bytes.Buffer)
		buf.ReadFrom(response.Body)
		context := buf.String()

		return true, context, response.StatusCode
	} else {

		return false, "", response.StatusCode
	}
}

func CreateDownload() *Download {

	downloadPtr := new(Download)
	return downloadPtr
}
