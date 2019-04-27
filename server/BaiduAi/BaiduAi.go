package BaiduAi

import (
	"net/http"
	"net/url"
	"io"
	"io/ioutil"
	"errors"
	"fmt"
)

func Speech2Text(bodys io.Reader, rate string) ([]byte, error) { 

	req, err := http.NewRequest("POST", "http://vop.baidu.com/server_api?cuid=80001&token=24.8c2ab6a8feca6534d5fcd68346e13203.2592000.1558181252.282335-16055618&lan=en", bodys) 
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
	
	req, err := http.NewRequest("GET", "http://tsn.baidu.com/text2audio?tex=" + tex.EscapedPath() + "&lan=en&cuid=80001&ctp=1&tok=24.349f1237eb7b5b64c4d7571a824c46b4.2592000.1558182477.282335-16055618", nil)
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

func init() {
    fmt.Println("init")
}