package main

import (
	_ "embed"
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
	"github.com/tidwall/gjson"
	"net/http"
	"os"
	"strconv"
	"strings"
)

var (
	host   string
	port   string
	domain string
	//go:embed index.html
	indexHtml     string
	directTypes   = []string{"img-original", "img-master", "c", "user-profile", "img-zip-ugoira"}
	imgTypes      = []string{"original", "regular", "small", "thumb", "mini"}
	docExampleImg = `![regular](http://example.com/98505703?t=regular)

![small](http://example.com/98505703?t=small)

![thumb](http://example.com/98505703?t=thumb)

![mini](http://example.com/98505703?t=mini)`
)

type Illust struct {
	origUrl string
	urls    map[string]gjson.Result
}

func handlePixivProxy(rw http.ResponseWriter, req *http.Request) {
	var err error
	var realUrl string
	c := &Context{
		rw:  rw,
		req: req,
	}
	path := req.URL.Path
	log.Info(req.Method, " ", req.URL.String())
	spl := strings.Split(path, "/")[1:]
	switch spl[0] {
	case "":
		c.String(200, indexHtml)
		return
	case "favicon.ico":
		c.WriteHeader(404)
		return
	case "api":
		handleIllustInfo(c)
		return
	}
	imgType := req.URL.Query().Get("t")
	if imgType == "" {
		imgType = "original"
	}
	if !in(imgTypes, imgType) {
		c.String(400, "invalid query")
		return
	}
	if in(directTypes, spl[0]) {
		realUrl = "https://i.pximg.net" + path
	} else {
		if _, err = strconv.Atoi(spl[0]); err != nil {
			c.String(400, "invalid query")
			return
		}
		illust, err := getIllust(spl[0])
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
		if len(spl) > 1 {
			realUrl = strings.Replace(realUrl, "_p0", "_p"+spl[1], 1)
		}
	}
	proxyHttpReq(c, realUrl, "fetch pixiv image error")
}

func handleIllustInfo(c *Context) {
	params := strings.Split(c.req.URL.Path, "/")
	pid := params[len(params)-1]
	if _, err := strconv.Atoi(pid); err != nil {
		c.String(400, "pid invalid")
		return
	}
	proxyHttpReq(c, "https://www.pixiv.net/ajax/illust/"+pid, "pixiv api error")
}

func getIllust(pid string) (*Illust, error) {
	b, err := httpGetBytes("https://www.pixiv.net/ajax/illust/" + pid)
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

func in(orig []string, str string) bool {
	for _, b := range orig {
		if b == str {
			return true
		}
	}
	return false
}

func checkEnv() {
	if os.Getenv("GPP_HOST") != "" {
		host = os.Getenv("GPP_HOST")
	}
	if os.Getenv("GPP_PORT") != "" {
		port = os.Getenv("GPP_PORT")
	}
	if os.Getenv("GPP_DOMAIN") != "" {
		domain = os.Getenv("GPP_DOMAIN")
	}
}

func init() {
	flag.StringVar(&host, "h", "127.0.0.1", "host")
	flag.StringVar(&port, "p", "18090", "port")
	flag.StringVar(&domain, "d", "", "your domain")
	log.SetFormatter(&easy.Formatter{
		TimestampFormat: "2006-01-02 15:04:05",
		LogFormat:       "[%lvl%][%time%]: %msg% \n",
	})
	log.SetLevel(log.InfoLevel)
}

func main() {
	flag.Parse()
	checkEnv()
	if domain != "" {
		indexHtml = strings.ReplaceAll(indexHtml, "{image-examples}", docExampleImg)
		indexHtml = strings.ReplaceAll(indexHtml, "http://example.com", domain)
	}
	http.HandleFunc("/", handlePixivProxy)
	log.Infof("started: %s:%s %s", host, port, domain)
	err := http.ListenAndServe(fmt.Sprintf("%s:%s", host, port), nil)
	if err != nil {
		log.Error("start failed: ", err)
	}
}
