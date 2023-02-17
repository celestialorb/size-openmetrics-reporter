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

var (
	binAddrMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Help: "The address of the section of the binary.",
		Name: "elf_binary_section_addr",
	}, []string{"label", "unit"})

	binSizeMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Help: "The size of the section of the binary.",
		Name: "elf_binary_section_size",
	}, []string{"label", "unit"})

	ramUsageMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Help: "The percentage usage of RAM by the binary from 0-1.",
		Name: "elf_binary_ram_usage",
	})
)

func main() {
	pattern := regexp.MustCompile(`^\.(?P<label>\w+)\s+(?P<size>\d+)\s+(?P<addr>\d+)$`)

	pflag.CommandLine.String("metrics.outfile", "out.prom", "The filename of the output OpenMetrics file.")
	pflag.CommandLine.String("report.infile", "memory.stats", "The filename of the input memory report file.")
	pflag.CommandLine.Bool("metrics.derived.ram", false, "Include a derived RAM usage metric.")
	pflag.CommandLine.Bool("metrics.sections.addrs", false, "Include metrics on the section addresses of the binary.")
	pflag.CommandLine.Bool("metrics.sections.sizes", true, "Include metrics on the section sizes of the binary.")
	pflag.Parse()
	err := viper.BindPFlags(pflag.CommandLine)
	if err != nil {
		logrus.Error(err)
	}

	logrus.WithFields(viper.AllSettings()).Debug("reconciled configuration")

	// Create our Prometheus registry and register our metrics.
	registry := prometheus.NewRegistry()
	registry.MustRegister(binAddrMetric)
	registry.MustRegister(binSizeMetric)
	registry.MustRegister(ramUsageMetric)

	report, err := os.Open(viper.GetString("report.infile"))
	if err != nil {
		logrus.Error(err)
	}

	scanner := bufio.NewScanner(report)
	scanner.Split(bufio.ScanLines)

	// Keep our data in memory in a map.
	addrs := map[string]float64{
		"dma":      float64(0),
		"heap":     float64(0),
		"relocate": float64(0),
	}
	sizes := map[string]float64{
		"dma":      float64(0),
		"heap":     float64(0),
		"relocate": float64(0),
	}

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

			// Update our in-memory map.
			label := parts[1]
			addrs[label] = addr
			sizes[label] = size
		}
	}
	report.Close()

	// Update our binary section address metrics if we're configured to do so.
	if viper.GetBool("metrics.sections.addrs") {
		for label, addr := range addrs {
			binAddrMetric.With(prometheus.Labels{
				"label": label,
				"unit":  "byte",
			}).Set(addr)
		}
	}

	// Update our binary section size metrics if we're configured to do so.
	if viper.GetBool("metrics.sections.sizes") {
		for label, size := range sizes {
			binSizeMetric.With(prometheus.Labels{
				"label": label,
				"unit":  "byte",
			}).Set(size)
		}
	}

	// Update our derived metrics if we're instructed to do so.
	if viper.GetBool("metrics.derived.ram") {
		logrus.Info("calculating derived RAM metric")

		usage := (addrs["heap"] - addrs["relocate"] + sizes["heap"])
		total := addrs["dma"] - addrs["relocate"]

		logrus.WithFields(logrus.Fields{
			"usage": usage,
			"total": total,
		}).Debug("calculated metric")

		// Calculate the RAM usage of the binary.
		logrus.Info("setting derived RAM metric")
		ramUsageMetric.Set(usage / total)
	}

	// Write out the Prometheus metrics to a file.
	err = prometheus.WriteToTextfile(viper.GetString("metrics.outfile"), registry)
	if err != nil {
		logrus.Error(err)
	}
}
