package BaiduAi

import (
	"net/http"
	"net/url"
	"io"
	"io/ioutil"
	"errors"
	"fmt"
	"encoding/json"
	"strings"
)

type ASR struct {
	Corpus_no    	string  `json:"corpus_no"`
	Err_msg        	string  `json:"err_msg"`
	Err_no    		int 	`json:"err_no"`
	Result    		[]string `json:"result"`
	Sn    			string `json:"sn"`
}

type TokenResult struct {
	Access_token 	string
	Expires_in      int
	Refresh_token   string
	Scope			string
	Session_key     string
	Session_secret  string 
}

var accessToken string

func GetAccessToken(api_key string, secret_key string) (string, error){
	req, err := http.NewRequest("GET", "https://openapi.baidu.com/oauth/2.0/token?grant_type=client_credentials&client_id="+ api_key + "&client_secret=" + secret_key + "&", nil) 
	if err != nil { 
	  return "", err 
	} 
	
	var httpclient *http.Client = &http.Client{} 

	resp, err := httpclient.Do(req)
	if err != nil { 
	  return "", err 
	} 
	
	defer resp.Body.Close() 

	var r io.Reader = resp.Body 

	body, err := ioutil.ReadAll(r) 
	if err != nil { 
	  return "", err 
	} 
	if resp.StatusCode != 200 { 
	  msg := fmt.Sprintf("handlePost statuscode=%d, body=%s", resp.StatusCode, body) 
	  return "", errors.New(msg) 
	}
	
	//fmt.Println(string(body))
	Token := &TokenResult{}
	err = json.Unmarshal(body, &Token)
	if err != nil {
		return "", nil
	}
	
	return Token.Access_token, nil	
}

func Speech2Text(bodys io.Reader, rate string) (*ASR, error) { 

	req, err := http.NewRequest("POST", "http://vop.baidu.com/server_api?cuid=80001&token="+ accessToken +"&lan=en", bodys) 
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
	
	ASRE := &ASR{}
	fmt.Println(string(body))
	err = json.Unmarshal(body, &ASRE)
	if err == nil {
		return ASRE, nil
	}
	
	return nil, nil 
} 

func Text2Speech(text string) ([]byte, error) {
	textEncode, err := url.Parse(text)
	fmt.Println(textEncode)
	
	req, err := http.NewRequest("GET", "http://tsn.baidu.com/text2audio?tex=" + textEncode.EscapedPath() + "&lan=zh&cuid=80001&ctp=1&aue=3&per=4&spd=7&tok=" + accessToken, nil)
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
	
	header := resp.Header
	if strings.Contains(header["Content-Type"][0], "audio") != true {
		fmt.Println(string(body))
		msg := fmt.Sprintf("handlePost statuscode=%d, body=%s", resp.StatusCode, body)
		return nil, errors.New(msg)
	}
	
	if resp.StatusCode != 200 {
		msg := fmt.Sprintf("handlePost statuscode=%d, body=%s", resp.StatusCode, body)
		return nil, errors.New(msg)
	}
	
	return body, nil
}

func init() {
	var err error
	
    accessToken, err = GetAccessToken("kvx3hZ4jNIEGHfWDzzFBeHX9", "UsKQnY1e3OYU66bMTVymi5tqQh1AMqH6")
	if err == nil {
		fmt.Println(accessToken)
	}
}