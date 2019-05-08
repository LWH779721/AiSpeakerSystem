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