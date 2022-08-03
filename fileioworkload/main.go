package main

import (
	"log"
	"os"
	"strconv"
	"time"
)

func main() {
	writeInterval := 10
	argLength := len(os.Args[1:])
	if argLength != 0 {
		var err error
		writeInterval, err = strconv.Atoi(os.Args[1])
		if err != nil {
			log.Fatal(err)
		}
	}

	counter := 0
	for true {
		counter++
		filename := strconv.Itoa(counter) + ".txt"

		f, err := os.Create(filename)
		if err != nil {
			log.Fatal(err)
		}
		err = f.Close()
		if err != nil {
			log.Fatal(err)
		}

		time.Sleep(time.Duration(writeInterval) * time.Second)
	}
}
