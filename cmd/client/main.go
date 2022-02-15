package main

import (
	"github.com/joho/godotenv"
	"log"
	"net"
	"strconv"
	"sync"
	"time"
)
import "fmt"
import "bufio"
import "os"

func init() {
	log.SetFlags(log.Lshortfile | log.Ltime | log.Ldate)
	err := godotenv.Load("./.env")
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	log.SetFlags(log.Lshortfile | log.Ldate | log.Ltime)
	tcpHost := os.Getenv("TCP_HOST_NAME")
	addr := os.Getenv("TCP_PORT")
	fakeClients := os.Getenv("FAKE_CLIENTS")

	clients, _ := strconv.ParseInt(fakeClients, 10, 64)

	wg := sync.WaitGroup{}
	wg.Add(int(clients))
	for i := int64(1); i <= clients; i++ {

		num := ""
		if i < 10 {
			num = fmt.Sprintf("00%d", i)
		} else if i < 100 {
			num = fmt.Sprintf("0%d", i)
		} else {
			num = fmt.Sprintf("%d", i)
		}
		time.Sleep(100 * time.Microsecond)
		go tcpClient(tcpHost, addr, num)
	}

	wg.Wait()
}

func tcpClient(host, addr, num string) {
	// connect to server
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", host, addr))
	if err != nil {
		if num == "001" || num == "999" || num == "9999" {
			log.Println(num, err)
		}
	} else {
		_, err := fmt.Fprintf(conn, fmt.Sprintf("s0m4k3y.Worker%s", num))
		if err != nil {
			if num == "001" || num == "999" || num == "8000" {
				log.Println(num, err)
			}
		} else {
			for {
				// wait for reply
				message, err := bufio.NewReader(conn).ReadString('\n')
				if err == nil {
					fmt.Printf("ID: %s Message from server: %s", num, message)
				}
			}
		}
	}
}
