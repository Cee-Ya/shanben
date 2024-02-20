package main

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const downloadDir = "/volume1/other/book"           // 下载目录
const baseUrl = "http://shanben.ioc.u-tokyo.ac.jp/" // 基础url
const syb = "/"                                     // 如果是windows系统, 请使用"\\" 如果是linux系统, 请使用"/"

// 爬取http://shanben.ioc.u-tokyo.ac.jp/list.php的古籍到指定目录
func main() {
	fmt.Println("开始爬取古籍")
	starTime := time.Now()
	// 爬取地址
	url := baseUrl + "list.php"
	// 使用goquery爬取
	download(url)
	fmt.Println(fmt.Sprintf("爬取完成, 耗时: %s 分钟", time.Now().Sub(starTime).Minutes()))
}

func buildRequest(url string, method string, body io.Reader) *http.Request {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		panic(err)
	}
	// 增加超时时间
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Cache-Control", "max-age=0")
	if method == "POST" {
		req.Header.Set("content-type", "application/json;charset=UTF-8")
		req.Header.Set("accept", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "PHPSESSID", Value: "v2cf91f30jkufcjr5dkqpd9co5"})
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/92.0.4515.159 Safari/537.36")
	req.Header.Set("If-None-Match", "5a309bab-26057")
	req.Header.Set("Referer", baseUrl)
	req.Header.Set("origin", baseUrl)
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-fetch-site", "same-site")
	return req
}

func get(url string) *http.Response {
	http.DefaultClient.Timeout = 30 * time.Second
	req := buildRequest(url, "GET", nil)
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	return response
}

func download(url string) {
	response := get(url)
	if response.StatusCode != 200 {
		panic(fmt.Errorf("download failed, status code: %d", response.StatusCode))
	}
	defer response.Body.Close()
	rs, _ := goquery.NewDocumentFromReader(response.Body)
	// 开始获取下一页链接 <img alt="下一頁"/>
	nextUrl := ""
	rs.Find("img[alt='下一頁']").Each(func(i int, selection *goquery.Selection) {
		// 父级的a标签href
		if href, ok := selection.Parent().Attr("href"); ok {
			nextUrl = baseUrl + href
		}
	})
	// 开始解析主体内容
	rs.Find("tbody").Each(func(i int, selection *goquery.Selection) {
		//只需要第三个
		if i == 2 {
			selection.Find("tr").Each(func(i int, selection *goquery.Selection) {
				// 跳过标题
				if i > 0 {
					// 索引号
					index := selection.Find("td").Eq(3).Text()
					// 分类
					tp := selection.Find("td").Eq(2).Text()
					// 书名
					name := selection.Find("td").Eq(1).Text()
					// 索引号
					if !IsExistFolder(downloadDir + syb + index) {
						createPath(index, downloadDir)
					}
					if !IsExistFolder(downloadDir + syb + index + syb + tp) {
						createPath(tp, downloadDir+syb+index)
					}
					if !IsExistFolder(downloadDir + syb + index + syb + tp + syb + name) {
						createPath(name, downloadDir+syb+index+syb+tp)
					}

					// 获取链接
					href, _ := selection.Find("a").Attr("href")
					// 获取标题
					title := selection.Find("a").Text()
					// 打开链接
					fmt.Println(fmt.Sprintf("打开链接: %s", title))
					response2 := get(baseUrl + href)
					if response2.StatusCode != 200 {
						panic(fmt.Errorf("打开链接%s失败, 状态码: %d", baseUrl+href, response2.StatusCode))
					}
					// 解析
					rsChild, err := goquery.NewDocumentFromReader(response2.Body)
					if err != nil {
						panic(fmt.Errorf("解析链接%s失败: %s", baseUrl+href, err))
					}
					// 获取下载链接
					rsChild.Find("#tree_body div span[width='343']").Each(func(i int, selection *goquery.Selection) {
						if hr, ok := selection.Find("a").Eq(1).Attr("href"); ok {
							fmt.Println(fmt.Sprintf("开始下载: %s ======>>", hr))
							// 判断文件是否存在
							if isExist(downloadDir + syb + index + syb + tp + syb + name + syb + hr[strings.LastIndex(hr, "/")+1:]) {
								fmt.Println(fmt.Sprintf("文件已存在: %s <<======", hr))
								return
							}
							downLoadImageToPath(baseUrl+hr, downloadDir+syb+index+syb+tp+syb+name)
							fmt.Println(fmt.Sprintf("下载完成: %s <<======", hr))
						}
					})
					response2.Body.Close()
					fmt.Println(fmt.Sprintf("关闭链接完成: %s", title))
				}
			})
		}
	})
	// 下一页继续爬取
	if nextUrl != "" {
		download(nextUrl)
	}
}

// IsExistFolder 是否存在文件夹
func IsExistFolder(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

// 创建文件夹
func createPath(path string, rootPath string) {
	if !IsExistFolder(rootPath) {
		err := os.MkdirAll(rootPath, os.ModePerm)
		if err != nil {
			return
		}
	}
	if !IsExistFolder(rootPath + syb + path) {
		err := os.MkdirAll(rootPath+syb+path, os.ModePerm)
		if err != nil {
			return
		}
	}
	fmt.Println(fmt.Sprintf("创建文件夹: %s", rootPath+syb+path))
}

// 判断该文件是否存在
func isExist(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

// 下载文件到指定目录
func downLoadImageToPath(url string, path string) {
	response := get(url)
	defer response.Body.Close()
	file, err := os.Create(path + syb + url[strings.LastIndex(url, "/")+1:])
	if err != nil {
		panic(err)
	}
	_, err = io.Copy(file, response.Body)
	if err != nil {
		panic(err)
	}
}
