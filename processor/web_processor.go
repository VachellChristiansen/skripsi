package processor

import (
	"bufio"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"skripsi/constant"
	"skripsi/helper"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

type WebProcessor interface {
	HandleFloodPredictionRequest(c echo.Context) error
}

type WebProcessorImpl struct {
	logger helper.LoggerHelper
}

func NewWebProcessor(l helper.LoggerHelper) WebProcessor {
	return &WebProcessorImpl{
		logger: l,
	}
}

func (p *WebProcessorImpl) HandleFloodPredictionRequest(c echo.Context) error {
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
	startDate, err := time.Parse("2006-01-02", c.FormValue("start_date"))
	endDate, err := time.Parse("2006-01-02", c.FormValue("end_date"))
	city := c.FormValue("city")
	if err != nil {
		return c.Render(http.StatusOK, "main", IndexData{
			Err:        "Parsing Date Error",
			StatusCode: http.StatusBadRequest,
			Timestamp:  time.Now().Unix(),
		})
	}

	if startDate.After(endDate) {
		return c.Render(http.StatusOK, "main", IndexData{
			Err:        "Start Date can't be later than End Date",
			StatusCode: http.StatusUnprocessableEntity,
		})
	}

	startDateRequest := strings.ReplaceAll(c.FormValue("start_date"), "-", "")
	endDateRequest := strings.ReplaceAll(c.FormValue("end_date"), "-", "")

	latlong, exists := cities[city]
	if !exists {
		return c.Render(http.StatusOK, "main", IndexData{
			Err:        "City is not available",
			StatusCode: http.StatusUnprocessableEntity,
		})
	}

	latitude := strings.Split(latlong, "&")[0]
	longitude := strings.Split(latlong, "&")[1]

	url := fmt.Sprintf("%s?start=%s&end=%s&latitude=%s&longitude=%s&%s", constant.NasaPowerAPIBaseURL, startDateRequest, endDateRequest, latitude, longitude, constant.NasaPowerAPIParams)
	err = p.PrepareNasaCSV(url)
	if err != nil {
		return c.Render(http.StatusOK, "main", IndexData{
			Err:        fmt.Sprintf("Preparing NASA data fails, %s", err.Error()),
			StatusCode: http.StatusInternalServerError,
		})
	}

	nasaData := [][]string{}
	err = p.PreprocessNasaCSV(&nasaData)
	if err != nil {
		return c.Render(http.StatusOK, "main", IndexData{
			Err:        fmt.Sprintf("Preprocessing NASA data fails, %s", err.Error()),
			StatusCode: http.StatusInternalServerError,
		})
	}

	bnpbData := [][]string{}
	bnpbDataOri := [][]string{}
	err = p.PreprocessBNPBCSV(&bnpbData, &bnpbDataOri, startDate, endDate, city)
	if err != nil {
		return c.Render(http.StatusOK, "main", IndexData{
			Err:        fmt.Sprintf("Preprocessing BNPB data fails, %s", err.Error()),
			StatusCode: http.StatusInternalServerError,
		})
	}

	statisticData := []map[string]interface{}{}
	err = p.PrepareStatistics(&bnpbData, &nasaData, startDate, endDate, city, &statisticData)
	if err != nil {
		return c.Render(http.StatusOK, "main", IndexData{
			Err:        fmt.Sprintf("Processing Statistic, %s", err.Error()),
			StatusCode: http.StatusInternalServerError,
		})
	}

	return c.Render(http.StatusOK, "main", IndexData{
		Data: map[string]interface{}{
			"NasaHeaders":    nasaData[0],
			"NasaStat":       nasaData[1:6],
			"NasaValues":     nasaData[6:],
			"BnpbHeaders":    bnpbData[0],
			"BnpbValues":     bnpbData[1:],
			"BnpbHeadersOri": bnpbDataOri[0],
			"BnpbValuesOri":  bnpbDataOri[1:],
			"StatisticData":  statisticData,
		},
		Message:    fmt.Sprintf("Preparation Done. Time Taken: %dms", time.Since(start).Milliseconds()),
		StatusCode: http.StatusOK,
	})
}

func (p *WebProcessorImpl) PrepareNasaCSV(url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return errors.New("Fetching data from Nasa Power API fails")
	}
	defer resp.Body.Close()

	wd, err := os.Getwd()
	if err != nil {
		return errors.New("Get working directory fails Prepare NASA")
	}

	tempFile, err := os.Create(filepath.Join(wd, "tmp/nasa_data.txt"))
	if err != nil {
		return errors.New("Creating temporary file fails")
	}

	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		return errors.New("Saving data to temporary file fails")
	}
	tempFile.Close()

	tempFile, err = os.Open(filepath.Join(wd, "tmp/nasa_data.txt"))
	if err != nil {
		return errors.New("Opening temporary file fails")
	}
	defer tempFile.Close()

	csvFile, err := os.Create(filepath.Join(wd, "tmp/nasa_data.csv"))
	if err != nil {
		return errors.New("Create file fails")
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
				return errors.New("Writing to CSV fails")
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return errors.New("Scanning temporary file fails")
	}

	return nil
}

func (p *WebProcessorImpl) PreprocessNasaCSV(nasaData *[][]string) error {
	wd, err := os.Getwd()
	if err != nil {
		return errors.New("Get working directory fails Preprocess NASA")
	}

	csvFile, err := os.Open(filepath.Join(wd, "tmp/nasa_data.csv"))
	if err != nil {
		return errors.New("Opening data csv file fails")
	}
	defer csvFile.Close()

	reader := csv.NewReader(csvFile)
	records, err := reader.ReadAll()
	if err != nil {
		return errors.New("Reading data csv file fails")
	}

	headers := records[0][2:]
	records = records[1:]
	headersWithDate := append([]string{"DATE"}, headers...)

	indexMap := make(map[string]int)
	for i, header := range headers {
		indexMap[header] = i
	}

	totalSlice := []float64{0, 0, 0, 0, 0, 0}
	minSlice := []float64{999, 999, 999, 999, 999, 999}
	maxSlice := []float64{-999, -999, -999, -999, -999, -999}

	for _, record := range records {
		year, _ := strconv.Atoi(record[0])
		doy, _ := strconv.Atoi(record[1])

		startOfYear := time.Date(year, 1, 1, 0, 0, 0, 0, time.Local)
		date := startOfYear.AddDate(0, 0, doy-1)
		stringDate := date.Format("2006/01/02")

		dataValue := record[2:]
		ws10m, _ := strconv.ParseFloat(dataValue[indexMap["WS10M"]], 32)
		rh2m, _ := strconv.ParseFloat(dataValue[indexMap["RH2M"]], 32)
		prectotcorr, _ := strconv.ParseFloat(dataValue[indexMap["PRECTOTCORR"]], 32)
		t2m, _ := strconv.ParseFloat(dataValue[indexMap["T2M"]], 32)
		t2mMax, _ := strconv.ParseFloat(dataValue[indexMap["T2M_MAX"]], 32)
		t2mMin, _ := strconv.ParseFloat(dataValue[indexMap["T2M_MIN"]], 32)

		values := []float64{0, 0, 0, 0, 0, 0}
		values[indexMap["WS10M"]] = ws10m
		values[indexMap["RH2M"]] = rh2m
		values[indexMap["PRECTOTCORR"]] = prectotcorr
		values[indexMap["T2M"]] = t2m
		values[indexMap["T2M_MAX"]] = t2mMax
		values[indexMap["T2M_MIN"]] = t2mMin

		for _, i := range indexMap {
			totalSlice[i] += values[i]
			if minSlice[i] > values[i] {
				minSlice[i] = values[i]
			}
			if maxSlice[i] < values[i] {
				maxSlice[i] = values[i]
			}
		}

		recordData := append([]string{stringDate}, record[2:]...)
		*nasaData = append(*nasaData, recordData)
	}

	meanSlice := []float64{0, 0, 0, 0, 0, 0}

	for _, i := range indexMap {
		meanSlice[i] = totalSlice[i] / float64(len(records))
	}

	varianceSlice := []float64{0, 0, 0, 0, 0, 0}
	for _, record := range records {
		dataValue := record[2:]
		ws10m, _ := strconv.ParseFloat(dataValue[indexMap["WS10M"]], 32)
		rh2m, _ := strconv.ParseFloat(dataValue[indexMap["RH2M"]], 32)
		prectotcorr, _ := strconv.ParseFloat(dataValue[indexMap["PRECTOTCORR"]], 32)
		t2m, _ := strconv.ParseFloat(dataValue[indexMap["T2M"]], 32)
		t2mMax, _ := strconv.ParseFloat(dataValue[indexMap["T2M_MAX"]], 32)
		t2mMin, _ := strconv.ParseFloat(dataValue[indexMap["T2M_MIN"]], 32)
		values := []float64{0, 0, 0, 0, 0, 0}
		values[indexMap["WS10M"]] = ws10m
		values[indexMap["RH2M"]] = rh2m
		values[indexMap["PRECTOTCORR"]] = prectotcorr
		values[indexMap["T2M"]] = t2m
		values[indexMap["T2M_MAX"]] = t2mMax
		values[indexMap["T2M_MIN"]] = t2mMin

		for _, i := range indexMap {
			varianceSlice[i] += math.Pow(values[i]-meanSlice[i], 2)
		}
	}

	stdDevSlice := []float64{0, 0, 0, 0, 0, 0}
	for _, i := range indexMap {
		varianceSlice[i] = varianceSlice[i] / float64(len(records))
		stdDevSlice[i] = math.Sqrt(varianceSlice[i])
	}

	meanSliceStr := []string{"", "", "", "", "", ""}
	minSliceStr := []string{"", "", "", "", "", ""}
	maxSliceStr := []string{"", "", "", "", "", ""}
	stdDevSliceStr := []string{"", "", "", "", "", ""}
	varianceSliceStr := []string{"", "", "", "", "", ""}
	for _, i := range indexMap {
		meanSliceStr[i] = strconv.FormatFloat(meanSlice[i], 'f', 2, 64)
		minSliceStr[i] = strconv.FormatFloat(minSlice[i], 'f', 2, 64)
		maxSliceStr[i] = strconv.FormatFloat(maxSlice[i], 'f', 2, 64)
		stdDevSliceStr[i] = strconv.FormatFloat(stdDevSlice[i], 'f', 2, 64)
		varianceSliceStr[i] = strconv.FormatFloat(varianceSlice[i], 'f', 2, 64)
	}
	meanSliceStr = append([]string{"MEAN"}, meanSliceStr...)
	minSliceStr = append([]string{"MIN"}, minSliceStr...)
	maxSliceStr = append([]string{"MAX"}, maxSliceStr...)
	stdDevSliceStr = append([]string{"STD DEV"}, stdDevSliceStr...)
	varianceSliceStr = append([]string{"VAR"}, varianceSliceStr...)

	*nasaData = append([][]string{headersWithDate, meanSliceStr, minSliceStr, maxSliceStr, stdDevSliceStr, varianceSliceStr}, *nasaData...)
	return nil
}

func (p *WebProcessorImpl) PreprocessBNPBCSV(bnpbData *[][]string, bnpbDataOri *[][]string, startDate, endDate time.Time, city string) error {
	wd, err := os.Getwd()
	if err != nil {
		return errors.New("Get working directory fails Preprocess BNPB")
	}

	csvFile, err := os.Open(filepath.Join(wd, "tmp/bnpb_data.csv"))
	if err != nil {
		return errors.New("Opening data csv file fails")
	}
	defer csvFile.Close()

	reader := csv.NewReader(csvFile)
	records, err := reader.ReadAll()
	if err != nil {
		return errors.New("Reading data csv file fails")
	}

	headers := []string{"CITY", "FLOOD DATE"}
	headersOri := records[1]
	records = records[2:]

	data := make(map[string][]string)
	dataOri := make(map[string][][]string)
	// 3, Tanggal. 6, Kapbupaten
	for _, record := range records {
		dateSlice := strings.Split(record[3], "/")
		date := strings.Join([]string{dateSlice[2], dateSlice[1], dateSlice[0]}, "/")
		year, _ := strconv.Atoi(dateSlice[2])
		month, _ := strconv.Atoi(dateSlice[1])
		day, _ := strconv.Atoi(dateSlice[0])
		dateValue := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)

		if dateValue.Before(startDate) || dateValue.After(endDate) {
			continue
		}
		if _, exists := data[record[6]]; !exists {
			data[record[6]] = []string{date}
			dataOri[record[6]] = [][]string{record}
		} else {
			data[record[6]] = append(data[record[6]], date)
			dataOri[record[6]] = append(dataOri[record[6]], record)
		}
	}

	mergedCityData := [][]string{}
	dateMap := make(map[string]bool)
	for cityName, dates := range data {
		cityLoweredCase := strings.ToLower(cityName)
		if strings.Contains(cityLoweredCase, city) {
			for _, date := range dates {
				if _, exists := dateMap[date]; !exists {

					mergedCityData = append(mergedCityData, []string{strings.ToUpper(city), date})
				} else {
					dateMap[date] = true
				}
			}
		}
	}
	mergedDataOri := [][]string{}
	for cityName, data := range dataOri {
		cityLoweredCase := strings.ToLower(cityName)
		if strings.Contains(cityLoweredCase, city) {
			for _, items := range data {
				mergedDataOri = append(mergedDataOri, items)
			}
		}
	}

	*bnpbData = append([][]string{headers}, mergedCityData...)
	*bnpbDataOri = append([][]string{headersOri}, mergedDataOri...)

	return nil
}

func (p *WebProcessorImpl) PrepareStatistics(bnpbData, nasaData *[][]string, startDate, endDate time.Time, city string, statisticData *[]map[string]interface{}) error {
	stats := []map[string]interface{}{}
	stats = append(stats, map[string]interface{}{"StartDate": startDate.Format("2006/01/02")})
	stats = append(stats, map[string]interface{}{"EndDate": endDate.Format("2006/01/02")})
	stats = append(stats, map[string]interface{}{"City": strings.ToUpper(city)})
	stats = append(stats, map[string]interface{}{"DayCount": int(endDate.Sub(startDate).Hours()/24) + 1})
	stats = append(stats, map[string]interface{}{"DataCount": len(*nasaData) - 6})
	stats = append(stats, map[string]interface{}{"FloodCount": len(*bnpbData) - 1})
	stats = append(stats, map[string]interface{}{"FloodPercentage": float64((float64(len(*bnpbData)) - 1) / (float64(len(*nasaData)) - 1) * 100)})

	*statisticData = stats
	return nil
}
