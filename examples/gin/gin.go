package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/WhisperRain/wechat"
	"github.com/WhisperRain/wechat/cache"
	"github.com/WhisperRain/wechat/message"
	"github.com/WhisperRain/wechat/oauth"
	"net/http"
)

const PersoncenterOauthSuccessUrl = "/oauth/personcenter/success"

var Wc = wechat.NewWechat(&wechat.Config{
	AppID:          "your app id",
	AppSecret:      "your app secret",
	Token:          "your token",
	EncodingAESKey: "your encoding aes key",
	Cache: cache.NewRedis(&cache.RedisOpts{
		Host:        "127.0.0.1:6379",
		Password:    "your redis password",
		MaxIdle:     10,
		IdleTimeout: 5,
	}),
})

func init() {
	Wc.Context.FastOauthEnable=true
}


func main() {
	router := gin.Default()

	router.Any("/", hello)

	{
		router.GET("/oauth/personcenter/begin", PersonCenterOauthBegin)
		router.GET(PersoncenterOauthSuccessUrl, PersonCenterOauthSuccess)
	}

	router.Run(":8001")
}

func hello(c *gin.Context) {

	// 传入request和responseWriter
	server := Wc.GetServer(c.Request, c.Writer)
	//设置接收消息的处理方法
	server.SetMessageHandler(func(msg message.MixMessage) *message.Reply {

		//回复消息：演示回复用户发送的消息
		text := message.NewText(msg.Content)
		return &message.Reply{MsgType: message.MsgTypeText, MsgData: text}
	})

	//处理消息接收以及回复
	err := server.Serve()
	if err != nil {
		fmt.Println(err)
		return
	}
	//发送回复的消息
	server.Send()
}

//PersonCenterBeginOauth  开始授权登录重定向，获取一次性code
func PersonCenterOauthBegin(c *gin.Context) {

	//跳转到授权认证的后接口，先获取微信用户信息，保存微信关注状态，然后绑定微信和游戏账户
	oauthSuccessUrl := getDomainUrl() + PersoncenterOauthSuccessUrl
	m := oauth.Direction{
		Ip:          c.ClientIP(),
		RedirectURI: oauthSuccessUrl,
		Scope:       "snsapi_userinfo",
		State:       "random_string",
	}

	//快速授权登录成功，直接执行成功以后的模块
	Wc.GetOauth().FastOauthWithCache(c.Writer, c.Request, m, func(user oauth.OauthUser) {
		OperationAfterOauthSuccess(c, user)
	})

}

//PersonCenterRedirectToFrontPage 授权登录拿到了code以后，用code缓存换取微信用户信息，并执行后续操作
func PersonCenterOauthSuccess(c *gin.Context) {
	o := Wc.GetOauth()

	//通过code换取access_token
	code := c.Query("code")
	if len(code) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "code 为空"})
		return
	}

	resToken, err := o.GetUserAccessToken(code)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	userInfo, err := o.GetUserInfo(resToken.AccessToken, resToken.OpenID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	//如果想要缓存手机号的话，可以创建userInfo的子类，在子类加入字段手机号

	//保存授权登录获取的用户信息到缓存
	go o.SaveOauthUserInfoToRedis(c.Request, c.ClientIP(), userInfo)

	OperationAfterOauthSuccess(c, userInfo)
}

func OperationAfterOauthSuccess(c *gin.Context, user oauth.OauthUser) {
	//TODO 授权登录成功以后的操作，比如带着用户信息重定向到前端网页
	c.JSON(http.StatusOK, gin.H{"data": user})
}


func getDomainUrl() string {
	return "http://127.0.0.1:8001"
}

