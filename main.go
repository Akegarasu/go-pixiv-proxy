package main

import (
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

var (
	host    string
	port    string
	headers = map[string]string{
		"Referer":    "https://www.pixiv.net",
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/81.0.4044.113 Safari/537.36",
	}
	client = &http.Client{
		Transport: &http.Transport{
			Proxy: func(request *http.Request) (u *url.URL, e error) {
				return http.ProxyFromEnvironment(request)
			},
		},
	}
	directTypes = []string{"img-original", "img-master", "c"}
	imgTypes    = []string{"original", "regular", "small", "thumb", "mini"}
	helpMsg     = `Usage:

1. http://example.com/$path
   - http://example.com/img-original/img/0000/00/00/00/00/00/12345678_p0.png

2. http://example.com/$pid[/$p][?img_type=original|regular|small|thumb|mini]
   - http://example.com/12345678    (p0)
   - http://example.com/12345678/0  (p0)
   - http://example.com/12345678/1  (p1)
   - http://example.com/12345678?t=small (small image)`
)

type Illust struct {
	origUrl string
	urls    map[string]gjson.Result
}

func httpGetReadCloser(u string) (io.ReadCloser, error) {
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

func handlePixivProxy(c *gin.Context) {
	var err error
	var realUrl string
	param := c.Param("pixiv")
	if param == "/" {
		c.String(200, helpMsg)
		return
	}
	imgType := c.DefaultQuery("t", "original")
	splitQuery := strings.Split(param, "/")[1:]
	if !in(imgTypes, imgType) {
		c.String(400, "invalid query")
		return
	}
	if in(directTypes, splitQuery[0]) {
		realUrl = "https://i.pximg.net/" + param[1:]
	} else {
		if _, err = strconv.Atoi(splitQuery[0]); err != nil {
			c.String(400, "invalid query")
			return
		} else {
			illust, err := getIllust(splitQuery[0])
			if err != nil {
				c.String(400, "pixiv api error")
				return
			}
			if r, ok := illust.urls[imgType]; ok {
				realUrl = r.String()
			} else {
				c.String(400, "this image type not exists")
				return
			}
		}
	}
	rd, err := httpGetReadCloser(realUrl)
	if err != nil {
		c.String(400, "get pixiv image error")
		return
	}
	defer rd.Close()
	_, _ = io.Copy(c.Writer, rd)
}

func handleIllustInfo(c *gin.Context) {
	pid := c.Param("pid")
	if _, err := strconv.Atoi(pid[1:]); err != nil {
		c.String(400, "pid invalid")
		return
	}
	rd, err := httpGetReadCloser("https://www.pixiv.net/ajax/illust/" + pid[1:])
	if err != nil {
		c.String(400, "get pixiv image error")
		return
	}
	defer rd.Close()
	_, _ = io.Copy(c.Writer, rd)
}

func in(orig []string, str string) bool {
	for _, b := range orig {
		if b == str {
			return true
		}
	}
	return false
}

func getBytes(url string) ([]byte, error) {
	body, err := httpGetReadCloser(url)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	b, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func getIllust(pid string) (*Illust, error) {
	b, err := getBytes("https://www.pixiv.net/ajax/illust/" + pid)
	if err != nil {
		return nil, err
	}
	g := gjson.ParseBytes(b)
	imgUrl := g.Get("body.urls.original").String()
	return &Illust{
		origUrl: imgUrl,
		urls:    g.Get("body.urls").Map(),
	}, nil
}

func init() {
	flag.StringVar(&host, "h", "127.0.0.1", "host")
	flag.StringVar(&port, "p", "18090", "port")
}

func main() {
	flag.Parse()
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	//r.GET("/json/*pid", handleIllustInfo)
	r.GET("/*pixiv", handlePixivProxy)
	err := r.Run(fmt.Sprintf("%s:%s", host, port))
	if err != nil {
		log.Error("start failed: ", err)
	}
}
