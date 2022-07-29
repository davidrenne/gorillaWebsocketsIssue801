package socketReaders

import (
	"log"
	"runtime/debug"
	"sync"
	"time"

	"gorillaWebSocket/client/protocols"

	"github.com/DanielRenne/GoCore/core"
	"github.com/DanielRenne/GoCore/core/atomicTypes"
	"github.com/DanielRenne/GoCore/core/extensions"
	goLogger "github.com/DanielRenne/GoCore/core/logger"
)

var SocketReaders sync.Map
var count atomicTypes.AtomicInt

func NewReaderSocket(deviceId string) {
	w := Websocket{
		DeviceId: deviceId,
	}
	w.Connect()
	_, ok := SocketReaders.Load(deviceId)
	if !ok {
		SocketReaders.Store(deviceId, &w)
	}

	// testing stuff to write bytes up
	go func() {
		w, ok := SocketReaders.Load(deviceId)
		if ok {
			websocket := w.(*Websocket)
			for {
				time.Sleep(time.Second * 2)
				err := websocket.Write([]byte("hello world: " + deviceId))
				if err != nil {
					return
				}
			}
		}
	}()
}

type Websocket struct {
	conn                 protocols.WebSocket
	DeviceId             string
	explicitlyDisconnect bool
}

//Connect satisfies the Device interface and connects to the devices protocol.
func (obj *Websocket) Connect() {
	go goLogger.GoRoutineLogger(func() {
		defer func() {
			if r := recover(); r != nil {
				log.Println("socketReaders/socket.go Panic Recovered at websocket.connectSocket:" + core.Debug.GetDump(r) + "\n\nPanic Stack: " + string(debug.Stack()))
				if obj.explicitlyDisconnect {
					log.Println("socketReaders/socket.go: tearing down websocket connection and exiting")
					return
				}
				obj.connectSocket()
				return
			}
		}()
		obj.connectSocket()
	}, "socketReader->ConnectSocket")
}

func (obj *Websocket) ExplicitlyDisconnect() {
	obj.explicitlyDisconnect = true
	obj.Disconnect()
}

func (obj *Websocket) Disconnect() {
	obj.conn.Close()
}

func (obj *Websocket) Write(data []byte) (err error) {
	err = obj.conn.Write(data)
	if err != nil {
		log.Println("socketReaders/socket.go Write error: " + err.Error())
	}
	return
}

func (obj *Websocket) OnConnect() {
	log.Println(obj.DeviceId + ": connected")
}

func (obj *Websocket) connectSocket() {
	ipAddress := "0.0.0.0"
	port := extensions.IntToString(8090)
	log.Println("Connecting to " + ipAddress + ":" + port)
	obj.Disconnect()
	var scheme string
	if port != "443" {
		scheme = "ws://"
	} else {
		scheme = "wss://"
	}
	url := scheme + ipAddress + ":" + port + "/ws"
	handleReads := func(payload []byte, errRead error) {
		if errRead != nil {
			if obj.explicitlyDisconnect {
				log.Println("socketReaders/socket.go: Reader dead.  " + errRead.Error() + ". tearing down websocket connection and exiting")
				return
			}
			time.Sleep(time.Millisecond * 2000)
			log.Println("socketReaders/socket.go: Read Error.  Reconnecting: " + errRead.Error())
			obj.connectSocket()
			return
		}
		log.Println("socketReaders/socket.go: Response (Length " + extensions.IntToString(len(payload)) + " bytes):" + string(payload))
	}

	err, resp := obj.conn.Connect(url, handleReads, false, true, obj.DeviceId)
	if err != nil {
		responseDetails := protocols.WebSocket{}.GenerateResponseDetails(resp)
		log.Println("socketReaders/socket.go: failed to connect: " + err.Error() + "\n" + responseDetails)
		if obj.explicitlyDisconnect {
			log.Println("socketReaders/socket.go: connect dead " + err.Error() + ". tearing down websocket connection and exiting")
			return
		}
		time.Sleep(time.Millisecond * 2000)
		obj.connectSocket()
		return
	}
	count.Increment()
	log.Println("total sockets:" + extensions.IntToString(count.Get()))
	obj.OnConnect()
}
