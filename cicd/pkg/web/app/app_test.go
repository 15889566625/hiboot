package app

import (
	"testing"
	"github.com/kataras/iris/httptest"
	"github.com/hidevopsio/hi/boot/pkg/log"
	"github.com/hidevopsio/hi/cicd/pkg/web/controllers"
	"os"
	"net/http"
	"github.com/iris-contrib/httpexpect"
	"github.com/hidevopsio/hi/cicd/pkg/auth"
	"github.com/hidevopsio/hi/boot/pkg/application"
	"github.com/hidevopsio/hi/cicd/pkg/ci"
	"time"
)

var userRequest controllers.UserRequest

func init() {
	userRequest = controllers.UserRequest{
		Url:      os.Getenv("SCM_URL"),
		Username: os.Getenv("SCM_USERNAME"),
		Password: os.Getenv("SCM_PASSWORD"),
	}
}

func newTestServer(t *testing.T) *httpexpect.Expect {
	boot := NewBoot()
	return httptest.New(t, boot.App())
}


func login(expired int64, unit time.Duration) (application.JwtToken, error) {
	u := &auth.User{}
	token, _, err := u.Login(userRequest.Url, userRequest.Username, userRequest.Password)
	jwtToken, err := application.GenerateJwtToken(application.MapJwt{
		"username": userRequest.Username,
		"password": token,
	}, expired, unit)
	return jwtToken, err
}


func requestCicdPipeline(e *httpexpect.Expect, jwtToken application.JwtToken, statusCode int)  {
	e.Request("POST", "/cicd/run").WithHeader(
		"Authorization", "Bearer " + string(jwtToken),
	).WithJSON(ci.Pipeline{
		Project:  "demo",
		App:      "hello-world",
		Profile:  "dev",
		Name: "java",
	}).Expect().Status(statusCode)
}


func TestUserLogin(t *testing.T) {
	log.Println("TestUserLogin()")

	e := newTestServer(t)

	response := e.Request("POST", "/user/login", ).WithJSON(
		userRequest).Expect().Status(http.StatusOK).JSON().Object()
	response.Value("message").Equal("Login successful.")
}

func TestUserLoginWithWrongCredentials(t *testing.T) {
	log.Println("TestUserLoginWithWrongCredentials()")

	e := newTestServer(t)

	request := controllers.UserRequest{
		Url:      os.Getenv("SCM_URL"),
		Username: "xxx",
		Password: "xxx",
	}

	e.Request("POST", "/user/login", ).WithJSON(
		request).Expect().Status(http.StatusForbidden)
}

func TestCicdRun(t *testing.T) {
	log.Println("TestCicdRun()")

	e := newTestServer(t)

	jwtToken, err := login(500, time.Millisecond)

	if err == nil {
		requestCicdPipeline(e, jwtToken, http.StatusOK)

		time.Sleep(1500 * time.Millisecond)

		requestCicdPipeline(e, jwtToken, http.StatusUnauthorized)
	}
}

func TestCicdRunWithoutToken(t *testing.T) {
	log.Println("TestCicdRunWithoutToken()")

	e := newTestServer(t)

	e.Request("POST", "/cicd/run").WithJSON(ci.Pipeline{
		Project: "demo",
		App:     "hello-world",
		Profile: "dev",
		Name: "java",
	}).Expect().Status(http.StatusUnauthorized)

}
