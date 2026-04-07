package main

import (
	"fmt"
	"log"

	"github.com/saschazar21/go-web-push-server/vapid"
)

func main() {
	key, err := vapid.GenerateVapidKey()

	if err != nil {
		log.Fatalf("%v", err)
	}

	var s string
	if s, err = key.EncodeToPEM(true); err != nil {
		log.Fatalf("%v", err)
	}

	fmt.Printf("Private Key PEM:\n\n%s\n\n", s)

	fmt.Printf("Public Key:\n\n%s\n", key.String())
}
