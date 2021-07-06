package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"gget/progress"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type asset struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	Size        int    `json:"size"`
	DownloadUrl string `json:"browser_download_url"`
}

type releases struct {
	Url     string  `json:"url"`
	Name    string  `json:"name"`
	Comment string  `json:"body"`
	Assets  []asset `json:"assets"`
}

func Map(vs []string, f func(string) string) []string {
	vsm := make([]string, len(vs))
	for i, v := range vs {
		vsm[i] = f(v)
	}
	return vsm
}
func DownloadFile(filepath string, url string, threads int) {
	defer timeTrack(time.Now(), "download")
	// Get the data
	resp, err := http.Head(url)
	check(err)
	// Create the file
	out, err := os.Create(filepath)
	check(err)
	defer out.Close()
	size := resp.ContentLength //获取文件长度
	_ = resp.Body.Close()
	//log.Printf("%+v\n", resp.Header)
	bar := &progress.Progressbar{Total: int(size)}
	bar.Run()
	defer bar.Print()
	wg := sync.WaitGroup{}
	wg.Add(threads)
	var mu sync.Mutex
	var skip = size / int64(threads)
	for i := 1; i <= threads; i++ {
		from := int64(i-1) * skip
		var to int64 = 0
		if i < threads {
			to = from + skip - 1
		} else {
			to = size - 1
		}
		go func() {
			downResp, _ := http.DefaultClient.Do(&http.Request{
				URL: resp.Request.URL,
				Header: map[string][]string{
					"Range": {fmt.Sprintf("bytes=%d-%d", from, to)},
				},
			})
			defer downResp.Body.Close()
			buf := make([]byte, 32*1024) //32kb的buf
			var writeCount int64 = 0
			for {
				rc, re := downResp.Body.Read(buf)
				if rc > 0 {
					mu.Lock()
					out.Seek(writeCount+from, 0)
					wc, we := out.Write(buf[0:rc])
					check(we)
					writeCount += int64(wc)
					mu.Unlock()
					if wc != rc {
						log.Fatal("Read and Write count mismatch")
					}
					if wc > 0 {
						bar.Add(wc)
					}
				}
				if re == io.EOF {
					break
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

var user = "Your Github Username"
var token = "Your Token"

func main() {
	var repo string
	var match string
	flag.StringVar(&repo, "r", "", "github repo to download")
	flag.StringVar(&match, "n", "", "text use to filter")
	flag.Parse()
	request, err := http.NewRequest("GET", "https://api.github.com/repos/"+repo+"/releases/latest", nil)
	check(err)
	request.SetBasicAuth(user, token)
	resp, err := http.DefaultClient.Do(request)
	check(err)
	bytes, err := ioutil.ReadAll(resp.Body)
	check(err)
	var i releases
	err = json.Unmarshal(bytes, &i)
	log.Printf("Latest release: %s \nNote:\n%s\n\n", i.Name, i.Comment)
	var filter []asset
	for _, e := range i.Assets {
		if strings.Index(e.Name, match) != -1 {
			filter = append(filter, e)
		}
	}
	if len(filter) > 1 {
		fmt.Println("There is more than one artifact match")
		PrintAssets(filter)
		os.Exit(1)
	} else if len(filter) == 0 {
		fmt.Println("There is no artifact match")
		PrintAssets(i.Assets)
		os.Exit(1)
	} else {
		fmt.Print("\n\r")
		//进入下载
		DownloadFile(filter[0].Name, filter[0].DownloadUrl, 4)
	}
}

func PrintAssets(assets []asset) {
	fmt.Printf("%-40s%-12s\t%s\n", "Name", "Size", "Url")
	for _, e := range assets {
		fmt.Printf("%-40s%-12d\t%s\n", e.Name, e.Size, e.DownloadUrl)
	}
}

func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("%s took %s", name, elapsed)
}
