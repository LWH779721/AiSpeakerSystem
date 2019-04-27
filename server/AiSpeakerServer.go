package main

import (
	"net/http"
	"github.com/gorilla/websocket"
	"errors"
	"fmt"
	"sync"
	"os"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"strings"
	"./BaiduAi"
)

// http升级websocket协议的配置
var wsUpgrader = websocket.Upgrader{
	// 允许所有CORS跨域请求
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var audioBuf = new(bytes.Buffer)

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
		
		err = binary.Write(audioBuf, binary.BigEndian, data) 
		if err != nil { 
		  panic(err) 
		}
		
		//因为百度 http 接口是上传整个文件，需要包含整句话语
		if audioBuf.Len() > 1024*200 {
			req := &wsMessage{
				msgType,
				audioBuf.Bytes(),
			}
			// 放入请求队列
			select {
			case wsConn.inChan <- req:
			case <- wsConn.closeChan:
				goto closed
			}
			
			audioBuf.Reset()
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

type ASR struct {
	Corpus_no    	string  `json:"corpus_no"`
	Err_msg        	string  `json:"err_msg"`
	Err_no    		int 	`json:"err_no"`
	Result    		[]string `json:"result"`
	Sn    			string `json:"sn"`
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
	
	i := 0

	// 这是一个同步处理模型（只是一个例子），如果希望并行处理可以每个请求一个gorutine，注意控制并发goroutine的数量!!!
	for {
		msg, err := wsConn.wsRead()
		if err != nil {
			fmt.Println("read fail")
			break
		}
		
		i++
		f, err := os.Create(string(i) + ".pcm")
		defer f.Close()
		f.Write(msg.data)
	
		buf := new(bytes.Buffer) 
		err = binary.Write(buf, binary.BigEndian, msg.data) 
		if err != nil { 
		  panic(err) 
		}

		body, err := BaiduAi.Speech2Text(buf, "16000")
		if err != nil { 
		  panic(err) 
		} 
		
		ASRE := &ASR{}
		fmt.Println(string(body))
		if err := json.Unmarshal(body, &ASRE); err == nil {
			var tex string
			if ASRE.Err_no == 0 {
				fmt.Println(ASRE.Result[0])
				tex = ASRE.Result[0] + "是什么？"
			} else {
				tex = "I don't know what you're talking about"
			}
			
			if (strings.Contains(tex, "play")) { //play
				var data = []byte { 0, 0}
				err = wsConn.wsWrite(msg.messageType, data)
			} else if (strings.Contains(tex, "pause")){ //pause
				var data = []byte { 1, 0}
				err = wsConn.wsWrite(msg.messageType, data)
			} else if (strings.Contains(tex, "resume")){ //resume
				var data = []byte { 2, 0}
				err = wsConn.wsWrite(msg.messageType, data)
			} else if (strings.Contains(tex, "stop")){ //stop
				var data = []byte { 3, 0}
				err = wsConn.wsWrite(msg.messageType, data)
			} else { // Not recognized
				var data = []byte { 4, 0}
				err = wsConn.wsWrite(msg.messageType, data)
			}
			
			if err != nil {
				fmt.Println("write fail")
				break
			}
				
			/*texEn, err := url.Parse(tex)
			//fmt.Println(texEn)
			body, err := BaiduAi.Text2Speech(texEn)
			if err != nil {
				panic(err)
			}
			
			err = wsConn.wsWrite(msg.messageType, body)
			if err != nil {
				fmt.Println("write fail")
				break
			}*/
		}
	}
}

func wsHandler(resp http.ResponseWriter, req *http.Request) {
	// 应答客户端告知升级连接为websocket
	wsSocket, err := wsUpgrader.Upgrade(resp, req, nil)
	if err != nil {
		return
	}
	wsConn := &wsConnection{
		wsSocket: wsSocket,
		inChan: make(chan *wsMessage, 1000),
		outChan: make(chan *wsMessage, 1000),
		closeChan: make(chan byte),
		isClosed: false,
	}

	// 处理器
	go wsConn.procLoop()
	// 读协程
	go wsConn.wsReadLoop()
	// 写协程
	go wsConn.wsWriteLoop()
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
	http.HandleFunc("/ws", wsHandler)
	http.ListenAndServe("0.0.0.0:7777", nil)
}