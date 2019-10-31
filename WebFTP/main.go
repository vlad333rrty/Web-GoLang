package main

import (
	"flag"
	"fmt"
	"github.com/jlaffaye/ftp"
	"html/template"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
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

func webError(writer http.ResponseWriter,err error){
	if err!=nil{
		t, _ := template.ParseFiles("error.html")
		_ = t.Execute(writer, err)
	}
}

type Data struct {
	CurrentDir    string
	Files         []string
	Folders       []string
	FilesNumber   int
	FoldersNumber int
}


func getDir(request *http.Request) (path string){
	r:=[]rune(request.URL.Path)
	path=string(r[5:])
	return
}

func getData(request *http.Request) ([]string,string){
	err:=request.ParseForm()
	if err!=nil{
		panic(err)
	}
	checkboxes:=request.Form

	files:=make([]string,0)
	path:=getParentDir(request)
	fmt.Println(path)

	for i:=range checkboxes {
		if checkboxes[i][0] == "on" {
			files = append(files, i)
		}
	}
	return files,path
}

func getParentDir(request *http.Request) string{
	return request.FormValue("!.#dir I evaluated#")
}

func main(){
	var client *ftp.ServerConn
	handleFolder:= func(writer http.ResponseWriter, request *http.Request) {
		data := Data{
			CurrentDir: getDir(request),
			Files:      make([]string, 0),
			Folders:    make([]string, 0),
		}

		list, err := client.NameList(data.CurrentDir)
		webError(writer, err)

		for i := range list {
			isFile :=strings.Contains(list[i], ".")
			if isFile{
				data.Files = append(data.Files, list[i])
			}else{
				data.Folders = append(data.Folders, list[i])
			}
		}

		data.FoldersNumber= len(data.Folders)
		data.FilesNumber= len(data.Files)

		t, _ := template.ParseFiles("home.html")
		_ = t.Execute(writer, data)
	}

	cd:= func(path string) {
		err:=client.ChangeDir(path)
		handleError(err,true)
	}

	upload:= func(writer http.ResponseWriter, request *http.Request){
		err:=request.ParseMultipartForm(10<<20)
		if err!=nil{
			panic(err)
		}

		file,handler,err:=request.FormFile("file_name")
		if err!=nil{
			panic(err)
		}

		defer file.Close()
		fmt.Printf("Uploaded File: %+v\n", handler.Filename)
		fmt.Printf("File Size: %+v\n", handler.Size)
		fmt.Printf("MIME Header: %+v\n", handler.Header)

		path :=getParentDir(request)

		cd(path)

		err=client.Stor(handler.Filename,file)
		webError(writer,err)

		http.Redirect(writer,request,"/home"+path,http.StatusSeeOther)
	}


	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		http.ServeFile(writer,request,"index.html")
	})
	http.HandleFunc("/FTPAuthentication/result", func(writer http.ResponseWriter, request *http.Request) {
		var(
			ip,password,port,login string
			err error
		)
		flag.StringVar(&ip,"ip",request.FormValue("ip"),"server ip")
		flag.StringVar(&password,"password",request.FormValue("password"),"user's password")
		flag.StringVar(&port,"port",request.FormValue("port"),"port")
		flag.StringVar(&login,"login",request.FormValue("login"),"user's login")
		flag.Parse()

		client,err=ftp.Dial(fmt.Sprintf("%s:%s", ip,port),ftp.DialWithTimeout(5*time.Second))
		webError(writer,err)

		err=client.Login(login,password)
		webError(writer,err)

		dir,err:=client.CurrentDir()
		handleError(err,false)


		folders:=make([]string,0)

		var rec func()

		rec= func() {
			folders = append(folders, dir)
			list,err:=client.NameList(dir)
			handleError(err,false)

			for i:=0;i< len(list);i++{
				if !strings.Contains(list[i],"."){
					err=client.ChangeDir(list[i])
					if handleError(err,false){
						dir,err=client.CurrentDir()
						handleError(err,false)
						rec()
						_=client.ChangeDirToParent()
					}

				}
			}
		}

		rec()

		for i:=0;i<len(folders);i++{
			http.HandleFunc("/home"+folders[i],handleFolder)
		}

		http.Redirect(writer,request,"/home/",http.StatusSeeOther)
	})


	http.HandleFunc("/home/uploadPage",upload)
	http.HandleFunc("/home/deletePage", func(writer http.ResponseWriter, request *http.Request) {
		files,path:=getData(request)
		cd(path)
		var err error
		fmt.Println()
		for i:=range files{
			err=client.Delete(files[i])
			if err!=nil{
				err=client.RemoveDir(files[i])
				webError(writer,err)
			}
		}
		http.Redirect(writer,request,"/home"+path,http.StatusSeeOther)
	})
	http.HandleFunc("/home/downloadPage", func(writer http.ResponseWriter, request *http.Request) {
		files,path:=getData(request)
		cd(path)
		var(
			response *ftp.Response
			err error
			wg sync.WaitGroup
		)
		for i:=range files{
			wg.Add(1)
			go func() {
				defer func() {wg.Done()}()
				response,err=client.Retr(files[i])

				defer response.Close()

				webError(writer,err)
				writer.Header().Set("Content-Disposition",fmt.Sprintf("attachment; filename=%s",files[i]))
				_,err=io.Copy(writer,response)
				webError(writer,err)
			}()
			wg.Wait()
		}
	})
	http.HandleFunc("/home/dirCreationPage", func(writer http.ResponseWriter, request *http.Request) {
		var err error
		err=request.ParseForm()
		webError(writer,err)
		dirname:=request.Form["name"][0]
		path:=getParentDir(request)
		err=client.ChangeDir(path)
		if handleError(err,false){
			err=client.MakeDir(dirname)
			webError(writer,err)
		}
		var absPath string
		if path=="/"{
			absPath=path+dirname
		}else{
			absPath = path+"/"+dirname
		}
		fmt.Println(absPath)
		http.HandleFunc("/home"+absPath,handleFolder)
		http.Redirect(writer,request,"/home"+path,http.StatusSeeOther)
	})

	err:=http.ListenAndServe("localhost:9000",nil)
	if err!=nil{
		log.Fatal(err)
	}
	handleError(client.Logout(),false)
}