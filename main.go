package main

import (
	"bufio"
	"bytes"
	"flag"
	"os"
	"time"

	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/prometheus/prometheus/storage/local"

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
	cfgMemoryStorage := local.MemorySeriesStorageOptions{
		MemoryChunks:       1024,
		MaxChunksToPersist: 1024,
		//PersistenceStoragePath:
		//PersistenceRetentionPeriod:
		//CheckpointInterval:         time.Minute*30,
		//CheckpointDirtySeriesLimit: 10000,
		Dirty:          true,
		PedanticChecks: true,
		SyncStrategy:   local.Always,
	}

	// Number of bytes to read before doing a sample append (approx)
	var ingestionBuffer int

	app := kingpin.New("prometheus-prefill", "command line utility to manually fill a prometheus data store")
	app.Version(Version)

	app.Flag("storage.path", "Directory path to create and fill the data store under.").Default("data").StringVar(&cfgMemoryStorage.PersistenceStoragePath)
	app.Flag("storage.retention-period", "Period of time to store data for").Default("360h").DurationVar(&cfgMemoryStorage.PersistenceRetentionPeriod)

	app.Flag("storage.checkpoint-interval", "Period of time to store data for").Default("30m").DurationVar(&cfgMemoryStorage.CheckpointInterval)
	app.Flag("storage.checkpoint-dirty-series-limit", "Period of time to store data for").Default("10000").IntVar(&cfgMemoryStorage.CheckpointDirtySeriesLimit)

	app.Flag("prefiller.buffer-size", "Amount of data to buffer for ingestion in bytes").Default("104857600").IntVar(&ingestionBuffer)

	logLevel := app.Flag("log.level", "Logging level").Default("info").String()

	kingpin.MustParse(app.Parse(os.Args[1:]))
	if err := flag.Set("log.level", *logLevel); err != nil {
		log.Fatalln("invalid --log-level")
	}

	log.Infoln("Prefilling into", cfgMemoryStorage.PersistenceStoragePath)
	localStorage := local.NewMemorySeriesStorage(&cfgMemoryStorage)

	sampleAppender := localStorage

	log.Infoln("Starting the storage engine")
	if err := localStorage.Start(); err != nil {
		log.Errorln("Error opening memory series storage:", err)
		return 1
	}
	defer func() {
		if err := localStorage.Stop(); err != nil {
			log.Errorln("Error stopping storage:", err)
		}
	}()

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

		for sampleAppender.NeedsThrottling() {
			log.Debugln("Waiting 100ms for appender to be ready for more data")
			time.Sleep(time.Millisecond * 100)
		}

		var (
			numOutOfOrder = 0
			numDuplicates = 0
		)

		for _, s := range model.Samples(decSamples) {
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
