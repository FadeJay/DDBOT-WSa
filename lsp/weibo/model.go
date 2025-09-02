package weibo

import (
	"html"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cnxysoft/DDBOT-WSa/lsp/concern_type"
	"github.com/cnxysoft/DDBOT-WSa/lsp/mmsg"
	localutils "github.com/cnxysoft/DDBOT-WSa/utils"
	"github.com/sirupsen/logrus"
)

const (
	News concern_type.Type = "news"
)

type UserInfo struct {
	Uid             int64  `json:"id"`
	Name            string `json:"screen_name"`
	ProfileImageUrl string `json:"profile_image_url"`
	ProfileUrl      string `json:"profile_url"`
}

func (u *UserInfo) Site() string {
	return Site
}

func (u *UserInfo) GetUid() interface{} {
	return u.Uid
}

func (u *UserInfo) GetName() string {
	return u.Name
}

func (u *UserInfo) Logger() *logrus.Entry {
	return logger.WithFields(logrus.Fields{
		"Site": Site,
		"Uid":  u.Uid,
		"Name": u.Name,
	})
}

type NewsInfo struct {
	*UserInfo
	LatestNewsTs int64   `json:"latest_news_time"`
	Cards        []*Card `json:"-"`
}

func (n *NewsInfo) Type() concern_type.Type {
	return News
}

func (n *NewsInfo) Logger() *logrus.Entry {
	return n.UserInfo.Logger().WithFields(logrus.Fields{
		"Type":     n.Type().String(),
		"CardSize": len(n.Cards),
	})
}

type ConcernNewsNotify struct {
	GroupCode int64 `json:"group_code"`
	*UserInfo
	Card *CacheCard
}

func (c *ConcernNewsNotify) Type() concern_type.Type {
	return News
}

func (c *ConcernNewsNotify) GetGroupCode() int64 {
	return c.GroupCode
}

func (c *ConcernNewsNotify) Logger() *logrus.Entry {
	return c.UserInfo.Logger().WithFields(localutils.GroupLogFields(c.GroupCode))
}

func (c *ConcernNewsNotify) ToMessage() (m *mmsg.MSG) {
	return c.Card.GetMSG()
}

func NewConcernNewsNotify(groupCode int64, info *NewsInfo) []*ConcernNewsNotify {
	var result []*ConcernNewsNotify
	for _, card := range info.Cards {
		result = append(result, &ConcernNewsNotify{
			GroupCode: groupCode,
			UserInfo:  info.UserInfo,
			Card:      NewCacheCard(card, info.GetName()),
		})
	}
	return result
}

type CacheCard struct {
	*Card
	Name string

	once     sync.Once
	msgCache *mmsg.MSG
}

func NewCacheCard(card *Card, name string) *CacheCard {
	return &CacheCard{Card: card, Name: name}
}

func (c *CacheCard) prepare() {
	m := mmsg.NewMSG()
	createdTime := getTimeString(c.Card.GetCreatedAt())
	if c.Card.GetRetweetedStatus() != nil {
		m.Textf("weibo-%v转发了%v的微博：\n%v",
			c.Name,
			c.Card.GetRetweetedStatus().GetUser().GetScreenName(),
			createdTime,
		)
	} else {
		m.Textf("weibo-%v发布了新微博：\n%v",
			c.Name,
			createdTime,
		)
	}
	switch c.Card.GetMblogtype() {
	case CardType_Normal, CardType_Text, CardType_Top:
		logger.Infof("found card_types: %v", c.Mblogtype.String())
		if len(c.Card.GetRawText()) > 0 {
			rawText := parseHTML(c.Card.GetRawText())
			m.Textf("\n%v", localutils.RemoveHtmlTag(rawText))
		} else {
			Text := parseHTML(c.Card.GetText())
			m.Textf("\n%v", localutils.RemoveHtmlTag(Text))
		}
		findPicForCard(c.Card.GetPicInfos(), m)
		if c.Card.GetRetweetedStatus() != nil {
			if len(c.Card.GetRetweetedStatus().GetRawText()) > 0 {
				rawText := parseHTML(c.Card.GetRetweetedStatus().GetRawText())
				m.Textf("\n\n原微博：\n%v", localutils.RemoveHtmlTag(rawText))
			} else {
				Text := parseHTML(c.Card.GetRetweetedStatus().GetText())
				m.Textf("\n\n原微博：\n%v", localutils.RemoveHtmlTag(Text))
			}
			if c.Card.GetRetweetedStatus().GetMixMediaInfo() != nil {
				findPicForMix(c.Card.GetRetweetedStatus().GetMixMediaInfo().GetItems(), m)
				findVideoForMix(c.Card.GetRetweetedStatus().GetMixMediaInfo().GetItems(), m)
			}
			findPicForCard(c.Card.GetRetweetedStatus().GetPicInfos(), m)
		}
		if c.GetPageInfo() != nil {
			m.ImageByUrl(c.GetPageInfo().GetPagePic(), "")
			switch c.GetPageInfo().GetObjectType() {
			case "video":
				m.Textf("%s\n%s - %s\n", c.GetPageInfo().GetMediaInfo().GetName(),
					time.Unix(c.GetPageInfo().GetMediaInfo().GetVideoPublishTime(), 0).Format(time.DateTime),
					c.GetPageInfo().GetMediaInfo().GetOnlineUsers())
			case "article":
				m.Textf("%s\n", c.GetPageInfo().GetContent1())
			default:
				logger.Debugf("found page_info new type: %s", c.GetPageInfo().GetObjectType())
			}
		} else if c.Card.GetMixMediaInfo() != nil {
			findPicForMix(c.Card.GetMixMediaInfo().GetItems(), m)
			findVideoForMix(c.Card.GetMixMediaInfo().GetItems(), m)
		}
		m.Text("\n" + getWeiboUrl(c.Card.GetUser().GetId(), c.Card.Mblogid))
	default:
		logger.WithField("Type", c.Mblogtype.String()).Debug("found new card_types")
	}
	c.msgCache = m
}

func (c *CacheCard) GetMSG() *mmsg.MSG {
	c.once.Do(c.prepare)
	return c.msgCache
}

func parseHTML(text string) string {
	text = strings.ReplaceAll(text, "<br />", "\n")
	text = html.UnescapeString(text)
	return text
}

func getWeiboUrl(uid int64, mblogId string) string {
	return "https://weibo.com/" + strconv.FormatInt(uid, 10) + "/" + mblogId
}

func getTimeString(t string) string {
	var ti string
	newsTime, err := time.Parse(time.RubyDate, t)
	if err == nil {
		ti = newsTime.Format("2006-01-02 15:04:05")
	} else {
		ti = t
	}
	return ti
}

func findPicForCard(picInfos map[string]*Card_PicInfo, m *mmsg.MSG) {
	for _, pic := range picInfos {
		switch pic.Type {
		case "pic":
			m.ImageByUrl(pic.GetLarge().GetUrl(), "")
		case "gif":
			m.ImageByUrl(pic.GetOriginal().GetUrl(), "")
		}
	}
}

func findPicForMix(Item []*Card_MixMediaInfo_Items, m *mmsg.MSG) {
	for _, item := range Item {
		raw := item.Data.AsMap()
		switch item.Type {
		case "pic", "gif":
			var pic Card_PicInfo
			b, _ := json.Marshal(raw)
			err := json.Unmarshal(b, &pic)
			if err != nil {
				logger.Errorf("found pic failed. %v,", err)
			}
			if item.Type == "gif" {
				m.ImageByUrl(pic.GetOriginal().GetUrl(), "")
				continue
			}
			m.ImageByUrl(pic.GetLarge().GetUrl(), "")
		}
	}
}

func findVideoForMix(Item []*Card_MixMediaInfo_Items, m *mmsg.MSG) {
	for _, item := range Item {
		raw := item.Data.AsMap()
		switch item.Type {
		case "video":
			var video Card_PageInfo
			b, _ := json.Marshal(raw)
			err := json.Unmarshal(b, &video)
			if err != nil {
				logger.Errorf("found video failed. %v,", err)
			}
			m.ImageByUrl(video.GetPagePic(), "")
			m.Textf("%s\n%s - %s\n", video.GetMediaInfo().GetName(),
				time.Unix(video.GetMediaInfo().GetVideoPublishTime(), 0).Format(time.DateTime),
				video.GetMediaInfo().GetOnlineUsers())
		}
	}
}
