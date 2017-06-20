package main

import (
	"bufio"
	"bytes"
	"flag"
	"os"
	"time"

	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/wrouesnel/prometheus-prefiller/local"

	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/model"
)

// Version of the binary.
var Version string

func main() {
	os.Exit(realMain())
}

func realMain() int {

	app := kingpin.New("prometheus-prefill", "command line utility to manually fill a prometheus data store")
	app.Version(Version)

	basePath := app.Flag("storage.path", "Directory path to create and fill the data store under.").Default("data").String()
	ingestionBuffer := app.Flag("prefiller.buffer-size", "Amount of data to buffer for ingestion in bytes").Default("104857600").Int()
	logLevel := app.Flag("log.level", "Logging level").Default("info").String()

	kingpin.MustParse(app.Parse(os.Args[1:]))
	if err := flag.Set("log.level", *logLevel); err != nil {
		log.Fatalln("invalid --log-level")
	}

	log.Infoln("Prefilling into", *basePath)
	//localStorage := local.NewMemorySeriesStorage(&cfgMemoryStorage)
	//sampleAppender := localStorage

	persistEngine, err := local.NewPersistence(*basePath,
		false, false, func() bool {return false}, 0)
	if err != nil {
		log.Fatalln("Could not initialize persistence:", err)
	}

	//defer func() {
	//	if err := localStorage.Stop(); err != nil {
	//		log.Errorln("Error stopping storage:", err)
	//	}
	//}()

	// Ingest samples line-by-line from stdin.
	rdr := bufio.NewScanner(os.Stdin)
	for {
		inpBuf := new(bytes.Buffer)
		if !rdr.Scan() {
			break
		}
		for rdr.Scan() {
			inpBuf.Write(append(rdr.Bytes(), '\n'))
			if inpBuf.Len() >= ingestionBuffer {
				log.Infoln("Ingestion buffer full: flushing", inpBuf.Len())
				break
			}
		}

		sdec := expfmt.SampleDecoder{
			Dec: expfmt.NewDecoder(inpBuf, expfmt.FmtText),
			Opts: &expfmt.DecodeOptions{
				Timestamp: model.Now(),
			},
		}

		decSamples := make(model.Vector, 0, 1)

		if err := sdec.Decode(&decSamples); err != nil {
			log.Errorln("Could not decode metric:", err)
			continue
		}

		log.Debugln("Ingested", len(decSamples), "metrics")

		//for sampleAppender.NeedsThrottling() {
		//	log.Debugln("Waiting 100ms for appender to be ready for more data")
		//	time.Sleep(time.Millisecond * 100)
		//}

		var (
			numOutOfOrder = 0
			numDuplicates = 0
		)

		for _, s := range model.Samples(decSamples) {
			rawFP := s.Metric.FastFingerprint()
			
			if err := sampleAppender.Append(s); err != nil {
				switch err {
				case local.ErrOutOfOrderSample:
					numOutOfOrder++
					log.With("sample", s).With("error", err).Debug("Sample discarded")
				case local.ErrDuplicateSampleForTimestamp:
					numDuplicates++
					log.With("sample", s).With("error", err).Debug("Sample discarded")
				default:
					log.With("sample", s).With("error", err).Warn("Sample discarded")
				}
			}
		}
	}

	log.Infoln("Shutting down cleanly.")

	return 0
}
