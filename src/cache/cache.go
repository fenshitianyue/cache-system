package cache

import (
  "fmt"
  "time"
)

type Item struct {
  Object interface{}  // 真正的数据项
  Expiration int64    // 生存时间
}

//判断数据项是否已经过期
func (item Item) Expired() bool {
  if item.Expiration == 0 {
    return false
  }
  return time.Now().UnixNano() > item.Expiration
}
