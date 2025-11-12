package fuzzer

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/glitchedgitz/grroxy-db/rawhttp"
)

const ModeClusterBomb = "cluster_bomb"
const ModePitchFork = "pitch_fork"

type FuzzerConfig struct {
	Request     string
	Host        string
	Port        string
	UseTLS      bool
	Markers     map[string]string
	Mode        string
	Concurrency int
	Timeout     time.Duration
}

type FuzzerResult struct {
	Request  string
	Response string
	Time     time.Duration
	Markers  map[string]string
	Error    string
}

type Fuzzer struct {
	Config  FuzzerConfig
	Results chan any
	State   string
	mu      sync.RWMutex
	wg      sync.WaitGroup

	http        *rawhttp.Client
	files       map[string]*bufio.Reader
	fileHandles map[string]*os.File
}

func NewFuzzer(config FuzzerConfig) *Fuzzer {
	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}

	if config.Concurrency == 0 {
		config.Concurrency = 40
	}

	return &Fuzzer{Config: config,
		http:    rawhttp.NewClient(rawhttp.Config{Timeout: config.Timeout}),
		Results: make(chan any, config.Concurrency)}
}

func (f *Fuzzer) Fuzz() error {

	if f.Config.Request == "" {
		return fmt.Errorf("request is empty")
	}

	if f.Config.Host == "" {
		return fmt.Errorf("host is empty")
	}

	if f.Config.Port == "" {
		if f.Config.UseTLS {
			f.Config.Port = "443"
		} else {
			f.Config.Port = "80"
		}
	}

	if f.files == nil {
		f.files = make(map[string]*bufio.Reader)
	}

	if f.fileHandles == nil {
		f.fileHandles = make(map[string]*os.File)
	}

	if f.Config.Mode == "" {
		f.Config.Mode = ModeClusterBomb
	}

	if f.Config.Concurrency == 0 {
		f.Config.Concurrency = 10
	}

	if f.Config.Mode != ModeClusterBomb && f.Config.Mode != ModePitchFork {
		return fmt.Errorf("invalid mode: %s", f.Config.Mode)
	}

	// defer close(f.results)

	for marker, wordlist := range f.Config.Markers {
		file, err := os.Open("./" + wordlist)
		if err != nil {
			return fmt.Errorf("failed to open wordlist: %w", err)
		}
		f.files[marker] = bufio.NewReader(file)
		f.fileHandles[marker] = file
	}

outerLoop:
	for {

		markers := make(map[string]string)
		hitEOF := false

		if f.isStopped() {
			break outerLoop
		}

		for marker := range f.files {
			markers[marker] = ""
		}

		for marker, reader := range f.files {
			if f.isStopped() {
				break outerLoop
			}

			word, err := reader.ReadString('\n')

			if err == io.EOF {
				hitEOF = true
			}

			if err != nil && err != io.EOF {
				f.Stop()
				log.Println("[fuzzer] failed to read wordlist: %w", err)
				break outerLoop
			}

			if strings.HasSuffix(word, "\r\n") {
				word = word[:len(word)-2]
			} else if strings.HasSuffix(word, "\n") {
				word = word[:len(word)-1]
			}

			log.Println("[fuzzer] reading word: ", word)

			markers[marker] = word

			if f.Config.Mode == ModeClusterBomb {
				f.wg.Add(1)
				go f.SendRequest(markers)
			}
		}

		if f.Config.Mode == ModePitchFork {
			// Only process if we didn't hit EOF (all markers were read successfully)
			if !hitEOF {
				f.wg.Add(1)
				go f.SendRequest(markers)
			}
		}

		if hitEOF {
			// f.Stop()
			break outerLoop
		}

		if f.isStopped() {
			break outerLoop
		}
	}

	// Wait for all pending goroutines to finish
	f.wg.Wait()
	// Close the results channel so the receiver knows we're done
	close(f.Results)

	return nil

}

func (f *Fuzzer) Stop() {
	f.mu.Lock()
	f.State = "stopped"
	f.mu.Unlock()

	for _, file := range f.fileHandles {
		file.Close()
	}

	// close(f.results)
}

func (f *Fuzzer) isStopped() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.State == "stopped"
}

func (f *Fuzzer) ReplaceMarkers(markers map[string]string) string {
	request := f.Config.Request
	for marker, word := range markers {
		request = strings.ReplaceAll(request, marker, word)
	}
	return request
}

func (f *Fuzzer) SendRequest(markers map[string]string) {
	defer f.wg.Done()
	request := f.ReplaceMarkers(markers)
	req := rawhttp.Request{
		RawBytes: []byte(request),
		Host:     f.Config.Host,
		Port:     f.Config.Port,
		UseTLS:   f.Config.UseTLS,
		Timeout:  f.Config.Timeout,
	}
	resp, err := f.http.Send(req)

	result := FuzzerResult{Request: request, Response: "", Time: 0, Markers: markers}
	if err == nil {
		result.Response = string(resp.RawBytes)
		result.Time = resp.ResponseTime
	} else {
		result.Response = err.Error()
	}

	f.Results <- result
}
