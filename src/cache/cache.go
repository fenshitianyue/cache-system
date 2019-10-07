package cache

import (
  "fmt"
  "time"
  "encoding/gob"
  "io"
  "sync"
  "os"
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

const (
  //没有过期的时间标志
  NoExpiration time.Duration = -1
  //默认的过期时间
  DefaultExpiration time.Duration = 0
)

type Cache struct {
  defaultExpiration    time.Duration
  items                map[string]Item
  mu                   sync.RWMutex
  gcInterval           time.Duration
  stopGc               chan bool
}

func (c *Cache) gcLoop() {
  ticker := time.NewTicker(c.gcInterval)
  for {
    select {
    case <-ticker.C:
      c.DeleteExpired()
    case <-c.stopGc:
      ticker.Stop()
      return
    }
  }
}

func (c *Cache) delete(k string) {
  delete(c.Items, k)
}

func (c *Cache) DeleteExpired() {
  now := time.Now().UnixNano
  c.mu.Lock()
  defer c.mu.Unlock()

  for k, v := range c.items {
    if v.Expiration > 0 && now > v.Expiration {
      c.delete(k)
    }
  }
}

func (c *Cache) Set(k string, v interface{}, d time.Duration) {
  var e int64
  if d == DefaultExpiration {
    d = c.defaultExpiration
  }
  if d > 0 {
    e = time.Now().Add(d).UnixNano()
  }
  c.mu.Lock()
  defer c.mu.Unlock()
  c.item[k] = Item {
    Object: v,
    Expiration: e,
  }
}

func (c *Cache) get(k string) (interface{}, bool) {
  item, found := c.items[k]
  if !found {
    return nil, false
  }
  if item.Expired() {
    return nil, false
  }
  return item.Object, true
}

func (c *Cache) Get(k string) (interface{}, bool) {
  c.mu.RLock()
  item, found := c.items[k]
  if !found {
    c.mu.RUnlock()
    return nil, false
  }
  if item.Expired() {
    return nil, false
  }
  c.mu.RUnlock()
  return item.Object, true
}

func (c *Cache) Add(k string, v interface{}, d time.Duration) error {
  c.mu.Lock()
  _, found := c.get(k)
  if found {
    c.mu.Unlock()
    return fmt.Errorf("Item %s already exists.", k)
  }
  c.set(k, v, d)
  c.mu.Unlock()
  return nil
}

func (c *Cache) Replace(k string, v interface{}, d time.Duration) error {
  c.mu.Lock()
  _, found := c.get(k)
  if !found {
    c.mu.Unlock()
    return fmt.Errorf("Item %s doesn't exist.", k)
  }
  c.set(k, v, d)
  c.mu.Unlock()
  return nil
}

func (c *Cache) Delete(k string) {
  c.mu.Lock()
  c.delete(k)
  c.mu.Unlock()
}



