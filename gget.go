package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
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

type Progbar struct {
	total int
}

func (p *Progbar) PrintProg(portion int) {
	bars := p.calcBars(portion) //算长度
	spaces := maxbars - bars - 1
	percent := 100 * (float32(portion) / float32(p.total))
	builder := strings.Builder{}
	for i := 0; i < bars; i++ {
		builder.WriteRune('=')
	}
	builder.WriteRune('>')
	for i := 0; i <= spaces; i++ {
		builder.WriteRune(' ')
	}
	//fmt.Sprintf()
	fmt.Printf(" \r[%s] %3.2f%% (%d/%d)", builder.String(), percent, portion, p.total)
}

func (p *Progbar) PrintComplete() {
	p.PrintProg(p.total)
	fmt.Print("\n")
}

func (p *Progbar) calcBars(portion int) int {
	if portion == 0 {
		return portion
	}
	return int(float32(maxbars) / (float32(p.total) / float32(portion)))
}

func DownloadFile(filepath string, url string) {
	// Get the data
	resp, err := http.Get(url)
	check(err)
	defer resp.Body.Close()
	// Create the file
	out, err := os.Create(filepath)
	check(err)
	defer out.Close()
	size := resp.ContentLength //获取文件长度
	bar := &Progbar{total: int(size)}
	written := make(chan int, 500)
	go func() {
		copied := 0
		c := 0
		tick := time.Tick(interval) //定时发送tick tock
		for {
			select {
			case c = <-written: //如果收到已写入的数据，就加到copied去
				copied += c
			case <-tick:
				bar.PrintProg(copied) //如果收到tick tock，就打印状态
			}
		}
	}()
	buf := make([]byte, 32*1024) //32kb的buf
	for {
		rc, re := resp.Body.Read(buf)
		if rc > 0 {
			wc, we := out.Write(buf[0:rc])
			check(we)
			if wc != rc {
				log.Fatal("Read and Write count mismatch")
			}
			if wc > 0 {
				written <- wc
			}
		}
		if re == io.EOF {
			break
		}
	}
	bar.PrintComplete()
	fmt.Println("Download complete")
}

const (
	maxbars  int = 100
	interval     = 500 * time.Millisecond
)

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {

	//DownloadFile("10mb.bin", "http://mirror.sg.leaseweb.net/speedtest/10mb.bin")
	//return

	var repo string
	var match string
	flag.StringVar(&repo, "r", "", "github repo to download")
	flag.StringVar(&match, "n", "", "text use to filter")
	flag.Parse()
	request, err := http.NewRequest("GET", "https://api.github.com/repos/"+repo+"/releases/latest", nil)
	check(err)
	request.SetBasicAuth("XGFan", "245d7b7432200011da900baa85fa389d2f8ab9a8")
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
	} else if len(filter) == 0 {
		fmt.Println("There is no artifact match")
		PrintAssets(i.Assets)
	} else {
		fmt.Print("\n\r")
		//进入下载
		DownloadFile(filter[0].Name, filter[0].DownloadUrl)
	}
}

func PrintAssets(assets []asset) {
	fmt.Printf("%-40s%-12s\t%s\n", "Name", "Size", "Url")
	for _, e := range assets {
		fmt.Printf("%-40s%-12d\t%s\n", e.Name, e.Size, e.DownloadUrl)
	}
}
