# AudioAiServer
* 使用go语言实现 AI语音服务器，
* 基础功能 语音转换成文件，文字转换成语音使用的是百度AI平台接口
* AI部分简单实现
* 嵌入式设备（蓝牙音箱）与服务器之间的数据交换通过websocket实现

# PromptToneGenerator.go
* 提示音生成器

## 使用方式
```
go run PromptToneGenerator.go "你好"
```

# AiSpeakerServer_h2.go
* HTTP2 协议版本