package main

import (
	"fmt"
	"html/template"
	"net/http"
	"net/smtp"
)

type Config struct {
	Host     string
	Port     int
	Sender   string
	Password string
}

type InfoLog struct {
	ReturnPage string
	Info       string
}

func main() {
	var (
		conn Config
		auth smtp.Auth
	)
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		http.ServeFile(writer, request, "index.html")
	})

	http.HandleFunc("/SMTPAuthentication/result", func(writer http.ResponseWriter, request *http.Request) {

		var (
			host string
			port int
		)

		host, port = "smtp.mail.ru", 587

		conn = Config{
			Host:     host,
			Port:     port,
			Sender:   request.FormValue("login"),
			Password: request.FormValue("password"),
		}

		auth = smtp.PlainAuth("", conn.Sender, conn.Password, conn.Host)

		http.Redirect(writer, request, "/home", http.StatusSeeOther)
	})

	http.HandleFunc("/home", func(writer http.ResponseWriter, request *http.Request) {
		http.ServeFile(writer, request, "home.html")
	})

	http.HandleFunc("/home/sendLetter", func(writer http.ResponseWriter, request *http.Request) {
		receiver := request.FormValue("receiver")
		subject := request.FormValue("subject")
		letter := request.FormValue("letter")
		err := smtp.SendMail(fmt.Sprintf("%s:%d", conn.Host, conn.Port),
			auth,
			conn.Sender,
			[]string{receiver},
			[]byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s\r\n", receiver, subject, letter)),
		)

		var msg string
		if err == nil {
			msg = "The message was successfully sent!"
		} else {
			msg = fmt.Sprintf("An error occurred while sending your message :( ...\n\n%s", err)
			fmt.Println(err)
		}

		t, _ := template.ParseFiles("track.html")
		_ = t.Execute(writer, InfoLog{
			ReturnPage: "/home",
			Info:       msg,
		})
	})

	_ = http.ListenAndServe("localhost:9000", nil)
}
