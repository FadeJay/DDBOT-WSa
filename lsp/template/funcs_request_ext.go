package template

import (
	"bytes"
	"github.com/cnxysoft/DDBOT-WSa/proxy_pool"
	"github.com/cnxysoft/DDBOT-WSa/requests"
	"github.com/google/uuid"
	"github.com/spf13/cast"
	"net/url"
	"os"
	"path"
	"reflect"
	"strings"
)

const (
	DDBOT_REQ_DEBUG      = "DDBOT_REQ_DEBUG"
	DDBOT_REQ_HEADER     = "DDBOT_REQ_HEADER"
	DDBOT_REQ_COOKIE     = "DDBOT_REQ_COOKIE"
	DDBOT_REQ_PROXY      = "DDBOT_REQ_PROXY"
	DDBOT_REQ_USER_AGENT = "DDBOT_REQ_USER_AGENT"
)

func preProcess(oParams []map[string]interface{}) (map[string]interface{}, []requests.Option) {
	var params map[string]interface{}
	if len(oParams) == 0 {
		return nil, nil
	} else if len(oParams) == 1 {
		params = oParams[0]
	} else {
		panic("given more than one params")
	}
	fn := func(key string, f func() []requests.Option) []requests.Option {
		var r []requests.Option

		if _, found := params[key]; found {
			r = f()
			delete(params, key)
		}
		return r
	}

	collectStringSlice := func(i interface{}) []string {
		v := reflect.ValueOf(i)
		if v.Kind() == reflect.String {
			return []string{v.String()}
		}
		return cast.ToStringSlice(i)
	}

	var item = []struct {
		key string
		f   func() []requests.Option
	}{
		{
			DDBOT_REQ_DEBUG,
			func() []requests.Option {
				return []requests.Option{requests.DebugOption()}
			},
		},
		{
			DDBOT_REQ_HEADER,
			func() []requests.Option {
				var result []requests.Option
				var header = collectStringSlice(params[DDBOT_REQ_HEADER])
				for _, h := range header {
					spt := strings.SplitN(h, "=", 2)
					if len(spt) >= 2 {
						result = append(result, requests.HeaderOption(spt[0], spt[1]))
					} else {
						logger.WithField("DDBOT_REQ_HEADER", h).Errorf("invalid header format")
					}
				}
				return result
			},
		},
		{
			DDBOT_REQ_COOKIE,
			func() []requests.Option {
				var result []requests.Option
				var cookie = collectStringSlice(params[DDBOT_REQ_COOKIE])
				for _, c := range cookie {
					spt := strings.SplitN(c, "=", 2)
					if len(spt) >= 2 {
						result = append(result, requests.CookieOption(spt[0], spt[1]))
					} else {
						logger.WithField("DDBOT_REQ_COOKIE", c).Errorf("invalid cookie format")
					}
				}
				return result
			},
		},
		{
			DDBOT_REQ_PROXY,
			func() []requests.Option {
				iproxy := params[DDBOT_REQ_PROXY]
				proxy, ok := iproxy.(string)
				if !ok {
					logger.WithField("DDBOT_REQ_PROXY", iproxy).Errorf("invalid proxy format")
					return nil
				}
				if proxy == "prefer_mainland" {
					return []requests.Option{requests.ProxyOption(proxy_pool.PreferMainland)}
				} else if proxy == "prefer_oversea" {
					return []requests.Option{requests.ProxyOption(proxy_pool.PreferOversea)}
				} else if proxy == "prefer_none" {
					return nil
				} else if proxy == "prefer_any" {
					return []requests.Option{requests.ProxyOption(proxy_pool.PreferAny)}
				} else {
					return []requests.Option{requests.RawProxyOption(proxy)}
				}
			},
		},
		{
			DDBOT_REQ_USER_AGENT,
			func() []requests.Option {
				iua := params[DDBOT_REQ_USER_AGENT]
				ua, ok := iua.(string)
				if !ok {
					logger.WithField("DDBOT_REQ_USER_AGENT", iua).Errorf("invalid ua format")
					return nil
				}
				return []requests.Option{requests.AddUAOption(ua)}
			},
		},
	}

	var result = []requests.Option{requests.AddUAOption()}
	for _, i := range item {
		result = append(result, fn(i.key, i.f)...)
	}
	return params, result
}

func httpGet(url string, oParams ...map[string]interface{}) (body []byte) {
	params, opts := preProcess(oParams)
	err := requests.Get(url, params, &body, opts...)
	if err != nil {
		logger.Errorf("template: httpGet error %v", err)
	}
	return
}

func httpPostJson(url string, oParams ...map[string]interface{}) (body []byte) {
	params, opts := preProcess(oParams)
	err := requests.PostJson(url, params, &body, opts...)
	if err != nil {
		logger.Errorf("template: httpGet error %v", err)
	}
	return
}

func httpPostForm(url string, oParams ...map[string]interface{}) (body []byte) {
	params, opts := preProcess(oParams)
	err := requests.PostForm(url, params, &body, opts...)
	if err != nil {
		logger.Errorf("template: httpGet error %v", err)
	}
	return
}

func downloadFile(inUrl string, loPath string, fileName string, oParams ...map[string]interface{}) string {
	// 声明变量
	var (
		Url        *url.URL
		err        error
		localPath  string
		resp       bytes.Buffer
		respHeader requests.RespHeader
	)
	params, opts := preProcess(oParams)
	// 检查URL
	if inUrl == "" {
		logger.Error("请提供URL进行下载")
		return ""
	} else {
		Url, err = url.Parse(inUrl)
		if err != nil {
			logger.Error("无效的URL")
			return ""
		}
	}
	// 设置下载路径
	if loPath == "" {
		logger.Trace("没有指定下载路径，将使用默认路径")
		localPath = "./downloads"
	} else {
		localPath = loPath
	}
	// 检查文件路径是否存在
	if _, err = os.Stat(localPath); os.IsNotExist(err) {
		if err = os.MkdirAll(localPath, 0755); err != nil {
			logger.Error("创建下载目录失败")
			return ""
		}
	}
	err = requests.GetWithHeader(Url.String(), params, &resp, &respHeader, opts...)
	if err != nil {
		logger.Error("下载文件失败")
		return ""
	}
	if fileName == "" {
		if respHeader.ContentDisposition != "" {
			fileName = respHeader.ContentDisposition
		} else {
			var vaild bool
			fileName, vaild = extractFilename(Url.String())
			if fileName == "" || !vaild {
				fileName = uuid.New().String()
			}
		}
	}
	filePath := localPath + "/" + fileName
	err = os.WriteFile(filePath, resp.Bytes(), 0644)
	if err != nil {
		logger.Error("保存文件失败")
		return ""
	}
	return filePath
}

// 提取文件名并验证有效性，返回 (文件名, 是否有效)
func extractFilename(urlStr string) (string, bool) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return "", false
	}

	cleanPath := path.Clean(u.Path)
	base := path.Base(cleanPath)

	// 验证文件名有效性（与判断逻辑一致）
	if base == "." || base == ".." || base == "/" || base == "" {
		return "", false
	}

	dotIndex := strings.LastIndex(base, ".")
	if dotIndex == -1 || dotIndex == 0 || dotIndex == len(base)-1 {
		return "", false
	}

	return base, true
}
