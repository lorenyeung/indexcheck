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
		components.BoolFlag{
			Name:         "retry",
			Description:  "Show retry queues in chart",
			DefaultValue: false,
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

	q := widgets.NewParagraph()
	q.Title = "CPU Usage (%)"
	q.Text = "Initializing"
	q.SetRect(0, 6, 36, 11)

	r := widgets.NewParagraph()
	r.Title = "Number of Metrics"
	r.Text = "Initializing"
	r.SetRect(37, 6, 77, 11)

	//GC statistics
	o2 := widgets.NewParagraph()
	o2.Title = "DB Sync statistics"
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

	var sysLoadOneData = make([]float64, 60)
	var sysLoadFiveData = make([]float64, 60)
	var sysLoadFifteenData = make([]float64, 60)
	var sysLoadPlotData = [][]float64{sysLoadOneData, sysLoadFiveData, sysLoadFifteenData}

	for i := 0; i < 60; i++ {
		dbActivePlotData[i] = 0
		dbMaxPlotData[i] = 0
		dbIdlePlotData[i] = 0
		dbMinIdlePlotData[i] = 0
	}

	for i := 0; i < 60; i++ {
		sysLoadOneData[i] = 0
		sysLoadFiveData[i] = 0
		sysLoadFifteenData[i] = 0
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

	//Sysload plot chart

	p3 := widgets.NewPlot()
	p3.Title = "Sys Load Graph"

	p3.Data = sysLoadPlotData
	p3.SetRect(78, 28, 146, 56)
	p3.DotMarkerRune = '.'
	p3.AxesColor = ui.ColorWhite
	p3.LineColors[0] = ui.ColorBlack
	p3.LineColors[1] = ui.ColorGreen
	p3.LineColors[2] = ui.ColorBlue
	p3.LineColors[3] = ui.ColorRed
	p3.DrawDirection = widgets.DrawLeft
	p3.HorizontalScale = 1

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

	//remote connections list
	l := widgets.NewList()
	l.Title = "Queue List"
	l.Rows = []string{}
	l.TextStyle = ui.NewStyle(ui.ColorYellow)
	l.WrapText = false
	l.SetRect(37, 11, 77, 45)

	ui.Render(bc, g2, g3, g4, l, o, o2, p1, p3, q, r)

	uiEvents := ui.PollEvents()
	ticker := time.NewTicker(time.Second * time.Duration(interval)).C
	offSetCounter := 0
	tickerCount := 1

	go func() {
		for {
			if time.Now().Second() == 0 {
				for i := range sysLoadPlotData {
					sysLoadPlotData[i] = make([]float64, 60)
				}
				for i := range dbConnPlotData {
					dbConnPlotData[i] = make([]float64, 60)
				}
				log.Debug("reset graphs")
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
			offSetCounter, rcPlotData, err = drawFunction(config, bc, barchartData, g2, g3, g4, l, o, o2, p1, dbConnPlotData, p2, rcPlotData, q, r, offSetCounter, tickerCount, interval, p3, sysLoadPlotData, c)
			if err != nil {
				return errorutils.CheckError(err)
			}
			tickerCount++
		}
	}
}

func drawFunction(config *config.ServerDetails, bc *widgets.BarChart, bcData []float64, g2 *widgets.Gauge, g3 *widgets.Gauge, g4 *widgets.Gauge, l *widgets.List, o *widgets.Paragraph, o2 *widgets.Paragraph, p1 *widgets.Plot, plotData [][]float64, p2 *widgets.Plot, rcPlotData map[string][]float64, q *widgets.Paragraph, r *widgets.Paragraph, offSetCounter int, ticker int, interval int, p3 *widgets.Plot, sysLoadplotData [][]float64, c *components.Context) (int, map[string][]float64, error) {
	responseTime := time.Now()
	data, lastUpdate, offset, err := helpers.GetMetricsData(config, offSetCounter, false, interval)
	if err != nil {
		return 0, nil, err

	}
	responseTimeCompute := time.Now()

	var freeSpace, totalSpace, heapFreeSpace, heapMaxSpace, heapTotalSpace *big.Float = big.NewFloat(1), big.NewFloat(100), big.NewFloat(100), big.NewFloat(100), big.NewFloat(100)
	var heapProc string
	var dbConnIdle, dbConnMinIdle, dbConnActive, dbConnMax, gcBinariesTotal, gcDurationSecs, lastGcRun, sysLoadOne, sysLoadFive, sysLoadFifteen string
	var gcSizeCleanedBytes, gcCurrentSizeBytes *big.Float = big.NewFloat(0), big.NewFloat(0)
	var numQueues int

	//maybe we can turn this into a hashtable for faster lookup
	//remote connection specifc
	var remoteConnMap []helpers.Data

	var remoteConnMap2 = make(map[string]helpers.Data)
	var remoteConnMapIds = []string{}
	var queueMetrics []helpers.Metrics

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

		case "sys_load_1":
			sysLoadOne = data[i].Metric[0].Value
		case "sys_load_5":
			sysLoadFive = data[i].Metric[0].Value
		case "sys_load_15":
			sysLoadFifteen = data[i].Metric[0].Value

		case "jfxr_db_sync_duration_seconds":
			gcDurationSecs = data[i].Metric[0].Value
			lastGcRun = "Last DB Run Duration:" + gcDurationSecs
		case "jfxr_db_sync_started_before_seconds":
			gcSizeCleanedBytes, _, err = big.ParseFloat(data[i].Metric[0].Value, 10, 0, big.ToNearestEven)
			if err != nil {
				log.Error(err.Error() + " at " + string(helpers.Trace().Fn) + " on line " + string(strconv.Itoa(helpers.Trace().Line)))
				gcSizeCleanedBytes = big.NewFloat(1)
			}
		case "jfxr_db_sync_ended_persist_before_seconds":
			gcBinariesTotal = data[i].Metric[0].Value
		case "jfxr_db_sync_ended_analyze_before_seconds":
			gcCurrentSizeBytes, _, err = big.ParseFloat(data[i].Metric[0].Value, 10, 0, big.ToNearestEven)
			if err != nil {
				gcCurrentSizeBytes = big.NewFloat(1)
				log.Error(err.Error() + " at " + string(helpers.Trace().Fn) + " on line " + string(strconv.Itoa(helpers.Trace().Line)))
			}
		case "queue_messages_total":
			numQueues = len(data[i].Metric)
			queueMetrics = data[i].Metric

		default:
			// do nothing
		}
		//gc
		var gcSizeCleanedBytesBigInt, gcCurrentSizeBytesBigInt = new(big.Int), new(big.Int)
		gcSizeCleanedBytesBigInt, _ = gcSizeCleanedBytes.Int(gcSizeCleanedBytesBigInt)
		gcCurrentSizeBytesBigInt, _ = gcCurrentSizeBytes.Int(gcCurrentSizeBytesBigInt)
		gcSizeCleanedBytesStr := gcSizeCleanedBytesBigInt.String()
		gcCurrentSizeBytesStr := gcCurrentSizeBytesBigInt.String()

		o2.Text = lastGcRun + "\nSeconds since DB data Persisted: " + gcBinariesTotal + " Duration: " + gcDurationSecs + "s\nSeconds since DB started: " + gcSizeCleanedBytesStr + " \nSeconds since sent to Impact Analysis: " + gcCurrentSizeBytesStr

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

	sysLoadOneInt, err := strconv.ParseFloat(sysLoadOne, 64)
	if err != nil {
		sysLoadOneInt = 0
	}

	sysLoadFiveInt, err := strconv.ParseFloat(sysLoadFive, 64)
	if err != nil {
		sysLoadFiveInt = 0
	}

	sysLoadFifteenInt, err := strconv.ParseFloat(sysLoadFifteen, 64)
	if err != nil {
		sysLoadFifteenInt = 0
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
	var listRow = make([]string, numQueues)
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
			//listRow[mapCount] = remoteConnMap[i].Metric[0].Value + " " + remoteConnMap[i].Metric[0].Labels.Pool + " " + strings.ReplaceAll(remoteConnMap[i].Help, " Connections", "") + " " + uniqId
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

	var queueChartSize int
	for i := 0; i < len(queueMetrics); i++ {
		if c.GetBoolFlagValue("retry") {
			listRow[queueChartSize] = queueMetrics[i].Labels.QueueName + " " + queueMetrics[i].Value
			queueChartSize++
		} else if !strings.Contains(queueMetrics[i].Labels.QueueName, "Retry") {
			listRow[queueChartSize] = queueMetrics[i].Labels.QueueName + " " + queueMetrics[i].Value
			queueChartSize++
		}

	}
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

	//Sys load data
	for i := 0; i < 60; i++ {
		if i == int(timeSecond) {
			//order: active, max, idle, minIdle
			sysLoadplotData[0][i] = float64(sysLoadOneInt)
			sysLoadplotData[1][i] = float64(sysLoadFiveInt)
			sysLoadplotData[2][i] = float64(sysLoadFifteenInt)
			log.Debug("current time:", i)
		}

		for i := 0; i < interval; i++ {
			if timeSecond+i < 60 {
				//order: active, max, idle, minIdle
				sysLoadplotData[0][timeSecond+i] = float64(sysLoadOneInt)
				sysLoadplotData[1][timeSecond+i] = float64(sysLoadFiveInt)
				sysLoadplotData[2][timeSecond+i] = float64(sysLoadFifteenInt)
				log.Debug("current time:", i)
			}
		}
	}
	p3.Data = sysLoadplotData

	p2.DataLabels = []string{"hello"}

	log.Debug("size of plot rc:", len(rcPlotFinalData))
	p2.Data = rcPlotFinalData

	//metrics data
	r.Text = "Count: " + strconv.Itoa(len(data)) + "\nHeap Proc: " + heapProc + "\nHeap Total: " + heapTotalSpace.String()

	o.Text = "Current time: " + time.Now().Format("2006.01.02 15:04:05") + "\nLast updated: " + lastUpdate + " (" + strconv.Itoa(offset) + " seconds) Data Compute time:" + time.Now().Sub(responseTimeCompute).String() + "\nResponse time: " + time.Now().Sub(responseTime).String() + " Polling interval: every " + strconv.Itoa(interval) + " seconds\nServer url: " + config.ServerId

	ui.Render(bc, g2, g3, g4, l, o, o2, p1, p3, q, r)
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
