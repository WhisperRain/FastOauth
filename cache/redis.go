package cache

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
)

//Redis redis cache
type Redis struct {
	conn *redis.Pool
}

//RedisOpts redis 连接属性
type RedisOpts struct {
	Host        string `yml:"host" json:"host"`
	Password    string `yml:"password" json:"password"`
	Database    int    `yml:"database" json:"database"`
	MaxIdle     int    `yml:"max_idle" json:"max_idle"`
	MaxActive   int    `yml:"max_active" json:"max_active"`
	IdleTimeout int32  `yml:"idle_timeout" json:"idle_timeout"` //second
}

//NewRedis 实例化
func NewRedis(opts *RedisOpts) *Redis {
	pool := &redis.Pool{
		MaxActive:   opts.MaxActive,
		MaxIdle:     opts.MaxIdle,
		IdleTimeout: time.Second * time.Duration(opts.IdleTimeout),
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", opts.Host,
				redis.DialDatabase(opts.Database),
				redis.DialPassword(opts.Password),
			)
		},
		TestOnBorrow: func(conn redis.Conn, t time.Time) error {
			if time.Since(t) < time.Minute {
				return nil
			}
			_, err := conn.Do("PING")
			return err
		},
	}
	return &Redis{pool}
}

//SetConn 设置conn
func (r *Redis) SetConn(conn *redis.Pool) {
	r.conn = conn
}

//Get 获取一个值
func (r *Redis) Get(key string) interface{} {
	conn := r.conn.Get()
	defer conn.Close()

	var data []byte
	var err error
	if data, err = redis.Bytes(conn.Do("GET", key)); err != nil {
		return nil
	}
	var reply interface{}
	if err = json.Unmarshal(data, &reply); err != nil {
		return nil
	}

	return reply
}

//Set 设置一个值
func (r *Redis) Set(key string, val interface{}, timeout time.Duration) (err error) {
	conn := r.conn.Get()
	defer conn.Close()

	var data []byte
	if data, err = json.Marshal(val); err != nil {
		return
	}

	_, err = conn.Do("SETEX", key, int64(timeout/time.Second), data)

	return
}

//IsExist 判断key是否存在
func (r *Redis) IsExist(key string) (bool, error) {
	conn := r.conn.Get()
	defer conn.Close()

	a, err := conn.Do("EXISTS", key)
	if err != nil {
		return false, err
	}
	i := a.(int64)
	if i > 0 {
		return true, nil
	}
	return false, nil
}

//Delete 删除
func (r *Redis) Delete(key string) error {
	conn := r.conn.Get()
	defer conn.Close()

	if _, err := conn.Do("DEL", key); err != nil {
		return err
	}

	return nil
}

//HGet 从hash map中获取一个值 ,reply 是指针类型
func (r *Redis) HGet(key, field string, reply interface{}) error {
	conn := r.conn.Get()
	defer conn.Close()

	var data []byte
	var err error
	if data, err = redis.Bytes(conn.Do("HGET", key, field)); err != nil {
		return err
	}

	if err = json.Unmarshal(data, reply); err != nil {
		return err
	}

	return nil
}


//HSetWxUser 设置微信用户信息
func (r *Redis) HSetWxUser(ip, agentKey string, user interface{}) error {

	exist, err := r.IsExist(ip)
	if err != nil {
		return err
	}

	conn := r.conn.Get()
	defer conn.Close()

	var data []byte
	if data, err = json.Marshal(user); err != nil {
		return err
	}

	//首次创建hash map，开启事务
	//1.设置 hash map 字段初始值
	//2.设置过期时间
	//提交事务
	if exist {

		//if手机网络为4G等等， 过期时间2天， expireTime过期时间应当近似等于ip的变化周期的两倍。
		expireTime := 2 * 24 * 3600 * time.Second
		//if手机网络为wifi， 过期时间 2*30天
		if strings.Contains(agentKey, "NetType/WIFI") {
			expireTime = 2 * 30 * 24 * 3600 * time.Second
		}

		//pl := RedisClient.TxPipeline()
		//pl.HSet(ip, agentKey, user)
		//pl.Expire(ip, expireTime)
		//_, err = pl.Exec()

		_ = conn.Send("MULTI")
		_ = conn.Send("HSET", ip, agentKey, data)
		_ = conn.Send("EXPIRE", int64(expireTime))
		_, err := conn.Do("EXEC")
		if err != nil {
			return err
		}
		return nil
	}

	_, err = conn.Do("HSET", ip, agentKey, data)

	return err
}

//Get 获取一个值
func (r *Redis) GetWithErrorBack(key string, reply interface{}) error {
	conn := r.conn.Get()
	defer conn.Close()

	var data []byte
	var err error
	if data, err = redis.Bytes(conn.Do("GET", key)); err != nil {
		return err
	}

	if err = json.Unmarshal(data, &reply); err != nil {
		return err
	}

	return nil
}

//HGet 从hash map中获取一个值 ,reply 是指针类型
func (r *Redis) DecrBy(key string,value int64) error {
	conn := r.conn.Get()
	defer conn.Close()

 	var err error
	if _, err = redis.Bytes(conn.Do("DECRBY", key, value)); err != nil {
		return err
	}

	return nil
}
