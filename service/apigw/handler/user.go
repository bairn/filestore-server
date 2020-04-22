package handler

import (
	"context"
	"filestore-server/common"
	"filestore-server/config"
	userProto "filestore-server/service/account/proto"
	dlProto "filestore-server/service/download/proto"
	upProto "filestore-server/service/upload/proto"
	"filestore-server/util"
	"github.com/gin-gonic/gin"
	"github.com/micro/go-micro"
	_ "github.com/micro/go-plugins/registry/kubernetes"
	"github.com/micro/go-plugins/wrapper/breaker/hystrix"
	"github.com/micro/go-plugins/wrapper/ratelimiter/ratelimit"
	"log"
	"net/http"
	ratelimit2 "github.com/juju/ratelimit"
)

var (
	userCli userProto.UserService
	upCli upProto.UploadService
	dlCli dlProto.DownloadService
)

func init()  {
	//TODO
	//配置请求容量及qps
	bRate := ratelimit2.NewBucketWithRate(100, 1000)
	service := micro.NewService(
		micro.Flags(common.CustomFlags...),
		micro.WrapClient(ratelimit.NewClientWrapper(bRate, false)), //加入限流功能, false为不等待(超限即返回请求失败)
		micro.WrapClient(hystrix.NewClientWrapper()),               // 加入熔断功能, 处理rpc调用失败的情况(cirucuit breaker)
	)
	// 初始化， 解析命令行参数等
	service.Init()

	//初始化，解析命令行参数
	service.Init()
	cli := service.Client()

	//初始化一个account服务的客户端
	userCli = userProto.NewUserService("go.micro.service.user", cli)
	//初始化一个upload服务的客户端
	upCli = upProto.NewUploadService("go.micro.service.upload", cli)
	//初始化一个download服务的客户端
}


func SignupHandler(c *gin.Context) {
	c.Redirect(http.StatusFound, "/static/view/signup.html")
}

func DoSignupHandler(c *gin.Context) {
	username := c.Request.FormValue("username")
	passwd := c.Request.FormValue("password")

	resp, err := userCli.Signup(context.TODO(), &userProto.ReqSignup{
		Username: username,
		Password: passwd,
	})

	if err != nil {
		log.Println(err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code" : resp.Code,
		"msg" : resp.Message,
	})
}

func SigninHandler(c *gin.Context) {
	c.Redirect(http.StatusFound, "/static/view/signin.html")
}

func DoSigninHandler(c *gin.Context) {
	username := c.Request.FormValue("username")
	password := c.Request.FormValue("password")

	rpcResp, err := userCli.Signin(context.TODO(), &userProto.ReqSignin{
		Username: username,
		Password: password,
	})

	if err != nil {
		log.Println(err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}

	if rpcResp.Code != common.StatusOK {
		c.JSON(200, gin.H{
			"msg" : "登陆失败",
			"code" : rpcResp.Code,
		})
		return
	}

	// // 动态获取上传入口地址
	// upEntryResp, err := upCli.UploadEntry(context.TODO(), &upProto.ReqEntry{})
	// if err != nil {
	// 	log.Println(err.Error())
	// } else if upEntryResp.Code != cmn.StatusOK {
	// 	log.Println(upEntryResp.Message)
	// }

	// // 动态获取下载入口地址
	// dlEntryResp, err := dlCli.DownloadEntry(context.TODO(), &dlProto.ReqEntry{})
	// if err != nil {
	// 	log.Println(err.Error())
	// } else if dlEntryResp.Code != cmn.StatusOK {
	// 	log.Println(dlEntryResp.Message)
	// }

	cliResp := util.RespMsg{
		Code: int(common.StatusOK),
		Msg:  "登陆成功",
		Data: struct {
			Location string
			Username string
			Token    string
			UploadEntry string
			DownloadEntry string
		}{
			Location: "/static/view/home.html",
			Username: username,
			Token:    rpcResp.Token,
			UploadEntry:   config.UploadLBHost,
			DownloadEntry: config.DownloadLBHost,
		},
	}
	c.Data(http.StatusOK, "application/json", cliResp.JSONBytes())
}

//查询用户信息
func UserInfoHandler(c *gin.Context) {
	username := c.Request.FormValue("username")

	resp, err := userCli.UserInfo(context.TODO(), &userProto.ReqUserInfo{Username: username})
	if err != nil {
		log.Println(err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}

	cliResp := util.RespMsg{
		Code: 0,
		Msg:  "ok",
		Data: gin.H{
			"Username" : username,
			"SignupAt" : resp.SignupAt,
			"LastActive" : resp.LastActiveAt,
		},
	}
	c.Data(http.StatusOK, "application/json", cliResp.JSONBytes())
}

