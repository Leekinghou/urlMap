package main

import (
	"sync"
)

// URLStore 是一个存储短网址和长网址的映射的结构体
type URLStore struct {
	// map 是从短网址到长网址
	urls map[string]string
	mu   sync.RWMutex
}

// NewURLStore URLStore 工厂函数，返回一个新的 URLStore
// NewURLStore 返回一个新的 URLStore
func NewURLStore() *URLStore {
	return &URLStore{
		urls: make(map[string]string),
	}
}

// Get 从 URLStore 中获取一个长网址
func (s *URLStore) Get(shortURL string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	longURL, ok := s.urls[shortURL]
	return longURL, ok
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

func (s *URLStore) Put(url string) string {
	for {
		// 生成短链接
		key := genKey(s.Count())
		if ok := s.Set(key, url); ok {
			return key
		}
		// 程序不会走到这里
		return ""
	}
}
