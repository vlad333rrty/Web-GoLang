package main

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
)

func handleError(err error, priority bool) bool {
	if err != nil {
		if priority {
			panic(err)
		} else {
			log.Println(err)
			return false
		}
	}
	return true
}

type Lore struct {
	Lore []string
	Dir  string
}

func main() {
	var (
		stdin   io.WriteCloser
		stdout  io.Reader
		out     = make([]byte, 65536)
		lore    Lore
		session *ssh.Session
	)
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		http.ServeFile(writer, request, "WebSSH/index.html")
	})
	http.HandleFunc("/SSHAuthentication/result", func(writer http.ResponseWriter, request *http.Request) {
		ip := request.FormValue("ip")
		password := request.FormValue("password")
		port := request.FormValue("port")
		login := request.FormValue("login")

		lore = Lore{
			Lore: make([]string, 0),
			Dir:  "",
		}

		var err error
		config := &ssh.ClientConfig{
			Config: ssh.Config{},
			User:   login,
			Auth: []ssh.AuthMethod{
				ssh.Password(password),
			},
			HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
				return error(nil)
			},
		}

		conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%s", ip, port), config)
		handleError(err, true)

		session, err = conn.NewSession()
		handleError(err, false)

		modes := ssh.TerminalModes{
			ssh.ECHO:          0,
			ssh.TTY_OP_ISPEED: 14400,
			ssh.TTY_OP_OSPEED: 14400,
		}

		if err = session.RequestPty("xterm", 80, 40, modes); err != nil {
			_ = session.Close()
			log.Fatal(err)
		}

		stdin, err = session.StdinPipe()
		handleError(err, true)

		stdout, err = session.StdoutPipe()

		handleError(err, true)
		handleError(session.Shell(), true)

		//reading greetings
		n, err := stdout.Read(out)
		handleError(err, false)

		s := string(out[:n])

		if strings.LastIndex(s, "\n") == -1 {
			lore.Dir = s
		} else {
			temp := strings.Split(s, "\n")
			for i := 0; i < len(temp); i++ {
				lore.Lore = append(lore.Lore, temp[i])
			}
		}

		http.Redirect(writer, request, "/console", http.StatusSeeOther)
	})

	http.HandleFunc("/console", func(writer http.ResponseWriter, request *http.Request) {
		n, err := stdout.Read(out)
		handleError(err, false)
		t, _ := template.ParseFiles("WebSSH/console.html")

		temp := string(out[:n])
		k := strings.LastIndex(temp, "\n")

		if k == -1 {
			lore.Dir = temp
		} else {
			lore.Lore = append(lore.Lore, temp[:k-1])
			lore.Dir = temp[k:]
		}
		_ = t.Execute(writer, lore)
	})

	http.HandleFunc("/SSHData/parseData", func(writer http.ResponseWriter, request *http.Request) {
		cmd := request.FormValue("cmd")
		if cmd == "clear" {
			lore.Lore = make([]string, 0)
			cmd = ""
		}
		_, err := fmt.Fprintf(stdin, "%s\n", cmd)
		handleError(err, false)
		if cmd == "exit" {
			err = session.Close()
			handleError(err, true)

			http.Redirect(writer, request, "/", http.StatusSeeOther)
		} else {
			http.Redirect(writer, request, "/console", http.StatusSeeOther)
		}
	})

	err := http.ListenAndServe("localhost:9000", nil)
	if err != nil {
		log.Fatal(err)
	}
}
