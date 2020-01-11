package oauth

import (
	"errors"
	"github.com/WhisperRain/wechat/cache"
	"log"
	"net/http"
	"strings"
)

type Direction struct {
	Ip, RedirectURI, Scope, State string
}

type OauthUser interface {
	GetOpenID() string
}

func (oauth *Oauth) GetRedisFromCache() (*cache.Redis, error) {
	if !oauth.FastOauthEnable {
		return nil, errors.New("FastOauth Not Enable")
	}

	c := oauth.Cache
	switch c.(type) {
	case *cache.Redis:
		return c.(*cache.Redis), nil
	default:
	}
	return nil, errors.New("no data")
}

//最快获取微信用户信息的跳转方法
func (oauth *Oauth) FastOauthWithCache(writer http.ResponseWriter, req *http.Request, m Direction, f func(user OauthUser)) {
	redisCache, err := oauth.GetRedisFromCache()
	if err != nil {
		return
	}

	agentKey, exist := FilterRedisKeyOfUserAgent(req)
	if !exist {
		_ = oauth.Redirect(writer, req, m.RedirectURI, m.Scope, m.State)
		return
	}

	var wechatUser OauthUser

	err1 := redisCache.HGet(m.Ip, agentKey, &wechatUser)
	if err1 != nil {
		log.Println(err1)
		_ = oauth.Redirect(writer, req, m.RedirectURI, m.Scope, m.State)
		return
	}

	if len(wechatUser.GetOpenID()) == 0 {
		_ = oauth.Redirect(writer, req, m.RedirectURI, m.Scope, m.State)
		return
	}

	// 取出openid对应的信任度
	weight, err2 := oauth.GetOpenidWeight(wechatUser.GetOpenID())
	if err2 != nil {
		log.Println(err2)
		_ = oauth.Redirect(writer, req, m.RedirectURI, m.Scope, m.State)
		return
	}
	if weight < 50 {
		_ = oauth.Redirect(writer, req, m.RedirectURI, m.Scope, m.State)
		return
	}

	//触发信任度检查机制
	//1. 更新redis中本人的访问时间
	//2. 5s中之后检查本人的消息回调的记录是否存在，不存在的话，此openid的信任度-20
	//快速登录检查扣分3次，回调检查扣分2次，快速登录扣分1次+回调扣分一次，信任度< 50，将会无法使用快速登录，等到缓存过期又可以重新使用快速登录
	go oauth.ChangeUserOpenidWeight(wechatUser.GetOpenID())


	f(wechatUser)
}

func (oauth *Oauth) SaveOauthUserInfoToRedis(req *http.Request, ip string, user OauthUser) {
	redisCache, err := oauth.GetRedisFromCache()
	if err != nil {
		return
	}

	agentKey, exist := FilterRedisKeyOfUserAgent(req)
	if !exist {
		log.Println("不标准的网络请求 FilterRedisKeyOfUserAgent")
	}

	err = redisCache.HSetWxUser(ip, agentKey, user)
	if err != nil {
		log.Println(err) //出于高可用，这里并不会return
	}

}

func FilterRedisKeyOfUserAgent(req *http.Request) (key string, exist bool) {
	agentStr := req.Header["User-Agent"]
	agentKey := ""
	for _, v := range agentStr {
		str1 := strings.Replace(v, " ", "", -1)
		//str2 := strings.ReplaceAll(str1, "]", "")
		//str3 := strings.ReplaceAll(str2, "{", "")
		//str4 := strings.ReplaceAll(str3, "}", "")
		agentKey += str1
	}
	if len(agentStr) > 0 {
		if len(agentStr) > 1024 {
			//长度太长的redis key，只截取其中一部分
			agentKey = agentKey[0:1024]
		}
		return agentKey, true
	}

	return "", false
}


