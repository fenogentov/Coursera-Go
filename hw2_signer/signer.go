package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
)

func ExecutePipeline(tasks ...job) {
	wg := &sync.WaitGroup{}
	chIN := make(chan interface{})

	for _, task := range tasks {
		chOUT := make(chan interface{})
		wg.Add(1)
		go func(task job, in chan interface{}, out chan interface{}, wg *sync.WaitGroup) {
			defer close(out)
			defer wg.Done()
			task(in, out)

		}(task, chIN, chOUT, wg)
		chIN = chOUT
	}
	wg.Wait()
}

func rawToSrting(dRAW interface{}) (string, error) {
	var dString string
	var err error

	switch x := dRAW.(type) {
	case int:
		fmt.Printf("%v \n", x.Value)
		d, ok := dRAW.(int)
		if !ok {
			fmt.Println("cant convert result data to string")
			//			err.Error("cant convert result data to string")
		} else {
			dString = strconv.FormatUint(uint64(d), 10)
		}
	case uint32:
		d, ok := dRAW.(uint32)
		if !ok {
			fmt.Println("cant convert result data to string")
			//			err.Error("cant convert result data to string")
		} else {
			dString = strconv.FormatUint(uint64(d), 10)
		}
	case string:
		d, ok := dRAW.(string)
		if !ok {
			fmt.Println("cant convert result data to string")
			//			err.Error("cant convert result data to string")
		} else {
			dString = d
		}
	}
	return dString, err
}

func SingleHash(in, out chan interface{}) {
	wgSH := &sync.WaitGroup{}
	lock := &sync.Mutex{}

	for dataRaw := range in {
		wgSH.Add(1)
		go func(d interface{}) {
			defer wgSH.Done()
			var h1, h2 string
			data, _ := rawToSrting(d)

			chH1 := make(chan string)
			go func(chH1 chan string) {
				chH1 <- DataSignerCrc32(data)
			}(chH1)

			lock.Lock()
			h2 = DataSignerMd5(data)
			lock.Unlock()

			h2 = DataSignerCrc32(h2)
			h1 = <-chH1

			out <- h1 + "~" + h2
		}(dataRaw)
	}
	wgSH.Wait()
}

func MultiHash(in, out chan interface{}) {
	wgMH := &sync.WaitGroup{}
	wgCRC := &sync.WaitGroup{}

	for dataRaw := range in {
		wgMH.Add(1)

		go func(d interface{}) {
			defer wgMH.Done()
			var arr [6]string

			data, ok := d.(string)
			if !ok {
				fmt.Println("cant convert result data to string")
				//t.Error("cant convert result data to string")
			}

			for i := 0; i < 6; i++ {
				wgCRC.Add(1)
				go func(i int) {
					defer wgCRC.Done()
					arr[i] = DataSignerCrc32(strconv.Itoa(i) + data)
				}(i)
			}
			wgCRC.Wait()

			sl := arr[:]
			out <- strings.Join(sl, "")

		}(dataRaw)
	}
	wgMH.Wait()
}

func CombineResults(in, out chan interface{}) {
	var sl []string
	for x := range in {
		data, ok := x.(string)
		if !ok {
			fmt.Println("cant convert result data to string")
			//t.Error("cant convert result data to string")
		}
		sl = append(sl, data)
	}
	sort.Strings(sl)
	out <- strings.Join(sl, "_")
}
