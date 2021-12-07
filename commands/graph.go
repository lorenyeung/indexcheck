package commands

import (
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jfrog/jfrog-cli-core/v2/plugins/components"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"

	helpers "github.com/lorenyeung/indexcheck/utils"

	"github.com/jfrog/jfrog-cli-core/v2/utils/config"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/jfrog/jfrog-client-go/utils/log"
)

type Alphabetic []string

func (list Alphabetic) Len() int { return len(list) }

func GetGraphCommand() components.Command {
	return components.Command{
		Name:        "graph",
		Description: "Graph open metrics API.",
		Aliases:     []string{"g"},
		Arguments:   getGraphArguments(),
		Flags:       getGraphFlags(),
		EnvVars:     getGraphEnvVar(),
		Action: func(c *components.Context) error {
			return GraphCmd(c)
		},
	}
}

func getGraphArguments() []components.Argument {
	return []components.Argument{}
}

func getGraphFlags() []components.Flag {
	return []components.Flag{
		components.StringFlag{
			Name:         "interval",
			Description:  "Polling interval in seconds",
			DefaultValue: "1",
		},
	}
}

func getGraphEnvVar() []components.EnvVar {
	return []components.EnvVar{}
}

type GraphConfiguration struct {
	interval int
}

func GraphCmd(c *components.Context) error {

	interval, err := strconv.Atoi(c.GetStringFlagValue("interval"))

	config, err := helpers.GetConfig()
	if err != nil {
		return err
	}

	if err := ui.Init(); err != nil {
		fmt.Printf("failed to initialize termui: %v", err)
		return err
	}
	defer ui.Close()

	//Meta statistics
	o := widgets.NewParagraph()
	o.Title = "Meta statistics"
	o.Text = "Current time: " + time.Now().Format("2006.01.02 15:04:05")
	o.SetRect(0, 0, 77, 6)

	p := widgets.NewParagraph()
	p.Title = "Total Remote Conns"
	p.Text = "Initializing"
	p.SetRect(0, 6, 25, 11)

	q := widgets.NewParagraph()
	q.Title = "CPU Usage (%)"
	q.Text = "Initializing"
	q.SetRect(26, 6, 51, 11)

	r := widgets.NewParagraph()
	r.Title = "Number of Metrics"
	r.Text = "Initializing"
	r.SetRect(52, 6, 77, 11)

	//GC statistics
	o2 := widgets.NewParagraph()
	o2.Title = "Garbage Collection statistics"
	o2.Text = "Initializing"
	o2.SetRect(0, 45, 77, 51)

	g2 := widgets.NewGauge()
	g2.Title = "Current Used Storage"
	g2.SetRect(0, 11, 36, 14)
	g2.Percent = 0
	g2.BarColor = ui.ColorGreen
	g2.LabelStyle = ui.NewStyle(ui.ColorBlue)
	g2.BorderStyle.Fg = ui.ColorWhite

	g3 := widgets.NewGauge()
	g3.Title = "Current Used Go Heap"
	g3.SetRect(0, 14, 36, 17)
	g3.Percent = 0
	g3.BarColor = ui.ColorGreen
	g3.LabelStyle = ui.NewStyle(ui.ColorBlue)
	g3.BorderStyle.Fg = ui.ColorWhite

	//DB connections
	g4 := widgets.NewGauge()
	g4.Title = "Active DB connections"
	g4.SetRect(0, 17, 36, 20)
	g4.Percent = 0
	g4.BarColor = ui.ColorGreen
	g4.LabelStyle = ui.NewStyle(ui.ColorBlue)
	g4.BorderStyle.Fg = ui.ColorWhite

	//DB connection plot chart
	p1 := widgets.NewPlot()
	p1.Title = "DB Connection Chart"
	//p1.Marker = widgets.MarkerDot

	var dbActivePlotData = make([]float64, 60)
	var dbMaxPlotData = make([]float64, 60)
	var dbIdlePlotData = make([]float64, 60)
	var dbMinIdlePlotData = make([]float64, 60)
	var dbConnPlotData = [][]float64{dbActivePlotData, dbMaxPlotData, dbIdlePlotData, dbMinIdlePlotData}

	for i := 0; i < 60; i++ {
		dbActivePlotData[i] = 0
		dbMaxPlotData[i] = 0
		dbIdlePlotData[i] = 0
		dbMinIdlePlotData[i] = 0
	}
	p1.Data = dbConnPlotData
	p1.SetRect(78, 0, 146, 28)
	p1.DotMarkerRune = '.'
	p1.AxesColor = ui.ColorWhite
	p1.LineColors[0] = ui.ColorBlack
	p1.LineColors[1] = ui.ColorGreen
	p1.LineColors[2] = ui.ColorBlue
	p1.LineColors[3] = ui.ColorRed
	p1.DrawDirection = widgets.DrawLeft
	p1.HorizontalScale = 1

	//Remote connection plot chart
	var rcPlotData = make(map[string][]float64)
	p2 := widgets.NewPlot()
	p2.Title = "Remote Connections Chart"
	//p2.Marker = widgets.MarkerDot

	var leasedPlotData = make([]float64, 60)
	var connPlotData = [][]float64{leasedPlotData}

	for i := 0; i < 60; i++ {
		leasedPlotData[i] = 0
	}
	p2.Data = connPlotData
	p2.SetRect(78, 28, 146, 56)
	p2.DotMarkerRune = '+'
	p2.AxesColor = ui.ColorWhite
	p2.DrawDirection = widgets.DrawLeft
	p2.HorizontalScale = 1

	//bar chart
	barchartData := []float64{1, 1, 1, 1}

	bc := widgets.NewBarChart()
	bc.Title = "DB Connections"
	bc.BarWidth = 5
	bc.Data = barchartData
	bc.SetRect(0, 20, 36, 34)
	bc.Labels = []string{"Active", "Max", "Idle", "MinIdle"}
	bc.BarColors[0] = ui.ColorBlack
	bc.BarColors[1] = ui.ColorGreen
	bc.BarColors[2] = ui.ColorBlue
	bc.BarColors[3] = ui.ColorRed
	bc.LabelStyles[0] = ui.NewStyle(ui.ColorWhite)
	bc.LabelStyles[1] = ui.NewStyle(ui.ColorWhite)
	bc.LabelStyles[2] = ui.NewStyle(ui.ColorWhite)
	bc.LabelStyles[3] = ui.NewStyle(ui.ColorWhite)
	bc.NumStyles[0] = ui.NewStyle(ui.ColorBlack)

	//remote conn barchart
	bc2 := widgets.NewBarChart()
	bc2.Title = "Remote Connections Barchart"
	bc2.BarWidth = 3
	bc2.Data = []float64{}
	bc2.SetRect(0, 34, 77, 45)
	bc2.Labels = []string{}
	bc2.BarColors[0] = ui.ColorGreen
	bc2.NumStyles[0] = ui.NewStyle(ui.ColorBlack)

	//remote connections list
	l := widgets.NewList()
	l.Title = "Remote Connections List"
	l.Rows = []string{}
	l.TextStyle = ui.NewStyle(ui.ColorYellow)
	l.WrapText = false
	l.SetRect(37, 11, 77, 34)

	ui.Render(bc, bc2, g2, g3, g4, l, o, o2, p, p1, p2, q, r)

	uiEvents := ui.PollEvents()
	ticker := time.NewTicker(time.Second * time.Duration(interval)).C
	offSetCounter := 0
	tickerCount := 1

	go func() {
		for {
			if time.Now().Second() == 0 {
				for i := range rcPlotData {
					rcPlotData[i] = make([]float64, 60)
				}
				for i := range dbConnPlotData {
					dbConnPlotData[i] = make([]float64, 60)
				}
				log.Info("reset graphs")
			}
			time.Sleep(time.Second * time.Duration(1))
		}
	}()

	for {
		select {
		case e := <-uiEvents:
			switch e.ID { // event string/identifier
			case "q", "<C-c>": // press 'q' or 'C-c' to quit
				return nil
			}

		// use Go's built-in tickers for updating and drawing data
		case <-ticker:
			var err error
			offSetCounter, rcPlotData, err = drawFunction(config, bc, bc2, barchartData, g2, g3, g4, l, o, o2, p, p1, dbConnPlotData, p2, rcPlotData, q, r, offSetCounter, tickerCount, interval)
			if err != nil {
				return errorutils.CheckError(err)
			}
			tickerCount++

		}
	}
}

func drawFunction(config *config.ServerDetails, bc *widgets.BarChart, bc2 *widgets.BarChart, bcData []float64, g2 *widgets.Gauge, g3 *widgets.Gauge, g4 *widgets.Gauge, l *widgets.List, o *widgets.Paragraph, o2 *widgets.Paragraph, p *widgets.Paragraph, p1 *widgets.Plot, plotData [][]float64, p2 *widgets.Plot, rcPlotData map[string][]float64, q *widgets.Paragraph, r *widgets.Paragraph, offSetCounter int, ticker int, interval int) (int, map[string][]float64, error) {
	responseTime := time.Now()
	data, lastUpdate, offset, err := helpers.GetMetricsData(config, offSetCounter, false, interval)
	if err != nil {
		return 0, nil, err

	}
	responseTimeCompute := time.Now()

	var freeSpace, totalSpace, heapFreeSpace, heapMaxSpace, heapTotalSpace *big.Float = big.NewFloat(1), big.NewFloat(100), big.NewFloat(100), big.NewFloat(100), big.NewFloat(100)
	var heapProc string
	var dbConnIdle, dbConnMinIdle, dbConnActive, dbConnMax, gcBinariesTotal, gcDurationSecs, lastGcRun string
	var gcSizeCleanedBytes, gcCurrentSizeBytes *big.Float = big.NewFloat(0), big.NewFloat(0)

	//maybe we can turn this into a hashtable for faster lookup
	//remote connection specifc
	var remoteConnMap []helpers.Data

	var remoteConnMap2 = make(map[string]helpers.Data)
	var remoteConnMapIds = []string{}

	for i := range data {

		var err error
		switch dataArg := data[i].Name; dataArg {
		case "sys_cpu_ratio":
			q.Text = data[i].Metric[0].Value
		case "go_memstats_heap_reserved_bytes":
			heapMaxSpace, _, err = big.ParseFloat(data[i].Metric[0].Value, 10, 0, big.ToNearestEven)
			if err != nil {
				//prevent cannot divide by zero error for all heap/space floats to prevent remote connection crashes
				heapMaxSpace = big.NewFloat(1)
				log.Error(err.Error() + " at " + string(helpers.Trace().Fn) + " on line " + string(strconv.Itoa(helpers.Trace().Line)))
			}
		case "go_memstats_heap_in_use_bytes":
			heapFreeSpace, _, err = big.ParseFloat(data[i].Metric[0].Value, 10, 0, big.ToNearestEven)
			if err != nil {
				heapFreeSpace = big.NewFloat(1)
				log.Error(err.Error() + " at " + string(helpers.Trace().Fn) + " on line " + string(strconv.Itoa(helpers.Trace().Line)))
			}
		case "go_memstats_heap_allocated_bytes":
			heapTotalSpace, _, err = big.ParseFloat(data[i].Metric[0].Value, 10, 0, big.ToNearestEven)
			if err != nil {
				heapTotalSpace = big.NewFloat(1)
				log.Error(err.Error() + " at " + string(helpers.Trace().Fn) + " on line " + string(strconv.Itoa(helpers.Trace().Line)))
			}
		case "jfrt_runtime_heap_processors_total":
			heapProc = data[i].Metric[0].Value
		case "app_disk_free_bytes":
			freeSpace, _, err = big.ParseFloat(data[i].Metric[0].Value, 10, 0, big.ToNearestEven)
			if err != nil {
				freeSpace = big.NewFloat(1)
				log.Error(err.Error() + " at " + string(helpers.Trace().Fn) + " on line " + string(strconv.Itoa(helpers.Trace().Line)))
			}
		case "app_disk_total_bytes":
			totalSpace, _, err = big.ParseFloat(data[i].Metric[0].Value, 10, 0, big.ToNearestEven)
			if err != nil {
				totalSpace = big.NewFloat(1)
				log.Error(err.Error() + " at " + string(helpers.Trace().Fn) + " on line " + string(strconv.Itoa(helpers.Trace().Line)))
			}
		case "db_connection_pool_in_use_total":
			dbConnActive = data[i].Metric[0].Value
		case "db_connection_pool_max_open_total":
			dbConnMax = data[i].Metric[0].Value
		case "jfrt_db_connections_min_idle_total":
			dbConnMinIdle = data[i].Metric[0].Value
		case "db_connection_pool_idle_total":
			dbConnIdle = data[i].Metric[0].Value
		case "jfrt_artifacts_gc_duration_seconds":
			gcDurationSecs = data[i].Metric[0].Value

			gcStart, err := strconv.ParseInt(data[i].Metric[0].Labels.Start, 10, 64)
			if err != nil {
				log.Error(err.Error() + " at " + string(helpers.Trace().Fn) + " on line " + string(strconv.Itoa(helpers.Trace().Line)))
			}
			startTimeEpoch := time.Unix(gcStart/1000, 0)
			gcEnd, err := strconv.ParseInt(data[i].Metric[0].Labels.End, 10, 64)
			if err != nil {
				log.Error(err.Error() + " at " + string(helpers.Trace().Fn) + " on line " + string(strconv.Itoa(helpers.Trace().Line)))
			}
			endTimeEpoch := time.Unix(gcEnd/1000, 0)

			lastGcRun = "Last GC Run:" + startTimeEpoch.Format("2006.01.02 15:04:05") + " -> " + endTimeEpoch.Format("2006.01.02 15:04:05") + "\nType: " + data[i].Metric[0].Labels.Type + " Status: " + data[i].Metric[0].Labels.Status
		case "jfrt_artifacts_gc_size_cleaned_bytes":
			gcSizeCleanedBytes, _, err = big.ParseFloat(data[i].Metric[0].Value, 10, 0, big.ToNearestEven)
			if err != nil {
				log.Error(err.Error() + " at " + string(helpers.Trace().Fn) + " on line " + string(strconv.Itoa(helpers.Trace().Line)))
				gcSizeCleanedBytes = big.NewFloat(1)
			}
		case "jfrt_artifacts_gc_binaries_total":
			gcBinariesTotal = data[i].Metric[0].Value
		case "jfrt_artifacts_gc_current_size_bytes":
			gcCurrentSizeBytes, _, err = big.ParseFloat(data[i].Metric[0].Value, 10, 0, big.ToNearestEven)
			if err != nil {
				gcCurrentSizeBytes = big.NewFloat(1)
				log.Error(err.Error() + " at " + string(helpers.Trace().Fn) + " on line " + string(strconv.Itoa(helpers.Trace().Line)))
			}

		default:
			// do nothing
		}
		//gc
		var gcSizeCleanedBytesBigInt, gcCurrentSizeBytesBigInt = new(big.Int), new(big.Int)
		gcSizeCleanedBytesBigInt, _ = gcSizeCleanedBytes.Int(gcSizeCleanedBytesBigInt)
		gcCurrentSizeBytesBigInt, _ = gcCurrentSizeBytes.Int(gcCurrentSizeBytesBigInt)
		gcSizeCleanedBytesStr := gcSizeCleanedBytesBigInt.String()
		gcCurrentSizeBytesStr := gcCurrentSizeBytesBigInt.String()

		gcSizeConv, err := helpers.StringToInt64(gcSizeCleanedBytesStr)
		if err != nil {
			log.Warn(err)
			gcSizeConv = 0
		}
		gcCurrentConv, err := helpers.StringToInt64(gcCurrentSizeBytesStr)
		if err != nil {
			log.Warn(err)
			gcCurrentConv = 0
		}
		o2.Text = lastGcRun + "\nNumber of binaries cleaned: " + gcBinariesTotal + " Duration: " + gcDurationSecs + "s\nCleaned up: " + helpers.ByteCountDecimal(gcSizeConv) + " Current size: " + helpers.ByteCountDecimal(gcCurrentConv)

		//repo specific connection check
		if strings.Contains(data[i].Name, "jfrt_http_connections") {
			remoteConnMap2[data[i].Name] = data[i]
			remoteConnMapIds = append(remoteConnMapIds, data[i].Name)
		}
	}

	sort.Sort(Alphabetic(remoteConnMapIds))
	for i := range remoteConnMapIds {
		somedata := remoteConnMap2[remoteConnMapIds[i]]
		remoteConnMap = append(remoteConnMap, somedata)
	}

	//heapMax is xmx confirmed. no idea what the other two are
	//2.07e8, 4.29e09, 1.5e09
	//fmt.Println(heapFreeSpace, heapMaxSpace, heapTotalSpace)

	//compute DB active guage
	dbConnActiveInt, err := strconv.Atoi(dbConnActive)
	if err != nil {
		dbConnActiveInt = 0
		//return 0, errors.New(err.Error() + " at " + string(helpers.Trace().Fn) + " on line " + string(strconv.Itoa(helpers.Trace().Line)))
	}
	dbConnMaxInt, err := strconv.Atoi(dbConnMax)
	if err != nil {
		//prevent integer divide by zero error
		dbConnMaxInt = 1
		//return 0, errors.New(err.Error() + " at " + string(helpers.Trace().Fn) + " on line " + string(strconv.Itoa(helpers.Trace().Line)))
	}
	dbConnIdleInt, err := strconv.Atoi(dbConnIdle)
	if err != nil {
		dbConnIdleInt = 0
		//return 0, errors.New(err.Error() + " at " + string(helpers.Trace().Fn) + " on line " + string(strconv.Itoa(helpers.Trace().Line)))
	}
	dbConnMinIdleInt, err := strconv.Atoi(dbConnMinIdle)
	if err != nil {
		dbConnMinIdleInt = 0
		//return 0, errors.New(err.Error() + " at " + string(helpers.Trace().Fn) + " on line " + string(strconv.Itoa(helpers.Trace().Line)))
	}
	pctDbConnActive := dbConnActiveInt / dbConnMaxInt * 100
	g4.Percent = pctDbConnActive

	//compute free space gauge
	pctFreeSpace := new(big.Float).Mul(big.NewFloat(100), new(big.Float).Quo(freeSpace, totalSpace))
	pctFreeSpaceStr := pctFreeSpace.String()
	pctFreeSplit := strings.Split(pctFreeSpaceStr, ".")
	pctFreeInt, _ := strconv.Atoi(pctFreeSplit[0])
	g2.Percent = 100 - pctFreeInt

	//compute free heap gauge
	pctFreeHeapSpace := new(big.Float).Mul(big.NewFloat(100), new(big.Float).Quo(heapFreeSpace, heapMaxSpace))
	pctFreeHeapSpaceStr := pctFreeHeapSpace.String()
	pctFreeHeapSplit := strings.Split(pctFreeHeapSpaceStr, ".")
	pctFreeHeapInt, _ := strconv.Atoi(pctFreeHeapSplit[0])
	g3.Percent = pctFreeHeapInt

	bc.Data = []float64{float64(dbConnActiveInt), float64(dbConnMaxInt), float64(dbConnIdleInt), float64(dbConnMinIdleInt)}

	//list data
	connMapsize := len(remoteConnMap)
	var listRow = make([]string, connMapsize)
	var bc2labels []string
	var totalLease, totalMax, totalAvailable, totalPending int
	mapCount := 0
	var remoteBcData []float64
	timeSecond := responseTime.Second()

	log.Debug("size of map before processing", len(rcPlotData))
	if connMapsize > 0 {
		log.Debug("remote connection print out:", remoteConnMap)
		for i := range remoteConnMap {
			id := strings.Split(remoteConnMap[i].Name, "jfrt_http_connections")
			uniqId := id[0] + string(remoteConnMap[i].Help[0])
			bc2labels = append(bc2labels, uniqId)
			listRow[mapCount] = remoteConnMap[i].Metric[0].Value + " " + remoteConnMap[i].Metric[0].Labels.Pool + " " + strings.ReplaceAll(remoteConnMap[i].Help, " Connections", "") + " " + uniqId
			mapCount++

			totalValue, err := strconv.Atoi(remoteConnMap[i].Metric[0].Value)
			if err != nil {
				totalValue = 0 //safety in case it can't convert
				log.Warn("Failed to convert number ", remoteConnMap[i].Metric[0].Value, " at ", helpers.Trace().Fn, " line ", helpers.Trace().Line)
			}

			//init the float for the map for plot
			if strings.Contains(remoteConnMap[i].Name, "jfrt_http_connections_leased_total") {
				if rcPlotData[uniqId] == nil {
					var rcPlotDataRow = make([]float64, 60)
					for i := 0; i < 60; i++ {
						if i == timeSecond {
							rcPlotDataRow[i] = float64(totalValue)
						} else {
							rcPlotDataRow[i] = 0
						}
					}
					rcPlotData[uniqId] = rcPlotDataRow
				} else {
					// float row already exists, need to append/update
					rcPlotDataRow := rcPlotData[uniqId]
					rcPlotDataRow[timeSecond] = float64(totalValue)
					for i := 0; i < interval; i++ {
						if timeSecond+i < 60 {
							rcPlotDataRow[timeSecond+i] = float64(totalValue)
						} else {
							for i := range rcPlotDataRow {
								rcPlotDataRow[i] = 0
							}
							rcPlotData[uniqId] = rcPlotDataRow
							log.Info("reset graph")
						}
					}
					rcPlotData[uniqId] = rcPlotDataRow
				}
			}

			//append bar chart
			remoteBcData = append(remoteBcData, float64(totalValue))

			switch typeTotal := remoteConnMap[i].Help; typeTotal {
			case "Leased Connections":
				totalLease = totalLease + totalValue

			case "Pending Connections":
				totalPending = totalPending + totalValue

			case "Max Connections":
				totalMax = totalMax + totalValue

			case "Available Connections":
				totalAvailable = totalAvailable + totalValue
			}
		}
	}

	bc2.Labels = bc2labels
	bc2.Data = remoteBcData
	l.Rows = listRow

	//remote connection data
	var rcPlotFinalData = make([][]float64, len(rcPlotData))
	var rcCount int = 0
	for i := range rcPlotData {
		log.Debug("rcPlot data:", rcPlotData[i])
		log.Debug("i:", i, " data size:", len(rcPlotData[i]))
		if len(rcPlotData[i]) == 0 {
			//skip
			log.Debug("Map is empty at this location:", i)
		} else {

			rcPlotFinalData[rcCount] = rcPlotData[i]

		}
		rcCount++
	}

	//Db connection plot data
	for i := 0; i < 60; i++ {
		if i == int(timeSecond) {
			//order: active, max, idle, minIdle
			plotData[0][i] = float64(dbConnActiveInt)
			plotData[1][i] = float64(0) //whats the point of plotting max
			plotData[2][i] = float64(dbConnIdleInt)
			plotData[3][i] = float64(dbConnMinIdleInt)
			log.Debug("current time:", i)
		}

		for i := 0; i < interval; i++ {
			if timeSecond+i < 60 {
				//order: active, max, idle, minIdle
				plotData[0][timeSecond+i] = float64(dbConnActiveInt)
				plotData[1][timeSecond+i] = float64(0) //whats the point of plotting max
				plotData[2][timeSecond+i] = float64(dbConnIdleInt)
				plotData[3][timeSecond+i] = float64(dbConnMinIdleInt)
				log.Debug("current time:", i)
			}
		}
	}
	p1.Data = plotData
	p2.DataLabels = []string{"hello"}

	log.Debug("size of plot rc:", len(rcPlotFinalData))
	p2.Data = rcPlotFinalData

	//total
	p.Text = "Leased:" + strconv.Itoa(totalLease) + " Max:" + strconv.Itoa(totalMax) + " Available:" + strconv.Itoa(totalAvailable) + " Pending:" + strconv.Itoa(totalPending)
	//metrics data
	r.Text = "Count: " + strconv.Itoa(len(data)) + "\nHeap Proc: " + heapProc + "\nHeap Total: " + heapTotalSpace.String()

	o.Text = "Current time: " + time.Now().Format("2006.01.02 15:04:05") + "\nLast updated: " + lastUpdate + " (" + strconv.Itoa(offset) + " seconds) Data Compute time:" + time.Now().Sub(responseTimeCompute).String() + "\nResponse time: " + time.Now().Sub(responseTime).String() + " Polling interval: every " + strconv.Itoa(interval) + " seconds\nServer url: " + config.ServerId

	ui.Render(bc, bc2, g2, g3, g4, l, o, o2, p, p1, p2, q, r)
	return offset, rcPlotData, nil
}

func Extend(slice []string, element string) []string {
	n := len(slice)
	slice = slice[0 : n+1]
	slice[n] = element
	return slice
}

func (list Alphabetic) Swap(i, j int) { list[i], list[j] = list[j], list[i] }

func (list Alphabetic) Less(i, j int) bool {
	var si string = list[i]
	var sj string = list[j]
	var si_lower = strings.ToLower(si)
	var sj_lower = strings.ToLower(sj)
	if si_lower == sj_lower {
		return si < sj
	}
	return si_lower < sj_lower
}
