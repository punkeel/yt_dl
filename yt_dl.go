package main

// @TODO: caching ? I don't care for now.
// @TODO: with caching: handle when the clients cancels its request (meh.)

import (
	"net/http"
	"fmt"
	"os/exec"
	"os"
	"io"
	"encoding/json"
	"net"
)

type YTInfo struct {
	Uploader  string
	Title     string
	Thumbnail string
}

func main() {
	http.HandleFunc("/mp3/", mp3Handler)
	http.HandleFunc("/infos/", infosHandler)
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/", fs)
	l, err := net.Listen("unix", "/tmp/yt_dl.sock")
	if err != nil {
		fmt.Printf("%s\n", err)
	} else {
		err := http.Serve(l, nil)
		if err != nil {
			panic(err)
		}
		// http.ListenAndServe(":8080", nil)
	}
}

func mp3Handler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/mp3/"):]
	if len(id) != 11 {
		fmt.Fprintf(w, "Invalid ID (!= 11 chr)")
		return
	}
	info := getInfo(id)

	if info.Title == "" {
		fmt.Fprintf(w, "Invalid video. (Wer is da title ?!)")
		return
	}
	cmdName := "./stream_mp3.sh"
	cmdArgs := []string{id, info.Title, info.Uploader}
	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Content-Disposition", "attachment; filename='" + info.Title + ".mp3'")
	runCommand(w, cmdName, cmdArgs);
}

func infosHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/infos/"):]
	if len(id) != 11 {
		fmt.Fprintf(w, `{"error": true}`)
		return
	}
	info := getInfo(id)
	w.Header().Set("Content-Type", "application/json")

	if info.Title == "" {
		fmt.Fprintf(w, `{"error": true}`)
		return
	}
	d, _ := json.Marshal(info)
	w.Write(d)
}

func runCommand(res http.ResponseWriter, cmdName string, cmdArgs []string) {
	cmd := exec.Command(cmdName, cmdArgs...)
	cmd.Stderr = os.Stderr

	pipeReader, pipeWriter := io.Pipe()
	cmd.Stdout = pipeWriter
	go writeCmdOutput(res, pipeReader)

	err := cmd.Start()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error starting Cmd", err)
		os.Exit(1)
	}

	err = cmd.Wait()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error waiting for Cmd", err)
		os.Exit(1)
	}
	pipeWriter.Close()
}

func getInfo(youtubeID string) YTInfo {
	cmd := exec.Command("youtube-dl", []string{"-j", youtubeID}...)
	out, err := cmd.Output()
	if err != nil {
		println("Err: ", err)
		os.Exit(1)
	}
	cmd.Start()
	var res YTInfo
	if err := json.Unmarshal(out, &res); err != nil {
		panic(err)
	}
	fmt.Println(res)
	return res
}

func writeCmdOutput(res http.ResponseWriter, pipeReader *io.PipeReader) {
	BUF_LEN := 512
	buffer := make([]byte, BUF_LEN)
	for {
		n, err := pipeReader.Read(buffer)
		if err != nil {
			pipeReader.Close()
			break
		}

		data := buffer[0:n]
		res.Write(data)
		if f, ok := res.(http.Flusher); ok {
			f.Flush()
		}
		//reset buffer
		for i := 0; i < n; i++ {
			buffer[i] = 0
		}
	}
}