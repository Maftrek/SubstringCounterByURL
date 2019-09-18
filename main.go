package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

type urlUUID struct {
	url  string
	uuid string
}

func processingURL(limit chan bool, errorsListen chan error, counters *sync.Map, url string) {
	defer func() {
		<-limit
	}()

	url = strings.ReplaceAll(url, "\n", "")

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		errorsListen <- err
		return
	}
	client := &http.Client{
		Timeout: time.Second * 5,
	}
	resp, err := client.Do(req)
	if err != nil {
		errorsListen <- err
		return
	}
	defer func() {
		if resp.Body != nil {
			resp.Body.Close()
		}
	}()

	if resp.StatusCode != 200 {
		errorsListen <- errors.New("status not ok")
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		errorsListen <- err
		return
	}

	uuid, err := exec.Command("uuidgen").Output()
	if err != nil {
		errorsListen <- err
		return
	}

	urlNew := urlUUID{
		url:  url,
		uuid: string(uuid),
	}
	counters.Store(urlNew, counting(body))
}

func counting(body []byte) int {
	return bytes.Count(body, []byte("Go"))
}

func getResult(counters *sync.Map) {
	fmt.Println("Result data: ")
	total := 0
	handler := func(k, v interface{}) bool {
		var ok bool
		var (
			url   urlUUID
			count int
		)
		if url, ok = k.(urlUUID); !ok {
			fmt.Println("url has the wrong type")
		}
		if count, ok = v.(int); !ok {
			fmt.Println("count has the wrong type")
		}
		res := fmt.Sprintf(`Count for %s: %d`, url.url, v)
		fmt.Println(res)
		total += count
		return true
	}
	counters.Range(handler)
	fmt.Println("Total:", total)
}

func timer(sing chan time.Time, finish chan bool) {
	for range time.Tick(time.Second) {
		select {
		case <-sing:
		case <-time.After(time.Minute):
			select {
			case finish <- true:
			default:
			}
		}
	}
}

func scaner(urls chan string, sing chan time.Time) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		text := scanner.Text()
		if len(text) > 0 {
			urls <- text
			sing <- time.Now()
			fmt.Println(text)
		}
	}
}

func main() {
	urls := make(chan string, 5)
	limit := make(chan bool, 5)
	errorsListen := make(chan error, 1)
	sing := make(chan time.Time, 1)
	finish := make(chan bool, 1)
	counters := &sync.Map{}

	// считывает введенные данные
	go scaner(urls, sing)

	// обрабатывает url
	go func() {
		for url := range urls {
			limit <- true
			go processingURL(limit, errorsListen, counters, url)
		}
	}()

	// отвечает за завершение приложения, если в течении минуты не было введено новых данных
	go timer(sing, finish)

	// слушатель ошибок
	go func() {
		for range time.Tick(time.Second) {
			err := <-errorsListen
			if err != nil {
				fmt.Println("err", err)
			}
		}
	}()

	// ожидание либо сигнала о завершении, либо получения каналом finish значения
	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-finish:
		fmt.Println("time")
		getResult(counters)
		fmt.Println("exit")
		os.Exit(1)
	case <-osSignals:
		getResult(counters)
		fmt.Println("exit")
		os.Exit(1)
	}
}
