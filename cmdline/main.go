package main

import (
	"bufio"
	"flag"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"

	"github.com/pwiecz/portal_patterns/lib"
)

func main() {
	fileBase := filepath.Base(os.Args[0])
	cpuprofile := flag.String("cpuprofile", "", "write CPU profile to this file")
	numWorkersFlag := flag.Int("num_workers", 0, "if applicable for given algorithm use that many worker threads. If <= 0 use as many as there are CPUs on the machine")
	showProgress := flag.Bool("progress", true, "show progress bar")
	output := flag.String("output", "-", "write output to this file, instead of printing it to stdout")
	flag.BoolVar(showProgress, "P", true, "show progress bar")
	cobwebCmd := NewCobwebCmd()
	herringboneCmd := NewHerringboneCmd()
	doubleHerringboneCmd := NewDoubleHerringboneCmd()
	threeCornersCmd := NewThreeCornersCmd()
	flipFieldCmd := NewFlipFieldCmd()
	homogeneousCmd := NewHomogeneousCmd()
	droneFlightCmd := NewDroneFlightCmd()

	defaultUsage := flag.Usage
	flag.Usage = func() {
		defaultUsage()
		cobwebCmd.Usage(fileBase)
		threeCornersCmd.Usage(fileBase)
		herringboneCmd.Usage(fileBase)
		doubleHerringboneCmd.Usage(fileBase)
		flipFieldCmd.Usage(fileBase)
		homogeneousCmd.Usage(fileBase)
		droneFlightCmd.Usage(fileBase)
	}
	flag.Parse()
	if len(flag.Args()) <= 1 {
		flag.Usage()
		os.Exit(0)
	}
	numWorkers := runtime.GOMAXPROCS(0)
	if *numWorkersFlag > 0 {
		numWorkers = *numWorkersFlag
	}
	var outputWriter io.Writer
	outputWriter = os.Stdout
	if *output != "-" {
		outputFile, err := os.Create(*output)
		if err != nil {
			log.Fatal("cannot write to file ", *output, " : ", err)
		}
		defer outputFile.Close()
		outputFileWriter := bufio.NewWriter(outputFile)
		defer outputFileWriter.Flush()
		outputWriter = outputFileWriter
	}
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}
	progressFunc := lib.PrintProgressBar
	if !*showProgress {
		progressFunc = func(int, int) {}
	}
	switch flag.Args()[0] {
	case "cobweb":
		cobwebCmd.Run(flag.Args()[1:], outputWriter, progressFunc)
	case "herringbone":
		herringboneCmd.Run(flag.Args()[1:], outputWriter, numWorkers, progressFunc)
	case "double_herringbone":
		doubleHerringboneCmd.Run(flag.Args()[1:], outputWriter, numWorkers, progressFunc)
	case "flip_field":
		flipFieldCmd.Run(flag.Args()[1:], numWorkers, outputWriter, progressFunc)
	case "three_corners":
		threeCornersCmd.Run(flag.Args()[1:], outputWriter, progressFunc)
	case "homogeneous":
		fallthrough
	case "homogenous":
		homogeneousCmd.Run(flag.Args()[1:], outputWriter, numWorkers, progressFunc)
	case "drone_flight":
		droneFlightCmd.Run(flag.Args()[1:], numWorkers, outputWriter, progressFunc)
	default:
		log.Fatalf("Unknown command: \"%s\"\n", flag.Args()[0])
	}
}
