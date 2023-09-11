package judger

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/LQQ4321/owo/config"
	"go.uber.org/zap"
)

var (
	logger *zap.SugaredLogger
)

func JudgerInit(instantiation *zap.SugaredLogger) {
	logger = instantiation
}

// 将windows端产生的测试文件转为linux端(具体操作是将\r\n转为\n)
func FileConversion(filePath, destination string) error {
	winFile, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer winFile.Close()

	linuxFile, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer linuxFile.Close()

	scanner := bufio.NewScanner(winFile)
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r\n") + "\n"
		_, err := linuxFile.WriteString(line)
		if err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

// read file,generate corresponding hash value
func GenerateHashValue(filePath string) string {
	contentByte, err := ioutil.ReadFile(filePath)
	if err != nil {
		logger.Infoln("read "+filePath+" file fail", err)
		return ""
	}
	var id int = -1
	for i := len(contentByte) - 1; i >= 0; i-- {
		if contentByte[i] != 10 && contentByte[i] != 32 {
			id = i
			break
		}
	}
	reader := bytes.NewReader(contentByte[:id+1])
	// there may be room for improvement here
	hash := sha256.New()
	if _, err := io.Copy(hash, reader); err != nil {
		logger.Infoln("reader copy to hash fail :", err)
		return ""
	}
	hashValue := hash.Sum(nil)
	return fmt.Sprintf("%x", hashValue)
}

// =====================================================================================

// 也许可以尝试一下一次编译加运行，但是对于多个测试数据来说，就要重复编译好多次了
func SendToJudger(data []byte) ([]Result, error) {
	req, err := http.NewRequest("POST", config.JUDGER_DSN, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body := new(bytes.Buffer)
	_, err = body.ReadFrom(resp.Body)
	if err != nil {
		return nil, err
	}
	var Results []Result
	// logger.Infoln(string(body.Bytes()))
	// 有时候由于请求体不规范，文件不存在或者其他原因，judger不会返回一个[]Result格式的回复，
	// 而是返回一个字符串，然后就会导致解析错误，这时候也应该算是ojserver错误
	err = json.Unmarshal(body.Bytes(), &Results)
	if err != nil {
		return nil, err
	}
	// logger.Infoln(Results)
	return Results, nil
}

// 编译，只有python3不需要编译，其他都需要编译。编译只需要一次
// 需要的参数：语言类型，源文件地址
// []string{"c++","/dev/shm/files/allContest/2/1/submit/1.cpp"}
func complier(language, filePath string) (*Result, error) {
	CmdInfo := &Cmd{}
	CmdInfo.Env = []string{"PATH=/usr/bin:/bin"}
	CmdInfo.ProcLimit = 50
	CmdInfo.CopyOutMax = 10 * 1024 * 1024
	CmdInfo.CpuLimit = 5000000000 //编译时间这里定死了，定为5s，因为java和go的编译时间较长
	CmdInfo.MemoryLimit = 104857600
	CmdInfo.Files = make([]FileInfo, 3)
	CmdInfo.CopyOutCached = []string{"a"}
	CmdInfo.Files[0] = &MemoryFile{Content: ""}
	CmdInfo.Files[1] = &Collector{Name: "stdout", Max: 10240}
	CmdInfo.Files[2] = &Collector{Name: "stderr", Max: 10240}
	if language == "c++" {
		CmdInfo.Args = []string{"/usr/bin/g++", "-std=c++17", "a.cpp", "-o", "a"}
		CmdInfo.CopyIn = map[string]FileInfo{
			"a.cpp": &LocalFile{Src: filePath},
		}
	} else if language == "c" {
		CmdInfo.Args = []string{"/usr/bin/gcc", "a.c", "-o", "a"}
		CmdInfo.CopyIn = map[string]FileInfo{
			"a.c": &LocalFile{Src: filePath},
		}
	} else if language == "golang" {
		CmdInfo.Args = []string{"/usr/bin/go", "build", "-o", "a", "a.go"}
		CmdInfo.Env = []string{"PATH=/usr/bin:/bin", "GOPATH=/w", "GOCACHE=/tmp/"}
		CmdInfo.CopyIn = map[string]FileInfo{
			"a.go": &LocalFile{Src: filePath},
		}
	} else if language == "java" {
		CmdInfo.Args = []string{"/usr/bin/javac", "Main.java"}
		CmdInfo.CopyIn = map[string]FileInfo{
			"Main.java": &LocalFile{Src: filePath},
		}
		CmdInfo.CopyOutCached = []string{"Main.class"}
	}
	var request struct {
		V []Cmd `json:"cmd"`
	}
	request.V = append(request.V, *CmdInfo)
	jsonData, err := json.Marshal(request)
	if err != nil {
		logger.Infoln(err)
		return nil, err
	}
	results, err := SendToJudger(jsonData)
	if err != nil {
		logger.Infoln(err)
		return nil, err
	}
	return &results[0], nil
}

func execute(req *Request) ([]Result, error) {
	var request struct {
		V []*Cmd `json:"cmd"`
	}
	req.Time <<= 20
	req.Memory <<= 20
	for _, v := range req.InputPath {
		c := &Cmd{}
		c.Args = []string{"a"} //c,c++,golang
		c.Env = []string{"PATH=/usr/bin:/bin"}
		c.Files = make([]FileInfo, 3)
		c.Files[0] = &LocalFile{Src: v}
		c.Files[1] = &Collector{Name: "stdout", Max: 10240}
		c.Files[2] = &Collector{Name: "stderr", Max: 10240}
		c.CpuLimit = req.Time      //c,c++
		c.MemoryLimit = req.Memory //c,c++
		c.ProcLimit = 50
		c.CopyIn = map[string]FileInfo{ //c,c++,golang
			"a": &PrepareFile{
				FileId: req.ExecuteFileId,
			},
		}
		c.CopyOutCached = []string{"stdout"}
		if req.Language == "c++" {

		} else if req.Language == "c" {

		} else if req.Language == "golang" {
			c.CpuLimit = req.Time * 2
			c.MemoryLimit = req.Memory * 2
		} else if req.Language == "java" {
			c.Args = []string{"/usr/bin/java", "Main"}
			c.CpuLimit = req.Time * 2
			c.MemoryLimit = req.Memory * 2
			c.CopyIn = map[string]FileInfo{
				"Main.class": &PrepareFile{
					FileId: req.ExecuteFileId,
				},
			}
		} else if req.Language == "python3" {
			c.Args = []string{"/usr/bin/python3", "a.py"}
			c.CpuLimit = req.Time * 2
			c.MemoryLimit = req.Memory * 2
			c.CopyIn = map[string]FileInfo{
				"a.py": &LocalFile{
					Src: req.SourceFilePath,
				},
			}
		} else {
			return []Result{}, fmt.Errorf("not exists language")
		}
		request.V = append(request.V, c)
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		logger.Infoln(err)
		return []Result{}, err
	}
	results, err := SendToJudger(jsonData)
	if err != nil {
		logger.Infoln(err)
		return nil, err
	}
	return results, nil
}
