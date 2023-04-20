package main

import (
	"flag"
	"fmt"
	"net/http"
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
	dataFile   = flag.String("file", "file/store.gob", "data store file name")
	hostname   = flag.String("host", "localhost:8080", "http host name")
)

var store = NewURLStore("file/store.gob")

func main() {
	//  flags 被解析后实例化 URLStore 对象
	flag.Parsed()
	store = NewURLStore(*dataFile)
	http.HandleFunc("/", Redirect)
	http.HandleFunc("/add", Add)
	http.ListenAndServe(":8080", nil)
}

func Redirect(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Path[1:]
	url := store.Get(key)
	if url == "" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, url, http.StatusFound)
}

func Add(w http.ResponseWriter, r *http.Request) {
	url := r.FormValue("url")
	if url == "" {
		fmt.Fprint(w, AddForm)
		return
	}
	key := store.Put(url)
	// 将 localhost:8080 替换成 *hostname
	fmt.Fprintf(w, "http://%s/%s", *hostname, key)
}
