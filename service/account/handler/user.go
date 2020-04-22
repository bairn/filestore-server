package handler

import (
	"context"
	"filestore-server/common"
	"filestore-server/config"
	"filestore-server/service/account/proto"
	dbcli "filestore-server/service/dbproxy/client"
	"filestore-server/util"
	"fmt"
	"time"
)

// GenToken : 生成token
func GenToken(username string) string {
	// 40位字符:md5(username+timestamp+token_salt)+timestamp[:8]
	ts := fmt.Sprintf("%x", time.Now().Unix())
	tokenPrefix := util.MD5([]byte(username + ts + "_tokensalt"))
	return tokenPrefix + ts[:8]
}

type User struct {
}

func (u *User) Signup(ctx context.Context,req *proto.ReqSignup,res *proto.RespSignup) error {
	username := req.Username
	passwd := req.Password

	if len(username) < 3 || len(passwd) < 5 {
		res.Code = common.StatusParamInvalid
		res.Message = "注册参数无效"
		return nil
	}

	enc_passwd := util.Sha1([]byte(passwd + config.PasswordSalt))
	dbResp, err := dbcli.UserSignup(username, enc_passwd)
	if err == nil && dbResp.Suc {
		res.Code = common.StatusOK
		res.Message = "注册成功"
	} else {
		res.Code = common.StatusRegisterFailed
		res.Message = "注册失败"
	}

	return nil
}

func (u *User) Signin(ctx context.Context, req *proto.ReqSignin, res *proto.RespSignin) error {
	username := req.Username
	password := req.Password

	encPasswd := util.Sha1([]byte(password + config.PasswordSalt))
	dbResp, err := dbcli.UserSignin(username, encPasswd)
	if err != nil || !dbResp.Suc {
		res.Code = common.StatusLoginFailed
		return nil
	}

	token := GenToken(username)
	upRes, err := dbcli.UpdateToken(username, token)
	if err != nil || !upRes.Suc {
		res.Code = common.StatusServerError
		return nil
	}

	res.Code = common.StatusOK
	res.Token = token
	return nil
}

func (u *User) UserInfo(ctx context.Context, req *proto.ReqUserInfo, res *proto.RespUserInfo) error {
	dbResp, err := dbcli.GetUserInfo(req.Username)
	if err != nil {
		res.Code = common.StatusServerError
		res.Message = "服务错误"
		return nil
	}

	if !dbResp.Suc {
		res.Code = common.StatusUserNotExists
		res.Message = "用户不存在"
		return nil
	}

	user := dbcli.ToTableUser(dbResp.Data)

	res.Code = common.StatusOK
	res.Username = user.Username
	res.SignupAt = user.SignupAt
	res.LastActiveAt = user.LastActiveAt
	res.Status = int32(user.Status)

	//TODO:需增加接口支持完善用户信心(email/phone)等
	res.Email = user.Email
	res.Phone = user.Phone

	return nil
}

