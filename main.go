package main

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/jasonlvhit/gocron"
	"github.com/labstack/echo/v4"
	"gopkg.in/gomail.v2"
)

var ring = redis.NewClient(&redis.Options{
	Addr:     "localhost:6379",
	Password: "", // no password set
	DB:       0,  // use default DB
})

func SetRedis(rdb *redis.Client, key string, value string, expiration int) {
	err := rdb.Set(ctx, key, value, 0).Err()
	if err != nil {
		log.Fatal(err)
	}
}

func GetRedis(rdb *redis.Client, key string) string {
	val, err := rdb.Get(ctx, key).Result()

	if err != nil {
		log.Fatal(err)
	}
	return val
}

var ctx = context.Background()

type Response struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

func SendMail(from string, to string, subject string, content string) {
	email := gomail.NewMessage()
	email.SetHeader("From", from)
	email.SetHeader("To", to)
	email.SetHeader("Subject", subject)
	email.SetBody("text/plain", content)

	sender := gomail.NewDialer("smtp.gmail.com", 587, "if-21029@students.ithb.ac.id", "#MieEoaNQo7K5jz$z2Uf&Xk88^DV%wFLc2v&$baMMNsZfzMfqJ5WGbUQT4pGhmuTPZrUk2GzdLAf*ay^dAhg@yo6KqKn#*2C")

	if err := sender.DialAndSend(email); err != nil {
		panic(err)
	}
}

func GetUserData(user_id int) {
	db := gormConn()
	var user Users
	user.User_ID = user_id
	result := db.First(&user)
	if result.Error == nil {
		SetRedis(ring, "userId", strconv.Itoa(user.User_ID), 0)
		SetRedis(ring, "userEmail", user.Email, 0)
	} else {
		panic(result.Error)
	}
}

func insertUser(c echo.Context) error {
	db := gormConn()

	user := new(Users)
	user.Username = c.FormValue("username")
	user.Email = c.FormValue("email")
	user.Password = c.FormValue("password")

	query := db.Select("username", "email", "password").Create(&user)
	if query.Error != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": "Gagal memasukkan data pengguna",
		})
	}
	subscription := new(Subscriptions)
	subscription.Jenis_Payment = "OvO"
	subscription.Layanan_ID = 1
	subscription.User_ID = user.User_ID
	subscription.Active = false
	query2 := db.Select("user_id", "layanan_id", "jenis_payment", "active").Create(&subscription)
	if query2.Error != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": "Gagal memasukkan data subscription",
		})
	}
	go SendMail("if-21047@students.ithb.ac.id", user.Email, "Account Successfully Created!", "Welcome "+user.Username+" To The Spotify Platform, Please Enjoy The Songs :)")
	GetUserData(user.User_ID)
	return c.JSON(http.StatusOK, user)

}

func Subscribe(c echo.Context) error {
	db := gormConn()
	id, _ := strconv.Atoi(c.QueryParam("layanan_id"))

	user_id := GetRedis(ring, "userId")
	email := GetRedis(ring, "userEmail")
	var response Response
	if err := ring.Get(ctx, "userData"); err != nil {
		result := db.Table("subscriptions").Where("user_id=? AND layanan_id=?", user_id, id).Update("active", true)
		if result.Error == nil {
			response.Status = http.StatusOK
			response.Message = "Success Subscribe"
			SendMail("if-21029@students.ithb.ac.id", email, "Subscription Activation Success", "Congratulations your monthly subscription to Spotify was successfully activated")
		} else {
			response.Status = http.StatusInternalServerError
			response.Message = "Fail Subscribe"
		}
	}
	return c.JSON(response.Status, response)
}

func Unsubscribe(c echo.Context) error {
	db := gormConn()
	id, _ := strconv.Atoi(c.QueryParam("layanan_id"))

	user_id := GetRedis(ring, "userId")
	email := GetRedis(ring, "userEmail")
	var response Response
	if err := ring.Get(ctx, "userData"); err != nil {
		result := db.Table("subscriptions").Where("user_id=? AND layanan_id=?", user_id, id).Update("active", false)
		if result.Error == nil {
			response.Status = http.StatusOK
			response.Message = "Successful Termination"
			SendMail("if-21029@students.ithb.ac.id", email, "Subscription Terminated", "I'm sorry to see you go, Please contact us if you'd like to communicate any issues.")
		} else {
			response.Status = http.StatusInternalServerError
			response.Message = "Fail Unsubscribe"
		}
	}
	return c.JSON(response.Status, response)
}

func CheckActive() bool {
	db := gormConn()
	user_id := GetRedis(ring, "userId")
	var subscription Subscriptions
	if user_id != "" {
		db.Where("user_id=?", user_id).First(&subscription)
	}
	return subscription.Active
}

func task() {
	active := CheckActive()
	if !active {
		SendMail("if-21029@students.ithb.ac.id", GetRedis(ring, "userEmail"), "Activate your Subscription", "Activate full Spotify Premium to enjoy all the features")
	}
}

func main() {
	router := echo.New()
	go GetUserData(1)
	time.Sleep(2 * time.Second)
	gocron.Start()
	gocron.Every(20).Seconds().Do(task)
	router.PUT("/subscribe", Subscribe)
	router.POST("/users", insertUser)
	router.PUT("/unsubscribe", Unsubscribe)
	router.Logger.Fatal(router.Start(":1323"))
}
