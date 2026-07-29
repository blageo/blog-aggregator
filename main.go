package main

import (
	"log"

	"example.com/blog-aggregator/internal/config"
)

func main() {
	cfg, err := config.Read()
	if err != nil {
		log.Fatal(err)
	}
	println(cfg.CurrentUserName)
}
