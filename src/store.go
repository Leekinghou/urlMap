package main

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"sync"
)

// URLStore 是一个存储短网址和长网址的映射的结构体
type URLStore struct {
	// map 是从短网址到长网址
	urls map[string]string
	mu   sync.RWMutex
	save chan record
}

type record struct {
	Key, URL string
}

// saveQueueLength 保存队列的长度
const saveQueueLength = 1000

/**
NewURLStore URLStore 工厂函数，返回一个新的 URLStore
当我们实例化 URLStore 的时候，我们将调用store.gob文件
并将它的名称作为参数： var store = NewURLStore("store.gob")
NewURLStore 返回一个新的 URLStore
*/
func NewURLStore(filename string) *URLStore {
	s := &URLStore{
		urls: make(map[string]string),
		// 保存队列:弥补性能瓶颈，Put将一个record发送到channel缓冲区保存，而非进行函数调用保存每一条记录到磁盘
		save: make(chan record, saveQueueLength),
	}
	if err := s.load(filename); err != nil {
		log.Println("Error opening URLStore:", err)
	}
	go s.saveLoop(filename)
	return s
}

// Get 从 URLStore 中获取一个长网址
func (s *URLStore) Get(shortURL string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.urls[shortURL]
}

// Set 将一个长网址和一个短网址存储到 URLStore 中
// (s *URLStore)的作用是为了让Set方法可以访问URLStore的属性
func (s *URLStore) Set(shortURL, longURL string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	// 如果短网址已经存在，返回 false
	if _, present := s.urls[shortURL]; present {
		return false
	}
	s.urls[shortURL] = longURL
	return true
}

// Delete 从 URLStore 中删除一个短网址
func (s *URLStore) Delete(shortURL string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.urls, shortURL)
}

// Count 返回 URLStore 中的短网址数量
func (s *URLStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.urls)
}

// All 返回 URLStore 中的所有短网址
func (s *URLStore) All() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	urls := make(map[string]string, len(s.urls))
	for k, v := range s.urls {
		urls[k] = v
	}
	return urls
}

// Put 将一个长网址存储到 URLStore 中，并返回一个短网址
func (s *URLStore) Put(url string) string {
	for {
		// 生成短链接
		key := genKey(s.Count())
		if ok := s.Set(key, url); ok {
			s.save <- record{key, url}
			return key
		}
		// 程序不会走到这里
		panic("shouldn't get here")
	}
}

// load:在 goto 启动的时候，我们磁盘上的数据存储必须读取到 URLStore 中
func (s *URLStore) load(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		log.Println("Error opening URLStore:", err)
		return err
	}
	defer f.Close()
	// 从文件中读取数据
	d := json.NewDecoder(f)
	// 解码是一个无限循环，只要没有错误就会一直继续下去
	for err == nil {
		var r record
		if err = d.Decode(&r); err == nil {
			s.Set(r.Key, r.URL)
		}
	}
	if err == io.EOF {
		return nil
	}
	// map hasn't been read correctly
	log.Println("Error decoding URLStore:", err)
	return err
}

// saveLoop:将数据保存到磁盘
func (s *URLStore) saveLoop(filename string) {
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Println("Error opening URLStore:", err)
	}
	defer f.Close()
	e := json.NewEncoder(f)
	for {
		r := <-s.save
		if err := e.Encode(r); err != nil {
			log.Println("Error saving URL:", err)
		}
	}
}
