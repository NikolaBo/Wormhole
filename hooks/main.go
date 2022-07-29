package main

import (
	"bufio"
	"errors"
	"log"
	"os"
	"strconv"
)

func main() {
	filename := "/mnt/azure/poststart.txt"
	argLength := len(os.Args[1:])
	if argLength != 0 {
		filename = "/mnt/azure/prestop.txt"
	}

	f, err := os.OpenFile(filename, os.O_RDWR, 0755)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			f, err = os.Create(filename)
			_, err = f.WriteString("1\n")
			if err != nil {
				log.Fatal(err)
			}
		} else {
			log.Fatal(err)
		}
	} else {
		scanner := bufio.NewScanner(f)
		scanner.Scan()
		text := scanner.Text()
		num, err := strconv.Atoi(text)
		if err != nil {
			log.Fatal(err)
		}
		num++
		f.Seek(0, 0)
		_, err = f.WriteString(strconv.Itoa(num) + "\n")
		if err != nil {
			log.Fatal(err)
		}
	}
	defer f.Close()
}
