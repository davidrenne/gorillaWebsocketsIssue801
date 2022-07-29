package main

import (
	"gorillaWebSocket/client/socketReaders"
	"time"

	"gopkg.in/mgo.v2/bson"
)

func main() {
	forever := make(chan int, 0)
	// Notice 150 threads on MAX OSX 12.4
	// go func() {
	// 	for i := 1; i <= 150; i++ {
	// 		socketReaders.NewReaderSocket(bson.NewObjectId().Hex())
	// 	}
	// }()

	// this will bomb out on my system because too many threads get opened
	go func() {
		for i := 1; i <= 5000; i++ {
			time.Sleep(time.Millisecond * 10)
			socketReaders.NewReaderSocket(bson.NewObjectId().Hex())
		}
	}()
	<-forever
}
