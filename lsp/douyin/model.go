package douyin

import (
	"github.com/cnxysoft/DDBOT-WSa/lsp/concern_type"
	"github.com/cnxysoft/DDBOT-WSa/lsp/mmsg"
	"github.com/cnxysoft/DDBOT-WSa/lsp/template"
	localutils "github.com/cnxysoft/DDBOT-WSa/utils"
	"github.com/sirupsen/logrus"
	"sync"
)

type UserInfo struct {
	Uid       string `json:"uid"`
	SecUid    string `json:"secUid"`
	NikeName  string `json:"nickname"`
	RealName  string `json:"realName"`
	Desc      string `json:"desc"`
	WebRoomId string `json:"web_rid"`
}

func (u UserInfo) GetUid() interface{} {
	return u.SecUid
}

func (u UserInfo) GetName() string {
	return u.NikeName
}

type LiveInfo struct {
	UserInfo
	IsLiving bool `json:"living"`

	once     sync.Once
	msgCache *mmsg.MSG
}

func (l *LiveInfo) Living() bool {
	return l.IsLiving
}

func (l *LiveInfo) Site() string {
	return Site
}

func (l *LiveInfo) Type() concern_type.Type {
	return Live
}

func (l *LiveInfo) Logger() *logrus.Entry {
	return logger.WithFields(logrus.Fields{
		"Site": Site,
		"Uid":  l.Uid,
		"Name": l.NikeName,
		"Room": l.WebRoomId,
		"Type": l.Type().String(),
	})
}

func (l *LiveInfo) GetMSG() *mmsg.MSG {
	l.once.Do(func() {
		var data = map[string]interface{}{
			"uid":    l.Uid,
			"name":   l.NikeName,
			"roomId": l.WebRoomId,
			"living": l.Living(),
			"url":    BaseLiveHost + "/" + l.WebRoomId,
		}
		var err error
		l.msgCache, err = template.LoadAndExec("notify.group.douyin.live.tmpl", data)
		if err != nil {
			logger.Errorf("acfun: LiveInfo LoadAndExec error %v", err)
		}
		return
	})
	return l.msgCache
}

type LiveNotify struct {
	*LiveInfo
	GroupCode int64
}

func (l LiveNotify) Site() string {
	return "douyin"
}

func (nl LiveNotify) Type() concern_type.Type {
	return Live
}

func (l *LiveNotify) GetUid() interface{} {
	return l.SecUid
}

func (l LiveNotify) Logger() *logrus.Entry {
	if &l == nil {
		return logger
	}
	return l.LiveInfo.Logger().WithFields(localutils.GroupLogFields(l.GroupCode))
}

func (n LiveNotify) GetGroupCode() int64 {
	return n.GroupCode
}

func (n LiveNotify) ToMessage() *mmsg.MSG {
	return n.LiveInfo.GetMSG()
}
