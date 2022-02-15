package main

import (
	"github.com/joho/godotenv"
	"log"
	"socketserver/handler"
)

func init() {
	log.SetFlags(log.Lshortfile | log.Ltime | log.Ldate)

	// line 13-16 not necessary if environment variables saved without file format
	err := godotenv.Load("./.env")
	if err != nil {
		log.Fatal(err)
	}

	handler.DisplayTitle()
}

func main() {
	handler.Listen()
}
