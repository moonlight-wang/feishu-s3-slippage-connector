package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// LarkAuth 飞书认证
type LarkAuth struct {
	verificationToken string
}

func NewLarkAuth(verificationToken string) *LarkAuth {
	return &LarkAuth{
		verificationToken: verificationToken,
	}
}

func (l *LarkAuth) VerifyToken(token string) bool {
	if l.verificationToken == "" {
		return true
	}
	return token == l.verificationToken
}

// LarkConnectorMiddleware 飞书连接器中间件
func LarkConnectorMiddleware(larkAuth *LarkAuth) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req LarkConnectorRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusOK, LarkConnectorResponse{
				Code:    400,
				Message: "Invalid request format: " + err.Error(),
			})
			c.Abort()
			return
		}

		if !larkAuth.VerifyToken(req.VerificationToken) {
			c.JSON(http.StatusOK, LarkConnectorResponse{
				Code:    401,
				Message: "Invalid verification token",
			})
			c.Abort()
			return
		}

		c.Set("lark_request", req)
		c.Next()
	}
}
