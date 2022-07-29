package protocols

import (
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/DanielRenne/GoCore/core/atomicTypes"
	"github.com/DanielRenne/GoCore/core/extensions"
	goLogger "github.com/DanielRenne/GoCore/core/logger"
	"github.com/davidrenne/tcpkeepalive"
	"github.com/gorilla/websocket"
)

type WebSocket struct {
	client          *websocket.Conn
	isConnected     atomicTypes.AtomicBool
	responseHandler ResponseHandler
	id              string
	sync.RWMutex

	//Public Setters
	DisableWriteTimeout bool
}

type ResponseHandler func([]byte, error)

func (self *WebSocket) Connect(url string, handler ResponseHandler, disableKeepAlive bool, longKeepAlive bool, id string) (err error, resp *http.Response) {
	self.id = id
	netDial := func(network, addr string) (conn net.Conn, err error) {
		conn, err = net.DialTimeout(network, addr, time.Duration(3000)*time.Millisecond)
		return
	}
	// var err error
	dialer := *websocket.DefaultDialer
	if strings.Index(url, "wss://") != -1 {
		dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	dialer.HandshakeTimeout = time.Duration(3000) * time.Millisecond
	dialer.NetDial = netDial
	self.responseHandler = handler

	client, resp, err := dialer.Dial(url, nil)

	if err != nil {
		log.Println("Error->protocols->webSocket.Connect->Failed to  dial:  " + url + "   " + err.Error())
		return
	}

	self.client = client

	self.SetIsConnected(true)

	_, ok := self.client.UnderlyingConn().(*tls.Conn)

	if !disableKeepAlive && !ok {
		kaConn, errKeepAlive := tcpkeepalive.EnableKeepAlive(self.client.UnderlyingConn())
		if errKeepAlive == nil {
			if longKeepAlive {
				kaConn.SetKeepAliveIdle(6 * time.Second)
				kaConn.SetKeepAliveCount(10)
				kaConn.SetKeepAliveInterval(3 * time.Second)
			} else {
				kaConn.SetKeepAliveIdle(2 * time.Second)
				kaConn.SetKeepAliveCount(2)
				kaConn.SetKeepAliveInterval(2 * time.Second)
			}
		} else {
			log.Println("Error->protocols->webSocket.Connect->Failed to set Keep Alive on Web Socket Connection:  " + url + "   " + errKeepAlive.Error())
		}
	}

	go goLogger.GoRoutineLogger(func() {
		// do setup work
		defer func() {
			if r := recover(); r != nil {
				log.Println("\n\nPanic Stack: " + string(debug.Stack()))
				self.SetIsConnected(false)
				return
			}
		}()
		for {
			_, message, err := self.client.ReadMessage()
			if err != nil {
				log.Println("Websocket->Connect-> Error Reading Message:" + err.Error())
				self.Close()
				handler(message, err)
				break

			}

			handler(message, nil)
		}
	}, "Reader->"+id)

	return nil, nil

}

func (WebSocket) GenerateResponseDetails(resp *http.Response) (details string) {
	if resp == nil {
		return
	}
	details += "StatusCode : " + extensions.IntToString(resp.StatusCode) + "\n"
	details += "Upgrade Header : " + resp.Header.Get("Upgrade") + "\n"
	details += "Upgrade Header : " + resp.Header.Get("Connection") + "\n"
	details += "Upgrade Header : " + resp.Header.Get("Sec-Websocket-Accept") + "\n"
	return
}

func (self *WebSocket) Write(payload []byte) (err error) {

	if self == nil || self.client == nil {
		return nil
	}

	self.Lock()
	err = self.client.WriteMessage(websocket.BinaryMessage, payload)
	self.Unlock()
	if err != nil {
		self.SetIsConnected(false)
	} else {
		self.SetIsConnected(true)
	}
	return err
}

func (self *WebSocket) WriteText(payload string) (err error) {

	if self.client != nil {
		self.Lock()
		err = self.client.WriteMessage(websocket.TextMessage, []byte(payload))
		self.Unlock()
		if err != nil {
			self.SetIsConnected(false)
		} else {
			self.SetIsConnected(true)
		}
	}
	return err
}

func (self *WebSocket) WriteJSON(payload interface{}) (err error) {

	if self == nil || self.client == nil {
		return nil
	}

	self.Lock()
	err = self.client.WriteJSON(&payload)
	self.Unlock()
	if err != nil {
		self.SetIsConnected(false)
	} else {
		self.SetIsConnected(true)
	}
	return err
}

func (self *WebSocket) WriteObj(key string, payload interface{}) (err error) {

	if self == nil || self.client == nil {
		return nil
	}

	self.Lock()
	err = self.client.WriteJSON(&payload)
	self.Unlock()
	if err != nil {
		self.SetIsConnected(false)
	} else {
		self.SetIsConnected(true)
	}
	return err
}

func (self *WebSocket) Close() (err error) {
	self.SetIsConnected(false)
	if self.client != nil {
		self.Lock()
		self.client.SetReadDeadline(time.Now().Add(1))
		err = self.client.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		self.client.Close()
		self.Unlock()
		return err
	}
	return
}

func (self *WebSocket) ConnectionStatus() (value bool) {
	value = self.isConnected.Get()
	return
}

func (self *WebSocket) SetIsConnected(value bool) {
	self.isConnected.Set(value)
}
