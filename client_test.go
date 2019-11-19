package xreq_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	. "github.com/ehyyoj/xreq"

	"github.com/stretchr/testify/assert"
)

const (
	host = "http://localhost:8080"
)

func init() {
	mux := http.NewServeMux()
	mux.HandleFunc("/query_params", queryParams)
	mux.HandleFunc("/post_form", postForm)
	mux.HandleFunc("/post_json", postJSON)
	mux.HandleFunc("/not_found", notFound)
	mux.HandleFunc("/internal_error", internalError)
	mux.HandleFunc("/set_header", setHeader)
	mux.HandleFunc("/upload_file", uploadFile)
	mux.HandleFunc("/multipart", multipart)
	mux.HandleFunc("/post_chunk", postChunk)
	go func() {
		if err := http.ListenAndServe(":8080", mux); err != nil {
			panic(err)
		}
	}()
	time.Sleep(time.Millisecond * 100)
}

func queryParams(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Write([]byte(r.URL.Query().Encode()))
}

func postForm(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		panic(err)
	}
	val := r.PostForm.Encode()
	w.WriteHeader(200)
	w.Write([]byte(val))
}

func postJSON(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	w.Write(data)
}

func notFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(404)
	w.Write([]byte("hello"))
}

func internalError(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(500)
	w.Write([]byte("internal error"))
}

func setHeader(w http.ResponseWriter, r *http.Request) {
	for _, c := range r.Cookies() {
		http.SetCookie(w, c)
	}

	for k, v := range r.Header {
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}
}

func uploadFile(w http.ResponseWriter, r *http.Request) {
	file, fh, err := r.FormFile("upload_file")
	if err != nil {
		panic(err)
	}
	w.Header().Set("filename", fh.Filename)
	w.Header().Set("size", strconv.Itoa(int(fh.Size)))
	data, err := ioutil.ReadAll(file)
	if err != nil {
		panic(err)
	}
	w.Header().Set("name", r.FormValue("name"))
	w.Header().Set("age", r.FormValue("age"))
	w.Write(data)
}

func multipart(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	age := r.FormValue("age")
	w.Write([]byte(fmt.Sprintf("name=%s&age=%s", name, age)))
}

func postChunk(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hello world"))
	w.(http.Flusher).Flush()
	time.Sleep(100 * time.Millisecond)
}

func TestTimeout(t *testing.T) {
	cli := NewClient(Config{
		Timeout: 1,
	})
	resp, err := cli.Get(host + "/query_params")

	err = errors.Unwrap(err)
	if e := err.(net.Error); !e.Timeout() {
		t.Errorf("should be network timeout")
	}
	if err == nil {
		resp.Body.Close()
	}
}

func TestGet(t *testing.T) {
	data, code, err := GetBytes(host+"/query_params?name=abc",
		WithQueryValue("age", "18"),
	)
	assert.Nil(t, err)
	assert.Equal(t, 200, code)
	if string(data) != "name=abc&age=18" && string(data) != "age=18&name=abc" {
		t.Errorf("actual: %s", string(data))
	}

	data, code, err = GetBytes(host+"/query_params",
		WithQueryValue("name", "abc"),
		WithQueryValue("age", "18"),
	)
	assert.Nil(t, err)
	assert.Equal(t, 200, code)
	if string(data) != "name=abc&age=18" && string(data) != "age=18&name=abc" {
		t.Errorf("actual: %s", string(data))
	}
}

func TestQuery(t *testing.T) {
	tests := []map[string]string{
		{
			"name": "jack",
			"age":  "18",
		},
		{
			"address": "深圳",
			"group":   "技术部",
		},
	}
	expected := [][]string{
		{"name=jack&age=18", "age=18&name=jack"},
		{"address=深圳&group=技术部", "group=技术部&address=深圳"},
	}

	for i, m := range tests {
		data, _, err := DoBytes(host+"/query_params",
			WithQuery(m),
		)
		assert.Nil(t, err)
		v, err := url.QueryUnescape(string(data))
		assert.Nil(t, err)

		var ok bool
		for _, e := range expected[i] {
			if v == e {
				ok = true
				break
			}
		}
		if !ok {
			t.Errorf("actual: %s, unescape: %s", string(data), v)
		}
	}
}

func TestPostForm(t *testing.T) {
	tests := []map[string]string{
		{
			"name": "jack",
			"age":  "18",
		},
		{
			"address": "深圳",
			"group":   "技术部",
		},
	}
	expected := [][]string{
		{"name=jack&age=18", "age=18&name=jack"},
		{"address=深圳&group=技术部", "group=技术部&address=深圳"},
	}

	for i, m := range tests {
		data, _, err := DoBytes(host+"/post_form",
			WithPostForm(m),
		)
		assert.Nil(t, err)
		v, err := url.QueryUnescape(string(data))
		assert.Nil(t, err)

		var ok bool
		for _, e := range expected[i] {
			if v == e {
				ok = true
				break
			}
		}
		if !ok {
			t.Errorf("actual: %s, unescape: %s", string(data), v)
		}
	}
}

func TestPostJSON(t *testing.T) {
	tests := []map[string]interface{}{
		{
			"name": "jack",
			"age":  18,
		},
		{
			"code": 0,
			"msg":  "success",
			"data": map[string]interface{}{
				"user": map[string]interface{}{
					"name": "jack",
					"age":  18,
				},
			},
		},
	}

	for _, m := range tests {
		data, _, err := DoBytes(host+"/post_json",
			WithPostJSON(m),
		)
		assert.Nil(t, err)

		expected, err := json.Marshal(m)
		assert.Nil(t, err)
		assert.Equal(t, string(expected), string(data))
	}
}

func TestCheckStatus(t *testing.T) {
	data, code, err := GetBytes(host+"/not_found",
		WithCheckStatus(true),
	)
	assert.NotNil(t, err)
	assert.Equal(t, 404, code)
	assert.Equal(t, "hello", string(data))
	assert.Equal(t, "http status code: 404", err.Error())
}

func TestJSONError(t *testing.T) {
	data, _, err := DoBytes(host+"/internal_error",
		WithPostJSON(make(chan int)),
	)
	err = errors.Unwrap(err)
	err = errors.Unwrap(err)
	if _, ok := err.(*json.UnsupportedTypeError); !ok {
		t.Errorf("expected json unsupported type error")
	}
	assert.Nil(t, data)
}

func TestRequest(t *testing.T) {
	_, err := Get(":test")
	assert.NotNil(t, err)

	req, err := http.NewRequest("GET", host+"/query_params", nil)
	assert.Nil(t, err)

	_, err = Get("",
		WithRequest(req),
	)
	assert.Nil(t, err)
}

func TestPostBody(t *testing.T) {
	data, _, err := DoBytes(host+"/post_json",
		WithMethod("POST"),
		WithBodyString("application/json", `{"name": "jack"}`),
	)
	assert.Nil(t, err)
	assert.Equal(t, `{"name": "jack"}`, string(data))

	data, _, err = DoBytes(host+"/post_json",
		WithMethod("POST"),
		WithBodyBytes("application/json", []byte(`{"name": "jack"}`)),
	)
	assert.Nil(t, err)
	assert.Equal(t, `{"name": "jack"}`, string(data))

	resp, err := Do(host+"/post_json",
		WithMethod("POST"),
		WithBodyReader("application/json", strings.NewReader(`{"name": "jack"}`)),
	)
	assert.Nil(t, err)
	data, err = ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	assert.Nil(t, err)
	assert.Equal(t, `{"name": "jack"}`, string(data))
}

func TestPost(t *testing.T) {
	resp, err := Post(host+"/post_json", "application/json", strings.NewReader(`{"name": "jack"}`))
	assert.Nil(t, err)
	data, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	assert.Nil(t, err)
	assert.Equal(t, `{"name": "jack"}`, string(data))

	data, _, err = PostBytes(host+"/post_json", "application/json", strings.NewReader(`{"name": "jack"}`))
	assert.Nil(t, err)
	assert.Equal(t, `{"name": "jack"}`, string(data))

	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	_, _, err = PostBytes(host+"/post_json", "application/json", strings.NewReader(`{"name": "jack"}`),
		WithContext(ctx),
	)
	assert.NotNil(t, err)

	ctx, cancel = context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	_, err = Post(host+"/post_json", "application/json", strings.NewReader(`{"name": "jack"}`),
		WithContext(ctx),
	)
	assert.NotNil(t, err)
}

func TestHeader(t *testing.T) {
	resp, err := Get(host+"/set_header",
		WithSetHeader("name", "jack"),
		WithSetHeader("age", "18"),
	)
	assert.Nil(t, err)
	resp.Body.Close()
	assert.Equal(t, "jack", resp.Header.Get("name"))
	assert.Equal(t, "18", resp.Header.Get("age"))

	header := make(http.Header)
	header.Set("name", "jack")
	header.Set("age", "18")
	resp, err = Get(host+"/set_header",
		WithHeader(header),
	)
	assert.Nil(t, err)
	resp.Body.Close()
	assert.Equal(t, "jack", resp.Header.Get("name"))
	assert.Equal(t, "18", resp.Header.Get("age"))
}

func TestAddCookie(t *testing.T) {
	sess := &http.Cookie{
		Name:  "session",
		Value: "abc",
	}
	user := &http.Cookie{
		Name:  "user",
		Value: "zzz",
	}

	resp, err := Get(host+"/set_header",
		WithAddCookie(sess),
		WithAddCookie(user),
	)
	assert.Nil(t, err)

	found := 0
	for _, c := range resp.Cookies() {
		if c.Name == "session" {
			assert.Equal(t, "abc", c.Value)
			found++
		}
		if c.Name == "user" {
			assert.Equal(t, "zzz", c.Value)
			found++
		}
	}
	assert.Equal(t, 2, found)
}

func TestUploadFile(t *testing.T) {
	params := map[string]string{
		"name": "jack",
		"age":  "18",
	}
	fileStr := "hello world世界！"
	resp, err := Do(host+"/upload_file",
		WithMultipartFile("upload_file", "hello.txt", []byte(fileStr), params),
	)
	assert.Nil(t, err)
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)
	assert.Equal(t, fileStr, string(data))
	assert.Equal(t, "hello.txt", resp.Header.Get("filename"))
	assert.Equal(t, strconv.Itoa(len(fileStr)), resp.Header.Get("size"))
	assert.Equal(t, "jack", resp.Header.Get("name"))
	assert.Equal(t, "18", resp.Header.Get("age"))
}

func TestMultipart(t *testing.T) {
	params := map[string]string{
		"name": "jack",
		"age":  "18",
	}
	resp, err := Do(host+"/multipart",
		WithMultipart(params),
	)
	assert.Nil(t, err)
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)
	assert.Equal(t, "name=jack&age=18", string(data))
}

func TestChunkRead(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()
	data, _, err := GetBytes(host+"/post_chunk",
		WithContext(ctx),
	)
	assert.Nil(t, err)
	assert.Equal(t, "hello world", string(data))
}

func BenchmarkXGet(b *testing.B) {
	for i := 0; i < b.N; i++ {
		resp, err := Get(host + "/query_params?name=jack")
		if err != nil {
			b.Errorf("get error: %s", err)
		}
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}
}

func BenchmarkXGetWithoutDrain(b *testing.B) {
	for i := 0; i < b.N; i++ {
		resp, err := Get(host + "/query_params?name=jack")
		if err != nil {
			b.Errorf("get error: %s", err)
		}
		resp.Body.Close()
	}
}

func BenchmarkStdGet(b *testing.B) {
	for i := 0; i < b.N; i++ {
		resp, err := http.Get(host + "/query_params?name=jack")
		if err != nil {
			b.Errorf("get error: %s", err)
		}
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}
}

func BenchmarkStdGetWithoutDrain(b *testing.B) {
	for i := 0; i < b.N; i++ {
		resp, err := Get(host + "/query_params?name=jack")
		if err != nil {
			b.Errorf("get error: %s", err)
		}
		resp.Body.Close()
	}
}

func BenchmarkStdPostJSON(b *testing.B) {
	body := `{"name": "jack", "age": 18}`
	for i := 0; i < b.N; i++ {
		resp, err := http.Post(host+"/post_json", "application/json", strings.NewReader(body))
		assert.Nil(b, err)
		data, err := ioutil.ReadAll(resp.Body)
		assert.Nil(b, err)
		assert.Equal(b, body, string(data))
		resp.Body.Close()
	}
}

func BenchmarkXPostJSON(b *testing.B) {
	body := `{"name": "jack", "age": 18}`
	for i := 0; i < b.N; i++ {
		data, code, err := DoBytes(host+"/post_json",
			WithBodyString("application/json", body),
		)
		assert.Nil(b, err)
		assert.Equal(b, 200, code)
		assert.Equal(b, body, string(data))
	}
}
