package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/components"
	"github.com/go-echarts/go-echarts/v2/opts"
)

type ContentMatrix struct {
	Tag                string
	Size               int64
	AvgSpeed           float32
	DownloadStartedAt  int64
	DownloadFinishedAt int64
	ProvidedBy         []string
}

type ConnectionInfo struct {
	BLEDiscoveredAt int64
	WifiConnectedAt int64
	RSSI            int
	Speed           int
	Frequency       int
	IPFSConnectedAt int64
	DisconnectedAt  int64
}

type DiscoveredNodeMatrix struct {
	ConnectionAlive                  bool
	ConnectionSuccessCount           int
	ConnectionFailureCount           int
	LastSuccessfulConnectionDuration int64
	BLEDiscoveredAt                  int64
	WifiConnectedAt                  int64
	RSSI                             int
	Speed                            int
	Frequency                        int
	IPFSConnectedAt                  int64
	DiscoveryDelays                  []int64 // from BLE Discovery to ipfs Connection
	ConnectionHistory                []ConnectionInfo
}

type matrix struct {
	ContentMatrix map[string]ContentMatrix
	NodeMatrix    map[string]DiscoveredNodeMatrix
	TotalUptime   int64
}
type BatteryMeasurements struct {
	BatteryMeasurement []Measurement `json:"BatteryMeasurement"`
}
type Measurement struct {
	DataTransfer       string `json:"DataTransfer"`
	TransferInterval   string `json:"TransferInterval"`
	BatteryConsumption string `json:"Battery Consumption"`
}

var matrixFiles = []string{"zero_host_downloader", "zero_client_uploader", "five_host_downloader", "five_client_uploader"}
var batteryMeasurementFiles = []string{"battery_measurements"}

func main() {
	for _, v := range matrixFiles {
		err := renderMatrixPage(v)
		if err != nil {
			log.Fatal("Page render failed ", err.Error())
		}
	}

	for _, v := range batteryMeasurementFiles {
		err := renderBatteryMeasurementPage(v)
		if err != nil {
			log.Fatal("Page render failed ", err.Error())
		}
	}

	fs := http.FileServer(http.Dir("html"))
	log.Println("running server at http://localhost:8089")
	http.ListenAndServe("localhost:8089", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s\n", r.RemoteAddr, r.Method, r.URL)
		fs.ServeHTTP(w, r)
	}))
}

func renderBatteryMeasurementPage(pageName string) error {
	file, err := ioutil.ReadFile(fmt.Sprintf("logs/%s.log", pageName))
	if err != nil {
		log.Fatal("matrix file missing ", err.Error())
	}
	data := &BatteryMeasurements{}
	err = json.Unmarshal(file, data)
	if err != nil {
		log.Fatal("matrix file missing ", err.Error())
	}
	page := components.NewPage()
	page.AddCharts(
		transferIntervalToBatteryPercentage(data),
	)
	page.PageTitle = "Datahop Battery Measurement Charts"
	f, err := os.Create(fmt.Sprintf("html/%s.html", pageName))
	if err != nil {
		log.Fatal("unable to create file ", err.Error())
	}
	return page.Render(io.MultiWriter(f))
}

func transferIntervalToBatteryPercentage(data *BatteryMeasurements) *charts.Bar {
	// create a new bar instance
	bar := charts.NewBar()
	// set some global options like Title/Legend/ToolTip or anything else
	bar.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{Title: "Battery Consumption after 3 hours of transfer"}),
		charts.WithTooltipOpts(opts.Tooltip{Show: true}),
		charts.WithLegendOpts(opts.Legend{Show: true, Left: "80%"}),
	)
	tenMbItems := make([]opts.BarData, 0)
	hundredMbItems := make([]opts.BarData, 0)
	for _, v := range data.BatteryMeasurement {
		if v.DataTransfer == "10" {
			tenMbItems = append(tenMbItems, opts.BarData{Value: v.BatteryConsumption})
			continue
		}
		if v.DataTransfer == "100" {
			hundredMbItems = append(hundredMbItems, opts.BarData{Value: v.BatteryConsumption})
		}
	}
	// Put data into instance
	bar.SetXAxis([]string{"40s", "120s"}).
		AddSeries("10Mb", tenMbItems).
		AddSeries("100Mb", hundredMbItems).
		SetSeriesOptions(
			charts.WithLabelOpts(opts.Label{
				Show:     true,
				Position: "insideTop",
			}),
		)

	return bar
}

func renderMatrixPage(pageName string) error {
	file, err := ioutil.ReadFile(fmt.Sprintf("logs/%s.log", pageName))
	if err != nil {
		log.Fatal("matrix file missing ", err.Error())
	}
	data := &matrix{}
	err = json.Unmarshal(file, data)
	if err != nil {
		log.Fatal("matrix file missing ", err.Error())
	}
	page := components.NewPage()
	page.AddCharts(
		bleToWifi(data),
		bleToIpfs(data),
		rssiSpeed(data),
		downloadSpeed(data),
	)
	page.PageTitle = "Datahop Matrix Charts"
	f, err := os.Create(fmt.Sprintf("html/%s.html", pageName))
	if err != nil {
		log.Fatal("unable to create file ", err.Error())
	}
	return page.Render(io.MultiWriter(f))
}

func bleToWifi(data *matrix) *charts.Line {
	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithTitleOpts(
			opts.Title{
				Title: "Seconds from BLE discovery to Successful Wifi connection",
			},
		),
		charts.WithXAxisOpts(
			opts.XAxis{
				Name: "Count",
			},
		),
		charts.WithYAxisOpts(
			opts.YAxis{
				Name: "Seconds",
			},
		),
		charts.WithTooltipOpts(opts.Tooltip{Show: true}),
		charts.WithLegendOpts(opts.Legend{Show: true, Left: "80%"}),
	)
	xAxis := []int{}
	yAxis := make([]opts.LineData, 0)
	for _, v := range data.NodeMatrix {
		for _, k := range v.ConnectionHistory {
			if k.WifiConnectedAt != 0 {
				xAxis = append(xAxis, len(xAxis))
				yAxis = append(yAxis, opts.LineData{Value: k.WifiConnectedAt - k.BLEDiscoveredAt})
			}
		}

	}

	line.SetXAxis(xAxis).AddSeries("BLE to Wifi", yAxis).
		SetSeriesOptions(
			charts.WithAreaStyleOpts(opts.AreaStyle{
				Opacity: 0.2,
			}),
			charts.WithLineChartOpts(opts.LineChart{
				Smooth: true,
			}),
		)
	return line
}

func bleToIpfs(data *matrix) *charts.Line {
	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithTitleOpts(
			opts.Title{
				Title: "Seconds from BLE discovery to Successful IPFS connection",
			},
		),
		charts.WithXAxisOpts(
			opts.XAxis{
				Name: "Count",
			},
		),
		charts.WithYAxisOpts(
			opts.YAxis{
				Name: "Seconds",
			},
		),
		charts.WithTooltipOpts(opts.Tooltip{Show: true}),
		charts.WithLegendOpts(opts.Legend{Show: true, Left: "80%"}),
	)
	xAxis := []int{}
	yAxis := make([]opts.LineData, 0)
	for _, v := range data.NodeMatrix {
		for _, k := range v.DiscoveryDelays {
			xAxis = append(xAxis, len(xAxis))
			yAxis = append(yAxis, opts.LineData{Value: k})
		}
	}

	line.SetXAxis(xAxis).AddSeries("BLE to IPFS", yAxis).
		SetSeriesOptions(
			charts.WithAreaStyleOpts(opts.AreaStyle{
				Opacity: 0.2,
			}),
			charts.WithLineChartOpts(opts.LineChart{
				Smooth: true,
			}),
		)
	return line
}

var (
	parallelAxisList = []opts.ParallelAxis{
		{Dim: 0, Name: "RSSI"},
		{Dim: 1, Name: "Speed"},
	}
)

func rssiSpeed(data *matrix) *charts.Parallel {
	parallel := charts.NewParallel()
	parallel.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: "RSSI Speed",
		}),
		charts.WithParallelAxisList(parallelAxisList),
		charts.WithTooltipOpts(opts.Tooltip{Show: true}),
		charts.WithLegendOpts(opts.Legend{Show: true, Left: "80%"}),
	)
	items := make([]opts.ParallelData, 0)
	for _, v := range data.NodeMatrix {
		for _, k := range v.ConnectionHistory {
			items = append(items, opts.ParallelData{Value: []interface{}{k.RSSI, k.Speed}})
		}
	}
	parallel.AddSeries("RSSI Speed", items)
	return parallel
}

func downloadSpeed(data *matrix) *charts.Line {
	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithTitleOpts(
			opts.Title{
				Title: "Download Speed",
			},
		),
		charts.WithXAxisOpts(
			opts.XAxis{
				Name: "Count",
			},
		),
		charts.WithYAxisOpts(
			opts.YAxis{
				Name: "MBps",
			},
		),
		charts.WithTooltipOpts(opts.Tooltip{Show: true}),
		charts.WithLegendOpts(opts.Legend{Show: true, Left: "80%"}),
	)
	xAxis := []int{}
	yAxis := make([]opts.LineData, 0)
	for _, v := range data.ContentMatrix {
		xAxis = append(xAxis, len(xAxis))
		s := fmt.Sprintf("%.1f", v.AvgSpeed)
		f, _ := strconv.ParseFloat(s, 64)
		yAxis = append(yAxis, opts.LineData{Value: f})
	}

	line.SetXAxis(xAxis).AddSeries("Download Speed", yAxis).
		SetSeriesOptions(
			charts.WithAreaStyleOpts(opts.AreaStyle{
				Opacity: 0.2,
			}),
		)
	return line
}
