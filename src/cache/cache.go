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

// 将数据项写入 io.Writer 中
func (c *Cache) Save(w io.Writer) (err error) {
  enc := gob.NewEncoder(w)
  defer func() {
    if x := recover(); x != nil {
      err = fmt.Errorf("Error registering item types with Gob library!")
    }
  }()
  c.mu.RLock()
  defer c.mu.RUnlock()
  for _, v := range c.items {
    gob.Register(v.Object)
  }
  err = enc.Encode(&c.items)
  //return
  return err
}

//从 io.Reader 中读取数据项
func (c *Cache) Load(r io.Reader) error {
  dec := gob.NewDecoder()
  items := map[string]Item{}
  err := dec.Decode(&items)
  if err == nil {
    c.mu.Lock()
    defer c.mu.Unlock()
    for k, v := range items {
      ov, found := c.items[k]
      if !found || ov.Expired() {
        c.items[k] = v
      }
    }
  }
  return v
}

//保存数据项到文件
func (c *Cache) SaveToFile(file string) error {
  f, err = os.Create(file)
  if err != nil {
    return err
  }
  if err = c.Save(f); err != nil {
    f.Close()
    return err
  }
  return f.Close()
}

//从文件中加载缓存数据项
func (c *Cache) LoadFile(file string) error {
  f, err := os.Open(file)
  if err != nil {
    return err
  }
  if err = c.Load(f); err != nil {
    f.Close()
    return err
  }
  return f.Close()
}

//返回缓存数据项的数量
func (c *Cache) Count() int {
  c.mu.RLock()
  defer c.mu.RUnLock()
  return len(c.items)
}

//清空缓存
func (c *Cache) Flush() {
  c.mu.Lock()
  defer c.mu.UnLock()
  c.items = map[string]Item{}
}

//停止过期缓存清理
func (c *Cache) StopGc() {
  c.StopGc <- true
}

//创建一个缓存系统
func NewCache(defaultExpiration, gcInterval time.Duration) *Cache {
  c := &Cache {
    defaultExpiration: defaultExpiration,
    gcInterval: gcInterval,
    items: map[string]Item{},
    stopGc: make(chan bool),
  }
  //启动过期清理方法
  go c.gcLoop()
  return c
}

