package processor

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"skripsi/constant"
	"skripsi/helper"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"gonum.org/v1/gonum/mat"
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

	kValue, err := strconv.Atoi(c.FormValue("k_value"))
	if err != nil {
		return c.Render(http.StatusOK, "main", IndexData{
			Err:        "K Value is not a valid number",
			StatusCode: http.StatusUnprocessableEntity,
		})
	}

	if kValue <= 0 || kValue > 500 {
		return c.Render(http.StatusOK, "main", IndexData{
			Err:        "Chosen K Value is not Valid (Must be 1 - 500)",
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

	nasaData := [][]float64{}
	nasaDataStr := [][]string{}
	err = p.PreprocessNasaCSV(&nasaDataStr, &nasaData)
	if err != nil {
		return c.Render(http.StatusOK, "main", IndexData{
			Err:        fmt.Sprintf("Preprocessing NASA data fails, %s", err.Error()),
			StatusCode: http.StatusInternalServerError,
		})
	}

	bnpbData := [][]string{}
	bnpbDataOri := [][]string{}
	floodData := []float64{}
	err = p.PreprocessBNPBCSV(&bnpbData, &bnpbDataOri, &floodData, startDate, endDate, city)
	if err != nil {
		return c.Render(http.StatusOK, "main", IndexData{
			Err:        fmt.Sprintf("Preprocessing BNPB data fails, %s", err.Error()),
			StatusCode: http.StatusInternalServerError,
		})
	}

	statisticData := []map[string]interface{}{}
	err = p.PrepareStatistics(&bnpbData, &nasaDataStr, startDate, endDate, city, &statisticData)
	if err != nil {
		return c.Render(http.StatusOK, "main", IndexData{
			Err:        fmt.Sprintf("Processing Statistic fails, %s", err.Error()),
			StatusCode: http.StatusInternalServerError,
		})
	}

	nasaWithFloodDataStr := p.MergeNASAWithFlood(nasaDataStr, floodData)

	var stationaryNasaData [][]float64
	stationaryDataMinLength := 99999
	maxDifferencingStep := -99999
	for i := 0; i < len(nasaData); i++ {
		stationary, err := p.adfTest(nasaData[i])
		if err != nil {
			return c.Render(http.StatusOK, "main", IndexData{
				Err:        fmt.Sprintf("Processing ADF test fails, %s", err.Error()),
				StatusCode: http.StatusInternalServerError,
			})
		}

		differencingStep := 0
		differencedNasaDataColumn := nasaData[i]
		for !stationary {
			differencingStep++
			differencedNasaDataColumn = p.differencing(differencedNasaDataColumn)
			stationary, err = p.adfTest(differencedNasaDataColumn)
			if err != nil {
				return c.Render(http.StatusOK, "main", IndexData{
					Err:        fmt.Sprintf("Processing ADF test fails, %s", err.Error()),
					StatusCode: http.StatusInternalServerError,
				})
			}
		}

		if maxDifferencingStep < differencingStep {
			maxDifferencingStep = differencingStep
		}
		if stationaryDataMinLength > len(differencedNasaDataColumn) {
			stationaryDataMinLength = len(differencedNasaDataColumn)
		}
		stationaryNasaData = append(stationaryNasaData, differencedNasaDataColumn)
	}

	var stationaryNasaWithFloodData [][]float64
	for _, data := range stationaryNasaData {
		differencedData := data
		for j := 0; j < len(data)-stationaryDataMinLength; j++ {
			differencedData = p.differencing(data)
		}
		stationaryNasaWithFloodData = append(stationaryNasaWithFloodData, differencedData)
	}
	stationaryNasaWithFloodData = append(stationaryNasaWithFloodData, floodData[len(floodData)-stationaryDataMinLength:len(floodData)])

	predictedValues, err := p.vectorAutoregression(stationaryNasaWithFloodData)
	if err != nil {
		return c.Render(http.StatusOK, "main", IndexData{
			Err:        fmt.Sprintf("Processing VAR Autoregression fails, %s", err.Error()),
			StatusCode: http.StatusInternalServerError,
		})
	}

	_, nrmseResult := p.evaluateVectorAutoregressionWithNRMSE(stationaryNasaWithFloodData, 1.0, 6.0)

	var predictedValuesStr []string
	for i, predictedValue := range predictedValues {
		predictedValuesStr = append(predictedValuesStr, fmt.Sprintf("%s: %0.4f", nasaDataStr[0][i+1], predictedValue))
	}

	knnResult, _ := p.knnClassification(predictedValues, stationaryNasaWithFloodData, kValue)
	flood := "No Flood"
	if knnResult == 0 {
		flood = "Flood"
	} else {
		flood = "No Flood"
	}

	p.logger.LogAndContinue("Done Processing Request")
	viewData := map[string]interface{}{
		"NasaHeaders":            nasaDataStr[0],
		"NasaStat":               nasaDataStr[1:6],
		"NasaValues":             nasaDataStr[6:],
		"NasaFloodHeaders":       append(nasaDataStr[0], "FLOOD"),
		"NasaFloodValues":        nasaWithFloodDataStr,
		"BnpbHeaders":            bnpbData[0],
		"BnpbValues":             bnpbData[1:],
		"BnpbHeadersOri":         bnpbDataOri[0],
		"BnpbValuesOri":          bnpbDataOri[1:],
		"NRMSEEvaluationHeaders": nrmseResult[0],
		"NRMSEEvaluationValues":  nrmseResult[1:],
		"StatisticData":          statisticData,
		"StartDate":              startDate.Format("2006/01/02"),
		"EndDate":                endDate.Format("2006/01/02"),
		"Latitude":               latitude,
		"Longitude":              longitude,
		"DifferencingStep":       strconv.Itoa(maxDifferencingStep),
		"PredictedValues":        predictedValuesStr,
		"KNNResult":              flood,
		"Timestamp":              time.Now().Unix(),
	}
	jsData, err := json.Marshal(viewData)
	if err != nil {
		return c.Render(http.StatusOK, "main", IndexData{
			Err:        fmt.Sprintf("Marshaling data into json fails, %s", err.Error()),
			StatusCode: http.StatusInternalServerError,
		})
	}

	return c.Render(http.StatusOK, "main", IndexData{
		Data:       viewData,
		JSData:     string(jsData),
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

func (p *WebProcessorImpl) PreprocessNasaCSV(nasaDataStr *[][]string, nasaData *[][]float64) error {
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
	headersWithDate := []string{"DATE", "WS10M", "RH2M", "PRECTOTCORR", "T2M", "T2M_MAX", "T2M_MIN"}

	indexMap := make(map[string]int)
	for i, header := range headers {
		indexMap[header] = i
	}

	totalSlice := []float64{0, 0, 0, 0, 0, 0}

	tempData := make([][]float64, 6)
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

		values := []float64{ws10m, rh2m, prectotcorr, t2m, t2mMax, t2mMin}
		valuesStr := []string{dataValue[indexMap["WS10M"]], dataValue[indexMap["RH2M"]], dataValue[indexMap["PRECTOTCORR"]], dataValue[indexMap["T2M"]], dataValue[indexMap["T2M_MAX"]], dataValue[indexMap["T2M_MIN"]]}

		for i, value := range values {
			totalSlice[i] += value
			tempData[i] = append(tempData[i], value)
		}

		recordData := append([]string{stringDate}, valuesStr...)
		*nasaDataStr = append(*nasaDataStr, recordData)
	}
	*nasaData = tempData

	maxSlice := []float64{}
	minSlice := []float64{}
	for _, item := range tempData {
		maxSlice = append(maxSlice, p.getMax(item))
		minSlice = append(minSlice, p.getMin(item))
	}

	meanSlice := []float64{0, 0, 0, 0, 0, 0}

	for i, total := range totalSlice {
		meanSlice[i] = total / float64(len(records))
	}

	varianceSlice := []float64{0, 0, 0, 0, 0, 0}
	stdDevSlice := []float64{0, 0, 0, 0, 0, 0}
	for _, record := range records {
		dataValue := record[2:]
		ws10m, _ := strconv.ParseFloat(dataValue[indexMap["WS10M"]], 32)
		rh2m, _ := strconv.ParseFloat(dataValue[indexMap["RH2M"]], 32)
		prectotcorr, _ := strconv.ParseFloat(dataValue[indexMap["PRECTOTCORR"]], 32)
		t2m, _ := strconv.ParseFloat(dataValue[indexMap["T2M"]], 32)
		t2mMax, _ := strconv.ParseFloat(dataValue[indexMap["T2M_MAX"]], 32)
		t2mMin, _ := strconv.ParseFloat(dataValue[indexMap["T2M_MIN"]], 32)
		values := []float64{ws10m, rh2m, prectotcorr, t2m, t2mMax, t2mMin}

		for i, value := range values {
			varianceSlice[i] += math.Pow(value-meanSlice[i], 2)
		}
	}
	for i, variance := range varianceSlice {
		varianceSlice[i] = variance / float64(len(records))
		stdDevSlice[i] = math.Sqrt(varianceSlice[i])
	}

	meanSliceStr := []string{"", "", "", "", "", ""}
	minSliceStr := []string{"", "", "", "", "", ""}
	maxSliceStr := []string{"", "", "", "", "", ""}
	stdDevSliceStr := []string{"", "", "", "", "", ""}
	varianceSliceStr := []string{"", "", "", "", "", ""}
	for i := 0; i < len(totalSlice); i++ {
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

	*nasaDataStr = append([][]string{headersWithDate, meanSliceStr, minSliceStr, maxSliceStr, stdDevSliceStr, varianceSliceStr}, *nasaDataStr...)
	return nil
}

func (p *WebProcessorImpl) PreprocessBNPBCSV(bnpbData, bnpbDataOri *[][]string, floodData *[]float64, startDate, endDate time.Time, city string) error {
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

	dayCount := int(endDate.Sub(startDate).Hours()/24) + 1
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

	var flood []float64
	for i := 0; i < dayCount; i++ {
		curr := startDate.Add(time.Hour * 24 * time.Duration(i))
		currStr := curr.Format("2006/01/02")
		if _, exists := dateMap[currStr]; exists {
			flood = append(flood, 1)
		} else {
			flood = append(flood, 0)
		}
	}

	*floodData = flood
	*bnpbData = append([][]string{headers}, mergedCityData...)
	*bnpbDataOri = append([][]string{headersOri}, mergedDataOri...)

	return nil
}

func (p *WebProcessorImpl) MergeNASAWithFlood(nasaDataStr [][]string, floodData []float64) (result [][]string) {
	for i, nasaData := range nasaDataStr[6:] {
		floodStr := strconv.FormatFloat(floodData[i], 'f', 2, 64)
		nasaData = append(nasaData, floodStr)
		result = append(result, nasaData)
	}

	return
}

func (p *WebProcessorImpl) PrepareStatistics(bnpbData, nasaData *[][]string, startDate, endDate time.Time, city string, statisticData *[]map[string]interface{}) error {
	stats := []map[string]interface{}{}
	stats = append(stats, map[string]interface{}{"StartDate": startDate.Format("2006/01/02")})
	stats = append(stats, map[string]interface{}{"EndDate": endDate.Format("2006/01/02")})
	stats = append(stats, map[string]interface{}{"City": strings.ToUpper(city)})
	stats = append(stats, map[string]interface{}{"DayCount": int(endDate.Sub(startDate).Hours()/24) + 1})
	stats = append(stats, map[string]interface{}{"DataCount": len(*nasaData) - 6})
	stats = append(stats, map[string]interface{}{"FloodCount": len(*bnpbData) - 1})
	stats = append(stats, map[string]interface{}{"FloodPercentage": fmt.Sprintf("%0.3f%s", float64((float64(len(*bnpbData))-1)/(float64(len(*nasaData))-1)*100), "%")})

	*statisticData = stats
	return nil
}

func (p *WebProcessorImpl) getMax(data []float64) (max float64) {
	max = data[0]
	for _, i := range data {
		if max < i {
			max = i
		}
	}
	return
}

func (p *WebProcessorImpl) getMin(data []float64) (min float64) {
	min = data[0]
	for _, i := range data {
		if min > i {
			min = i
		}
	}
	return
}

func (p *WebProcessorImpl) adfCriticalValue(dataLength float64, significance int) (criticalValue float64) {
	coefficients := map[int][]float64{
		1:  {-3.43035, -6.5393, -16.786, -79.433},
		5:  {-2.86154, -2.8903, -4.234, -40.040},
		10: {-2.56677, -1.5384, -2.809, -31.223},
	}

	chosenCoefficients := coefficients[significance]
	criticalValue = chosenCoefficients[0] + chosenCoefficients[1]/dataLength + chosenCoefficients[2]/dataLength + chosenCoefficients[3]/dataLength
	return
}

func (p *WebProcessorImpl) adfTest(originalData []float64) (bool, error) {
	criticalValue := p.adfCriticalValue(float64(len(originalData)), 5)
	var responseVectorAsSlice, designMatrixAsSlice []float64
	for i := 0; i < len(originalData)-2; i++ {
		responseVectorAsSlice = append(responseVectorAsSlice, originalData[i+2]-originalData[i+1])
		designMatrixAsSlice = append(designMatrixAsSlice, 1)
		designMatrixAsSlice = append(designMatrixAsSlice, originalData[i+1])
		designMatrixAsSlice = append(designMatrixAsSlice, originalData[i+1]-originalData[i])
	}
	designMatrixRows, designMatrixCols := len(originalData)-2, 3

	designMatrix := mat.NewDense(designMatrixRows, designMatrixCols, designMatrixAsSlice)
	responseVector := mat.NewDense(designMatrixRows, 1, responseVectorAsSlice)
	transposedDesignMatrix := designMatrix.T()
	xtxMatrix := mat.NewDense(designMatrixCols, designMatrixCols, nil)
	xtxMatrix.Mul(transposedDesignMatrix, designMatrix)
	xtyMatrix := mat.NewDense(3, 1, nil)
	xtyMatrix.Mul(transposedDesignMatrix, responseVector)

	var inverseXtxMatrix mat.Dense
	err := inverseXtxMatrix.Inverse(xtxMatrix)
	if err != nil {
		return false, err
	}

	olsResult := mat.NewDense(3, 1, nil)
	olsResult.Mul(&inverseXtxMatrix, xtyMatrix)
	olsData := olsResult.RawMatrix().Data
	return criticalValue > olsData[1], nil
}

func (p *WebProcessorImpl) differencing(data []float64) (result []float64) {
	for i := 0; i < len(data)-1; i++ {
		result = append(result, data[i+1]-data[i])
	}

	return
}

func (p *WebProcessorImpl) vectorAutoregression(data [][]float64) ([]float64, error) {
	var responseVectorAsSlice, designMatrixAsSlice []float64
	for i := 0; i < len(data[0])-1; i++ {
		designMatrixAsSlice = append(designMatrixAsSlice, 1)
		for j := 0; j < len(data)-1; j++ {
			designMatrixAsSlice = append(designMatrixAsSlice, data[j][i])
			responseVectorAsSlice = append(responseVectorAsSlice, data[j][i+1])
		}
	}
	designMatrixRows, designMatrixCols := len(data[0])-1, len(data)

	designMatrix := mat.NewDense(designMatrixRows, designMatrixCols, designMatrixAsSlice)
	responseVector := mat.NewDense(designMatrixRows, designMatrixCols-1, responseVectorAsSlice)
	transposedDesignMatrix := designMatrix.T()
	xtxMatrix := mat.NewDense(designMatrixCols, designMatrixCols, nil)
	xtxMatrix.Mul(transposedDesignMatrix, designMatrix)
	xtyMatrix := mat.NewDense(len(data), len(data)-1, nil)
	xtyMatrix.Mul(transposedDesignMatrix, responseVector)

	var inverseXtxMatrix mat.Dense
	err := inverseXtxMatrix.Inverse(xtxMatrix)
	if err != nil {
		return nil, err
	}

	olsResult := mat.NewDense(len(data), len(data)-1, nil)
	olsResult.Mul(&inverseXtxMatrix, xtyMatrix)
	olsData := olsResult.RawMatrix().Data

	var lastRowData []float64
	for i := 0; i < len(data)-1; i++ {
		lastRowData = append(lastRowData, data[i][len(data[i])-1])
	}

	var predictedValues []float64
	for i := 0; i < len(data)-1; i++ {
		predictedValues = append(predictedValues, olsData[i])
		for j := 1; j < len(data); j++ {
			index := i + (j * (len(data) - 1))
			coefficient := olsData[index]
			predictedValues[i] += coefficient * lastRowData[j-1]
		}
	}

	return predictedValues, nil
}

func (p *WebProcessorImpl) knnClassification(dataPoints []float64, nasaData [][]float64, kValue int) (int, float64) {
	var distances []float64
	for i := 0; i < len(nasaData[0]); i++ {
		var distance float64
		for j := 0; j < len(nasaData)-1; j++ {
			distance += math.Pow((nasaData[j][i] - dataPoints[j]), 2)
		}
		distance = math.Sqrt(distance)
		distances = append(distances, distance)
	}

	indices := make([]int, len(distances))
	for i := range indices {
		indices[i] = i
	}

	sort.Slice(indices, func(i, j int) bool {
		return distances[indices[i]] < distances[indices[j]]
	})

	sortedDistances := make([]float64, len(distances))
	sortedFlood := make([]float64, len(nasaData[6]))

	for i, idx := range indices {
		sortedDistances[i] = distances[idx]
		sortedFlood[i] = nasaData[6][idx]
	}

	var kScore float64
	for i := 0; i < kValue; i++ {
		kScore += sortedFlood[i]
	}

	result := 0
	if kScore/float64(kValue) > 0.5 {
		result = 1
	}

	return result, kScore / float64(kValue)
}

func (p *WebProcessorImpl) evaluateVectorAutoregressionWithNRMSE(data [][]float64, start, stop float64) (result [][]float64, resultStr [][]string) {
	var lengthSplit []int
	for i := start; i < stop+1; i++ {
		split := math.Floor(float64(len(data[0])) * ((100 - i*5) / 100))
		lengthSplit = append(lengthSplit, int(split))
	}

	var dataRange []float64
	for i := 0; i < len(data)-1; i++ {
		dataRange = append(dataRange, p.getMax(data[i])-p.getMin(data[i]))
	}

	// Rolling VAR
	for _, split := range lengthSplit {
		mse := make([]float64, 6)
		for j := split; j < len(data[0]); j++ {
			var varData [][]float64
			for k := 0; k < len(data); k++ {
				varData = append(varData, data[k][:j])
			}

			varResult, _ := p.vectorAutoregression(varData)
			for k := 0; k < len(varResult); k++ {
				mse[k] += math.Pow(varResult[k]-data[k][j], 2)
			}
		}

		for j := 0; j < len(mse); j++ {
			mse[j] = math.Sqrt(mse[j]/float64(split)) / dataRange[j]
		}
		result = append(result, mse)
	}

	resultStr = append(resultStr, []string{"TRAIN-TEST-SPLIT", "WS10M", "RH2M", "PRECTOTCORR", "T2M", "T2M_MAX", "T2M_MIN"})

	for i := 0; i < len(lengthSplit); i++ {
		testLength := 5 * (i + 1)
		trainLength := 100 - testLength
		trainTest := fmt.Sprintf("%d-%d", trainLength, testLength)
		strSlice := oneDimFloatToOneDimString(result[i])
		resultStr = append(resultStr, []string{trainTest, strSlice[0], strSlice[1], strSlice[2], strSlice[3], strSlice[4], strSlice[5]})
	}
	return
}

func matPrint(X mat.Matrix) {
	fa := mat.Formatted(X, mat.Prefix(""), mat.Squeeze())
	fmt.Printf("%v\n", fa)
}

func oneDimFloatToOneDimString(input []float64) (output []string) {
	for i := 0; i < len(input); i++ {
		output = append(output, fmt.Sprintf("%0.5f", input[i]))
	}
	return
}

func twoDimFloatToTwoDimString(input [][]float64) (output [][]string) {
	for i := 0; i < len(input); i++ {
		var tempStrSlice []string
		for j := 0; j < len(input[i]); j++ {
			tempStrSlice = append(tempStrSlice, fmt.Sprintf("%0.5f", input[i][j]))
		}
		output = append(output, tempStrSlice)
	}
	return
}
