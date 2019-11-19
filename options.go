package xreq

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	urlpkg "net/url"
	"strings"
)

// Option is a type define use for pass closure as parameters.
type Option func(o *Options)

// Options define some option of HTTP.
type Options struct {
	*http.Request

	Err    error
	Values urlpkg.Values

	checkStatus bool
}

// WithHeader set up the entire http.Header.
func WithHeader(header http.Header) Option {
	return func(o *Options) {
		o.Request.Header = header
	}
}

// WithSetHeader set key-value into http.Header.
func WithSetHeader(k, v string) Option {
	return func(o *Options) {
		o.Request.Header.Set(k, v)
	}
}

// WithContext set context to the http.Request
// it use to timeout or cancel.
//
// Example:
//
// ctx, cancel := context.WithCancel(context.Background(), time.Second*3)
// defer cancel()
// data, _, err := xreq.GetBytes(url,
//     WithContext(ctx),
// )
func WithContext(ctx context.Context) Option {
	return func(o *Options) {
		o.Request = o.Request.Clone(ctx)
	}
}

// WithMethod set the http method.
func WithMethod(method string) Option {
	return func(o *Options) {
		o.Request.Method = method
	}
}

// WithBodyBytes set []byte into the request body.
func WithBodyBytes(contentType string, data []byte) Option {
	return WithBodyReader(contentType, bytes.NewBuffer(data))
}

// WithBodyString set string into the request body.
func WithBodyString(contentType string, body string) Option {
	return WithBodyReader(contentType, bytes.NewBufferString(body))
}

// WithBodyReader set io.Reader into the request body.
func WithBodyReader(contentType string, body io.Reader) Option {
	return func(o *Options) {
		req := o.Request
		req.Header.Set("Content-Type", contentType)
		setBody(req, body)
	}
}

func setBody(req *http.Request, body io.Reader) {
	req.Body = ioutil.NopCloser(body)
	switch v := body.(type) {
	case *bytes.Buffer:
		req.ContentLength = int64(v.Len())
		buf := v.Bytes()
		req.GetBody = func() (io.ReadCloser, error) {
			r := bytes.NewReader(buf)
			return ioutil.NopCloser(r), nil
		}
	case *bytes.Reader:
		req.ContentLength = int64(v.Len())
		snapshot := *v
		req.GetBody = func() (io.ReadCloser, error) {
			r := snapshot
			return ioutil.NopCloser(&r), nil
		}
	case *strings.Reader:
		req.ContentLength = int64(v.Len())
		snapshot := *v
		req.GetBody = func() (io.ReadCloser, error) {
			r := snapshot
			return ioutil.NopCloser(&r), nil
		}
	default:
		// From http.NewRequestWithContext() comment:
		// This is where we'd set it to -1 (at least
		// if body != NoBody) to mean unknown, but
		// that broke people during the Go 1.8 testing
		// period. People depend on it being 0 I
		// guess. Maybe retry later. See Issue 18117.
	}
}

// WithQuery set the URL query
func WithQuery(params map[string]string) Option {
	return func(o *Options) {
		for k, v := range params {
			o.Values.Set(k, v)
		}
	}
}

// WithQueryValue set key-value into query.
// Example:
//
// body, err :=	Get("http://localhost/api",
//			WithQuery("name", "jack"),
//			WithQuery("id", "18"))
// and the request URL will be "http://localhost/api?name=jack&id=18"
func WithQueryValue(key, value string) Option {
	return func(o *Options) {
		o.Values.Set(key, value)
	}
}

// WithPostForm set the entire post form
func WithPostForm(params map[string]string) Option {
	return func(o *Options) {
		vals := make(urlpkg.Values)
		for k, v := range params {
			vals.Set(k, v)
		}

		o.Request.Method = http.MethodPost
		o.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		body := strings.NewReader(vals.Encode())
		setBody(o.Request, body)
	}
}

// WithPostJSON marshal v to the JSON bytes and set to the request body.
func WithPostJSON(v interface{}) Option {
	return func(o *Options) {
		data, err := json.Marshal(v)
		if err != nil {
			o.Err = fmt.Errorf("json marshal error: %w", err)
			return
		}

		o.Request.Method = http.MethodPost
		o.Request.Header.Set("Content-Type", "application/json")
		body := bytes.NewBuffer(data)
		setBody(o.Request, body)
	}
}

// WithAddCookie set http.Cookie.
func WithAddCookie(cookie *http.Cookie) Option {
	return func(o *Options) {
		if cookie != nil {
			o.Request.AddCookie(cookie)
		}
	}
}

// WithRequest replace the http.Request entirely.
func WithRequest(req *http.Request) Option {
	return func(o *Options) {
		o.Request = req
	}
}

// WithCheckStatus treat non-200 as error
// NOTE it only effected which method with bytes return,
// method with *http.Response return does not effected.
func WithCheckStatus(check bool) Option {
	return func(o *Options) {
		o.checkStatus = check
	}
}

// WithMultipart set the multipart/form-data without file.
func WithMultipart(params map[string]string) Option {
	return func(o *Options) {
		buf := new(bytes.Buffer)
		writer := multipart.NewWriter(buf)
		for k, v := range params {
			if err := writer.WriteField(k, v); err != nil {
				o.Err = fmt.Errorf("write field error: %w", err)
				return
			}
		}
		if err := writer.Close(); err != nil {
			o.Err = fmt.Errorf("writer close error: %w", err)
			return
		}

		o.Request.Header.Set("Content-Type", writer.FormDataContentType())
		o.Request.Method = http.MethodPost
		setBody(o.Request, buf)
	}
}

// WithWithMultipartFile use multipart/form-data format to upload file.
func WithMultipartFile(fieldname, filename string, data []byte, params ...map[string]string) Option {
	return func(o *Options) {
		buf := new(bytes.Buffer)
		writer := multipart.NewWriter(buf)

		if len(params) > 0 {
			for k, v := range params[0] {
				if err := writer.WriteField(k, v); err != nil {
					o.Err = fmt.Errorf("write field error: %w", err)
					return
				}
			}
		}

		part, err := writer.CreateFormFile(fieldname, filename)
		if err != nil {
			o.Err = fmt.Errorf("create form file error: %w", err)
			return
		}
		if _, err = part.Write(data); err != nil {
			o.Err = fmt.Errorf("write form file error: %w", err)
			return
		}
		if err = writer.Close(); err != nil {
			o.Err = fmt.Errorf("writer close error: %w", err)
			return
		}

		o.Request.Header.Set("Content-Type", writer.FormDataContentType())
		o.Request.Method = http.MethodPost
		setBody(o.Request, buf)
	}
}
