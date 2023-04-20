package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/rpc"
	"os"
	"sync"
)

/**
* 所有的从服务器都使用 ProxyStore
* 主服务器使用 URLStore
* 但创建方法相似： 都实现了使用相同签名的 Get 和 Put 方法
* 所以我们能定义一个接口 Store 来归纳它们的行为
* */
type Store interface {
	Put(url, key *string) error
	Get(key, url *string) error
}

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

// ProxyStore 用于 RPC 服务的 URLStore，我们可以构建另一种类型来代表 RPC 客户端，并将发送请求到 RPC 服务器端
type ProxyStore struct {
	urls   *URLStore
	client *rpc.Client
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
	s := &URLStore{urls: make(map[string]string)}
	// 获得一个空的 filename 时不去尝试写入或读取磁盘
	if filename != "" {
		s.save = make(chan record, saveQueueLength)
		if err := s.load(filename); err != nil {
			log.Println("Error opening URLStore:", err)
		}
		go s.saveLoop(filename)
	}
	return s
}

// Get 从 URLStore 中获取一个长网址
// 因为 key 和 url 是指针，必须在它们前面添加一个 * 来获取它们的值，就像 *key；u 是一个值，我们可以将它分配给指针，这样： *url = u
func (s *URLStore) Get(key, url *string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if u, ok := s.urls[*key]; ok {
		*url = u
		return nil
	}
	return errors.New("key not found")
}

// Set 将一个长网址和一个短网址存储到 URLStore 中
// (s *URLStore)的作用是为了让Set方法可以访问URLStore的属性
func (s *URLStore) Set(shortURL, longURL *string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// 如果短网址已经存在，返回 false
	if _, present := s.urls[*shortURL]; present {
		return errors.New("shortURL already exists")
	}
	s.urls[*shortURL] = *longURL
	return nil
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
func (s *URLStore) Put(url, key *string) error {
	for {
		// 生成短链接
		*key = genKey(s.Count())
		if err := s.Set(key, url); err == nil {
			break
		}
		if s.save != nil {
			s.save <- record{*key, *url}
		}
	}
	return nil
}

// load:在项目启动的时候，我们磁盘上的数据存储必须读取到 URLStore 中
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
			s.Set(&r.Key, &r.URL)
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

func NewProxyStore(addr string) *ProxyStore {
	client, err := rpc.DialHTTP("tcp", addr)
	if err != nil {
		log.Println("Error constructing ProxyStore: ", err)
	}
	return &ProxyStore{urls: NewURLStore(""), client: client}
}

// Get 可以在 RPC 客户端调用这些Get将请求直接传递给 RPC 服务器端
func (s *ProxyStore) Get(key, url *string) error {
	// 首先检查缓存中是否有 key
	if err := s.urls.Get(key, url); err == nil {
		return nil
	}
	// 如果没有找到，就从远程服务器获取
	if err := s.client.Call("Store.Get", key, url); err != nil {
		return err
	}
	// 将远程服务器的数据保存到本地
	s.urls.Set(key, url)
	return nil
}

func (s *ProxyStore) Put(url, key *string) error {
	// rpc call to master
	if err := s.client.Call("Store.Put", url, key); err != nil {
		return err
	}
	// rpc update local cache
	s.urls.Set(key, url)
	return nil
}
