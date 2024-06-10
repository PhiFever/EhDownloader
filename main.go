package main

import (
	"EhDownloader/eh"
	"EhDownloader/utils"
	"fmt"
	"github.com/fatih/color"
	"github.com/spf13/cast"
	"github.com/urfave/cli/v2"
	"os"
	"regexp"
	"time"
)

const infoJsonPath = "galleryInfo.json"

var (
	onlyInfo        bool
	outputDir       string
	url             string
	listFilePath    string
	galleryUrlRegex = regexp.MustCompile(`^https://e-hentai.org/g/[a-z0-9]*/[a-z0-9]{10}/$`)
)

type GalleryDownloader struct {
	InfoJsonPath string
}

func (gd *GalleryDownloader) Download(outputDir string, url string, onlyInfo bool) error {
	if galleryUrlRegex.MatchString(url) {
		return eh.DownloadGallery(outputDir, gd.InfoJsonPath, url, onlyInfo)
	}
	return fmt.Errorf("未知的url格式：%s", url)
}

func getExecutionTime(startTime time.Time, endTime time.Time) string {
	//按时:分:秒格式输出
	duration := endTime.Sub(startTime)
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%d时%d分%d秒", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%d分%d秒", minutes, seconds)
	} else {
		return fmt.Sprintf("%d秒", seconds)
	}
}

func main() {
	//设置输出颜色
	successColor := color.New(color.Bold, color.FgGreen).FprintlnFunc()
	failColor := color.New(color.Bold, color.FgRed).FprintlnFunc()
	errCount := 0

	app := &cli.App{
		Name:      "EhDownloader",
		UsageText: "EhDownloader -u <url> | -l <file>",
		Version:   "0.9.1",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "info", Aliases: []string{"i"}, Destination: &onlyInfo, Usage: "只下载画廊信息"},
			&cli.StringFlag{Name: "url", Aliases: []string{"u"}, Destination: &url, Usage: "画廊网址"},
			&cli.StringFlag{Name: "list", Aliases: []string{"l"}, Destination: &listFilePath, Usage: "包含画廊网址的文件"},
			&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Destination: &outputDir, Value: "images", Usage: "输出目录"},
		},
		Action: func(c *cli.Context) error {
			var galleryUrlList []string
			if url != "" {
				galleryUrlList = append(galleryUrlList, url)
			} else if listFilePath != "" {
				var err error
				galleryUrlList, err = utils.ReadListFile(listFilePath)
				if err != nil {
					return err
				}
			}

			//记录开始时间
			startTime := time.Now()

			//创建下载器
			downloader := GalleryDownloader{InfoJsonPath: infoJsonPath}
			for _, u := range galleryUrlList {
				successColor(os.Stdout, "开始下载gallery:", u)
				err := downloader.Download(outputDir, u, onlyInfo)
				if err != nil {
					failColor(os.Stderr, "下载失败:", err, "\n")
					errCount++
				} else {
					successColor(os.Stdout, "gallery下载完毕:", u, "\n")
				}
			}

			//记录结束时间
			endTime := time.Now()
			//计算执行时间，单位为秒
			successColor(os.Stdout, "所有gallery下载完毕，共耗时:", getExecutionTime(startTime, endTime))
			if errCount > 0 {
				return fmt.Errorf("有" + cast.ToString(errCount) + "个下载失败")
			}

			return nil
		},
	}
	if err := app.Run(os.Args); err != nil {
		failColor(os.Stderr, err)
	}

}
