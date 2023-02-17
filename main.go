package main

import (
	"bufio"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func main() {
	pattern := regexp.MustCompile(`^\.(?P<label>\w+)\s+(?P<size>\d+)\s+(?P<addr>\d+)$`)

	pflag.CommandLine.String("metrics.outfile", "out.prom", "The filename of the output OpenMetrics file.")
	pflag.CommandLine.String("report.infile", "memory.stats", "The filename of the input memory report file.")
	pflag.Parse()
	err := viper.BindPFlags(pflag.CommandLine)
	if err != nil {
		logrus.Error(err)
	}

	logrus.WithFields(viper.AllSettings()).Debug("reconciled configuration")

	// Create our Prometheus registry and register our metrics.
	registry := prometheus.NewRegistry()
	binAddrMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Help: "The address of the section of the binary.",
		Name: "elf_binary_section_addr",
	}, []string{"label", "unit"})
	binSizeMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Help: "The size of the section of the binary.",
		Name: "elf_binary_section_size",
	}, []string{"label", "unit"})
	registry.MustRegister(binAddrMetric)
	registry.MustRegister(binSizeMetric)

	report, err := os.Open(viper.GetString("report.infile"))
	if err != nil {
		logrus.Error(err)
	}

	scanner := bufio.NewScanner(report)
	scanner.Split(bufio.ScanLines)

	// Scan each line.
	for scanner.Scan() {
		line := scanner.Text()

		// If the line starts with a period, it's a data line.
		if strings.HasPrefix(line, ".") {
			parts := pattern.FindStringSubmatch(line)
			if len(parts) <= 0 {
				continue
			}

			// Parse the binary section address from the line.
			addr, err := strconv.ParseFloat(parts[3], 64)
			if err != nil {
				logrus.Fatal(err)
			}

			// Parse the binary section size from the line.
			size, err := strconv.ParseFloat(parts[2], 64)
			if err != nil {
				logrus.Fatal(err)
			}

			// Update our metrics.
			binAddrMetric.With(prometheus.Labels{
				"label": parts[1],
				"unit":  "byte",
			}).Set(addr)

			binSizeMetric.With(prometheus.Labels{
				"label": parts[1],
				"unit":  "byte",
			}).Set(size)
		}
	}

	report.Close()

	err = prometheus.WriteToTextfile(viper.GetString("metrics.outfile"), registry)
	if err != nil {
		logrus.Error(err)
	}
}
