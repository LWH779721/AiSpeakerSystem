package Util

import (
	"os"
)

var i int = 0

func Save2Fle(audio []byte) (error){ 
	i++
	
	f, err := os.Create(string(i) + ".pcm")
	defer f.Close()
	if err != nil {
		return err
	}
	
	f.Write(audio)
	
	return nil
}

func GeneratorMp3file(mp3 []byte, name string) (error){ 
	
	f, err := os.Create(name + ".mp3")
	defer f.Close()
	if err != nil {
		return err
	}
	
	f.Write(mp3)
	
	return nil
}