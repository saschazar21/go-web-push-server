package main

import (
	"fmt"
	"log"

	"github.com/saschazar21/go-web-push-server/webpush"
)

func main() {
	key, err := webpush.GenerateVapidKey()

	if err != nil {
		log.Fatalf("%v", err)
	}

	fmt.Println(key.EncodeToPEM(true))
}
