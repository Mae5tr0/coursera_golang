package main

import (
	"fmt"
	"strings"
	"sort"
	"sync"
)

func worker(wg *sync.WaitGroup, j job, in, out chan interface{}) {
	defer wg.Done()
	defer close(out)
	j(in, out)
}

func ExecutePipeline(jobs ...job) {
	in := make(chan interface{})
	out := make(chan interface{})

	wg := &sync.WaitGroup{}

	for _, job := range jobs {
		wg.Add(1)
		go worker(wg, job, in, out)

		in = out
		out = make(chan interface{})
	}
	wg.Wait()
}

func calcCrc32(data string, out chan string) {
	out <- DataSignerCrc32(data)
}

func calcCrc32Md5(data string, mu *sync.Mutex, out chan string) {
	mu.Lock()
	md5 := DataSignerMd5(data)
	mu.Unlock()

	out <- DataSignerCrc32(md5)
}

func workerSingleHash(data string, wg *sync.WaitGroup, mu *sync.Mutex, out chan interface{}) {
	defer wg.Done()
	crc32Ch := make(chan string)
	crc32Ch2 := make(chan string)

	go calcCrc32(data, crc32Ch)
	go calcCrc32Md5(data, mu, crc32Ch2)

	out <- fmt.Sprintf("%s~%s", 
		<- crc32Ch, 
		<- crc32Ch2,		
	)
}

func SingleHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}
	mu := &sync.Mutex{}
	for rawData := range in {
		wg.Add(1)
		data := fmt.Sprintf("%d", rawData.(int))
		fmt.Println("SingleHash", data)
		go workerSingleHash(data, wg, mu, out)
	}
	wg.Wait()
}

func workerMultiHash(data string, wg *sync.WaitGroup, out chan interface{}) {
	defer wg.Done()

	subW := &sync.WaitGroup{}
	crc32Items := make([]string, 6)
	for i := 0; i < 6; i++ {
		subW.Add(1)
		go func(data string, wg *sync.WaitGroup, num int, result []string) {
			defer wg.Done()
			result[num] = DataSignerCrc32(data)
		}(fmt.Sprintf("%d%s", i, data), subW, i, crc32Items)
	}
	subW.Wait()
	var result strings.Builder
	for _, crc32 := range crc32Items {
		result.WriteString(crc32)
	}
	out <- result.String()
}
 
func MultiHash(in, out chan interface{}) {	
	wg := &sync.WaitGroup{}
	for rawData := range in {
		wg.Add(1)
		fmt.Println("MultiHash", rawData.(string))
		go workerMultiHash(rawData.(string), wg, out)		
	}
	wg.Wait()
}

func CombineResults(in, out chan interface{}) {
	elements := []string{}
	for data := range in {
		fmt.Println("CombineResults", data.(string))
		elements = append(elements, data.(string))		
	}
	sort.Sort(sort.StringSlice(elements))
	out <- strings.Join(elements, "_")
}