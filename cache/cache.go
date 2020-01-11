package cache

import "time"

//Cache interface
type Cache interface {
	Get(key string)  interface{}
	Set(key string, val interface{}, timeout time.Duration) error
	IsExist(key string) (bool, error)
	Delete(key string) error
}