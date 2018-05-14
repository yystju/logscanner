package main

import (
	"archive/zip"
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
)

type LineFilterType []string

func (lft *LineFilterType) String() string {
	return strings.Join(*lft, ",")
}

func (lft *LineFilterType) Set(value string) error {
	*lft = append(*lft, value)
	return nil
}

var (
	arg_folder               string
	arg_filefilter           string
	arg_is_filefilter_regexp bool
	arg_linefilter           LineFilterType
	arg_is_linefilter_regexp bool
	arg_output               string
	arg_log_pathname         string

	logger *log.Logger

	log_file *os.File
)

func InitParam() {
	flag.StringVar(&arg_folder, "d", "./logs/", "The log folder to parse.")
	flag.StringVar(&arg_filefilter, "f", "", "The file name filter.")
	flag.BoolVar(&arg_is_filefilter_regexp, "fr", false, "The flag which represents whether the file filter is regular expression or not.")
	flag.Var(&arg_linefilter, "l", "The line filters. This flag can be repeated for multiple times with different filter contents.")
	flag.BoolVar(&arg_is_linefilter_regexp, "lr", false, "The flag which represents whether the line filter is regular expression or not.")
	flag.StringVar(&arg_output, "o", "-", "The output file. default \"-\" indicates the standard output stream (stdout).")
	flag.StringVar(&arg_log_pathname, "g", "-", "The log file. The default \"-\" indicates that the executable name will be used as log name.")
	flag.Parse()
}

func init() {
	InitParam()

	//logger
	execName, err := os.Executable()

	if err != nil {
		panic(err)
	}

	if arg_log_pathname != "-" {
		log_file, err = os.OpenFile(arg_log_pathname, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	} else {
		log_file, err = os.OpenFile(fmt.Sprintf("%s.log", execName), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	}

	if err != nil {
		panic(err)
	}

	logger = log.New(log_file, "", log.Lshortfile)

	logger.Printf("PARAMS: d : %s, f : %s, fr : %t, l : %s, lr : %t, o : %s, g : %s\n", arg_folder, arg_filefilter, arg_is_filefilter_regexp, arg_linefilter, arg_is_linefilter_regexp, arg_output, arg_log_pathname)
}

type Visitor func(name string, lineno int, line string) bool

func table_scan(name string, filters []string, visitor Visitor) {
	f, err := os.OpenFile(name, os.O_RDONLY, 0666)

	if err != nil {
		panic(err)
	}

	defer f.Close()

	table_scan_reader(f, name, filters, visitor)
}

func table_scan_reader(r io.Reader, name string, filters []string, visitor Visitor) {
	s := bufio.NewScanner(r)

	no := 1

	for s.Scan() {
		text := s.Text()

		var match bool

		if arg_is_linefilter_regexp {
			match = true

			for _, filter := range filters {
				if !match {
					break
				}

				m, _ := regexp.MatchString(filter, text)

				match = match && m
			}
		} else {
			match = true

			for _, filter := range filters {
				if !match {
					break
				}

				match = match && (strings.Index(text, filter) > -1)
			}
		}

		if match {
			if !visitor(name, no, text) {
				break
			}
		}

		no++
	}

	if err := s.Err(); err != nil {
		panic(err)
	}
}

func ScanDirByLineInternal(f os.FileInfo, folder string, fileFilter string, lineFilters []string, visitor Visitor, wait *sync.WaitGroup) {
	logger.Println("GOROUTINES : ", runtime.NumGoroutine())

	pathName := filepath.Join(folder, f.Name())

	isZip, _ := regexp.MatchString(".*\\.zip", f.Name())

	if isZip {
		zipReader, err := zip.OpenReader(pathName)

		if err != nil {
			panic(err)
		}

		for _, zipf := range zipReader.File {
			if !zipf.FileInfo().IsDir() {
				rc, err := zipf.Open()

				if err != nil {
					panic(err)
				}

				table_scan_reader(rc, pathName, lineFilters, visitor)

				rc.Close()
			}
		}

		zipReader.Close()
	} else {
		table_scan(pathName, lineFilters, visitor)
	}

	wait.Done()
}
func ScanDirByLine(folder string, fileFilter string, lineFilters []string, visitor Visitor) {
	files, err := ioutil.ReadDir(folder)

	if err != nil {
		panic(err)
	}

	wait := new(sync.WaitGroup)

	for _, f := range files {
		var isFileMatched bool

		if arg_is_filefilter_regexp {
			isFileMatched, _ = regexp.MatchString(fileFilter, f.Name())
		} else {
			isFileMatched = (strings.Index(f.Name(), fileFilter) > -1)
		}

		if !isFileMatched {
			continue
		}

		wait.Add(1)

		go ScanDirByLineInternal(f, folder, fileFilter, lineFilters, visitor, wait)
	}

	wait.Wait()
}

func main() {
	defer log_file.Close()

	logger.Printf("GOMAXPROCS : %d\n", runtime.GOMAXPROCS(0))

	var outs io.Writer = os.Stdout

	if arg_output != "-" {
		o, err := os.OpenFile(arg_output, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)

		if err != nil {
			panic(err)
		}

		defer o.Close()

		outs = o
	}

	var filterArray []string = arg_linefilter

	ScanDirByLine(arg_folder, arg_filefilter, filterArray, func(name string, linno0 int, line0 string) bool {
		fmt.Fprintf(outs, "%s[%d]%s\n", name, linno0, line0)
		return true
	})
}
