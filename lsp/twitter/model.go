package twitter

import (
	"github.com/cnxysoft/DDBOT-WSa/lsp/concern_type"
	"github.com/sirupsen/logrus"
	"strconv"
	"strings"
	"time"
)

// 这里面可以定义推送中使用的结构

type NewsInfo struct {
	*UserInfo
	Tweet *Tweet
}

func (e *NewsInfo) Site() string {
	return Site
}

func (e *NewsInfo) Type() concern_type.Type {
	return Tweets
}

func (e *NewsInfo) GetUid() interface{} {
	return e.UserInfo.Id
}

func (e *NewsInfo) Logger() *logrus.Entry {
	return logger.WithFields(logrus.Fields{
		"Site": e.Site(),
		"Type": e.Type(),
		"Uid":  e.GetUid(),
		"Name": e.UserInfo.Name,
	})
}

type LatestTweetIds struct {
	TweetId  []string
	PinnedId string
}

func (l *LatestTweetIds) GetLatestTweetTs() *time.Time {
	if len(l.TweetId) == 0 {
		return nil
	}
	ts, err := ParseSnowflakeTimestamp(l.TweetId[0])
	if err != nil {
		logger.WithError(err).Error("ParseSnowflakeTimestamp")
		return &time.Time{}
	}
	return &ts
}

func (l *LatestTweetIds) GetLatestTweetId() string {
	if len(l.TweetId) == 0 {
		return ""
	}
	return l.TweetId[len(l.TweetId)-1]
}

func (l *LatestTweetIds) SetLatestTweetId(tweetId string) {
	if l.ExistTweetId(tweetId) {
		return
	}
	maxIds := 20
	if len(l.TweetId) < maxIds {
		l.TweetId = append(l.TweetId, tweetId)
	} else if len(l.TweetId) >= maxIds {
		l.TweetId = l.TweetId[:len(l.TweetId)-1]
	}
}

func (l *LatestTweetIds) ExistTweetId(tweetId string) bool {
	for _, TweetId := range l.TweetId {
		if TweetId == tweetId {
			return true
		}
	}
	return false
}

func (l *LatestTweetIds) SetPinnedTweet(tweetId string) bool {
	if tweetId == "" {
		logger.Debug("SetPinnedTweet failed: Empty tweetId")
		return false
	}
	l.PinnedId = tweetId
	return true
}

func (l *LatestTweetIds) GetPinnedTweet() string {
	return l.PinnedId
}

type UserInfo struct {
	Id   string
	Name string
}

func (u *UserInfo) GetUid() interface{} {
	return u.Id
}

func (u *UserInfo) GetName() string {
	return u.Name
}

const (
	TWEET   = 1
	RETWEET = 2
)

type TweetItem struct {
	Type        int
	Title       string
	Description string
	Link        string
	Media       []string
	Published   time.Time
	Author      *UserInfo
}

func (e *TweetItem) GetId() string {
	return ExtractTweetID(e.Link)
}

func ExtractTweetID(url string) string {
	// 分割URL路径
	parts := strings.Split(url, "/")
	for i, part := range parts {
		if part == "status" && i+1 < len(parts) {
			// 去除可能的锚点或参数
			if hashIndex := strings.Index(parts[i+1], "#"); hashIndex != -1 {
				return parts[i+1][:hashIndex]
			}
			return parts[i+1]
		}
	}
	return ""
}

// 雪花ID解析参数（需与生成器配置保持一致）
const (
	epoch              = int64(1288834974657)                           // 起始时间戳（毫秒）
	datacenterIdBits   = uint(5)                                        // 数据中心位数
	workerIdBits       = uint(5)                                        // 工作节点位数
	sequenceBits       = uint(12)                                       // 序列号位数
	timestampLeftShift = datacenterIdBits + workerIdBits + sequenceBits // 时间戳偏移量
)

// ParseSnowflakeTimestamp 解析雪花ID中的时间戳
func ParseSnowflakeTimestamp(id string) (time.Time, error) {
	Id, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	timestamp := (Id >> timestampLeftShift) + epoch
	return time.UnixMilli(timestamp), nil
}
