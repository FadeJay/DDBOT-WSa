package douyin

import (
	"github.com/Sora233/MiraiGo-Template/config"
	"github.com/cnxysoft/DDBOT-WSa/lsp/concern"
	"net/http/cookiejar"
	"strings"
)

var (
	UserAgent   = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36 Edg/135.0.0.0"
	AcSignature = "_02B4Z6wo00f01-ThxWAAAIDD08UBP6F0nbvkwcHAAJGspSwE1jnxTP0Wq5Y4sRag6wu4giEpx30lcg4IWLBVTRergX7k0f2GDXwakMgLwo0Njci.GvR70Env.4qyrfAKawTsOC2BWoklKiWv71"
	AcNonce     = "0689b07b40013ec76b2e8"
)

func init() {
	concern.RegisterConcern(NewConcern(concern.GetNotifyChan()))
}

func setCookies() {
	ua := config.GlobalConfig.GetString("douyin.userAgent")
	as := config.GlobalConfig.GetString("douyin.AcSignature")
	an := config.GlobalConfig.GetString("douyin.AcNonce")
	Cookie, _ = cookiejar.New(nil)
	if ua != "" {
		UserAgent = ua
	}
	if as != "" {
		AcSignature = as
	}
	if an != "" {
		AcNonce = an
	}
}

func DPath(path string) string {
	if strings.HasPrefix(path, "/") {
		return BasePath[path] + path
	} else {
		return BasePath[path] + "/" + path
	}
}
