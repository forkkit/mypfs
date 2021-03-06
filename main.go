//
// Copyright (c) 2015 Jon Carlson.  All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.
//

package main

//
// Runs an HTTP static file server and file upload receiver from/to
// the directory that this is executed from.
//

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
)

var (
	action         string = "help"
	port           int    = 8080
	timeoutMinutes int64  = 10
	insecure       bool   = false
	version        string = "0.9.4"
	secretUsername string // Required for admittance to site
)

// uploadHandler returns an HTML upload form
func uploadHandler(w http.ResponseWriter, r *http.Request) error {
	if r.Method == "GET" {
		fmt.Fprintf(w, `<html>
<head>
  <title>GoLang HTTP Fileserver</title>
  <!--
  <meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
  <meta http-equiv="Pragma" content="no-cache" />
  <meta http-equiv="Expires" content="-1" />
  <meta http-equiv="Cache-Control" content="no-cache" />
  -->
</head>

<body>

<h4>Choose a file to upload</h4>

<form action="/fs-receive" method="post" enctype="multipart/form-data">
  <input type="file" name="file" id="file">
  <br> <br>
  <input type="submit" name="submit" value="Submit">
</form>

</body>
</html>`)
	}
	return nil
}

// receiveHandler accepts the file and saves it to the current working directory
func receiveHandler(w http.ResponseWriter, r *http.Request) error {

	// the FormFile function takes in the POST input id file
	file, header, err := r.FormFile("file")
	if err != nil {
		fmt.Fprintln(w, err)
		return err
	}

	defer file.Close()

	out, err := os.Create(header.Filename)
	if err != nil {
		fmt.Fprintf(w, "Unable to create the file for writing. Check your write access privilege")
		return err
	}

	defer out.Close()

	// write the content from POST to the file
	_, err = io.Copy(out, file)
	if err != nil {
		fmt.Fprintln(w, err)
		return err
	}

	log.Println("File received:", header.Filename)

	fmt.Fprintf(w, `<html>
File uploaded successfully: %s 
<p><a href="/">Back</a></p>
<html>`, header.Filename)
	return nil
}

func init() {
	err := myPfsCmd.Execute()
	if err != nil {
		os.Exit(1)
	}

	if insecure {
		// skip the random username generation
	} else {
		// Generate the token required to access this server via HTTP
		rand.Seed(time.Now().UTC().UnixNano())
		secretUsername = randomString(8)
	}
}

func main() {

	var portStr = ":" + strconv.Itoa(port)

	dir, err := os.Getwd()
	if err != nil {
		fmt.Println("err=", err)
		os.Exit(1)
	}

	if action == "help" {
		os.Exit(0)
	} else if action == "version" {
		os.Exit(0)
	}

	if action == "up" {
		printAddressAndPort()
		log.Printf("Allowing uploads to the current directory for %v minutes on port %v\n", timeoutMinutes, port)

		// Show the upload form
		http.Handle("/", errorHandler(authBasic(uploadHandler)))
		// Handle the incoming file
		http.Handle("/fs-receive", errorHandler(authBasic(receiveHandler)))

	} else if action == "down" {
		printAddressAndPort()
		log.Printf("Allowing downloads from the current directory for %v minutes on port %v\n", timeoutMinutes, port)

		// Show the download page using a customized FileServer with no
		// added Upload Header (because we are not allowing uploads)
		http.Handle("/", errorHandler(authBasic(errorableHandler(FileServer(Dir(dir), false /*addUploadHeader*/)))))

	} else if action == "up/down" {
		printAddressAndPort()
		log.Printf("Allowing downloads from (and uploads to) the current directory for %v minutes on port %v\n", timeoutMinutes, port)

		// Display the upload form
		http.Handle("/fs-upload", errorHandler(authBasic(uploadHandler)))
		// Handle the incoming file
		http.Handle("/fs-receive", errorHandler(authBasic(receiveHandler)))

		// Show the download page using a customized FileServer
		// copied from net/http/fs.go. This version adds a header
		// to the top when we list a directory (in dirList() func)
		http.Handle("/", errorHandler(authBasic(errorableHandler(FileServer(Dir(dir), true /*addUploadHeader*/)))))
	}

	go func() {
		time.Sleep(time.Duration(timeoutMinutes) * time.Minute)
		log.Println("Fileserver timed out.  Exiting.")
		os.Exit(0)
	}()

	l, err := net.Listen("tcp4", portStr)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	log.Fatal(http.Serve(l, nil))
}

func printAddressAndPort() {
	addrs, err := net.InterfaceAddrs()

	if err != nil {
		log.Fatal("Error getting local address", err)
	}

	var localAddress string = "<your-local-ip-address>"
	for _, address := range addrs {
		// check the address type and if it is not a loopback then display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				localAddress = ipnet.IP.String()
			}
		}
	}

	fmt.Println()
	if insecure {
		fmt.Printf("Use this address: http://%s:%v\n (No secret username required)", localAddress, port)
	} else {
		fmt.Printf("Use this address: http://%s:%v\n (Enter %s for username when requested. Ignore password) \n", localAddress, port, secretUsername)
	}
	fmt.Println()

}
