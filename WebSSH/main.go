package main

import (
	"flag"
	"fmt"
	"golang.org/x/crypto/ssh"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
)

func handleError(err error,priority bool)bool{
	if err!=nil{
		if priority{
			panic(err)
		}else{
			log.Println(err)
			return false
		}
	}
	return true
}

type Lore []string

func main(){
	var(
		stdin io.WriteCloser
		stdout io.Reader
		out=make([]byte,65536)
		lore=Lore{}
	)
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		http.ServeFile(writer,request,"index.html")
	})
	http.HandleFunc("/SSHAuthentication/result", func(writer http.ResponseWriter, request *http.Request) {
		var ip,password,port,login string
		flag.StringVar(&ip,"ip",request.FormValue("ip"),"server ip")
		flag.StringVar(&password,"password",request.FormValue("password"),"user's password")
		flag.StringVar(&port,"port",request.FormValue("port"),"port")
		flag.StringVar(&login,"login",request.FormValue("login"),"user's login")
		flag.Parse()

		var err error
		config:=&ssh.ClientConfig{
			Config:            ssh.Config{},
			User:              login,
			Auth:              []ssh.AuthMethod{
				ssh.Password(password),
			},
			HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
				return error(nil)
			},
		}

		conn,err:=ssh.Dial("tcp",fmt.Sprintf("%s:%s",ip,port),config)
		handleError(err,true)

		session,err:=conn.NewSession()
		handleError(err,false)

		stdin,err=session.StdinPipe()
		handleError(err,true)

		stdout,err=session.StdoutPipe()

		handleError(session.Shell(),true)

		http.Redirect(writer,request,"/console",http.StatusSeeOther)
	})

	http.HandleFunc("/console", func(writer http.ResponseWriter, request *http.Request) {
		t,_:=template.ParseFiles("console.html")
		_=t.Execute(writer,lore)
	})

	http.HandleFunc("/SSHData/parseData", func(writer http.ResponseWriter, request *http.Request) {
		cmd:=request.FormValue("cmd")
		_,_=fmt.Fprintf(stdin,"%s\n",cmd)

		fmt.Println(request.Form)

		n,err:=stdout.Read(out)

		handleError(err,true)

		lore = append(lore, string(out[:n]))

		http.Redirect(writer,request,"/console",http.StatusSeeOther)
	})

	err:=http.ListenAndServe("localhost:9000",nil)
	if err!=nil{
		log.Fatal(err)
	}
}