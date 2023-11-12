package main

import (
	"io"
	"net/http"
	"net/url"

	log "github.com/sirupsen/logrus"
)

var (
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
)

type Context struct {
	rw  http.ResponseWriter
	req *http.Request
}

func (c *Context) write(b []byte, status int) {
	c.rw.WriteHeader(status)
	_, err := c.rw.Write(b)
	if err != nil {
		log.Error(err)
	}
}

func (c *Context) String(status int, s string) {
	c.write([]byte(s), status)
}

func (c *Context) WriteHeader(statusCode int) {
	c.rw.WriteHeader(statusCode)
}

func proxyHttpReq(c *Context, url string, errMsg string) {
	resp, err := httpGet(url)
	if err != nil {
		c.String(500, errMsg)
		return
	}
	defer resp.Body.Close()
	copyHeader(c.rw.Header(), resp.Header)
	_, _ = io.Copy(c.rw, resp.Body)
}

func httpGet(u string) (*http.Response, error) {
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("Cookie", cookies)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func httpGetReadCloser(u string) (io.ReadCloser, error) {
	resp, err := httpGet(u)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

func httpGetBytes(url string) ([]byte, error) {
	body, err := httpGetReadCloser(url)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	b, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}
