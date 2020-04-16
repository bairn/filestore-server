package main

import (
	"bufio"
	"encoding/json"
	"filestore-server/config"
	"filestore-server/db"
	"filestore-server/mq"
	"filestore-server/store/oss"
	"fmt"
	"log"
	"os"
)

//处理文件的转移的真正逻辑
func ProcessTransfer(msg []byte) bool {
	pubData := mq.TransferData{}
	err := json.Unmarshal(msg, &pubData)
	if err != nil {
		log.Println(err.Error())
		return false
	}
	fmt.Printf("%+v\n", pubData)

	filed, err := os.Open(pubData.CurLocation)
	if err != nil {
		log.Println(err.Error())
		return false
	}

	err = oss.Bucket().PutObject(
		pubData.DestLocation,
		bufio.NewReader(filed),
	)
	if err != nil {
		log.Println(err.Error())
		return false
	}

	suc := db.UpdateFileLocation(pubData.FileHash, pubData.DestLocation)
	if !suc {
		return false
	}

	return true
}

func main() {
	log.Println("开始监听转移任务队列...")

	mq.StartConsume(
		config.TransOSSQueueName,
		"transfer_oss",
		ProcessTransfer,
		)
}
