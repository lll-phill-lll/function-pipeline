package main

import (
	"log"
	"os"
	"sort"
	"strconv"
	"sync"
)

const TH = 6
var Logger *log.Logger

// ExecutePipeline runs each of given jobs as a pipeline
func ExecutePipeline(jobs ...job) {
	// define logger
	Logger = log.New(os.Stdout,
		"",
		log.Ldate|log.Lmicroseconds|log.Lshortfile)

	jobsNum := len(jobs)
	// create a channel for each job
	chans := make([]chan interface{}, jobsNum + 1)
	for i := range chans {
		chans[i] = make(chan interface{})
	}
	Logger.Println("Created", len(chans), "channels for", jobsNum, "jobs")

	var wg sync.WaitGroup
	for i, singleJob := range jobs {
		wg.Add(1)
		go func(sJ job, i, o chan interface{}) {
			defer  wg.Done()
			sJ(i, o)
			close(o)
		} (singleJob, chans[i], chans[i + 1])
	}
	wg.Wait()
}

// mutex to avoid md5 overheating
var m sync.Mutex

// SingleHash computes crc32(data)+"~"+crc32(md5(data)) for each value from in channel
func SingleHash(in, out chan interface{}) {
	var wg sync.WaitGroup
	for input := range in {
		value, ok := input.(int)
		if !ok {
			Logger.Println("Can't convert", input, "to int")
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			// compute crc32 of value
			crc32chan := make (chan string)
			go func (ch chan string){
				crc32 := DataSignerCrc32(strconv.Itoa(value))
				Logger.Println("crc32 of", value, "is", crc32)
				ch <- crc32
			}(crc32chan)

			// compute md5 and crc32 of value
			md5crc32chan := make(chan string)
			go func(ch chan string)	 {
				m.Lock()
				md5 := DataSignerMd5(strconv.Itoa(value))
				Logger.Println("md5 of", value, "is", md5)
				m.Unlock()

				md5crc32 := DataSignerCrc32(md5)
				Logger.Println("crc32 of md5 of", value, "is", md5crc32)
				ch <- md5crc32
			}(md5crc32chan)

			// collect hashes
			singleHash := <-crc32chan + "~" + <-md5crc32chan
			Logger.Println("SingleHash of", value, "is", singleHash)
			out <- singleHash
		}()
	}
	wg.Wait()
}

// MultiHash computes crc32(th+data) where th = (0...5) and concatenates hashes
func MultiHash(in, out chan interface{}) {
	var wg sync.WaitGroup
	for input := range in {
		value, ok := input.(string)
		if !ok {
			Logger.Println("Can't convert", input, "to string")
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			chans := make([]chan string, TH)

			for i := 0; i != TH; i++ {
				chans[i] = make(chan string)

				go func(ch chan string, i int) {
					crc32TH := DataSignerCrc32(strconv.Itoa(i) + value)
					Logger.Println("crc32 for", i, "+", value, "is", crc32TH)
					ch <- crc32TH
				}(chans[i], i)
			}

			crc32TH := ""
			for i := 0; i != TH; i++ {
				crc32TH += <-chans[i]
			}
			Logger.Println("Full crc32 for", value, "is", crc32TH)

			out <- crc32TH
		}()
	}
	wg.Wait()
}

// CombineResults sorts all the hashes and concatenates separating with _
func CombineResults(in, out chan interface{}) {
	ans := ""
	hashes := make ([]string, 0)

	for input := range in {
		value, ok := input.(string)
		if !ok {
			Logger.Println("Can't convert", input, "to string")
		}
		hashes = append(hashes, value)
	}
	sort.Strings(hashes)

	for i, hash := range hashes {
		ans += hash
		if i < len(hashes) - 1 {
			ans += "_"
		}
	}
	out <- ans
}

// For local tests

// writeJob creates simple job - writes numbers to channel (don't forget to use readJob
// or something that reads from the channel to avoid a deadlock)
func writeJob(in, out chan interface{}) {
	Logger.Println("writeJob started")
	for i := 0; i != 10; i++ {
		out <- i
	}
	Logger.Println("writeJob finished")
}

// readJob is a pair for writeJob, it reads numbers from channel
func readJob(in, out chan interface{}) {
	Logger.Println("readJob started")
	for a := range in {
		Logger.Println(a)
	}
	Logger.Println("readJob finished")
}

// printResult prints the final hash
func printResult(in, out chan interface{}) {
	Logger.Println("printResult started")
	Logger.Println((<-in).(string))
	Logger.Println("printResult finished")
}

func main() {
	ExecutePipeline(job(writeJob), job(SingleHash), job(MultiHash), job(CombineResults), job(printResult))
}
