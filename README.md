[![Build Status](https://travis-ci.org/wrouesnel/prometheus-prefiller.svg?branch=master)](https://travis-ci.org/wrouesnel/prometheus-prefiller)
[![Coverage Status](https://coveralls.io/repos/github/wrouesnel/prometheus-prefiller/badge.svg?branch=master)](https://coveralls.io/github/wrouesnel/prometheus-prefiller?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/wrouesnel/prometheus-prefiller)](https://goreportcard.com/report/github.com/wrouesnel/prometheus-prefiller)

**DEPRECATTED** : this was a proof-of-concept hacked together quite quickly, and had some pretty serious performance issues when dealing with a lot of metrics. Use https://github.com/Cleafy/promqueen for a much better solution.

# Prometheus Prefiller

A prototype tool for pre-initializing a [Prometheus](http://prometheus.io/)
on-disk data store.

Useful for setting up standalone Prometheus instances that should be preloaded
with data. Be careful loading very deep time-series without setting an
appropriate retention time in the Prometheus instance when it starts.

## Getting Started
```bash

$ for i in {1..1000} ; \
    do d=$(date +%s) ; \
    echo test_metric $i $(($(($d*1000 - 8640000 )) + $(($i * 15000)) )) ; \
    echo test_metric_b $i $(($(($d*1000 - 8640000 )) + $(($i * 15000)) ))  ; \
    done > test.prom
$ cat test.prom | ./prometheus-prefiller --log.level debug
$ echo "global: {}" > prometheus.yml
$ prometheus
```

You will now be able to query `test_metric` and `test_metric_b` over the last
day.

## Usage

Metrics are ingested in the prometheus text ingestion format, described 
[here](https://prometheus.io/docs/instrumenting/exposition_formats/) and visible
from any Prometheus exporter endpoint.

Basic format:
```
metric_name value timestamp
```

With labels (probably what you want):
```
metric_name{label="value", label2="value2"} value timestamp
```

Unlike regular prometheus metrics, you *should* be appending a unix timestamp
in milliseconds to backfill the data - Prometheus's memory storage engine will
only accept monotonically increasing timestamps, so your data needs to be sorted
- this is not a backfilling solution. Identical timestamps are allowed but they
must never go backwards.

The Prometheus engine writes a lot of logging by default - I've not supressed
it since this is a quick and dirty tool to deal with quite a specific requirement
for the time being.

