package main

import (
	"net/http"
	"github.com/gorilla/websocket"
	"errors"
	"fmt"
	"time"
	"sync"
	"bytes"
	"encoding/binary"
	"strings"
	"./BaiduAi"
	"./Response"
	//"./Util"
)

// http升级websocket协议的配置
var wsUpgrader = websocket.Upgrader{
	// 允许所有CORS跨域请求
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// 客户端读写消息
type wsMessage struct {
	messageType int
	data []byte
}

// 客户端连接
type wsConnection struct {
	wsSocket *websocket.Conn // 底层websocket
	inChan chan *wsMessage	// 读队列
	outChan chan *wsMessage // 写队列

	mutex sync.Mutex	// 避免重复关闭管道
	isClosed bool
	closeChan chan byte  // 关闭通知
}

func (wsConn *wsConnection)wsReadLoop() {
	
	for {
		// 读一个message
		msgType, data, err := wsConn.wsSocket.ReadMessage()
		if err != nil {
			goto error
		}
		
		req := &wsMessage{
			msgType,
			data,
		}
		// 放入请求队列
		select {
		case wsConn.inChan <- req:
		case <- wsConn.closeChan:
			goto closed
		}
	}
error:
	wsConn.wsClose()
closed:
}

func (wsConn *wsConnection)wsWriteLoop() {
	for {
		select {
		// 取一个应答
		case msg := <- wsConn.outChan:
			// 写给websocket
			if err := wsConn.wsSocket.WriteMessage(msg.messageType, msg.data); err != nil {
				goto error
			}
		case <- wsConn.closeChan:
			goto closed
		}
	}
error:
	wsConn.wsClose()
closed:
}

func sendAudio(wsConn *wsConnection, text string){
	var buffer bytes.Buffer 

	buffer.Write([]byte {0})
	audioBuffer, err := BaiduAi.Text2Speech(text)
	if err != nil {
		panic(err)
	}
	
	buffer.Write(audioBuffer)
	err = wsConn.wsWrite(websocket.BinaryMessage, buffer.Bytes())
	if err != nil {
		fmt.Println("write fail")
	}	
}

func sendCmd(wsConn *wsConnection, text []byte){
	var buffer bytes.Buffer 

	buffer.Write([]byte {1})
	
	buffer.Write(text)
	err := wsConn.wsWrite(websocket.BinaryMessage, buffer.Bytes())
	if err != nil {
		fmt.Println("write fail")
	}	
}

func handeRecognise(wsConn *wsConnection, text string){
	var err error
	
	if (strings.Contains(text, "play")) { //play
		sendCmd(wsConn, Response.PlayMusic())
	} else if (strings.Contains(text, "next")) {
		sendCmd(wsConn, Response.NextMusic())
	} else if (strings.Contains(text, "pre")) {
		sendCmd(wsConn, Response.PreMusic())
	} else if (strings.Contains(text, "pause")){ //pause
		sendCmd(wsConn, Response.PauseMusic())
	} else if (strings.Contains(text, "resume")){ //resume
		sendCmd(wsConn, Response.ResumeMusic())
	} else if (strings.Contains(text, "stop")){ //stop
		sendCmd(wsConn, Response.StopMusic())
	} else if (strings.Contains(text, "your name")){
		tex := "My name is test"
		sendAudio(wsConn, tex)
	} else { 
		tex := "what's " + text
		sendAudio(wsConn, tex)
	}
	
	if err != nil {
		fmt.Println("write fail")
	}	
}

func (wsConn *wsConnection)procLoop() {
	// 启动一个gouroutine发送心跳
	/*go func() {
		for {
			time.Sleep(2 * time.Second)
			if err := wsConn.wsWrite(websocket.TextMessage, []byte("heartbeat from server")); err != nil {
				fmt.Println("heartbeat fail")
				wsConn.wsClose()
				break
			}
		}
	}()*/
	
	start := false
	end := false
	buf := new(bytes.Buffer) 
	
	for {
		msg, err := wsConn.wsRead()
		if err != nil {
			if start == false {
				continue
			} else {
				end = true
			}
		} else {
			start = true
		}
		 
		if end == false { 
			err = binary.Write(buf, binary.BigEndian, msg.data) 
			if err != nil { 
			  panic(err) 
			}
		} else {
			body, err := BaiduAi.Speech2Text(buf, "16000")
			if err != nil { 
			  panic(err) 
			} 
		
			var tex string
			if body.Err_no == 0 {
				fmt.Println(body.Result[0])
				tex = body.Result[0]
				handeRecognise(wsConn, tex)
			} else {
				tex = "I don't know what you're talking about"
				var buffer bytes.Buffer 

				buffer.Write([]byte {0})

				audioBuffer, err := BaiduAi.Text2Speech(tex)
				if err != nil {
					panic(err)
				}
				
				buffer.Write(audioBuffer)

				err = wsConn.wsWrite(websocket.BinaryMessage, buffer.Bytes())
				if err != nil {
					fmt.Println("write fail")
					break
				}
			}
			
			start = false
			end = false
			buf.Reset()
		}
	}
}

func aiHandler(resp http.ResponseWriter, req *http.Request) {
	// 应答客户端告知升级连接为websocket
	wsSocket, err := wsUpgrader.Upgrade(resp, req, nil)
	if err != nil {
		return
	}
	
	aiConn := &wsConnection{
		wsSocket: wsSocket,
		inChan: make(chan *wsMessage, 150000),
		outChan: make(chan *wsMessage, 150000),
		closeChan: make(chan byte),
		isClosed: false,
	}

	// 处理器
	go aiConn.procLoop()
	// 读协程
	go aiConn.wsReadLoop()
	// 写协程
	go aiConn.wsWriteLoop()
}

func (wsConn *wsConnection)wsWrite(messageType int, data []byte) error {
	select {
	case wsConn.outChan <- &wsMessage{messageType, data,}:
	case <- wsConn.closeChan:
		return errors.New("websocket closed")
	}
	return nil
}

func (wsConn *wsConnection)wsRead() (*wsMessage, error) {
	select {
	case msg := <- wsConn.inChan:
		return msg, nil
	case <- wsConn.closeChan:
	case <-time.After(400 * time.Millisecond):
		//fmt.Println("timeout")
		return nil, errors.New("timeout")
	}
	return nil, errors.New("websocket closed")
}

func (wsConn *wsConnection)wsClose() {
	wsConn.wsSocket.Close()

	wsConn.mutex.Lock()
	defer wsConn.mutex.Unlock()
	if !wsConn.isClosed {
		wsConn.isClosed = true
		close(wsConn.closeChan)
	}
}

func main() {
	//http.HandleFunc("/ai", aiHandler)
	//http.ListenAndServe("0.0.0.0:7777", nil)
	
	//websocket + openssl
	http.HandleFunc("/ai", aiHandler)
	http.ListenAndServeTLS(":8000", "cert.pem", "key.pem", nil) 
}