package Utils

import (
	"time"

	"github.com/golang-jwt/jwt"
)

type JwtUtil struct {
	SigningKey []byte //密钥
	ExpTime    int
}

/*
初始化参数
 sign: 密钥
 exp: 过期时长，秒
*/
func (this *JwtUtil) Init(sign string, exp int) {
	this.SigningKey = []byte(sign)
	this.ExpTime = exp
}

/*
产生token串
  data: 待加密串
  return：
    string：token字符串
    error： 错误
*/
func (this *JwtUtil) EncodeJwtString(data map[string]interface{}) (string, error) {

	//设置默认值
	if this.ExpTime == 0 {
		this.Init("Wh@66777060", 3600)
	}

	//在加密数据中加入过期时间，用现在时间+过期时长
	data["exp"] = time.Now().Unix() + int64(this.ExpTime)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims(data))
	tokenString, err := token.SignedString(this.SigningKey)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

/*
解析token串
  parameter:
    tokenString: 待解析串
  return:
    map[string]interface{}： 解析数据
    error: 是否有效
*/
func (this *JwtUtil) DecodeJwtString(tokenString string) (map[string]interface{}, error) {

	//设置默认值
	if this.ExpTime == 0 {
		this.Init("Wh@66777060", 3600)
	}

	//在这里如果也使用jwt.ParseWithClaims的话，第二个参数就写jwt.MapClaims{}
	//例如jwt.ParseWithClaims(tokenString, jwt.MapClaims{},func(t *jwt.Token) (interface{}, error){}

	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		return this.SigningKey, nil
	})
	if err != nil {
		return nil, err
	}
	return token.Claims.(jwt.MapClaims), nil
}
