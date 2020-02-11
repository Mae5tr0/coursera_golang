package main

import (
	"fmt"
)

func main() {
	inputData := []int{0, 1, 1, 2, 3, 5, 8}

	hashSignJobs := []job{
		job(func(in, out chan interface{}) {
			fmt.Println("Generator")
			for _, fibNum := range inputData {
				out <- fibNum
			}
		}),
		job(SingleHash),
		job(MultiHash),
		job(CombineResults),
		job(func(in, out chan interface{}) {			
			for dataRaw := range in {	
				fmt.Println("Printer", dataRaw.(string))				
			}
		}),
	}

	ExecutePipeline(hashSignJobs...)
}