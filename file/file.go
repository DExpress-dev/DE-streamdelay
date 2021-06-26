// stream_check project download.go
package file

import (
	"sync"
)

type FileManager struct {
	m3u8ContextLock sync.Mutex //临界区
	localM3u8Path   string     //本地m3u8地址
	LastM3u8Context string     //最近的M3u8的文件内容
}

//M3U8的内容是否一致
func (fileMgrPtr *FileManager) FileContextEqual(context string) bool {

	fileMgrPtr.m3u8ContextLock.Lock()
	defer fileMgrPtr.m3u8ContextLock.Unlock()

	if fileMgrPtr.LastM3u8Context == context {
		return true
	} else {
		fileMgrPtr.LastM3u8Context = context
		return false
	}
}

//设置最新的m3u8内容
func (fileMgrPtr *FileManager) SetLastM3u8Context(context string) {

	fileMgrPtr.m3u8ContextLock.Lock()
	defer fileMgrPtr.m3u8ContextLock.Unlock()

	fileMgrPtr.LastM3u8Context = context
}

//新建
func CreateFileManager(localPath string) *FileManager {

	fileMgrPtr := new(FileManager)

	fileMgrPtr.localM3u8Path = localPath
	fileMgrPtr.LastM3u8Context = ""

	return fileMgrPtr
}
