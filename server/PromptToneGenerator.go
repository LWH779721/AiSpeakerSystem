package main

import(
	"./BaiduAi"
	"./Util"
	"os"
	"fmt"
)

func main(){
	tex := "I don't know what you're talking about"
	
	if (len(os.Args) > 1){
		tex = os.Args[1];
	}
	
	fmt.Println(tex)
	
	audioBuffer, err := BaiduAi.Text2Speech(tex)
	if err != nil {
		panic(err)
	}
	
	Util.GeneratorMp3file(audioBuffer, "prompt")
}