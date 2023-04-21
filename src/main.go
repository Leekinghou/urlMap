package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/rpc"
)

const AddForm = `
<!DOCTYPE html>
<html>
<head>
	<title>添加URL</title>
</head>
<body>
	<h1>添加URL</h1>
	<form method="POST" action="/add">
		URL: <input type="text" name="url">
		<input type="submit" value="Add">
	</form>
</body>
</html>

`

var (
	listenAddr = flag.String("http", ":8080", "http listen address")
	dataFile   = flag.String("file", "file/store.json", "data store file name")
	hostname   = flag.String("host", "localhost:8081", "http host name")
	masterAddr = flag.String("master", "", "RPC master address")
	rpcEnabled = flag.Bool("rpc", true, "enable RPC server")
)

var store Store

func main() {
	//  flags 被解析后实例化 URLStore 对象
	flag.Parsed()
	//  如果 masterAddr 不为空，则使用 ProxyStore代表从服务器
	if *masterAddr != "" {
		store = NewProxyStore(*masterAddr)
	} else { // 否则使用 URLStore 代表主服务器
		store = NewURLStore(*dataFile)
	}
	// 如果 rpcEnabled 为 true，则注册 Store 服务
	if *rpcEnabled {
		rpc.RegisterName("Store", store)
		rpc.HandleHTTP()
	}
	http.HandleFunc("/", Redirect)
	http.HandleFunc("/add", Add)
	http.ListenAndServe(*listenAddr, nil)
}

func Redirect(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Path[1:]
	// 如果 key 为空，则返回 404
	if key == "" {
		http.NotFound(w, r)
		return
	}
	var url string
	if err := store.Get(&key, &url); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, url, http.StatusFound)
	log.Printf("Redirect: %s -> %s", key, url)
}

func Add(w http.ResponseWriter, r *http.Request) {
	url := r.FormValue("url")
	if url == "" {
		fmt.Fprint(w, AddForm)
		return
	}
	var key string
	if err := store.Put(&url, &key); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// 将 localhost:8080 替换成 *hostname
	fmt.Fprintf(w, "http://%s/%s", *hostname, key)
	log.Printf("Add: %s -> %s", key, url)
}
