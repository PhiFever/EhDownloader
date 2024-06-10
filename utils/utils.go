package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/carlmjohnson/requests"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	parallelism = 5 //页面处理的并发量
	DelayMs     = 330
)

type ImageInfo struct {
	Title string
	Url   string
	Refer string
}

func ErrorCheck(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func ToSafeFilename(in string) string {
	//https://stackoverflow.com/questions/1976007/what-characters-are-forbidden-in-windows-and-linux-directory-names
	//全部替换为_
	rp := strings.NewReplacer(
		"/", "_",
		`\`, "_",
		"<", "_",
		">", "_",
		":", "_",
		`"`, "_",
		"|", "_",
		"?", "_",
		"*", "_",
	)
	rt := rp.Replace(in)
	return rt
}

// BuildCache 用于生成utf-8格式的缓存文件 data为待写入数据结构
func BuildCache(saveDir string, cacheFile string, data interface{}) error {
	dir, _ := filepath.Abs(saveDir)
	err := os.MkdirAll(dir, os.ModePerm)
	ErrorCheck(err)

	// 打开文件用于写入数据
	file, err := os.Create(filepath.Join(dir, cacheFile))
	if err != nil {
		fmt.Println("File creation error:", err)
		return err
	}
	defer func(file *os.File) {
		err := file.Close()
		ErrorCheck(err)
	}(file)

	// 创建 JSON 编码器，并指定输出流为文件
	encoder := json.NewEncoder(file)
	// 设置编码器的输出流为 UTF-8
	encoder.SetIndent("", "    ") // 设置缩进，可选
	encoder.SetEscapeHTML(false)  // 禁用转义 HTML
	err = encoder.Encode(data)
	if err != nil {
		fmt.Println("JSON encoding error:", err)
		return err
	}

	return nil
}

// LoadCache 用于加载utf-8格式的缓存文件 result是一个指向目标数据结构的指针
func LoadCache(filePath string, result interface{}) error {
	// 打开utf-8格式的文件用于读取数据
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("File open error:", err)
		return err
	}
	defer func(file *os.File) {
		err := file.Close()
		ErrorCheck(err)
	}(file)

	// 创建 JSON 解码器
	decoder := json.NewDecoder(file)
	// 设置解码器的输入流为 UTF-8
	err = decoder.Decode(result)
	if err != nil {
		return err
	}
	return nil
}

func FileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return err == nil || os.IsExist(err)
}

// GetFileTotal 用于获取指定目录下指定后缀的文件数量
func GetFileTotal(dirPath string, fileSuffixes []string) int {
	var count int // 用于存储文件数量的变量

	// 使用Walk函数遍历指定目录及其子目录中的所有文件和文件夹
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// 检查是否为文件
		if !info.IsDir() {
			// 获取文件的扩展名
			ext := filepath.Ext(path)
			// 将扩展名转换为小写，以便比较
			ext = strings.ToLower(ext)
			// 检查文件扩展名是否在指定的后缀列表中
			for _, suffix := range fileSuffixes {
				if ext == suffix {
					count++
					break // 找到匹配的后缀，停止循环
				}
			}
		}
		return nil
	})

	if err != nil {
		fmt.Println("遍历目录出错:", err)
	}

	return count
}

// GetBeginIndex 用于获取指定目录下指定格式和后缀的文件中最大的序号，用于计算剩余图片数（目前只支持`数字_数字.后缀`的格式）
func GetBeginIndex(dirPath string, fileSuffixes []string) int {
	files, err := os.ReadDir(dirPath)
	if err != nil {
		fmt.Println("Error reading directory:", err)
		return 0
	}

	maxIndex := 0

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		for _, suffix := range fileSuffixes {
			if strings.HasSuffix(file.Name(), suffix) {
				name := strings.TrimSuffix(file.Name(), suffix)
				parts := strings.Split(name, "_")
				if len(parts) != 2 {
					continue
				}

				index, err := strconv.Atoi(parts[0])
				if err != nil {
					continue
				}

				if index > maxIndex {
					maxIndex = index
				}
			}
		}
	}

	return maxIndex
}

// ReadListFile 用于按行读取列表文件，返回一个字符串切片
func ReadListFile(filePath string) ([]string, error) {
	var list []string
	file, err := os.Open(filePath)
	if err != nil {
		return list, err
	}
	defer func(file *os.File) {
		err := file.Close()
		ErrorCheck(err)
	}(file)

	var line string
	for {
		_, err := fmt.Fscanln(file, &line)
		if err != nil {
			break
		}
		list = append(list, line)
	}
	return list, nil
}

// SaveFile 用于保存文件
func SaveFile(filePath string, data []byte) error {
	file, err := os.Create(filePath)
	//fmt.Println(filePath)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		err := file.Close()
		ErrorCheck(err)
	}(file)

	_, err = file.Write(data)
	if err != nil {
		return err
	}

	return nil
}

// SaveImagesWithRequest 通过requests库更方便的保存imageInfoList中的所有图片
func SaveImagesWithRequest(c *http.Client, h http.Header, imageInfoList []ImageInfo, saveDir string) {
	dir, err := filepath.Abs(saveDir)
	ErrorCheck(err)
	err = os.MkdirAll(dir, os.ModePerm)
	ErrorCheck(err)

	for _, data := range imageInfoList {
		filePath, err := filepath.Abs(filepath.Join(dir, data.Title))
		ErrorCheck(err)
		err = requests.
			URL(data.Url).
			Client(c).
			ToFile(filePath).
			Headers(h).
			Fetch(context.Background())
		if err != nil {
			log.Printf("Error saving image: %s by error %v", data.Title, err)
		} else {
			log.Println("Image saved:", data.Title)
		}
		time.Sleep(time.Millisecond * time.Duration(DelayMs))
	}

}

func SaveImagesWithMultiRequest(c *http.Client, h http.Header, imageInfoList []ImageInfo, saveDir string) {
	dir, err := filepath.Abs(saveDir)
	ErrorCheck(err)
	err = os.MkdirAll(dir, os.ModePerm)
	ErrorCheck(err)

	// Use a buffered channel as a semaphore to limit the number of goroutines running simultaneously
	semaphore := make(chan struct{}, parallelism)
	var wg sync.WaitGroup

	for _, data := range imageInfoList {
		wg.Add(1)
		// Acquire a semaphore slot before starting the goroutine
		semaphore <- struct{}{}

		go func(data ImageInfo) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release the semaphore slot when done

			filePath, err := filepath.Abs(filepath.Join(dir, data.Title))
			ErrorCheck(err)
			err = requests.
				URL(data.Url).
				Client(c).
				ToFile(filePath).
				Headers(h).
				Fetch(context.Background())
			if err != nil {
				log.Printf("Error saving image: %s by error %v", data.Title, err)
			} else {
				log.Println("Image saved:", data.Title)
			}
			time.Sleep(time.Millisecond * time.Duration(DelayMs))
		}(data)
	}

	// Wait for all goroutines to complete
	wg.Wait()
}

// ExtractSubstringFromText 按照Pattern在text里匹配，找到了就返回匹配到的部分
func ExtractSubstringFromText(pattern string, text string) (number string, err error) {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return "", err
	}

	match := regex.FindStringSubmatch(text)
	if match != nil {
		number = match[1]
		return number, nil
	} else {
		return "", fmt.Errorf("在pattern中未找到匹配的数字")
	}
}

func CheckUpdate(lastUpdateTime string, newTime string) bool {
	layout := "2006-01-02" //时间格式模板
	parsedDate1, err := time.Parse(layout, lastUpdateTime)
	if err != nil {
		fmt.Println("日期解析错误:", err)
		return true
	}
	parsedDate2, err := time.Parse(layout, newTime)
	if err != nil {
		fmt.Println("日期解析错误:", err)
		return true
	}

	if parsedDate1.Before(parsedDate2) {
		return true
	} else if parsedDate1.After(parsedDate2) {
		fmt.Println("解析的日期晚于当前日期，galleryInfo.json文件异常")
		return true
	} else {
		return false
	}
}

// ElementInSlice 判断slice中是否存在某个item
func ElementInSlice(value interface{}, array interface{}) bool {
	switch reflect.TypeOf(array).Kind() {
	case reflect.Slice:
		s := reflect.ValueOf(array)
		for i := 0; i < s.Len(); i++ {
			if reflect.DeepEqual(value, s.Index(i).Interface()) {
				return true
			}
		}
	}
	return false
}

// checkSequentialFileNames 检查指定目录中是否包含从1到maxNumber的连续数字命名的文件。
func checkSequentialFileNames(directory string, maxNumber int) (bool, error) {
	// 读取目录中的所有文件和子目录（不会递归到子目录）
	files, err := ioutil.ReadDir(directory)
	if err != nil {
		return false, err
	}

	// 创建一个map来跟踪存在的文件名
	fileNames := make(map[int]bool)
	for _, file := range files {
		if file.IsDir() {
			continue // 忽略目录
		}
		// 去除文件名中的扩展名
		name := file.Name()
		nameWithoutExt := name[:len(name)-len(filepath.Ext(name))]
		// 将文件名转换为整数
		number, err := strconv.Atoi(nameWithoutExt)
		if err == nil { // 如果转换成功，标记该数字
			fileNames[number] = true
		}
	}

	// 检查从1到maxNumber的每个数字是否都有对应的文件
	for i := 1; i <= maxNumber; i++ {
		if !fileNames[i] {
			return false, nil
		}
	}

	return true, nil
}
