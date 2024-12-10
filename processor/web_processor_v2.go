package processor

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"skripsi/constant"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/exp/rand"
	"gonum.org/v1/gonum/mat"
)

func (p *WebProcessorImpl) HandleFloodPredictionRequestV2(c echo.Context) error {
	var lagOrder int

	startDateLimit := time.Date(2007, 12, 31, 0, 0, 0, 0, time.Local)
	endDateLimit := time.Date(2024, 10, 1, 0, 0, 0, 0, time.Local)
	p.logger.LogAndContinue("Start Processing Request")
	start := time.Now()
	cities := map[string]string{
		"jakarta barat":   "-6.1674&106.7637",
		"jakarta timur":   "-6.2250&106.9004",
		"jakarta pusat":   "-6.1805&106.8284",
		"jakarta selatan": "-6.2615&106.8106",
		"jakarta utara":   "-6.1481&106.8998",
		"bogor":           "-6.2600&106.4800",
		"depok":           "-6.2350&106.4900",
		"tangerang":       "-6.1000&106.3000",
		"bekasi":          "-6.3350&107.1329",
	}

	// Begin Validation
	startDate, err := time.Parse(DateHyphenYMD, c.FormValue("start_date"))
	if err != nil {
		return c.Render(http.StatusOK, MainPage, IndexData{
			Err:        "Parsing Date Error",
			StatusCode: http.StatusBadRequest,
			Timestamp:  time.Now().Unix(),
		})
	}
	endDate, err := time.Parse(DateHyphenYMD, c.FormValue("end_date"))
	if err != nil {
		return c.Render(http.StatusOK, MainPage, IndexData{
			Err:        "Parsing Date Error",
			StatusCode: http.StatusBadRequest,
			Timestamp:  time.Now().Unix(),
		})
	}

	city := c.FormValue("city")
	if startDate.After(endDate) {
		return c.Render(http.StatusOK, MainPage, IndexData{
			Err:        "Start Date can't be later than End Date",
			StatusCode: http.StatusUnprocessableEntity,
		})
	}

	if int(endDate.Sub(startDate).Hours()/24) < 180 {
		return c.Render(http.StatusOK, MainPage, IndexData{
			Err:        "Day Count can't be lower than 180 days to ensure proper calculation",
			StatusCode: http.StatusUnprocessableEntity,
		})
	}

	if startDate.Before(startDateLimit) || endDate.After(endDateLimit) {
		return c.Render(http.StatusOK, MainPage, IndexData{
			Err:        "Date can only be within 2008/01/01 until 2024/09/30",
			StatusCode: http.StatusUnprocessableEntity,
		})
	}

	kValue, err := strconv.Atoi(c.FormValue("k_value"))
	if err != nil {
		return c.Render(http.StatusOK, MainPage, IndexData{
			Err:        "K Value is not a valid number",
			StatusCode: http.StatusUnprocessableEntity,
		})
	}

	if kValue <= 0 || kValue > 500 {
		return c.Render(http.StatusOK, MainPage, IndexData{
			Err:        "Chosen K Value is not Valid (Must be 1 - 500)",
			StatusCode: http.StatusUnprocessableEntity,
		})
	}

	if FlagV2 {
		lagOrder, err = strconv.Atoi(c.FormValue("lag_order"))
		if err != nil {
			return c.Render(http.StatusOK, MainPage, IndexData{
				Err:        "Lag Order is not a valid number",
				StatusCode: http.StatusUnprocessableEntity,
			})
		}

		if lagOrder <= 0 || lagOrder > 10 {
			return c.Render(http.StatusOK, MainPage, IndexData{
				Err:        "Chosen Lag Order is not Valid (Must be 1 - 10)",
				StatusCode: http.StatusUnprocessableEntity,
			})
		}
	} else {
		lagOrder = 5
	}

	smoteK, err := strconv.Atoi(c.FormValue("smote_k"))
	if err != nil {
		return c.Render(http.StatusOK, MainPage, IndexData{
			Err:        "SMOET K Value is not a valid number",
			StatusCode: http.StatusUnprocessableEntity,
		})
	}

	if smoteK <= 0 || smoteK > 10 {
		return c.Render(http.StatusOK, MainPage, IndexData{
			Err:        "Chosen SMOTE K Value is not Valid (Must be 1 - 10)",
			StatusCode: http.StatusUnprocessableEntity,
		})
	}

	startDateRequest := strings.ReplaceAll(c.FormValue("start_date"), "-", "")
	endDateRequest := strings.ReplaceAll(c.FormValue("end_date"), "-", "")

	latlong, exists := cities[city]
	if !exists {
		return c.Render(http.StatusOK, MainPage, IndexData{
			Err:        "City is not available",
			StatusCode: http.StatusUnprocessableEntity,
		})
	}
	// End Validation

	latitude := strings.Split(latlong, "&")[0]
	longitude := strings.Split(latlong, "&")[1]

	weathers := Weathers{}
	nasa := NasaData{}
	bnpb := BnpbData{}
	news := NewsData{}
	url := fmt.Sprintf("%s?start=%s&end=%s&latitude=%s&longitude=%s&%s", constant.NasaPowerAPIBaseURL, startDateRequest, endDateRequest, latitude, longitude, constant.NasaPowerAPIParams)
	weathers.PrepareNasa(url)
	if weathers.Err != nil {
		return c.Render(http.StatusOK, MainPage, IndexData{
			Err:        "Fetching Data from NASA Power API Fails",
			StatusCode: http.StatusUnprocessableEntity,
		})
	}

	cmd := exec.Command("python", "/home/vasti/Hobby/skripsi/granger_causality_test.py")
	cmd.Run()

	weathers.InjectNasa(&nasa)
	if weathers.Err != nil {
		return c.Render(http.StatusOK, MainPage, IndexData{
			Err:        "Preparing Data from NASA Power API Fails",
			StatusCode: http.StatusUnprocessableEntity,
		})
	}
	nasa.Stats()

	weathers.InjectBnpb(&bnpb, startDate, endDate, city)
	if weathers.Err != nil {
		return c.Render(http.StatusOK, MainPage, IndexData{
			Err:        "Preparing Data from BNPB Fails",
			StatusCode: http.StatusUnprocessableEntity,
		})
	}

	weathers.InjectNews(&news, startDate, endDate, city)
	if weathers.Err != nil {
		return c.Render(http.StatusOK, MainPage, IndexData{
			Err:        "Preparing Data from News Fails",
			StatusCode: http.StatusUnprocessableEntity,
		})
	}

	differencedWeathers := weathers.Differencing()

	prediction := differencedWeathers.VectorAutoregression(lagOrder)
	prediction.FillString()
	vectorAutoregressionEvaluation := differencedWeathers.VectorAutoregressionEval(6, 5, lagOrder)
	neighbors, knnResult := differencedWeathers.KNearestNeighbor(kValue, prediction, false)
	knnEval := differencedWeathers.KNearestNeighborEval(6, 5, kValue, lagOrder, false)

	oversampled := differencedWeathers.SmoteOversampling(smoteK, nasa)
	smoteNeighbors, smoteKnnResult := oversampled.KNearestNeighbor(kValue, prediction, true)
	smoteKnnEval := oversampled.KNearestNeighborEval(6, 5, kValue, lagOrder, true)

	statistics := Statistics{
		Ref: StatisticsReference{
			Nasa:                &nasa,
			Bnpb:                &bnpb,
			News:                &news,
			Weathers:            &weathers,
			DifferencedWeathers: &differencedWeathers,
			Smote:               &oversampled,
		},
	}
	predictionMap := []KeyValue{
		{Key: "WS10M", Value: prediction.WindSpeedStr},
		{Key: "RH2M", Value: prediction.RelHumidityStr},
		{Key: "PRECTOTCORR", Value: prediction.PrecipitationStr},
		{Key: "T2M", Value: prediction.TempAverageStr},
		{Key: "T2M_MAX", Value: prediction.TempMaxStr},
		{Key: "T2M_MIN", Value: prediction.TempMinStr},
	}

	statistics.FillStatistics(startDate, endDate, city)
	weathers.FillString()
	differencedWeathers.FillString()
	neighbors.FillString()
	oversampled.FillString()
	smoteNeighbors.FillString()
	p.logger.LogAndContinue("Done Processing Request")
	structToJsonFile(weathers, "weathers.json")
	viewData := map[string]interface{}{
		"NasaHeaders":                       []string{"DATE", "WS10M", "RH2M", "PRECTOTCORR", "T2M", "T2M MAX", "T2M MIN"},
		"NasaStats":                         []Nasa{nasa.Max, nasa.Min, nasa.Mean, nasa.Variance, nasa.StdDev},
		"NasaValues":                        nasa.Items,
		"BnpbHeaders":                       []string{"Kode Identitas Bencana", "ID Kabupaten", "Tanggal Kejadian", "Kejadian", "Lokasi", "Kabupaten", "Provinsi", "Penyebab"},
		"BnpbValues":                        bnpb.Items,
		"NewsHeaders":                       []string{"Kota", "Tanggal", "Link Berita"},
		"NewsValues":                        news.Items,
		"WeatherAndFloodHeaders":            []string{"DATE", "WS10M", "RH2M", "PRECTOTCORR", "T2M", "T2M MAX", "T2M MIN", "FLOOD"},
		"WeatherAndFloodValues":             weathers.Items,
		"DifferencedWeatherAndFloodHeaders": []string{"DATE", "WS10M", "RH2M", "PRECTOTCORR", "T2M", "T2M MAX", "T2M MIN", "FLOOD"},
		"DifferencedWeatherAndFloodValues":  differencedWeathers.Items,
		"DifferencedWeatherAndFloodStats":   differencedWeathers.Diff,
		"VectorAutoregressionHeaders":       []string{"TRAIN-TEST (%)", "WS10M", "RH2M", "PRECTOTCORR", "T2M", "T2M MAX", "T2M MIN"},
		"VectorAutoregressionValues":        vectorAutoregressionEvaluation.Items,
		"VectorAutoregressionResult":        predictionMap,
		"KNNHeaders":                        []string{"WS10M", "RH2M", "PRECTOTCORR", "T2M", "T2M MAX", "T2M MIN", "DISTANCE", "FLOOD"},
		"KNNValues":                         neighbors.Items,
		"KNNResult":                         knnResult,
		"KNNEvalHeaders":                    []string{"TRAIN-TEST (%)", "TP", "FP", "TN", "FN", "ACCURACY", "PRECISION", "RECALL", "F1-SCORE"},
		"KNNEvalValues":                     knnEval,
		"SMOTEHeaders":                      []string{"DATE", "WS10M", "RH2M", "PRECTOTCORR", "T2M", "T2M MAX", "T2M MIN", "FLOOD"},
		"SMOTEValues":                       oversampled.SynthItems,
		"SMOTEKNNHeaders":                   []string{"WS10M", "RH2M", "PRECTOTCORR", "T2M", "T2M MAX", "T2M MIN", "DISTANCE", "FLOOD"},
		"SMOTEKNNValues":                    smoteNeighbors.Items,
		"SMOTEKNNResult":                    smoteKnnResult,
		"SMOTEKNNEvalHeaders":               []string{"TRAIN-TEST (%)", "TP", "FP", "TN", "FN", "ACCURACY", "PRECISION", "RECALL", "F1-SCORE"},
		"SMOTEKNNEvalValues":                smoteKnnEval,
		"Statistics":                        statistics,
		"Latitude":                          latitude,
		"Longitude":                         longitude,
		"Timestamp":                         time.Now().Unix(),
	}

	return c.Render(http.StatusOK, MainPage, IndexData{
		Data:       viewData,
		Message:    fmt.Sprintf("Preparation Done. Time Taken: %dms", time.Since(start).Milliseconds()),
		StatusCode: http.StatusOK,
	})
}

func (w *Weathers) PrepareNasa(url string) {
	resp, err := http.Get(url)
	if err != nil {
		w.Err = err
		fmt.Printf("[NASA-FETCH] error fetching from url: %v", err)
		return
	}
	defer resp.Body.Close()

	wd, err := os.Getwd()
	if err != nil {
		w.Err = err
		fmt.Printf("[NASA-FETCH] error getting working directory: %v", err)
		return
	}

	tempFile, err := os.Create(filepath.Join(wd, "tmp/nasa_data.txt"))
	if err != nil {
		w.Err = err
		fmt.Printf("[NASA-FETCH] error creating temp file: %v", err)
		return
	}

	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		w.Err = err
		fmt.Printf("[NASA-FETCH] error saving repsonse data to temp file: %v", err)
		return
	}
	tempFile.Close()

	tempFile, err = os.Open(filepath.Join(wd, "tmp/nasa_data.txt"))
	if err != nil {
		w.Err = err
		fmt.Printf("[NASA-FETCH] error opening temp file: %v", err)
		return
	}
	defer tempFile.Close()

	csvFile, err := os.Create(filepath.Join(wd, "tmp/nasa_data.csv"))
	if err != nil {
		w.Err = err
		fmt.Printf("[NASA-FETCH] error creating csv: %v", err)
		return
	}
	defer csvFile.Close()

	scanner := bufio.NewScanner(tempFile)
	lineCount := 0

	for scanner.Scan() {
		lineCount++

		if lineCount > 14 {
			csvLine := scanner.Text()
			csvLine = strings.ReplaceAll(csvLine, "\t", ",")
			_, err := csvFile.WriteString(csvLine + "\n")
			if err != nil {
				w.Err = err
				fmt.Printf("[NASA-FETCH] error writing to csv: %v", err)
				return
			}
		}
	}

	if err := scanner.Err(); err != nil {
		w.Err = err
		fmt.Printf("[NASA-FETCH] error scanning temp file io content: %v", err)
		return
	}
}

func (w *Weathers) InjectNasa(nasa *NasaData) {
	wd, err := os.Getwd()
	if err != nil {
		w.Err = err
		fmt.Printf("[NASA-INJECT] error getting working directory: %v", err)
		return
	}

	csvFile, err := os.Open(filepath.Join(wd, "tmp/nasa_data.csv"))
	if err != nil {
		w.Err = err
		fmt.Printf("[NASA-INJECT] error opening csv: %v", err)
		return
	}
	defer csvFile.Close()

	reader := csv.NewReader(csvFile)
	records, err := reader.ReadAll()
	if err != nil {
		w.Err = err
		fmt.Printf("[NASA-INJECT] error reading csv file: %v", err)
	}

	headers := records[0][2:]
	headersIndex := make(map[string]int)
	for i, header := range headers {
		headersIndex[header] = i
	}

	records = records[1:]
	for _, record := range records {
		year, _ := strconv.Atoi(record[0])
		doy, _ := strconv.Atoi(record[1])

		startOfYear := time.Date(year, 1, 1, 0, 0, 0, 0, time.Local)
		date := startOfYear.AddDate(0, 0, doy-1)
		dateStr := date.Format(DateHyphenYMD)

		data := record[2:]
		ws10m, _ := strconv.ParseFloat(data[headersIndex["WS10M"]], 64)
		rh2m, _ := strconv.ParseFloat(data[headersIndex["RH2M"]], 64)
		prectotcorr, _ := strconv.ParseFloat(data[headersIndex["PRECTOTCORR"]], 64)
		t2m, _ := strconv.ParseFloat(data[headersIndex["T2M"]], 64)
		t2mMax, _ := strconv.ParseFloat(data[headersIndex["T2M_MAX"]], 64)
		t2mMin, _ := strconv.ParseFloat(data[headersIndex["T2M_MIN"]], 64)

		nasa.Items = append(nasa.Items, Nasa{
			DateStr:          dateStr,
			WindSpeed:        ws10m,
			RelHumidity:      rh2m,
			Precipitation:    prectotcorr,
			TempAverage:      t2m,
			TempMax:          t2mMax,
			TempMin:          t2mMin,
			WindSpeedStr:     data[headersIndex["WS10M"]],
			RelHumidityStr:   data[headersIndex["RH2M"]],
			PrecipitationStr: data[headersIndex["PRECTOTCORR"]],
			TempAverageStr:   data[headersIndex["T2M"]],
			TempMaxStr:       data[headersIndex["T2M_MAX"]],
			TempMinStr:       data[headersIndex["T2M_MIN"]],
		})
		w.Items = append(w.Items, Weather{
			Date:          date,
			WindSpeed:     ws10m,
			RelHumidity:   rh2m,
			Precipitation: prectotcorr,
			TempAverage:   t2m,
			TempMax:       t2mMax,
			TempMin:       t2mMin,
		})
	}
}

func (w *Weathers) InjectBnpb(bnpb *BnpbData, startDate, endDate time.Time, city string) {
	wd, err := os.Getwd()
	if err != nil {
		w.Err = err
		fmt.Printf("[BNPB-INJECT] error getting working directory: %v", err)
		return
	}

	csvFile, err := os.Open(filepath.Join(wd, "tmp/bnpb_data.csv"))
	if err != nil {
		w.Err = err
		fmt.Printf("[BNPB-INJECT] error opening csv: %v", err)
		return
	}
	defer csvFile.Close()

	reader := csv.NewReader(csvFile)
	records, err := reader.ReadAll()
	if err != nil {
		w.Err = err
		fmt.Printf("[BNPB-INJECT] error reading csv: %v", err)
		return
	}

	indexCode := 1
	indexCityID := 2
	indexDate := 3
	indexOccurence := 4
	indexLocation := 5
	indexCity := 6
	indexProvince := 7
	indexCause := 9
	floodDates := make(map[string][]string)

	for _, record := range records {
		if !strings.Contains(strings.ToLower(record[indexCity]), city) {
			continue
		}

		dateStr := record[indexDate]
		date, _ := time.Parse("02/01/2006", dateStr)
		if date.After(endDate) || date.Before(startDate) {
			continue
		}

		if _, exists := floodDates[dateStr]; !exists {
			floodDates[dateStr] = record
		}
	}

	for i, d := range w.Items {
		dateStr := d.Date.Format("02/01/2006")
		if record, exists := floodDates[dateStr]; exists {
			bnpb.Items = append(bnpb.Items, Bnpb{
				Code:      record[indexCode],
				CityID:    record[indexCityID],
				Date:      record[indexDate],
				Occurence: record[indexOccurence],
				Location:  record[indexLocation],
				City:      record[indexCity],
				Province:  record[indexProvince],
				Cause:     record[indexCause],
			})
			w.Items[i].Flood = true
		}
	}
}

func (w *Weathers) InjectNews(news *NewsData, startDate, endDate time.Time, city string) {
	wd, err := os.Getwd()
	if err != nil {
		w.Err = err
		fmt.Printf("[NEWS-INJECT] error getting working directory: %v", err)
		return
	}

	csvFile, err := os.Open(filepath.Join(wd, "tmp/data_berita_banjir.csv"))
	if err != nil {
		w.Err = err
		fmt.Printf("[NEWS-INJECT] error opening csv: %v", err)
		return
	}
	defer csvFile.Close()

	reader := csv.NewReader(csvFile)
	records, err := reader.ReadAll()
	if err != nil {
		w.Err = err
		fmt.Printf("[NEWS-INJECT] error reading csv: %v", err)
		return
	}

	indexCity := 0
	indexDate := 1
	indexLink := 2
	floodDates := make(map[string][]string)

	for _, record := range records {
		if !strings.Contains(strings.ToLower(record[indexCity]), city) {
			continue
		}

		dateStr := record[indexDate]
		date, _ := time.Parse("2006/01/02", dateStr)
		if date.After(endDate) || date.Before(startDate) {
			continue
		}

		if _, exists := floodDates[dateStr]; !exists {
			floodDates[dateStr] = record
		}
	}

	for i, d := range w.Items {
		dateStr := d.Date.Format("2006/01/02")
		if record, exists := floodDates[dateStr]; exists {
			news.Items = append(news.Items, News{
				City: record[indexCity],
				Date: record[indexDate],
				Link: template.HTML(fmt.Sprintf("<a class=\"text-blue-800\" href=\"%s\">Link</a>", record[indexLink])),
			})
			w.Items[i].Flood = true
		}
	}
}

func (w *Weathers) Differencing() (differencedWeathers Weathers) {
	var (
		steps                                                                                                                                        int
		windSpeed, relHumidity, precipitation, tempAverage, tempMax, tempMin                                                                         []float64
		critValWindSpeed, critValRelHumidity, critValPrecipitation, critValTempAverage, critValTempMax, critValTempMin                               float64
		gammaWindSpeed, gammaRelHumidity, gammaPrecipitation, gammaTempAverage, gammaTempMax, gammaTempMin                                           float64
		windSpeedStationarity, relHumidityStationarity, precipitationStationarity, tempAverageStationarity, tempMaxStationarity, tempMinStationarity bool
	)

	for i := 0; i < len(w.Items); i++ {
		windSpeed = append(windSpeed, w.Items[i].WindSpeed)
		relHumidity = append(relHumidity, w.Items[i].RelHumidity)
		precipitation = append(precipitation, w.Items[i].Precipitation)
		tempAverage = append(tempAverage, w.Items[i].TempAverage)
		tempMax = append(tempMax, w.Items[i].TempMax)
		tempMin = append(tempMin, w.Items[i].TempMin)
	}

	stationary := false
	for !stationary {
		windSpeedStationarity, critValWindSpeed, gammaWindSpeed, _ = adfTest(windSpeed)
		relHumidityStationarity, critValRelHumidity, gammaRelHumidity, _ = adfTest(relHumidity)
		precipitationStationarity, critValPrecipitation, gammaPrecipitation, _ = adfTest(precipitation)
		tempAverageStationarity, critValTempAverage, gammaTempAverage, _ = adfTest(tempAverage)
		tempMaxStationarity, critValTempMax, gammaTempMax, _ = adfTest(tempMax)
		tempMinStationarity, critValTempMin, gammaTempMin, _ = adfTest(tempMin)

		if windSpeedStationarity && relHumidityStationarity && precipitationStationarity && tempAverageStationarity && tempMaxStationarity && tempMinStationarity {
			stationary = true
			break
		}

		windSpeed = difference(windSpeed)
		relHumidity = difference(relHumidity)
		precipitation = difference(precipitation)
		tempAverage = difference(tempAverage)
		tempMax = difference(tempMax)
		tempMin = difference(tempMin)
		steps++
	}

	for i := 0; i < len(windSpeed); i++ {
		differencedWeathers.Items = append(differencedWeathers.Items, Weather{
			Date:          w.Items[steps+i].Date,
			WindSpeed:     windSpeed[i],
			RelHumidity:   relHumidity[i],
			Precipitation: precipitation[i],
			TempAverage:   tempAverage[i],
			TempMax:       tempMax[i],
			TempMin:       tempMin[i],
			Flood:         w.Items[steps+i].Flood,
		})
	}
	differencedWeathers.Diff.Step = steps
	differencedWeathers.Diff.CriticalValues = Weather{
		WindSpeed:     critValWindSpeed,
		RelHumidity:   critValRelHumidity,
		Precipitation: critValPrecipitation,
		TempAverage:   critValTempAverage,
		TempMax:       critValTempMax,
		TempMin:       critValTempMin,
	}
	differencedWeathers.Diff.Gamma = Weather{
		WindSpeed:     gammaWindSpeed,
		RelHumidity:   gammaRelHumidity,
		Precipitation: gammaPrecipitation,
		TempAverage:   gammaTempAverage,
		TempMax:       gammaTempMax,
		TempMin:       gammaTempMin,
	}

	differencedWeathers.Diff.CriticalValuesGammaMap = []KeyValue{
		{Key: "WS10M", Value: fmt.Sprintf("Critical Value: %s > %s", strconv.FormatFloat(differencedWeathers.Diff.CriticalValues.WindSpeed, 'f', 5, 64), strconv.FormatFloat(differencedWeathers.Diff.Gamma.WindSpeed, 'f', 5, 64))},
		{Key: "RH2M", Value: fmt.Sprintf("Critical Value: %s > %s", strconv.FormatFloat(differencedWeathers.Diff.CriticalValues.RelHumidity, 'f', 5, 64), strconv.FormatFloat(differencedWeathers.Diff.Gamma.RelHumidity, 'f', 5, 64))},
		{Key: "PRECTOTCORR", Value: fmt.Sprintf("Critical Value: %s > %s", strconv.FormatFloat(differencedWeathers.Diff.CriticalValues.Precipitation, 'f', 5, 64), strconv.FormatFloat(differencedWeathers.Diff.Gamma.Precipitation, 'f', 5, 64))},
		{Key: "T2M", Value: fmt.Sprintf("Critical Value: %s > %s", strconv.FormatFloat(differencedWeathers.Diff.CriticalValues.TempAverage, 'f', 5, 64), strconv.FormatFloat(differencedWeathers.Diff.Gamma.TempAverage, 'f', 5, 64))},
		{Key: "T2M_MAX", Value: fmt.Sprintf("Critical Value: %s > %s", strconv.FormatFloat(differencedWeathers.Diff.CriticalValues.TempMax, 'f', 5, 64), strconv.FormatFloat(differencedWeathers.Diff.Gamma.TempMax, 'f', 5, 64))},
		{Key: "T2M_MIN", Value: fmt.Sprintf("Critical Value: %s > %s", strconv.FormatFloat(differencedWeathers.Diff.CriticalValues.TempMin, 'f', 5, 64), strconv.FormatFloat(differencedWeathers.Diff.Gamma.TempMin, 'f', 5, 64))},
	}

	return
}

func (w *Weathers) VectorAutoregression(lagOrder int) (prediction Weather) {
	numOfVariables := 6
	matrixForm := make([][]float64, len(w.Items))
	for i, d := range w.Items {
		matrixForm[i] = []float64{d.WindSpeed, d.RelHumidity, d.Precipitation, d.TempAverage, d.TempMax, d.TempMin}
	}
	rowCount := len(matrixForm) - lagOrder

	responseSlice := make([][]float64, rowCount)
	for i := 0; i < rowCount; i++ {
		responseSlice[i] = matrixForm[lagOrder+i]
	}

	regressorSlice := make([][]float64, rowCount)
	for i := 0; i < rowCount; i++ {
		row := []float64{1.0}
		for lag := 1; lag <= lagOrder; lag++ {
			row = append(row, matrixForm[lagOrder+i-lag]...)
		}
		regressorSlice[i] = row
	}

	responseMatrix := mat.NewDense(len(responseSlice), len(responseSlice[0]), flatten(responseSlice))
	regressorMatrix := mat.NewDense(len(regressorSlice), len(regressorSlice[0]), flatten(regressorSlice))

	var xTx mat.Dense
	xTx.Mul(regressorMatrix.T(), regressorMatrix)

	var xTxInv mat.Dense
	if err := xTxInv.Inverse(&xTx); err != nil {
		return
	}

	var xTy mat.Dense
	xTy.Mul(regressorMatrix.T(), responseMatrix)

	var B mat.Dense
	B.Mul(&xTxInv, &xTy)

	result := make([][]float64, B.RawMatrix().Rows)
	for i := 0; i < len(result); i++ {
		result[i] = B.RawRowView(i)
	}
	result = transpose(result)

	predictionSlice := make([]float64, 6)
	for i, d := range result {
		predictionSlice[i] = d[0]
		for j := 1; j < len(d); j++ {
			previousIndex := len(matrixForm) - ((j-1)/6 + 1)
			predictionSlice[i] += d[j] * matrixForm[previousIndex][(j-1)-numOfVariables*((j-1)/numOfVariables)]
		}
	}

	prediction.WindSpeed = predictionSlice[0]
	prediction.RelHumidity = predictionSlice[1]
	prediction.Precipitation = predictionSlice[2]
	prediction.TempAverage = predictionSlice[3]
	prediction.TempMax = predictionSlice[4]
	prediction.TempMin = predictionSlice[5]

	return
}

func (w *Weathers) VectorAutoregressionEval(step, magnitude, lagOrder int) (evaluatedNrmse Weathers) {
	if magnitude*step > 100 {
		return
	}

	max, min := w.GetMaxMin()
	nrmseEval := make([]Weather, step)

	for i := 1; i <= step; i++ {
		testPerc := fmt.Sprintf("%d", i*magnitude)
		trainPerc := fmt.Sprintf("%d", 100-i*magnitude)

		test := magnitude * i
		testSize := len(w.Items) * test / 100
		trainSize := len(w.Items) - testSize

		rmse := Weather{}
		nrmse := Weather{}
		predictionCount := 0

		for j := trainSize; j < len(w.Items)-1; j++ {
			trainSlice := w.Items[:j]
			trainDataset := Weathers{
				Items: trainSlice,
			}

			predicted := trainDataset.VectorAutoregression(lagOrder)
			actual := w.Items[j]

			rmse.WindSpeed += math.Pow(predicted.WindSpeed-actual.WindSpeed, 2)
			rmse.RelHumidity += math.Pow(predicted.RelHumidity-actual.RelHumidity, 2)
			rmse.Precipitation += math.Pow(predicted.Precipitation-actual.Precipitation, 2)
			rmse.TempAverage += math.Pow(predicted.TempAverage-actual.TempAverage, 2)
			rmse.TempMax += math.Pow(predicted.TempMax-actual.TempMax, 2)
			rmse.TempMin += math.Pow(predicted.TempMin-actual.TempMin, 2)

			predictionCount++
		}

		predictionSize := float64(predictionCount)
		rmse.WindSpeed = math.Sqrt(rmse.WindSpeed / predictionSize)
		rmse.RelHumidity = math.Sqrt(rmse.RelHumidity / predictionSize)
		rmse.Precipitation = math.Sqrt(rmse.Precipitation / predictionSize)
		rmse.TempAverage = math.Sqrt(rmse.TempAverage / predictionSize)
		rmse.TempMax = math.Sqrt(rmse.TempMax / predictionSize)
		rmse.TempMin = math.Sqrt(rmse.TempMin / predictionSize)

		nrmse.WindSpeed = rmse.WindSpeed / (max.WindSpeed - min.WindSpeed)
		nrmse.RelHumidity = rmse.RelHumidity / (max.RelHumidity - min.RelHumidity)
		nrmse.Precipitation = rmse.Precipitation / (max.Precipitation - min.Precipitation)
		nrmse.TempAverage = rmse.TempAverage / (max.TempAverage - min.TempAverage)
		nrmse.TempMax = rmse.TempMax / (max.TempMax - min.TempMax)
		nrmse.TempMin = rmse.TempMin / (max.TempMin - min.TempMin)

		nrmseEval[i-1] = nrmse
		nrmseEval[i-1].FillString()
		nrmseEval[i-1].DateStr = fmt.Sprintf("%s - %s", trainPerc, testPerc)
	}
	evaluatedNrmse.Items = nrmseEval
	return
}

func (w *Weathers) KNearestNeighbor(kValue int, new Weather, withSynth bool) (neighbors Weathers, result string) {
	tempW := Weathers{}
	if withSynth {
		tempW.Items = make([]Weather, len(w.SynthItems))

		for i, d := range w.SynthItems {
			distance := math.Pow(new.WindSpeed-d.WindSpeed, 2)
			distance += math.Pow(new.RelHumidity-d.RelHumidity, 2)
			distance += math.Pow(new.Precipitation-d.Precipitation, 2)
			distance += math.Pow(new.TempAverage-d.TempAverage, 2)
			distance += math.Pow(new.TempMax-d.TempMax, 2)
			distance += math.Pow(new.TempMin-d.TempMin, 2)

			d.Distance = math.Sqrt(distance)
			tempW.Items[i] = d
		}
	} else {
		tempW.Items = make([]Weather, len(w.Items))

		for i, d := range w.Items {
			distance := math.Pow(new.WindSpeed-d.WindSpeed, 2)
			distance += math.Pow(new.RelHumidity-d.RelHumidity, 2)
			distance += math.Pow(new.Precipitation-d.Precipitation, 2)
			distance += math.Pow(new.TempAverage-d.TempAverage, 2)
			distance += math.Pow(new.TempMax-d.TempMax, 2)
			distance += math.Pow(new.TempMin-d.TempMin, 2)

			d.Distance = math.Sqrt(distance)
			tempW.Items[i] = d
		}
	}

	tempW.SortByDistance()
	vote := make([]int, 2)
	neighbors.Items = make([]Weather, kValue)
	for i, d := range tempW.Items {
		if i >= kValue {
			break
		}
		if !d.Flood {
			vote[0] += 1
		} else {
			vote[1] += 1
		}
		neighbors.Items[i] = d
	}

	if vote[0] >= vote[1] {
		result = "No Flood"
	} else {
		result = "Flood"
	}

	return
}

func (w *Weathers) KNearestNeighborMinority(kValue int, new Weather) (neighbors Weathers, result string) {
	tempW := Weathers{}
	tempW.Items = make([]Weather, len(w.Items))
	for i, d := range w.Items {
		distance := math.Pow(new.WindSpeed-d.WindSpeed, 2)
		distance += math.Pow(new.RelHumidity-d.RelHumidity, 2)
		distance += math.Pow(new.Precipitation-d.Precipitation, 2)
		distance += math.Pow(new.TempAverage-d.TempAverage, 2)
		distance += math.Pow(new.TempMax-d.TempMax, 2)
		distance += math.Pow(new.TempMin-d.TempMin, 2)

		w.Items[i].Distance = math.Sqrt(distance)
		d.Distance = math.Sqrt(distance)
		tempW.Items[i] = d
	}

	tempW.SortByDistance()
	vote := make([]int, 2)
	neighbors.Items = make([]Weather, kValue)
	for i, d := range tempW.Items {
		if !d.Flood {
			continue
		}
		if i >= kValue {
			break
		}
		if !d.Flood {
			vote[0] += 1
		} else {
			vote[1] += 1
		}
		neighbors.Items[i] = d
	}

	if vote[0] >= vote[1] {
		result = "No Flood"
	} else {
		result = "Flood"
	}

	return
}

func (w *Weathers) KNearestNeighborEval(step, magnitude, kValue, lagOrder int, withSynth bool) (confusionMatrix []ConfusionMatrix) {
	if magnitude*step > 100 {
		return
	}

	tempW := Weathers{}
	if withSynth {
		tempW.Items = make([]Weather, len(w.SynthItems))
		copy(tempW.Items, w.SynthItems)
	} else {
		tempW.Items = make([]Weather, len(w.Items))
		copy(tempW.Items, w.Items)
	}

	confusionMatrix = make([]ConfusionMatrix, step)
	for i := 1; i <= step; i++ {
		testPerc := fmt.Sprintf("%d", i*magnitude)
		trainPerc := fmt.Sprintf("%d", 100-i*magnitude)

		test := magnitude * i
		testSize := len(tempW.Items) * test / 100
		trainSize := len(tempW.Items) - testSize

		predictionCount := 0

		for j := trainSize; j < len(tempW.Items)-1; j++ {
			trainSlice := tempW.Items[:j]
			trainDataset := Weathers{
				Items:      trainSlice,
				SynthItems: trainSlice,
			}

			predicted := trainDataset.VectorAutoregression(lagOrder)
			_, knnResult := trainDataset.KNearestNeighbor(kValue, predicted, withSynth)
			actual := tempW.Items[j]

			flood := false
			if knnResult == "Flood" {
				flood = true
			}

			if actual.Flood && flood {
				confusionMatrix[i-1].TruePositive += 1
			}
			if !actual.Flood && flood {
				confusionMatrix[i-1].FalsePositive += 1
			}
			if actual.Flood && !flood {
				confusionMatrix[i-1].FalseNegative += 1
			}
			if !actual.Flood && !flood {
				confusionMatrix[i-1].TrueNegative += 1
			}

			predictionCount++
		}
		confusionMatrix[i-1].Metrics()
		confusionMatrix[i-1].FillString()
		confusionMatrix[i-1].TrainTestStr = fmt.Sprintf("%s - %s", trainPerc, testPerc)
	}

	return
}

func (w *Weathers) SmoteOversampling(kValue int, nasa NasaData) (oversampledData Weathers) {
	minoritySample := w.GetMinoritySample()

	var syntheticData []Weather
	for _, d := range minoritySample.Items {
		neighbors, _ := minoritySample.KNearestNeighborMinority(kValue+1, d)
		for _, e := range neighbors.Items {
			if isSame(d, e) {
				continue
			}
			synthData := d.InterpolateSyntheticData(e)
			syntheticData = append(syntheticData, synthData)
		}
	}

	var pointerSynth []*Weather
	for i := range syntheticData {
		pointerSynth = append(pointerSynth, &syntheticData[i])
	}

	var wg sync.WaitGroup
	for _, d := range pointerSynth {
		wg.Add(1)
		go func(p *Weather) {
			defer wg.Done()
			for _, e := range minoritySample.Items {
				p.GetCosineSimilarity(e, &nasa)
			}
			p.AvgCosineSimilarity = getMean(p.CosineSimilarity)
		}(d)
	}
	wg.Wait()

	avgCosineSimilarities := make([]float64, len(syntheticData))
	for i := range syntheticData {
		avgCosineSimilarities[i] = syntheticData[i].AvgCosineSimilarity
	}
	fmt.Println("City Average Cosine Similarity: ", getMean(avgCosineSimilarities))

	structToJsonFile(syntheticData, "Synth.json")

	oversampledData.Items = make([]Weather, len(w.Items))
	copy(oversampledData.Items, w.Items)
	oversampledData.Oversample.SynthData = syntheticData
	oversampledData.Oversample.SynthDataCount = len(syntheticData)
	oversampledData.SynthItems = append(syntheticData, oversampledData.Items...)

	return
}

func (w *Weathers) GetMaxMin() (max, min Weather) {
	max.WindSpeed = w.Items[0].WindSpeed
	max.RelHumidity = w.Items[0].RelHumidity
	max.Precipitation = w.Items[0].Precipitation
	max.TempAverage = w.Items[0].TempAverage
	max.TempMax = w.Items[0].TempMax
	max.TempMin = w.Items[0].TempMin

	min.WindSpeed = w.Items[0].WindSpeed
	min.RelHumidity = w.Items[0].RelHumidity
	min.Precipitation = w.Items[0].Precipitation
	min.TempAverage = w.Items[0].TempAverage
	min.TempMax = w.Items[0].TempMax
	min.TempMin = w.Items[0].TempMin

	for _, d := range w.Items {
		if max.WindSpeed < d.WindSpeed {
			max.WindSpeed = d.WindSpeed
		}
		if max.RelHumidity < d.RelHumidity {
			max.RelHumidity = d.RelHumidity
		}
		if max.Precipitation < d.Precipitation {
			max.Precipitation = d.Precipitation
		}
		if max.TempAverage < d.TempAverage {
			max.TempAverage = d.TempAverage
		}
		if max.TempMax < d.TempMax {
			max.TempMax = d.TempMax
		}
		if max.TempMin < d.TempMin {
			max.TempMin = d.TempMin
		}

		if min.WindSpeed > d.WindSpeed {
			if d.WindSpeed != -999.0 {
				min.WindSpeed = d.WindSpeed
			}
		}
		if min.RelHumidity > d.RelHumidity {
			if d.RelHumidity != -999.0 {
				min.RelHumidity = d.RelHumidity
			}
		}
		if min.Precipitation > d.Precipitation {
			if d.Precipitation != -999.0 {
				min.Precipitation = d.Precipitation
			}
		}
		if min.TempAverage > d.TempAverage {
			if d.TempAverage != -999.0 {
				min.TempAverage = d.TempAverage
			}
		}
		if min.TempMax > d.TempMax {
			if d.TempMax != -999.0 {
				min.TempMax = d.TempMax
			}
		}
		if min.TempMin > d.TempMin {
			if d.TempMin != -999.0 {
				min.TempMin = d.TempMin
			}
		}
	}
	return
}

type ByDistance []Weather

func (a ByDistance) Len() int {
	return len(a)
}

func (a ByDistance) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ByDistance) Less(i, j int) bool {
	return a[i].Distance < a[j].Distance
}

func (w *Weathers) SortByDistance() {
	sort.Sort(ByDistance(w.Items))
}

func (w *Weathers) GetMinoritySample() (minoritySample Weathers) {
	for _, d := range w.Items {
		if d.Flood {
			minoritySample.Items = append(minoritySample.Items, d)
		}
	}
	return
}

func (w *Weathers) FillString() {
	for i := 0; i < len(w.Items); i++ {
		w.Items[i].FillString()
	}

	for i := 0; i < len(w.SynthItems); i++ {
		w.SynthItems[i].FillString()
	}
}

func (w *Weather) InterpolateSyntheticData(neighbor Weather) (synth Weather) {
	lambda := rand.Float64()
	synth.WindSpeed = w.WindSpeed + (lambda * (neighbor.WindSpeed - w.WindSpeed))
	synth.RelHumidity = w.RelHumidity + (lambda * (neighbor.RelHumidity - w.RelHumidity))
	synth.Precipitation = w.Precipitation + (lambda * (neighbor.Precipitation - w.Precipitation))
	synth.TempAverage = w.TempAverage + (lambda * (neighbor.TempAverage - w.TempAverage))
	synth.TempMax = w.TempMax + (lambda * (neighbor.TempMax - w.TempMax))
	synth.TempMin = w.TempMin + (lambda * (neighbor.TempMin - w.TempMin))
	synth.Flood = true
	return
}

func (w *Weather) GetCosineSimilarity(pair Weather, nasa *NasaData) {
	fmt.Println("Working The Cosine Similarity")
	// Min Max Scaler
	minMaxScaler := func(n, min, max float64) float64 {
		return ((n - min) / (max - min))
	}

	tempW := Weather{
		WindSpeed:     minMaxScaler(w.WindSpeed, nasa.Min.WindSpeed, nasa.Max.WindSpeed),
		RelHumidity:   minMaxScaler(w.RelHumidity, nasa.Min.RelHumidity, nasa.Max.RelHumidity),
		Precipitation: minMaxScaler(w.Precipitation, nasa.Min.Precipitation, nasa.Max.Precipitation),
		TempAverage:   minMaxScaler(w.TempAverage, nasa.Min.TempAverage, nasa.Max.TempAverage),
		TempMax:       minMaxScaler(w.TempMax, nasa.Min.TempMax, nasa.Max.TempMax),
		TempMin:       minMaxScaler(w.TempMin, nasa.Min.TempMin, nasa.Max.TempMin),
	}
	tempPair := Weather{
		WindSpeed:     minMaxScaler(pair.WindSpeed, nasa.Min.WindSpeed, nasa.Max.WindSpeed),
		RelHumidity:   minMaxScaler(pair.RelHumidity, nasa.Min.RelHumidity, nasa.Max.RelHumidity),
		Precipitation: minMaxScaler(pair.Precipitation, nasa.Min.Precipitation, nasa.Max.Precipitation),
		TempAverage:   minMaxScaler(pair.TempAverage, nasa.Min.TempAverage, nasa.Max.TempAverage),
		TempMax:       minMaxScaler(pair.TempMax, nasa.Min.TempMax, nasa.Max.TempMax),
		TempMin:       minMaxScaler(pair.TempMin, nasa.Min.TempMin, nasa.Max.TempMin),
	}

	// Cosine Similarity
	numerator := (tempW.WindSpeed*tempPair.WindSpeed +
		tempW.RelHumidity*tempPair.RelHumidity +
		tempW.Precipitation*tempPair.Precipitation +
		tempW.TempAverage*tempPair.TempAverage +
		tempW.TempMax*tempPair.TempMax +
		tempW.TempMin*tempPair.TempMin)
	denominator := (math.Sqrt(
		math.Pow(tempW.WindSpeed, 2)+
			math.Pow(tempW.RelHumidity, 2)+
			math.Pow(tempW.Precipitation, 2)+
			math.Pow(tempW.TempAverage, 2)+
			math.Pow(tempW.TempMax, 2)+
			math.Pow(tempW.TempMin, 2)) *
		math.Sqrt(
			math.Pow(tempPair.WindSpeed, 2)+
				math.Pow(tempPair.RelHumidity, 2)+
				math.Pow(tempPair.Precipitation, 2)+
				math.Pow(tempPair.TempAverage, 2)+
				math.Pow(tempPair.TempMax, 2)+
				math.Pow(tempPair.TempMin, 2)))

	fmt.Printf("Num: %.3f, Denom: %.3f \n", numerator, denominator)

	if denominator == 0 {
		w.CosineSimilarity = append(w.CosineSimilarity, 0)
	}
	w.CosineSimilarity = append(w.CosineSimilarity, numerator/denominator)
}

func (w *Weather) FillString() {
	w.DateStr = w.Date.Format("02/01/2006")
	w.WindSpeedStr = strconv.FormatFloat(w.WindSpeed, 'f', 2, 64)
	w.RelHumidityStr = strconv.FormatFloat(w.RelHumidity, 'f', 2, 64)
	w.PrecipitationStr = strconv.FormatFloat(w.Precipitation, 'f', 2, 64)
	w.TempAverageStr = strconv.FormatFloat(w.TempAverage, 'f', 2, 64)
	w.TempMaxStr = strconv.FormatFloat(w.TempMax, 'f', 2, 64)
	w.TempMinStr = strconv.FormatFloat(w.TempMin, 'f', 2, 64)
	w.DistanceStr = strconv.FormatFloat(w.Distance, 'f', 2, 64)
	if w.Flood {
		w.FloodStr = template.HTML(fmt.Sprintf("<p class=\"text-emerald-700\">%v</p>", w.Flood))
	} else {
		w.FloodStr = template.HTML(fmt.Sprintf("<p class=\"text-rose-700\">%v</p>", w.Flood))
	}
}

func (n *NasaData) Stats() {
	sum := Nasa{}

	n.Max.WindSpeed = n.Items[0].WindSpeed
	n.Max.RelHumidity = n.Items[0].RelHumidity
	n.Max.Precipitation = n.Items[0].Precipitation
	n.Max.TempAverage = n.Items[0].TempAverage
	n.Max.TempMax = n.Items[0].TempMax
	n.Max.TempMin = n.Items[0].TempMin

	n.Min.WindSpeed = n.Items[0].WindSpeed
	n.Min.RelHumidity = n.Items[0].RelHumidity
	n.Min.Precipitation = n.Items[0].Precipitation
	n.Min.TempAverage = n.Items[0].TempAverage
	n.Min.TempMax = n.Items[0].TempMax
	n.Min.TempMin = n.Items[0].TempMin

	for _, d := range n.Items {
		if n.Max.WindSpeed < d.WindSpeed {
			n.Max.WindSpeed = d.WindSpeed
		}
		if n.Max.RelHumidity < d.RelHumidity {
			n.Max.RelHumidity = d.RelHumidity
		}
		if n.Max.Precipitation < d.Precipitation {
			n.Max.Precipitation = d.Precipitation
		}
		if n.Max.TempAverage < d.TempAverage {
			n.Max.TempAverage = d.TempAverage
		}
		if n.Max.TempMax < d.TempMax {
			n.Max.TempMax = d.TempMax
		}
		if n.Max.TempMin < d.TempMin {
			n.Max.TempMin = d.TempMin
		}

		if n.Min.WindSpeed > d.WindSpeed {
			if d.WindSpeed != -999.0 {
				n.Min.WindSpeed = d.WindSpeed
			}
		}
		if n.Min.RelHumidity > d.RelHumidity {
			if d.RelHumidity != -999.0 {
				n.Min.RelHumidity = d.RelHumidity
			}
		}
		if n.Min.Precipitation > d.Precipitation {
			if d.Precipitation != -999.0 {
				n.Min.Precipitation = d.Precipitation
			}
		}
		if n.Min.TempAverage > d.TempAverage {
			if d.TempAverage != -999.0 {
				n.Min.TempAverage = d.TempAverage
			}
		}
		if n.Min.TempMax > d.TempMax {
			if d.TempMax != -999.0 {
				n.Min.TempMax = d.TempMax
			}
		}
		if n.Min.TempMin > d.TempMin {
			if d.TempMin != -999.0 {
				n.Min.TempMin = d.TempMin
			}
		}

		sum.WindSpeed += d.WindSpeed
		sum.RelHumidity += d.RelHumidity
		sum.Precipitation += d.Precipitation
		sum.TempAverage += d.TempAverage
		sum.TempMax += d.TempMax
		sum.TempMin += d.TempMin
	}

	n.Mean.WindSpeed = sum.WindSpeed / float64(len(n.Items))
	n.Mean.RelHumidity = sum.RelHumidity / float64(len(n.Items))
	n.Mean.Precipitation = sum.Precipitation / float64(len(n.Items))
	n.Mean.TempAverage = sum.TempAverage / float64(len(n.Items))
	n.Mean.TempMax = sum.TempMax / float64(len(n.Items))
	n.Mean.TempMin = sum.TempMin / float64(len(n.Items))

	sumSquares := Nasa{}
	for _, d := range n.Items {
		sumSquares.WindSpeed += math.Pow((d.WindSpeed - n.Mean.WindSpeed), 2)
		sumSquares.RelHumidity += math.Pow((d.RelHumidity - n.Mean.RelHumidity), 2)
		sumSquares.Precipitation += math.Pow((d.Precipitation - n.Mean.Precipitation), 2)
		sumSquares.TempAverage += math.Pow((d.TempAverage - n.Mean.TempAverage), 2)
		sumSquares.TempMax += math.Pow((d.TempMax - n.Mean.TempMax), 2)
		sumSquares.TempMin += math.Pow((d.TempMin - n.Mean.TempMin), 2)
	}

	n.Variance.WindSpeed = sumSquares.WindSpeed / float64(len(n.Items))
	n.Variance.RelHumidity = sumSquares.RelHumidity / float64(len(n.Items))
	n.Variance.Precipitation = sumSquares.Precipitation / float64(len(n.Items))
	n.Variance.TempAverage = sumSquares.TempAverage / float64(len(n.Items))
	n.Variance.TempMax = sumSquares.TempMax / float64(len(n.Items))
	n.Variance.TempMin = sumSquares.TempMin / float64(len(n.Items))

	n.StdDev.WindSpeed = math.Sqrt(n.Variance.WindSpeed)
	n.StdDev.RelHumidity = math.Sqrt(n.Variance.RelHumidity)
	n.StdDev.Precipitation = math.Sqrt(n.Variance.Precipitation)
	n.StdDev.TempAverage = math.Sqrt(n.Variance.TempAverage)
	n.StdDev.TempMax = math.Sqrt(n.Variance.TempMax)
	n.StdDev.TempMin = math.Sqrt(n.Variance.TempMin)

	n.Max.FillString()
	n.Min.FillString()
	n.Mean.FillString()
	n.Variance.FillString()
	n.StdDev.FillString()

	n.Max.DateStr = "MAX"
	n.Min.DateStr = "MIN"
	n.Mean.DateStr = "MEAN"
	n.Variance.DateStr = "VARIANCE"
	n.StdDev.DateStr = "STDDEV"
}

func (n *Nasa) FillString() {
	n.WindSpeedStr = strconv.FormatFloat(n.WindSpeed, 'f', 3, 64)
	n.RelHumidityStr = strconv.FormatFloat(n.RelHumidity, 'f', 3, 64)
	n.PrecipitationStr = strconv.FormatFloat(n.Precipitation, 'f', 3, 64)
	n.TempAverageStr = strconv.FormatFloat(n.TempAverage, 'f', 3, 64)
	n.TempMaxStr = strconv.FormatFloat(n.TempMax, 'f', 3, 64)
	n.TempMinStr = strconv.FormatFloat(n.TempMin, 'f', 3, 64)
}

func (n *ConfusionMatrix) Metrics() {
	accNumerator := float64(n.TruePositive + n.TrueNegative)
	accDenominator := float64(n.TruePositive + n.TrueNegative + n.FalsePositive + n.FalseNegative)
	if accNumerator == 0 || accDenominator == 0 {
		n.Accuracy = 0
	} else {
		n.Accuracy = accNumerator / accDenominator
	}

	precNumerator := float64(n.TruePositive)
	precDenominator := float64(n.TruePositive + n.FalsePositive)
	if precNumerator == 0 || precDenominator == 0 {
		n.Precision = 0
	} else {
		n.Precision = precNumerator / precDenominator
	}

	recNumerator := float64(n.TruePositive)
	recDenominator := float64(n.TruePositive + n.FalseNegative)
	if recNumerator == 0 || recDenominator == 0 {
		n.Recall = 0
	} else {
		n.Recall = recNumerator / recDenominator
	}

	f1Numerator := (n.Precision * n.Recall)
	f1Denominator := (n.Precision + n.Recall)
	if f1Numerator == 0 || f1Denominator == 0 {
		n.F1Score = 0
	} else {
		n.F1Score = 2 * (f1Numerator / f1Denominator)
	}
}

func (n *ConfusionMatrix) FillString() {
	n.TruePositiveStr = strconv.Itoa(n.TruePositive)
	n.TrueNegativeStr = strconv.Itoa(n.TrueNegative)
	n.FalsePositiveStr = strconv.Itoa(n.FalsePositive)
	n.FalseNegativeStr = strconv.Itoa(n.FalseNegative)
	n.AccuracyStr = strconv.FormatFloat(n.Accuracy, 'f', 4, 64)
	n.PrecisionStr = strconv.FormatFloat(n.Precision, 'f', 4, 64)
	n.RecallStr = strconv.FormatFloat(n.Recall, 'f', 4, 64)
	n.F1ScoreStr = strconv.FormatFloat(n.F1Score, 'f', 4, 64)
}

func (s *Statistics) FillStatistics(startDate, endDate time.Time, city string) {
	s.City = strings.ToUpper(city)
	s.StartDate = startDate.Format("01/02/2006")
	s.EndDate = endDate.Format("01/02/2006")
	s.Nasa.DataCount = strconv.Itoa(len(s.Ref.Nasa.Items))
	s.Nasa.DayCount = strconv.Itoa(int(endDate.Sub(startDate).Hours()/24) + 1)
	s.Bnpb.FloodCount = strconv.Itoa(len(s.Ref.Bnpb.Items))
	s.News.FloodCount = strconv.Itoa(len(s.Ref.News.Items))
	s.Weathers.DataCount = strconv.Itoa(len(s.Ref.Weathers.Items))
	s.DifferencedWeathers.DataCount = strconv.Itoa(len(s.Ref.DifferencedWeathers.Items))
	s.Smote.DataCount = strconv.Itoa(len(s.Ref.Smote.SynthItems))
	s.Smote.OversampledCount = strconv.Itoa(s.Ref.Smote.Oversample.SynthDataCount)

	var weathersFloodCount, differencedWeathersFloodCount, smoteFloodCount int
	for _, d := range s.Ref.Weathers.Items {
		if d.Flood {
			weathersFloodCount++
		}
	}
	for _, d := range s.Ref.DifferencedWeathers.Items {
		if d.Flood {
			differencedWeathersFloodCount++
		}
	}
	for _, d := range s.Ref.Smote.SynthItems {
		if d.Flood {
			smoteFloodCount++
		}
	}
	var weathersFloodPercentage, differencedWeathersFloodPercentage, smoteFloodPercentage float64
	weathersFloodPercentage = float64(weathersFloodCount) / float64(len(s.Ref.Weathers.Items))
	differencedWeathersFloodPercentage = float64(differencedWeathersFloodCount) / float64(len(s.Ref.DifferencedWeathers.Items))
	smoteFloodPercentage = float64(smoteFloodCount) / float64(len(s.Ref.Smote.Items))

	s.Weathers.FloodCount = strconv.Itoa(weathersFloodCount)
	s.Weathers.FloodPercentage = strconv.FormatFloat(weathersFloodPercentage*100, 'f', 2, 64) + "%"
	s.DifferencedWeathers.FloodCount = strconv.Itoa(differencedWeathersFloodCount)
	s.DifferencedWeathers.FloodPercentage = strconv.FormatFloat(differencedWeathersFloodPercentage*100, 'f', 2, 64) + "%"
	s.Smote.FloodCount = strconv.Itoa(smoteFloodCount)
	s.Smote.FloodPercentage = strconv.FormatFloat(smoteFloodPercentage*100, 'f', 1, 64) + "%"

	s.FillMap()
}

func (s *Statistics) FillMap() {
	s.NasaMap = []KeyValue{
		{Key: "City", Value: s.City},
		{Key: "Start Date", Value: s.StartDate},
		{Key: "End Date", Value: s.EndDate},
		{Key: "Day Count", Value: s.Nasa.DayCount},
		{Key: "Data Count", Value: s.Nasa.DataCount},
	}

	s.BnpbMap = []KeyValue{
		{Key: "City", Value: s.City},
		{Key: "Start Date", Value: s.StartDate},
		{Key: "End Date", Value: s.EndDate},
		{Key: "Flood Count", Value: s.Bnpb.FloodCount},
	}

	s.NewsMap = []KeyValue{
		{Key: "City", Value: s.City},
		{Key: "Start Date", Value: s.StartDate},
		{Key: "End Date", Value: s.EndDate},
		{Key: "Flood Count", Value: s.News.FloodCount},
	}

	s.WeathersMap = []KeyValue{
		{Key: "City", Value: s.City},
		{Key: "Start Date", Value: s.StartDate},
		{Key: "End Date", Value: s.EndDate},
		{Key: "Data Count", Value: s.Weathers.DataCount},
		{Key: "Flood Count", Value: s.Weathers.FloodCount},
		{Key: "Flood Percentage", Value: s.Weathers.FloodPercentage},
	}

	s.DifferencedWeathersMap = []KeyValue{
		{Key: "City", Value: s.City},
		{Key: "Start Date", Value: s.StartDate},
		{Key: "End Date", Value: s.EndDate},
		{Key: "Data Count", Value: s.DifferencedWeathers.DataCount},
		{Key: "Flood Count", Value: s.DifferencedWeathers.FloodCount},
		{Key: "Flood Percentage", Value: s.DifferencedWeathers.FloodPercentage},
	}

	s.SmoteMap = []KeyValue{
		{Key: "City", Value: s.City},
		{Key: "Start Date", Value: s.StartDate},
		{Key: "End Date", Value: s.EndDate},
		{Key: "Data Count", Value: s.Smote.DataCount},
		{Key: "Oversampled Data Count", Value: s.Smote.OversampledCount},
		{Key: "Flood Count", Value: s.Smote.FloodCount},
		{Key: "Flood Percentage", Value: s.Smote.FloodPercentage},
	}
}

func adfCriticalValue(dataLength, degreeOfSignificance int) (criticalValue float64) {
	coefficients := map[int][]float64{
		1:  {-3.43035, -6.5393, -16.786, -79.433},
		5:  {-2.86154, -2.8903, -4.234, -40.040},
		10: {-2.56677, -1.5384, -2.809, -31.223},
	}

	chosenCoefficients := coefficients[degreeOfSignificance]
	criticalValue = chosenCoefficients[0] + chosenCoefficients[1]/float64(dataLength) + chosenCoefficients[2]/float64(dataLength) + chosenCoefficients[3]/float64(dataLength)
	return
}

func adfTest(input []float64) (isStationary bool, criticalValue float64, gamma float64, err error) {
	criticalValue = adfCriticalValue(len(input), 5)
	var responseVectorAsSlice, designMatrixAsSlice []float64
	for i := 0; i < len(input)-2; i++ {
		responseVectorAsSlice = append(responseVectorAsSlice, input[i+2]-input[i+1])
		designMatrixAsSlice = append(designMatrixAsSlice, 1)
		designMatrixAsSlice = append(designMatrixAsSlice, input[i+1])
		designMatrixAsSlice = append(designMatrixAsSlice, input[i+1]-input[i])
	}
	designMatrixRows, designMatrixCols := len(input)-2, 3

	designMatrix := mat.NewDense(designMatrixRows, designMatrixCols, designMatrixAsSlice)
	responseVector := mat.NewDense(designMatrixRows, 1, responseVectorAsSlice)
	transposedDesignMatrix := designMatrix.T()
	xtxMatrix := mat.NewDense(designMatrixCols, designMatrixCols, nil)
	xtxMatrix.Mul(transposedDesignMatrix, designMatrix)
	xtyMatrix := mat.NewDense(3, 1, nil)
	xtyMatrix.Mul(transposedDesignMatrix, responseVector)

	var inverseXtxMatrix mat.Dense
	err = inverseXtxMatrix.Inverse(xtxMatrix)
	if err != nil {
		return false, 0.0, 0.0, err
	}

	olsResult := mat.NewDense(3, 1, nil)
	olsResult.Mul(&inverseXtxMatrix, xtyMatrix)
	olsData := olsResult.RawMatrix().Data
	return criticalValue > olsData[1], criticalValue, olsData[1], nil
}

func difference(input []float64) (output []float64) {
	if len(input) < 2 {
		return []float64{}
	}

	output = make([]float64, len(input)-1)

	for i := 0; i < len(input)-1; i++ {
		output[i] = input[i+1] - input[i]
	}

	return
}

func flatten(input [][]float64) (output []float64) {
	flat := make([]float64, 0, len(input)*len(input[0]))
	for _, row := range input {
		flat = append(flat, row...)
	}
	return flat
}

func isSame(a, b Weather) bool {
	if a.WindSpeed == b.WindSpeed &&
		a.RelHumidity == b.RelHumidity &&
		a.Precipitation == b.Precipitation &&
		a.TempAverage == b.TempAverage &&
		a.TempMax == b.TempMax &&
		a.TempMin == b.TempMin {
		return true
	}
	return false
}

// c is between a and b
func inBetween(a, b, c float64) bool {
	if a < c && b > c {
		return true
	} else if b < c && a > c {
		return true
	}
	return false
}

func getMean(s []float64) float64 {
	var sum float64
	for _, i := range s {
		sum += i
	}
	return sum / float64(len(s))
}

func structToJsonFile(v interface{}, filename string) error {
	jsonData, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(filename, jsonData, 0644)
	if err != nil {
		return err
	}

	return nil
}
