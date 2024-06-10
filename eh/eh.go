package eh

import (
	"EhDownloader/utils"
	"bytes"
	"context"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/carlmjohnson/requests"
	"github.com/spf13/cast"
	"github.com/ybbus/httpretry"
	"log"
	"math"
	"math/rand/v2"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	chromeUserAgent = `Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36`
	imageInOnePage  = 40
)

type GalleryInfo struct {
	URL        string              `json:"gallery_url"`
	Title      string              `json:"gallery_title"`
	TotalImage int                 `json:"total_image"`
	TagList    map[string][]string `json:"tag_list"`
}

func generateIndexURL(urlStr string, page int) string {
	u, err := url.Parse(urlStr)
	if err != nil {
		fmt.Println("Error parsing URL:", err)
		return ""
	}

	if page == 0 {
		return u.String()
	}

	q := u.Query()
	q.Set("p", cast.ToString(page))
	u.RawQuery = q.Encode()

	return u.String()
}

func buildHtmlRequestHeaders() http.Header {
	return http.Header{
		"Accept":                    {"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7"},
		"Accept-Language":           {"zh-CN,zh;q=0.9,en;q=0.8"},
		"Cache-Control":             {"no-cache"},
		"Dnt":                       {"1"},
		"Pragma":                    {"no-cache"},
		"Priority":                  {"u=0, i"},
		"Sec-Ch-Ua":                 {`"Chromium";v="124", "Google Chrome";v="124", "Not-A.Brand";v="99"`},
		"Sec-Ch-Ua-Mobile":          {"?0"},
		"Sec-Ch-Ua-Platform":        {`"Windows"`},
		"Sec-Fetch-Dest":            {"document"},
		"Sec-Fetch-Mode":            {"navigate"},
		"Sec-Fetch-Site":            {"none"},
		"Sec-Fetch-User":            {"?1"},
		"Sec-Gpc":                   {"1"},
		"Upgrade-Insecure-Requests": {"1"},
		"User-Agent":                {chromeUserAgent},
	}
}

func getGalleryInfo(c *http.Client, galleryUrl string) GalleryInfo {
	var galleryInfo GalleryInfo
	galleryInfo.TagList = make(map[string][]string)
	galleryInfo.URL = galleryUrl

	var buffer bytes.Buffer
	err := requests.URL(galleryUrl).
		Client(c).
		Headers(buildHtmlRequestHeaders()).
		ToBytesBuffer(&buffer).
		Fetch(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	doc, err := goquery.NewDocumentFromReader(&buffer)
	if err != nil {
		log.Fatal(err)
	}
	galleryInfo.Title = doc.Find("h1#gn").Text()
	pageText := doc.Find("#gdd > table > tbody > tr:nth-child(6) > td.gdt2").Text()
	reMaxPage := regexp.MustCompile(`(\d+) pages`)
	if reMaxPage.MatchString(pageText) {
		//转换为int
		galleryInfo.TotalImage = cast.ToInt(reMaxPage.FindStringSubmatch(pageText)[1])
	}

	doc.Find("div#taglist table").Each(func(_ int, s *goquery.Selection) {
		s.Find("tr").Each(func(_ int, s *goquery.Selection) {
			key := strings.TrimSpace(s.Find("td.tc").Text())
			localKey := strings.ReplaceAll(key, ":", "")
			s.Find("td div").Each(func(_ int, s *goquery.Selection) {
				value := strings.TrimSpace(s.Text())
				galleryInfo.TagList[localKey] = append(galleryInfo.TagList[localKey], value)
			})
		})
	})

	return galleryInfo
}

func getImagePageUrlList(c *http.Client, indexUrl string) []string {
	var imagePageUrls []string
	var buffer bytes.Buffer
	err := requests.
		URL(indexUrl).
		Client(c).
		UserAgent(chromeUserAgent).
		ToBytesBuffer(&buffer).
		Fetch(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	doc, err := goquery.NewDocumentFromReader(&buffer)
	if err != nil {
		log.Fatal(err)
	}

	doc.Find("div#gdt div.gdtm a").Each(func(_ int, s *goquery.Selection) {
		imgUrl, _ := s.Attr("href")
		imagePageUrls = append(imagePageUrls, imgUrl)
	})

	return imagePageUrls
}

func getImageUrl(c *http.Client, imagePageUrl string) string {
	var imageUrl string
	var buffer bytes.Buffer
	err := requests.
		URL(imagePageUrl).
		Client(c).
		UserAgent(chromeUserAgent).
		ToBytesBuffer(&buffer).
		Fetch(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	doc, err := goquery.NewDocumentFromReader(&buffer)
	if err != nil {
		log.Fatal(err)
	}
	imageUrl, _ = doc.Find("img#img").Attr("src")
	return imageUrl
}

func buildJPEGRequestHeaders() http.Header {
	return http.Header{
		"Accept":             {"image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8"},
		"Accept-Encoding":    {"gzip, deflate, br"},
		"Accept-Language":    {"zh-CN,zh;q=0.9"},
		"Connection":         {"keep-alive"},
		"Dnt":                {"1"},
		"Referer":            {"https://e-hentai.org/"},
		"Sec-Ch-Ua":          {`"Chromium";v="124", "Google Chrome";v="124", "Not-A.Brand";v="99"`},
		"Sec-Ch-Ua-Mobile":   {"?0"},
		"Sec-Ch-Ua-Platform": {`"Windows"`},
		"Sec-Fetch-Dest":     {"image"},
		"Sec-Fetch-Mode":     {"no-cors"},
		"Sec-Fetch-Site":     {"cross-site"},
		"Sec-Gpc":            {"1"},
		"User-Agent":         {chromeUserAgent},
	}
}

func getImageInfoFromPage(c *http.Client, imagePageUrl string) (string, string) {
	imageIndex := imagePageUrl[strings.LastIndex(imagePageUrl, "-")+1:]
	imageUrl := getImageUrl(c, imagePageUrl)
	imageSuffix := imageUrl[strings.LastIndex(imageUrl, "."):]
	imageTitle := fmt.Sprintf("%s%s", imageIndex, imageSuffix)
	return imageTitle, imageUrl
}

// SaveImageWithRequest 通过requests库更方便的保存imageInfo所指向的图片
func SaveImageWithRequest(c *http.Client, h http.Header, imageInfo utils.ImageInfo, saveDir string) {
	dir, _ := filepath.Abs(saveDir)
	_ = os.MkdirAll(dir, os.ModePerm)
	filePath, _ := filepath.Abs(filepath.Join(dir, imageInfo.Title))
	err := requests.URL(imageInfo.Url).
		Client(c).
		ToFile(filePath).
		Headers(h).
		Fetch(context.Background())
	if err != nil {
		log.Printf("Error saving image: %s by error %v", imageInfo.Title, err)
	} else {
		log.Println("Image saved:", imageInfo.Title)
	}

}

func DownloadGallery(outputDir string, infoJsonPath string, galleryUrl string, onlyInfo bool) error {
	//目录号
	beginIndex := 0
	//余数
	remainder := 0

	// create a new http client with retry
	c := httpretry.NewDefaultClient(
		// retry up to 5 times
		httpretry.WithMaxRetryCount(5),
		// retry on status >= 500, if err != nil, or if response was nil (status == 0)
		httpretry.WithRetryPolicy(func(statusCode int, err error) bool {
			return err != nil || statusCode >= 500 || statusCode == 0
		}),
		// every retry should wait one more second
		httpretry.WithBackoffPolicy(func(attemptNum int) time.Duration {
			return time.Duration(attemptNum+1) * 1 * time.Second
		}),
	)

	//获取画廊信息，快速判断网络联通情况
	galleryInfo := getGalleryInfo(http.DefaultClient, galleryUrl)
	fmt.Println("Total Image:", galleryInfo.TotalImage)
	baseDir := filepath.Join(outputDir, utils.ToSafeFilename(galleryInfo.Title))
	fmt.Println(baseDir)

	//FIXME:处理此逻辑不应该通过检测数量的方法
	//应该是先检查连续性，再从最后断开的地方开始下载
	if utils.FileExists(filepath.Join(baseDir, infoJsonPath)) {
		fmt.Println("发现下载记录")
		//获取已经下载的图片数量
		downloadedImageCount := utils.GetFileTotal(baseDir, []string{".jpg", ".png"})
		fmt.Println("Downloaded image count:", downloadedImageCount)
		//计算剩余图片数量
		remainImageCount := galleryInfo.TotalImage - downloadedImageCount
		if remainImageCount == 0 {
			fmt.Println("本gallery已经下载完毕")
			return nil
		} else if remainImageCount < 0 {
			return fmt.Errorf("下载记录有误！")
		} else {
			fmt.Println("剩余图片数量:", remainImageCount)
			beginIndex = int(math.Floor(float64(downloadedImageCount) / float64(imageInOnePage)))
			remainder = downloadedImageCount - imageInOnePage*beginIndex
		}
	} else {
		//生成缓存文件
		err := utils.BuildCache(baseDir, infoJsonPath, galleryInfo)
		if err != nil {
			return err
		}
	}

	if onlyInfo {
		fmt.Println("画廊信息获取完毕，程序自动退出。")
		return nil
	}
	sumPage := int(math.Ceil(float64(galleryInfo.TotalImage) / float64(imageInOnePage)))
	for i := beginIndex; i < sumPage; i++ {
		fmt.Println("\nCurrent index:", i)
		indexUrl := generateIndexURL(galleryUrl, i)
		log.Printf("Current index url: %s", indexUrl)
		imagePageUrlList := getImagePageUrlList(c, indexUrl)
		if i == beginIndex {
			//如果是第一次处理目录，需要去掉前面的余数
			imagePageUrlList = imagePageUrlList[remainder:]
		}

		// Use a buffered channel as a semaphore to limit the number of goroutines running simultaneously
		semaphore := make(chan struct{}, utils.Parallelism)
		var wg sync.WaitGroup
		for _, imagePageUrl := range imagePageUrlList {
			wg.Add(1)
			// Acquire a semaphore slot before starting the goroutine
			semaphore <- struct{}{}
			go func(imagePageUrl string) {
				defer wg.Done()
				defer func() { <-semaphore }()
				imageTitle, imageUrl := getImageInfoFromPage(c, imagePageUrl)
				imageInfo := utils.ImageInfo{
					Title: imageTitle,
					Url:   imageUrl,
				}
				SaveImageWithRequest(c, buildJPEGRequestHeaders(), imageInfo, baseDir)
			}(imagePageUrl)

			//防止被ban，每保存一篇目录中的所有图片就sleep 1-3 seconds
			sleepTime := rand.Float64()*1 + 2
			log.Println("Sleep ", cast.ToString(sleepTime), " seconds...")
			time.Sleep(time.Duration(sleepTime) * time.Second)
		}

		// Wait for all goroutines to complete
		wg.Wait()

	}

	success, err := utils.CheckSequentialFileNames(baseDir, galleryInfo.TotalImage)
	if err != nil {
		return err
	}
	if success {
		fmt.Println("图片下载完毕")
	} else {
		fmt.Println("图片下载完毕，但是图片文件名不连续，自行查找问题")
	}
	return nil
}
