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
	"errors"
)

type YTInfo struct {
	Uploader  string
	Title     string
	Thumbnail string
}

func main() {
	http.HandleFunc("/mp3/", mp3Handler)
	http.HandleFunc("/infos/", infosHandler)
	http.HandleFunc("/env", envHandler)
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/", fs)

	port, exists := os.Getenv("PORT")
	if exists {
		fmt.Printf("Starting server on port %s\n", port)
		http.ListenAndServe(":" + port, nil)
	}else {
		socket := "/tmp/yt_dl.sock"
		fmt.Printf("Starting server on socket %s\n", socket)
		l, err := net.Listen("unix", socket)
		if err != nil {
			fmt.Printf("%s\n", err)
		} else {
			err := http.Serve(l, nil)
			if err != nil {
				panic(err)
			}

		}
	}
}

func envHandler(w http.ResponseWriter, r *http.Request) {
	for _, e := range os.Environ() {
		fmt.Fprintln(w, e)
	}
}

func mp3Handler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/mp3/"):]
	if len(id) != 11 {
		fmt.Fprintf(w, "Invalid ID (!= 11 chr)")
		return
	}
	info, err := getInfo(id)
	if err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}
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
	w.Header().Set("Content-Type", "application/json")
	info, err := getInfo(id)
	if err != nil || info.Title == "" {
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

func getInfo(youtubeID string) (YTInfo, error) {
	cmd := exec.Command("youtube-dl", []string{"-j", youtubeID}...)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	cmd.Start()
	var res YTInfo
	if string(rune(out[0])) != '{' {
		return nil, errors.New("yt-dl: could not fetch infos")
	}
	if err := json.Unmarshal(out, &res); err != nil {
		panic(err)
	}
	fmt.Println(res)
	return res, nil
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
