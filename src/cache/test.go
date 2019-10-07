package main

import (
  "fmt"
  "time"
)

type Item struct {
  //Object interface{}
  Object int64
  Expiration int64
}

func (item Item) Expired() bool {
  if item.Expiration == 0 {
    return false
  }
  return time.Now().UnixNano() > item.Expiration
}

func main() {
  // i := Item{1, 2}
  // if i.Expired() == false {
  //   fmt.Println("缓存过期！")
  // } else {
  //   fmt.Println(i.Expired())
  //   fmt.Println(i)
  // }
  fmt.Println(time.Now().UnixNano())
}
