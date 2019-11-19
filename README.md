### xReq
xReq is a easy-to-use, flexiable and extendable Go HTTP Client. 

**Quick Start**
```golang
package main

import (
	"fmt"

	"github.com/ehyyoj/xreq"
)

func main() {
	data, code, err := xreq.GetBytes("http://localhost:8080/hello",
            xreq.WithQueryValue("name", "jack"),
            xreq.WithSetHeader("x-request-id", "123"),
	)
	if err != nil {
		panic(err)
	}
	fmt.Printf("response body: %s, status code: %d", string(data), code)
}
```

**Post a JSON request**
```golang
import (
	"fmt"
	"time"

	"github.com/ehyyoj/xreq"
)

type User struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func main() {
	user := &User{
		Name: "jack",
		Age:  18,
	}

	// use context.Context to set request timeout.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()
	data, code, err := xreq.DoBytes("http://localhost:8080/hello",
		xreq.WithPostJSON(user),
		xreq.WithContext(ctx), 
	)
	fmt.Println("response:", string(data), code, err)
}
```

**Send Post form with header**
```golang
package main

import (
	"fmt"
	"time"

	"github.com/ehyyoj/xreq"
)

func main() {
	client := xreq.NewClient(xreq.Config{
		Retry:   3,
		Timeout: time.Second * 2,
	}, xreq.WithCheckStatus(true))
	params := make(map[string]string)
	params["name"] = "jack"
	params["age"] = "18"
	data, code, err := client.DoBytes("http://localhost:8080/hello",
		xreq.WithPostForm(params),
		xreq.WithSetHeader("x-request-id", "123"),
		xreq.WithQueryValue("pageIndex", "2"),
		xreq.WithQueryValue("pageSize", "10"),
	)
	fmt.Println("response:", string(data), code, err)
}
```
