package oauth

import (
	"log"
	"math"
	"strconv"
	"time"
)

func (oauth *Oauth) ChangeUserOpenidWeight(loginOpenid string) {
	defer func() {
		nerr := recover()
		if nerr != nil {
			log.Println(nerr.(error))
		}
	}()

	redisCache, err := oauth.GetRedisFromCache()
	if err != nil {
		return
	}

	// 5s中之后检查本人的快速授权登录的记录是否存在，不存在的话，此openid的信任度-30
	time.Sleep(time.Second * 5)
	localServerKey := "wechatserver:" + loginOpenid
	var weightStr string
	err = redisCache.GetWithErrorBack(localServerKey, &weightStr)

	if err != nil && err.Error() != "redis: nil" {
		//如果数据不存在，那么err==redis: nil
		log.Println(err)
	}
	if len(weightStr) == 0 {
		weightStr = "0"
	}

	wechatCallBack, err := strconv.Atoi(weightStr)
	if err != nil {
		log.Println(err)
	}
	duration := int(time.Now().Unix()) - wechatCallBack

	if math.Abs(float64(duration)) > 10 {
		//openid对应的信任度-20
		//快速登录检查扣分3次，回调检查扣分2次，快速登录扣分1次+回调扣分一次，信任度< 50，将会无法使用快速登录，等到缓存过期又可以重新使用快速登录
		err := oauth.DecreaseOpenidWeight(loginOpenid, 20)
		if err != nil {
			log.Println(err)
		}
	}

}

func (oauth *Oauth) SetInitUserOpenidWeight(loginOpenid string) error {
	defer func() {
		nerr := recover()
		if nerr != nil {
			log.Println(nerr.(error))
		}
	}()

	redisCache, err := oauth.GetRedisFromCache()
	if err != nil {
		return err
	}

	//重新授权登录获取到的openid，信任度初始值为100
	weightKey := "openidweight:" + loginOpenid
	err = redisCache.Set(weightKey, "100", 0)
	if err != nil {
		return err
	}

	return nil
}

func (oauth *Oauth) DecreaseOpenidWeight(openid string, num int64) error {

	redisCache, err := oauth.GetRedisFromCache()
	if err != nil {
		return err
	}

	redisKey := "openidweight:" + openid

	var weightStr string
	err = redisCache.GetWithErrorBack(redisKey, &weightStr)
	if err != nil {
		return err
	}
	if len(weightStr) == 0 {
		weightStr = "0"
	}

	oldWeight, err := strconv.Atoi(weightStr)
	if err != nil {
		return err
	}

	if oldWeight < 50 {
		return nil
	}

	err = redisCache.DecrBy(redisKey, num)
	if err != nil {
		return err
	}
	return nil
}

func (oauth *Oauth) GetOpenidWeight(openid string) (int, error) {
	redisCache, err := oauth.GetRedisFromCache()
	if err != nil {
		return 0, err
	}

	redisKey := "openidweight:" + openid
	var weightStr string
	err = redisCache.GetWithErrorBack(redisKey, &weightStr)
	if err != nil {
		return 0, err
	}
	if len(weightStr) == 0 {
		weightStr = "0"
	}

	oldWeight, err := strconv.Atoi(weightStr)
	if err != nil {
		return 0, err
	}

	return oldWeight, nil

}
