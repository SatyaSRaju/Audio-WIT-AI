package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	"goji.io"
	"goji.io/pat"
)

func main() {

	log.Print("Entering Main")

	mux := goji.NewMux()
	mux.HandleFunc(pat.Post("/transcode"), transcodeAudio())
	//http.ListenAndServe("localhost:8080", mux)
	//log.Println("Listening on port 8080")
	httpErr := http.ListenAndServe(GetPort(), mux)
	if httpErr != nil {
		log.Printf("Http Listen", httpErr)
	}

}

type LogWriter struct {
	logger *log.Logger
}

func NewLogWriter(l *log.Logger) *LogWriter {
	lw := &LogWriter{}
	lw.logger = l
	return lw
}

func (lw LogWriter) Write(p []byte) (n int, err error) {
	lw.logger.Println(p)
	return len(p), nil
}

func GetPort() string {
	var port = os.Getenv("PORT")
	// Set a default port if there is nothing in the environment

	log.Printf("The assigned Port is %s", port)
	if port == "" {
		port = "8080"
		fmt.Println("INFO: No PORT environment variable detected, defaulting to " + port)
	}
	return ":" + port
}

func ErrorWithJSON(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	fmt.Fprintf(w, "{message: %q}", message)
}

func ResponseWithJSON(w http.ResponseWriter, json []byte, code int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	w.Write(json)
}

type AudioFile struct {
	SourceFile   string `json:"sourceFile"`
	TargetFile   string `json:"targetFile"`
	Filelocation string `json:"fileLocation`
}

func transcodeAudio() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		mStartTime := time.Now()

		var audioData AudioFile
		log.Printf("Entering Transcode Audio Method")

		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&audioData)
		if err != nil {
			ErrorWithJSON(w, "Incorrect body", http.StatusBadRequest)
			return
		}

		log.Printf("Before calling FFMPEG For Audio Transcode")

		workingDir, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
			return

		}
		fmt.Println("The Current Dir is ->", workingDir)
		filePath := workingDir + "/" + audioData.TargetFile

		t := time.Now()
		fmt.Println("Starting Transcode job --> ", t.Format(time.RFC850))

		//cmd := exec.Command("FFREPORT=file=ffreport.log:level=32 ffmpeg", "-i", audioData.SourceFile, audioData.TargetFile)
		out, err := exec.Command("ffmpeg", "-y", "-i", audioData.SourceFile, audioData.TargetFile).CombinedOutput()
		//cmd.Start()
		//cmd.Wait()
		//go func() {

		if err != nil {
			//cmd.Stdout = NewLogWriter(log.New(io.Writer, "transcodeLogger", log.Lshortfile))
			log.Fatalf("The ffmpeg command failed: %v %v", err, string(out))

			return

		}

		t = time.Now()
		fmt.Println("Completed Transcode job --> ", t.Format(time.RFC850))

		//}()
		fmt.Println("Reading byte stream of Transcoded file ->", t.Format(time.RFC850))
		audioDataByte, err := ioutil.ReadFile(filePath)
		if err != nil {
			log.Fatal(err)
			return

		}
		data := bytes.NewReader([]byte(audioDataByte))

		fmt.Println("Sending Audio byte stream to AI Engine ->", t.Format(time.RFC850))

		witURL := "https://api.wit.ai/speech?v=20160526"
		req, err := http.NewRequest("POST", witURL, data)
		req.Header.Set("content-type", "audio/wav")
		req.Header.Set("Authorization", "Bearer S6BKVBJARBHDHE6XASGVJVKALIST4QPJ")
		client := &http.Client{}
		resp, err := client.Do(req)

		if err != nil {
			// handle error
		}
		defer resp.Body.Close()
		fmt.Println("Response from AI Engine", t.Format(time.RFC850))

		witResp, err := ioutil.ReadAll(resp.Body)
		fmt.Println(string(witResp))
		if err != nil {
			log.Fatal(err)
			return

		}

		ResponseWithJSON(w, witResp, http.StatusOK)
		fmt.Println("Start time is -->", mStartTime)
		fmt.Println("End time is ->", time.Now())
		mTimeDiff := time.Now().Sub(mStartTime)

		fmt.Println("The total time it took for method invocation is ->", mTimeDiff)

		errFile := os.Remove(audioData.TargetFile)
		if errFile != nil {
			log.Printf("Unable to Delete the File -> %s", err)
			return

		}

	}

}
