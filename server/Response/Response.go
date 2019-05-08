package Response

import (
	"encoding/json"
	"fmt"
)

type Play struct {
	Cmd				string  
	Url             string
}

var musicList = [...]string{"http://media.youban.com/dec1015/1292227744707363914.mp3", "http://sc1.111ttt.cn/2018/1/03/13/396131232171.mp3", "http://sc1.111ttt.cn/2018/1/03/13/396131203208.mp3","http://sc1.111ttt.cn/2017/1/11/11/304112002493.mp3"}
var index int = 0  

func PlayMusic() ([]byte) {
	play := Play{ "/playback/play", musicList[index]}
	
	b, err := json.Marshal(play)
    if err != nil {
        fmt.Println("json err:", err)
    }
	
    fmt.Println(string(b))
	
	return b
}

func NextMusic() ([]byte) {
	index++
	if index >= len(musicList) {
		index = 0
	}
	
	play := Play{ "/playback/play", musicList[index]}
	
	b, err := json.Marshal(play)
    if err != nil {
        fmt.Println("json err:", err)
    }
	
    fmt.Println(string(b))
	
	return b
}

func PreMusic() ([]byte) {
	index--
	if index < 0 {
		index = len(musicList) - 1
	}
	
	play := Play{ "/playback/play", musicList[index]}
	
	b, err := json.Marshal(play)
    if err != nil {
        fmt.Println("json err:", err)
    }
	
    fmt.Println(string(b))
	
	return b
}

type Pause struct {
	Cmd				string  
}

func PauseMusic() ([]byte) {
	pause := Pause{ "/playback/pause"}
	
	b, err := json.Marshal(pause)
    if err != nil {
        fmt.Println("json err:", err)
    }
	
    fmt.Println(string(b))
	
	return b
}

type Resume struct {
	Cmd				string  
}

func ResumeMusic() ([]byte) {
	resume := Resume{ "/playback/resume"}
	
	b, err := json.Marshal(resume)
    if err != nil {
        fmt.Println("json err:", err)
    }
	
    fmt.Println(string(b))
	
	return b
}

type Stop struct {
	Cmd				string  
}

func StopMusic() ([]byte) {
	stop := Stop{ "/playback/stop"}
	
	b, err := json.Marshal(stop)
    if err != nil {
        fmt.Println("json err:", err)
    }
	
    fmt.Println(string(b))
	
	return b
}
