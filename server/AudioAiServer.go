package main

import (
	"net/http"
	"github.com/gorilla/websocket"
	"errors"
	"fmt"
	"sync"
	"io"
	"os"
	"bytes"
	"encoding/binary" 
	"io/ioutil"
	"encoding/json"
	"net/url"
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

func getText(bodys io.Reader, rate string) ([]byte, error) { 

	req, err := http.NewRequest("POST", "http://vop.baidu.com/server_api?dev_pid=1536&cuid=80001&token=24.8c2ab6a8feca6534d5fcd68346e13203.2592000.1558181252.282335-16055618", bodys) 
	if err != nil { 
	  return nil, err 
	} 
	
	var httpclient *http.Client = &http.Client{} 
	req.Header.Add("Content-Type", "audio/pcm;rate="+rate) 
	resp, err := httpclient.Do(req) 

	if err != nil { 
	  return nil, err 
	} 
	defer resp.Body.Close() 

	var r io.Reader = resp.Body 

	body, err := ioutil.ReadAll(r) 
	if err != nil { 
	  return nil, err 
	} 
	if resp.StatusCode != 200 { 
	  msg := fmt.Sprintf("handlePost statuscode=%d, body=%s", resp.StatusCode, body) 
	  return nil, errors.New(msg) 
	} 
	return body, nil 
} 

func Text2Speech(tex *url.URL) ([]byte, error) {
	
	req, err := http.NewRequest("GET", "http://tsn.baidu.com/text2audio?tex=" + tex.EscapedPath() + "&lan=zh&cuid=80001&ctp=1&tok=24.349f1237eb7b5b64c4d7571a824c46b4.2592000.1558182477.282335-16055618", nil)
	if err != nil {
		return nil, err
	}

	var httpclient *http.Client = &http.Client{}
	resp, err := httpclient.Do(req)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var r io.Reader = resp.Body

	body, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		msg := fmt.Sprintf("handlePost statuscode=%d, body=%s", resp.StatusCode, body)
		return nil, errors.New(msg)
	}
	return body, nil
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

		body, err := getText(buf, "16000")
		if err != nil { 
		  panic(err) 
		} 
		
		ASRE := &ASR{}
		fmt.Println(string(body))
		if err := json.Unmarshal(body, &ASRE); err == nil {
			if ASRE.Err_no == 0 {
				fmt.Println(ASRE.Result[0])
				tex := ASRE.Result[0] + "是什么？"
				texEn, err := url.Parse(tex)
				//fmt.Println(texEn)
				
				body, err := Text2Speech(texEn)
				if err != nil {
					panic(err)
				}
				
				err = wsConn.wsWrite(msg.messageType, body)
				if err != nil {
					fmt.Println("write fail")
					break
				}
			}
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