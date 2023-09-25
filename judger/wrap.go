package judger

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/LQQ4321/owo/config"
)

type FileInfo interface {
	isFile()
}

type LocalFile struct {
	Src string `json:"src"`
}

func (f *LocalFile) isFile() {

}

type MemoryFile struct {
	Content string `json:"content"`
}

func (f *MemoryFile) isFile() {

}

type Collector struct {
	Name string `json:"name"`
	Max  int    `json:"max"`
	Pipe bool   `json:"pipe"`
}

func (f *Collector) isFile() {

}

type PrepareFile struct {
	FileId string `json:"fileId"`
}

func (f *PrepareFile) isFile() {

}

type Cmd struct {
	Args     []string   `json:"args"`
	Env      []string   `json:"env"`
	Files    []FileInfo `json:"files"`
	CpuLimit int64      `json:"cpuLimit"` //ns
	// ClockLimit        int64               `json:"clockLimit"`  //ns
	MemoryLimit int64 `json:"memoryLimit"` //byte
	// StackLimit        int64               `json:"stackLimit"`  //byte
	ProcLimit         int64               `json:"procLimit"`
	StrictMemoryLimit bool                `json:"strictMemoryLimit"`
	CopyIn            map[string]FileInfo `json:"copyIn"`
	// CopyOut           []string            `json:"copyOut"`
	CopyOutCached []string `json:"copyOutCached"`
	CopyOutMax    int64    `json:"copyOutMax"`
}

type FileError struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Message string `json:"message"`
}

type Result struct {
	Status     string            `json:"status"`
	Error      string            `json:"error"`
	ExitStatus int               `json:"exitStatus"`
	Time       int64             `json:"time"`
	Memory     int64             `json:"memory"`
	Runtime    int64             `json:"runtime"`
	Files      map[string]string `json:"files"`
	FileIds    map[string]string `json:"fileIds"`
	FileError  []FileError       `json:"fileError"`
}

func (r *Result) String() string {
	res := make([]string, 0)
	res = append(res, fmt.Sprintf("Status : %s", r.Status))
	res = append(res, fmt.Sprintf("Error : %s", r.Error))
	res = append(res, fmt.Sprintf("Time : %d", r.Time))
	res = append(res, fmt.Sprintf("Memory : %d", r.Memory))
	res = append(res, fmt.Sprintf("Runtime : %v", r.Runtime))
	res = append(res, fmt.Sprintf("Files : %v", r.Files))
	res = append(res, fmt.Sprintf("FileIds : %v", r.FileIds))
	res = append(res, fmt.Sprintf("FileError : %v", r.FileError))
	return strings.Join(res, "  #  ")
}
func (r *Result) executeFileId() string {
	// ok是false的情况还没处理
	if v, ok := r.FileIds["a"]; ok {
		return v
	} else if v, ok := r.FileIds["Main.class"]; ok {
		return v
	}
	return ""
}
func (r *Result) OutputFileId() string {
	if v, ok := r.FileIds["stdout"]; ok {
		return v
	}
	return ""
}
func (r *Result) Parse() []string {
	res := make([]string, 0)
	res = append(res, r.Status)
	res = append(res, r.FileIds["stdout"])
	res = append(res, strconv.FormatInt(r.Time>>20, 10))
	res = append(res, strconv.FormatInt(r.Memory>>20, 10))
	res = append(res, strconv.FormatInt(r.Runtime>>20, 10))
	return res
}

// go-judge的作者是不是因为Request是传参，为了避免复制，所以用指针
// 而Response是通过管道传输，不产生复制(猜的)，所以不用指针
func judger(req *Request, ch chan<- Response) {
	if req.Optional&1 == 1 {
		result, err := complier(req.Language, req.SourceFilePath)
		if err != nil {
			ch <- Response{
				Error: fmt.Errorf("complier func error"),
			}
			return
		}
		if req.Optional == 1 || result.Status != "Accepted" {
			ch <- Response{
				Results: []Result{*result},
			}
			return
		}
		fileId := result.executeFileId()
		if fileId == "" {
			ch <- Response{
				Error: fmt.Errorf("cannot parse file id from result"),
			}
			return
		}
		defer func() { //删除编译产生的文件
			if err := os.Remove(config.SHARE_JUDGER + fileId); err != nil {
				logger.Errorln(err)
			}
		}()
		req.ExecuteFileId = fileId
	}
	results, err := execute(req)
	if err != nil {
		ch <- Response{
			Error: fmt.Errorf("execute func error"),
		}
		return
	}
	ch <- Response{
		Results: results,
	}
}

func judgerServer(req *Request, proc *process) {
	ch := make(chan Response)
	defer close(ch)
	// judger协程是由judgerServr协程产生的，所以judgerServer协程关闭，judger协程也会关闭？？？
	go judger(req, ch)
	select {
	case res, ok := <-ch:
		close(proc.done)
		if !ok {
			proc.Response = Response{
				Error: fmt.Errorf("judgerServer chan fail"),
			}
			return
		}
		// 得到回复，首先把定时器关了先
		proc.Response = res
	case <-proc.done: //如果超时了，
		return
	}
}
