package twitter

import (
	"fmt"
	"github.com/cnxysoft/DDBOT-WSa/lsp/mmsg"
	"github.com/cnxysoft/DDBOT-WSa/proxy_pool"
	"github.com/cnxysoft/DDBOT-WSa/requests"
	localutils "github.com/cnxysoft/DDBOT-WSa/utils"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strings"
	"time"
)

type NewNotify struct {
	groupCode int64
	*NewsInfo
}

func (n *NewNotify) GetGroupCode() int64 {
	return n.groupCode
}

func (n *NewNotify) ToMessage() *mmsg.MSG {
	defer func() {
		if err := recover(); err != nil {
			logger.WithField("stack", string(debug.Stack())).
				WithField("tweet", n.Tweet).
				Errorf("concern notify recoverd %v", err)
		}
	}()
	// 构造消息
	message := mmsg.NewMSG()
	if n.Tweet.ID == "" {
		return message
	}
	var CreatedAt time.Time
	if n.Tweet.RtType() == RETWEET {
		CreatedAt = time.Now().UTC()
		message.Textf(fmt.Sprintf("X-%s转发了%s的推文：\n",
			n.Name, n.Tweet.OrgUser.Name))
	} else {
		CreatedAt = n.Tweet.CreatedAt
		message.Textf(fmt.Sprintf("X-%s发布了新推文：\n", n.Name))
	}
	message.Text(CSTTime(CreatedAt).Format(time.DateTime) + "\n")
	// msg加入推文
	if n.Tweet.Content != "" {
		content := n.Tweet.Content
		if n.Tweet.Media != nil || content[len(content)-1] != '\n' {
			content += "\n"
		}
		message.Text(content)
	}
	var addedUrl bool
	// msg加入媒体
	addMedia(n.Tweet, message, true, &addedUrl)
	// msg加入被引用推文
	if QuoteTweet := n.Tweet.QuoteTweet; QuoteTweet != nil {
		var CreatedAt time.Time
		quoteTxt := "\n%v引用了%v的推文：\n"
		CreatedAt = QuoteTweet.CreatedAt
		// 检查是否需要插入cut
		addCut(message, &quoteTxt)
		message.Textf(fmt.Sprintf(quoteTxt, n.Tweet.OrgUser.Name, QuoteTweet.OrgUser.Name))
		message.Text(CSTTime(CreatedAt).Format(time.DateTime) + "\n")
		// msg加入推文
		if QuoteTweet.Content != "" {
			message.Text(QuoteTweet.Content + "\n")
		}
		// msg加入媒体
		addMedia(QuoteTweet, message, false, &addedUrl)
	}
	addTweetUrl(message, n.Tweet.Url, &addedUrl)
	return message
}

func addMedia(tweet *Tweet, message *mmsg.MSG, mainTweet bool, addedUrl *bool) {
	for _, m := range tweet.Media {
		unescape := m.Url
		if strings.HasPrefix(unescape, "/") {
			Url, err := setMirrorHost(tweet.MirrorHost, *m)
			if err != nil {
				logger.WithField("stack", string(debug.Stack())).
					WithField("tweetId", tweet.ID).
					Errorf("concern notify recoverd %v", err)
				continue
			}
			if Url.Hostname() != "" {
				if Url.Hostname() == XImgHost || Url.Hostname() == XVideoHost {
					unescape, err = processMediaURL(m.Url)
					if err != nil {
						logger.WithField("stack", string(debug.Stack())).
							WithField("tweetId", tweet.ID).
							Errorf("concern notify recoverd: %v", err)
						continue
					}
				}
				switch m.Type {
				case "image":
					if tweet.MirrorHost == XImgHost {
						unescape = strings.TrimLeft(unescape, "/pic/")
					}
					fullURL, err := Url.Parse(unescape)
					if err != nil {
						logger.WithField("stack", string(debug.Stack())).
							WithField("tweetId", tweet.ID).
							Errorf("concern notify recoverd %v", err)
					}
					m.Url = fullURL.String()
					addCut(message, nil)
					message.Append(
						mmsg.NewImageByUrl(m.Url,
							requests.ProxyOption(proxy_pool.PreferOversea),
							requests.AddUAOption(UserAgent),
							requests.WithCookieJar(Cookie)))
				case "video", "gif":
					if strings.Contains(unescape, "video.twimg.com") {
						idx := strings.Index(unescape, "video.twimg.com")
						unescape, err = processMediaURL(unescape[idx:])
						if err != nil {
							logger.WithField("stack", string(debug.Stack())).
								WithField("tweetId", tweet.ID).
								Errorf("concern notify recoverd: %v", err)
							continue
						}
						m.Url = unescape
					}
					if mainTweet {
						addTweetUrl(message, tweet.Url, addedUrl)
					}
					message.Cut()
					message.Append(
						mmsg.NewVideoByUrl(m.Url,
							requests.ProxyOption(proxy_pool.PreferOversea),
							requests.AddUAOption(UserAgent),
							requests.WithCookieJar(Cookie)))
				case "video(m3u8)":
					var fullURL *url.URL
					var err error
					if tweet.MirrorHost == XVideoHost {
						idx := findNthIndex(unescape, '/', 3)
						if idx != -1 {
							unescape = unescape[idx+1:]
						}
					} else if strings.Contains(unescape, "https%3A%2F%2Fvideo.twimg.com") {
						idx := strings.Index(unescape, "https%3A%2F%2F")
						unescape, err = processMediaURL(unescape[idx:])
						if err != nil {
							logger.WithField("stack", string(debug.Stack())).
								WithField("tweetId", tweet.ID).
								Errorf("concern notify recoverd: %v", err)
							continue
						}
						idx = findNthIndex(unescape, '?', 1)
						if idx != -1 {
							unescape = unescape[:idx]
						}
						m.Url = unescape
					} else {
						fullURL, err = Url.Parse(unescape)
						if err != nil {
							logger.WithField("stack", string(debug.Stack())).
								WithField("tweetId", tweet.ID).
								Errorf("concern notify recoverd %v", err)
						}
						m.Url = fullURL.String()
					}
					var proxyStr string
					proxy, err := proxy_pool.Get(proxy_pool.PreferOversea)
					if err != nil {
						logger.WithField("stack", string(debug.Stack())).
							WithField("tweetId", tweet.ID).
							Warnf("concern notify recoverd: proxy setting failed: %v", err)
					} else {
						proxyStr = proxy.ProxyString()
					}
					if _, err = os.Stat("./res"); os.IsNotExist(err) {
						if err = os.MkdirAll("./res", 0755); err != nil {
							logger.Error("创建下载目录失败")
							continue
						}
					}
					filePath, _ := filepath.Abs("./res/" + uuid.New().String() + ".mp4")
					err = convertWithProxy(m.Url, filePath, proxyStr)
					if err != nil {
						logger.WithField("stack", string(debug.Stack())).
							WithField("tweetId", tweet.ID).
							Errorf("concern notify recoverd: convertWithProxy failed: %v", err)
						continue
					}
					if mainTweet {
						addTweetUrl(message, tweet.Url, addedUrl)
					}
					message.Cut()
					message.Append(mmsg.NewVideoByLocal(filePath))
					go func(path string) {
						time.Sleep(time.Second * 128)
						os.Remove(path)
					}(filePath)
				}
			}
		}
	}
}

func addCut(msg *mmsg.MSG, quo *string) {
	ele := msg.Elements()
	if ele[len(ele)-1].Type() == mmsg.Video {
		msg.Cut()
		if quo != nil {
			*quo = strings.TrimPrefix(*quo, "\n")
		}
	}
}

func addTweetUrl(msg *mmsg.MSG, url string, added *bool) {
	if !*added {
		*added = true
		msg.Text(url + "\n")
	}
}

func convertWithProxy(m3u8URL, outputPath, proxyURL string) error {
	cmd := exec.Command("ffmpeg",
		"-v", "error",
		"-i", m3u8URL,
		"-c", "copy",
		"-f", "mp4",
		outputPath)
	if proxyURL != "" {
		cmd.Env = append(os.Environ(), "http_proxy="+proxyURL, "https_proxy="+proxyURL, "rw_timeout=30000000")
	}

	cmd.Stdout = nil
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func findNthIndex(s string, sep byte, n int) int {
	count := 0
	for i := range s {
		if s[i] == sep {
			count++
			if count == n {
				return i
			}
		}
	}
	return -1
}

func setMirrorHost(mirrorHost string, m Media) (url.URL, error) {
	if mirrorHost == "" || mirrorHost == XImgHost || mirrorHost == XVideoHost {
		logger.WithField("mediaUrl", m.Url).
			Trace("No MirrorHost was found, using the default Host of X.")
		if m.Type == "image" {
			mirrorHost = XImgHost
		} else {
			mirrorHost = XVideoHost
		}
	}
	Url := url.URL{
		Scheme: "https",
		Host:   mirrorHost,
	}
	return Url, nil
}

func (n *NewNotify) Logger() *logrus.Entry {
	return n.NewsInfo.Logger().WithFields(localutils.GroupLogFields(n.groupCode))
}

// 检测是否包含URI编码特征
func isURIEncoded(s string) bool {
	// 匹配URI编码特征（%后跟两个十六进制字符）
	re := regexp.MustCompile(`%(?i)[0-9a-f]{2}`)
	return re.MatchString(s)
}

// 处理Twitter媒体URL
func processMediaURL(encodedURL string) (string, error) {
	// 判断是否需要解码
	if !isURIEncoded(encodedURL) {
		return encodedURL, nil
	}

	// 解除所有层级编码
	decodedURL, err := safeDecodeURIComponent(encodedURL)
	if err != nil {
		return "", fmt.Errorf("多级URI解码失败: %v", err)
	}

	return decodedURL, nil
}

// 安全的URI解码器
func safeDecodeURIComponent(s string) (string, error) {
	maxIterations := 10
	decoded := s
	for i := 0; i < maxIterations; i++ {
		nextDecoded, err := url.QueryUnescape(decoded)
		if err != nil {
			return decoded, err
		}
		if nextDecoded == decoded {
			break
		}
		decoded = nextDecoded
	}
	return decoded, nil
}
