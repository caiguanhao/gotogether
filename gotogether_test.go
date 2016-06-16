package gotogether_test

import (
	"crypto/md5"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/caiguanhao/gotogether"
)

func Example_enumerable() {
	var items []interface{}

	items = append(items, 100)
	items = append(items, 200)

	gotogether.Enumerable(items).Each(func(item interface{}) {
		seconds := item.(int)
		duration := time.Millisecond * time.Duration(seconds)
		time.Sleep(duration)
		fmt.Println(item, duration, "OK")
	})

	fmt.Println("All OK")

	// is the same as:

	gotogether.Enumerable(items).Parallel(func(item interface{}) {
		seconds := item.(int)
		duration := time.Millisecond * time.Duration(seconds)
		time.Sleep(duration)
		fmt.Println(item, duration, "OK")
	}).Run()

	fmt.Println("All OK")
	// Output:
	// 100 100ms OK
	// 200 200ms OK
	// All OK
	// 100 100ms OK
	// 200 200ms OK
	// All OK
}

func Example_enumerableQueue() {
	gotogether.Enumerable([]interface{}{100, 200, 101, 201, 102, 202}).QueueWithIndex(func(item interface{}, i int) {
		seconds := item.(int)
		duration := time.Millisecond * time.Duration(seconds)
		time.Sleep(duration)
		fmt.Println(item, duration, i, "OK")
	}).WithConcurrency(2).Run()

	fmt.Println("All OK")

	// Output:
	// 100 100ms 0 OK
	// 200 200ms 1 OK
	// 101 101ms 2 OK
	// 102 102ms 4 OK
	// 201 201ms 3 OK
	// 202 202ms 5 OK
	// All OK
}

func Example_enumerableWithIndex() {
	var items []interface{}

	items = append(items, "a")
	items = append(items, "b")

	gotogether.Enumerable(items).EachWithIndex(func(item interface{}, i int) {
		duration := time.Millisecond * 100 * time.Duration(i+1)
		time.Sleep(duration)
		fmt.Println(item, duration, "OK")
	})

	fmt.Println("All OK")

	// is the same as:

	gotogether.Enumerable(items).ParallelWithIndex(func(item interface{}, i int) {
		duration := time.Millisecond * 100 * time.Duration(i+1)
		time.Sleep(duration)
		fmt.Println(item, duration, "OK")
	}).Run()

	fmt.Println("All OK")
	// Output:
	// a 100ms OK
	// b 200ms OK
	// All OK
	// a 100ms OK
	// b 200ms OK
	// All OK
}

func Example_parallel() {
	gotogether.Parallel{
		func() {
			time.Sleep(100 * time.Millisecond)
			fmt.Println("1 OK")
		},
		func() {
			time.Sleep(300 * time.Millisecond)
			fmt.Println("3 OK")
		},
		func() {
			time.Sleep(200 * time.Millisecond)
			fmt.Println("2 OK")
		},
	}.Run()

	fmt.Println("All OK")
	// Output:
	// 1 OK
	// 2 OK
	// 3 OK
	// All OK
}

func Example_queue() {
	// make some files for test
	const TEST_FIXTURES_DIR = "test/fixtures"
	os.MkdirAll(TEST_FIXTURES_DIR, 0755)
	for i := 0; i < 6; i++ {
		ioutil.WriteFile(fmt.Sprintf("%s/%d", TEST_FIXTURES_DIR, i), []byte{byte(48 + i)}, 0644)
	}

	totalErrors := 0

	gotogether.Queue{
		Concurrency: 1, // set to 1 for constant output of this test; it can can be any integer great than 0
		AddJob: func(jobs *chan interface{}, done *chan interface{}, errs *chan error) {
			*errs <- filepath.Walk(TEST_FIXTURES_DIR, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.Mode().IsRegular() {
					return nil
				}
				select {
				case *jobs <- []interface{}{path, info.Size()}:
				case <-*done:
					return errors.New("walk canceled")
				}
				return nil
			})
		},
		OnAddJobError: func(err *error) {
			fmt.Fprintln(os.Stderr, *err)
			totalErrors++
		},
		DoJob: func(job *interface{}) (ret interface{}, err error) {
			jobInfo := (*job).([]interface{})
			path, size := jobInfo[0], jobInfo[1]

			var data []byte
			data, err = ioutil.ReadFile(path.(string))
			if err != nil {
				return
			}

			ret = []interface{}{path, fmt.Sprintf("%X", md5.Sum(data)), size}
			return
		},
		OnJobError: func(err *error) {
			fmt.Fprintln(os.Stderr, *err)
			totalErrors++
		},
		OnJobSuccess: func(ret *interface{}) {
			rets := (*ret).([]interface{})
			path, md5, size := rets[0].(string), rets[1].(string), rets[2].(int64)
			fmt.Printf("Result:  %s (%d bytes) = %s\n", path, size, md5)
		},
	}.Run()

	if totalErrors == 0 {
		fmt.Println("All OK")
	}
	// Output:
	// Result:  test/fixtures/0 (1 bytes) = CFCD208495D565EF66E7DFF9F98764DA
	// Result:  test/fixtures/1 (1 bytes) = C4CA4238A0B923820DCC509A6F75849B
	// Result:  test/fixtures/2 (1 bytes) = C81E728D9D4C2F636F067F89CC14862C
	// Result:  test/fixtures/3 (1 bytes) = ECCBC87E4B5CE2FE28308FD9F2A7BAF3
	// Result:  test/fixtures/4 (1 bytes) = A87FF679A2F3E71D9181A67B7542122C
	// Result:  test/fixtures/5 (1 bytes) = E4DA3B7FBBCE2345D7772B0674A318D5
	// All OK
}
