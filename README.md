# Foundation Library
The Foundation Go Library is your gateway to effortless interaction with the Foundation API. Designed with simplicity and efficiency in mind, this library abstracts away the complexity of direct API calls, providing a clean and intuitive interface for developers.

## Installation

```bash
go get github.com/teleology-io/foundation-go
```

## Usage Example
```go
package main

import (
	"fmt"

	"github.com/teleology-io/foundation-go"
)

func main() {
	f := foundation.Create("https://foundation-api.teleology.io", "<your_api_key>", foundation.Str(""))

	// get realtime updates
	f.Subscribe(func(event string, data interface{}, err error) {
		fmt.Printf("Subscribe: %s, Data: %+v, Error: %+v\n", event, data, err)
	})

	fmt.Println(f.GetEnvironment())
	fmt.Println(f.GetConfiguration())
	fmt.Println(f.GetVariable("open_enrollment", foundation.Str(""), false))
}
```
