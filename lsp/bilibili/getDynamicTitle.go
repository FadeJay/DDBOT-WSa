package bilibili

import (
	"bytes"
	"github.com/PuerkitoBio/goquery"
	"github.com/cnxysoft/DDBOT-WSa/proxy_pool"
	"github.com/cnxysoft/DDBOT-WSa/requests"
)

func reqDynamicPage(DynamicId string) string {
	Url := DynamicUrl(DynamicId)
	opt := []requests.Option{
		requests.AddUAOption(),
		requests.ProxyOption(proxy_pool.PreferNone),
		requests.RetryOption(3),
		requests.RequestAutoHostOption(),
	}
	var resp bytes.Buffer
	requests.Get(Url, nil, &resp, opt...)
	return getDynamicTitle(resp.Bytes())
}

func getDynamicTitle(data []byte) string {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(data))
	if err != nil {
		logger.Errorf("get dynamic title err: %v", err)
		return ""
	}
	title := doc.Find(".opus-module-title__text").Text()
	if title == "" {
		logger.Errorf("get dynamic title err: %v", err)
		return ""
	}
	return title
}
