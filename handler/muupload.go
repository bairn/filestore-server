package handler

import (
	rPool "filestore-server/cache/redis"
	"filestore-server/db"
	"filestore-server/util"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type MultipartUploadInfo struct {
	FileHash string
	FileSize int
	UploadId string
	ChunkSize int
	ChunkCount int
}

func InitalMultipartUploadHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	username := r.Form.Get("username")
	filehash := r.Form.Get("filehash")
	filesize, err := strconv.Atoi(r.Form.Get("filesize"))
	if err != nil {
		w.Write(util.NewRespMsg(-1, "params invalid", nil).JSONBytes())
		return
	}

	rConn := rPool.RedisPool().Get()
	defer rConn.Close()

	upInfo := MultipartUploadInfo{
		FileHash:   filehash,
		FileSize:   filesize,
		UploadId:   username + fmt.Sprintf("%x", time.Now().UnixNano()),
		ChunkSize:  5 * 1024 * 1024,//5MB
		ChunkCount: int(math.Ceil(float64(filesize)/(5 * 1024 * 1024))),
	}

	_, err = rConn.Do("HSET", "MP_"+upInfo.UploadId, "chunkcount", upInfo.ChunkCount)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	_, err = rConn.Do("HSET", "MP_" + upInfo.UploadId, "filehash", upInfo.FileHash)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	_, err = rConn.Do("HSET", "MP_" + upInfo.UploadId, "filesize", upInfo.FileSize)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	w.Write(util.NewRespMsg(0, "Ok", upInfo).JSONBytes())
}


//上传文件分块
func UploadPartHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	//username := r.Form.Get("username")
	uploadId := r.Form.Get("uploadid")
	chunkIndex := r.Form.Get("index")

	rConn := rPool.RedisPool().Get()
	defer rConn.Close()

	fpath := "/tmp/data/" + uploadId + "/"
	os.MkdirAll(fpath, 0744)
	fd, err := os.Create(fpath + chunkIndex)
	if err != nil {
		w.Write(util.NewRespMsg(-1, "Upload part failed", nil).JSONBytes())
		return
	}
	defer fd.Close()

	buf := make([]byte, 1024 * 1024)
	for {
		n, err := r.Body.Read(buf)
		fd.Write(buf[:n])
		if err != nil {
			break
		}
	}

	_, err = rConn.Do("HSET", "MP_"+uploadId, "chkidx_"+chunkIndex, 1)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	w.Write(util.NewRespMsg(0, "Ok", nil).JSONBytes())
}

//通知上传合并
func CompleteUploadHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	upid := r.Form.Get("uploadid")
	username := r.Form.Get("username")
	filehash := r.Form.Get("filehash")
	filesize := r.Form.Get("filesize")
	filename := r.Form.Get("filename")

	rConn := rPool.RedisPool().Get()
	defer rConn.Close()

	data, err := redis.Values(rConn.Do("HGETALL", "MP_"+upid))
	if err != nil {
		fmt.Println(err)
		w.Write(util.NewRespMsg(-1, "complete upload failed", nil).JSONBytes())
		return
	}

	totalCount := 0
	chunkCont := 0
	for i:=0;i<len(data);i+=2 {
		k := string(data[i].([]byte))
		v := string(data[i+1].([]byte))

		if k == "chunkcount" {
			totalCount, _ = strconv.Atoi(v)
		} else if strings.HasPrefix(k, "chkidx_") && v == "1" {
			chunkCont += 1
		}
	}

	if totalCount != chunkCont {
		fmt.Printf("totalCount:%d chunkCont:%d\n", totalCount , chunkCont)
		w.Write(util.NewRespMsg(-2, "invalid request", nil).JSONBytes())
		return
	}

	//TODO: 合并分块

	fsize, _ := strconv.Atoi(filesize)
	db.OnFileUploadFinished(filehash, filename, int64(fsize), "")
	db.OnUserFileUploadFinished(username, filehash, filename, int64(fsize))

    w.Write(util.NewRespMsg(0, "OK", nil).JSONBytes())
}