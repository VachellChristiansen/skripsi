package processor

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
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
	"golang.org/x/exp/rand"
	"gonum.org/v1/gonum/mat"
)

type WebProcessor interface {
	HandleFloodPredictionRequestV2(c echo.Context) error
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

const (
	DateHyphenYMD = "2006-01-02"
)

func (p *WebProcessorImpl) HandleFloodPredictionRequestV2(c echo.Context) error {
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
		return c.Render(http.StatusOK, "main", IndexData{
			Err:        "Parsing Date Error",
			StatusCode: http.StatusBadRequest,
			Timestamp:  time.Now().Unix(),
		})
	}
	endDate, err := time.Parse(DateHyphenYMD, c.FormValue("end_date"))
	if err != nil {
		return c.Render(http.StatusOK, "main", IndexData{
			Err:        "Parsing Date Error",
			StatusCode: http.StatusBadRequest,
			Timestamp:  time.Now().Unix(),
		})
	}

	city := c.FormValue("city")
	if startDate.After(endDate) {
		return c.Render(http.StatusOK, "main", IndexData{
			Err:        "Start Date can't be later than End Date",
			StatusCode: http.StatusUnprocessableEntity,
		})
	}

	if int(endDate.Sub(startDate).Hours()/24) < 180 {
		return c.Render(http.StatusOK, "main", IndexData{
			Err:        "Day Count can't be lower than 180 days to ensure proper calculation",
			StatusCode: http.StatusUnprocessableEntity,
		})
	}

	if startDate.Before(startDateLimit) || endDate.After(endDateLimit) {
		return c.Render(http.StatusOK, "main", IndexData{
			Err:        "Date can only be within 2008/01/01 until 2024/09/30",
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

	lagOrder, err := strconv.Atoi(c.FormValue("lag_order"))
	if err != nil {
		return c.Render(http.StatusOK, "main", IndexData{
			Err:        "Lag Order is not a valid number",
			StatusCode: http.StatusUnprocessableEntity,
		})
	}

	if lagOrder <= 0 || lagOrder > 10 {
		return c.Render(http.StatusOK, "main", IndexData{
			Err:        "Chosen Lag Order is not Valid (Must be 1 - 10)",
			StatusCode: http.StatusUnprocessableEntity,
		})
	}

	smoteK, err := strconv.Atoi(c.FormValue("smote_k"))
	if err != nil {
		return c.Render(http.StatusOK, "main", IndexData{
			Err:        "SMOET K Value is not a valid number",
			StatusCode: http.StatusUnprocessableEntity,
		})
	}

	if smoteK <= 0 || smoteK > 10 {
		return c.Render(http.StatusOK, "main", IndexData{
			Err:        "Chosen SMOTE K Value is not Valid (Must be 1 - 10)",
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
		return c.Render(http.StatusOK, "main", IndexData{
			Err:        "Fetching Data from NASA Power API Fails",
			StatusCode: http.StatusUnprocessableEntity,
		})
	}

	weathers.InjectNasa(&nasa)
	if weathers.Err != nil {
		return c.Render(http.StatusOK, "main", IndexData{
			Err:        "Preparing Data from NASA Power API Fails",
			StatusCode: http.StatusUnprocessableEntity,
		})
	}
	nasa.Stats()

	weathers.InjectBnpb(&bnpb, startDate, endDate, city)
	if weathers.Err != nil {
		return c.Render(http.StatusOK, "main", IndexData{
			Err:        "Preparing Data from BNPB Fails",
			StatusCode: http.StatusUnprocessableEntity,
		})
	}

	weathers.InjectNews(&news, startDate, endDate, city)
	if weathers.Err != nil {
		return c.Render(http.StatusOK, "main", IndexData{
			Err:        "Preparing Data from News Fails",
			StatusCode: http.StatusUnprocessableEntity,
		})
	}

	differencedWeathers := weathers.Differencing()

	prediction := differencedWeathers.VectorAutoregression(lagOrder)
	prediction.FillString()
	vectorAutoregressionEvaluation := differencedWeathers.VectorAutoregressionEval(6, 5, lagOrder)
	neighbors, knnResult := differencedWeathers.KNearestNeighbor(kValue, prediction)
	knnEval := differencedWeathers.KNearestNeighborEval(6, 5, kValue, lagOrder)

	oversampled := differencedWeathers.SmoteOversampling(smoteK)
	smoteNeighbors, smoteKnnResult := oversampled.KNearestNeighbor(kValue, prediction)

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
		"SMOTEValues":                       oversampled.Items,
		"SMOTEKNNHeaders":                   []string{"WS10M", "RH2M", "PRECTOTCORR", "T2M", "T2M MAX", "T2M MIN", "DISTANCE", "FLOOD"},
		"SMOTEKNNValues":                    smoteNeighbors.Items,
		"SMOTEKNNResult":                    smoteKnnResult,
		"Statistics":                        statistics,
		"Latitude":                          latitude,
		"Longitude":                         longitude,
		"Timestamp":                         time.Now().Unix(),
	}

	return c.Render(http.StatusOK, "main", IndexData{
		Data:       viewData,
		Message:    fmt.Sprintf("Preparation Done. Time Taken: %dms", time.Since(start).Milliseconds()),
		StatusCode: http.StatusOK,
	})
}

func (p *WebProcessorImpl) HandleFloodPredictionRequest(c echo.Context) error {
	startDateLimit := time.Date(2007, 12, 31, 0, 0, 0, 0, time.Local)
	endDateLimit := time.Date(2024, 10, 1, 0, 0, 0, 0, time.Local)
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
	if err != nil {
		return c.Render(http.StatusOK, "main", IndexData{
			Err:        "Parsing Date Error",
			StatusCode: http.StatusBadRequest,
			Timestamp:  time.Now().Unix(),
		})
	}
	endDate, err := time.Parse("2006-01-02", c.FormValue("end_date"))
	if err != nil {
		return c.Render(http.StatusOK, "main", IndexData{
			Err:        "Parsing Date Error",
			StatusCode: http.StatusBadRequest,
			Timestamp:  time.Now().Unix(),
		})
	}

	city := c.FormValue("city")
	if startDate.After(endDate) {
		return c.Render(http.StatusOK, "main", IndexData{
			Err:        "Start Date can't be later than End Date",
			StatusCode: http.StatusUnprocessableEntity,
		})
	}

	if int(endDate.Sub(startDate).Hours()/24) < 180 {
		return c.Render(http.StatusOK, "main", IndexData{
			Err:        "Day Count can't be lower than 180 days to ensure proper calculation",
			StatusCode: http.StatusUnprocessableEntity,
		})
	}

	if startDate.Before(startDateLimit) || endDate.After(endDateLimit) {
		return c.Render(http.StatusOK, "main", IndexData{
			Err:        "Date can only be within 2008/01/01 until 2024/09/30",
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

	smoteK, err := strconv.Atoi(c.FormValue("smote_k"))
	if err != nil {
		return c.Render(http.StatusOK, "main", IndexData{
			Err:        "SMOET K Value is not a valid number",
			StatusCode: http.StatusUnprocessableEntity,
		})
	}

	if smoteK <= 0 || smoteK > 10 {
		return c.Render(http.StatusOK, "main", IndexData{
			Err:        "Chosen SMOTE K Value is not Valid (Must be 1 - 10)",
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

	newsData := [][]string{}
	newsDataOri := [][]interface{}{}
	newsFloodData := []float64{}
	err = p.PreprocessFloodNewsCSV(&newsData, &newsDataOri, &newsFloodData, startDate, endDate, city)
	if err != nil {
		return c.Render(http.StatusOK, "main", IndexData{
			Err:        fmt.Sprintf("Preprocessing Flood News data fails, %s", err.Error()),
			StatusCode: http.StatusInternalServerError,
		})
	}

	mergedFloodData, mergedFlood := p.MergeFloodData(newsData, bnpbData, newsFloodData, floodData)

	statisticData := []map[string]interface{}{}
	p.PrepareStatistics(&mergedFloodData, &nasaDataStr, startDate, endDate, city, &statisticData)

	nasaWithFloodDataStr := p.MergeNASAWithFlood(nasaDataStr, mergedFlood)

	var stationaryNasaData [][]float64
	stationaryDataMinLength := 99999
	maxDifferencingStep := -99999
	criticalValues := make([]float64, 6)
	adfScores := make([]float64, 6)
	for i := 0; i < len(nasaData); i++ {
		stationary, criticalValue, adfScore, err := p.adfTest(nasaData[i])
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
			stationary, criticalValue, adfScore, err = p.adfTest(differencedNasaDataColumn)
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

		criticalValues[i] = criticalValue
		adfScores[i] = adfScore
		stationaryNasaData = append(stationaryNasaData, differencedNasaDataColumn)
	}

	var stationaryNasaWithFloodData [][]float64
	for i, data := range stationaryNasaData {
		differencedData := data
		update := false
		var criticalValue, adfScore float64
		for j := 0; j < len(data)-stationaryDataMinLength; j++ {
			differencedData = p.differencing(data)
			_, criticalValue, adfScore, err = p.adfTest(differencedData)
			if err != nil {
				return c.Render(http.StatusOK, "main", IndexData{
					Err:        fmt.Sprintf("Processing ADF test fails, %s", err.Error()),
					StatusCode: http.StatusInternalServerError,
				})
			}
			update = true
		}
		if update {
			criticalValues[i] = criticalValue
			adfScores[i] = adfScore
		}
		stationaryNasaWithFloodData = append(stationaryNasaWithFloodData, differencedData)
	}
	stationaryNasaWithFloodData = append(stationaryNasaWithFloodData, mergedFlood[len(mergedFlood)-stationaryDataMinLength:])

	var stationaryStatisticData []map[string]interface{}
	p.PrepareDifferencedStatistics(stationaryNasaWithFloodData, startDate, endDate, city, &stationaryStatisticData)
	predictedValues, err := p.vectorAutoregression(stationaryNasaWithFloodData)
	if err != nil {
		return c.Render(http.StatusOK, "main", IndexData{
			Err:        fmt.Sprintf("Processing VAR Autoregression fails, %s", err.Error()),
			StatusCode: http.StatusInternalServerError,
		})
	}

	_, nrmseResult, _, confMatrixResult := p.evaluateVARAndKNN(stationaryNasaWithFloodData, 1.0, 6.0, kValue)
	stationaryNasaWithFloodDataStr := twoDimFloatToTwoDimString(transpose(stationaryNasaWithFloodData))

	var predictedValuesStr []string
	for _, predictedValue := range predictedValues {
		predictedValuesStr = append(predictedValuesStr, fmt.Sprintf("%0.4f", predictedValue))
	}

	knnResult, nearestData, nearestDistances := p.knnClassification(predictedValues, stationaryNasaWithFloodData, kValue)
	flood := "No Flood"
	if knnResult == 1 {
		flood = "Flood"
	}

	knnData := nearestData[:len(nearestData)-1]
	knnData = append(knnData, nearestDistances)
	knnDataStr := twoDimFloatToTwoDimString(transpose(knnData))

	minoritySample := p.getMinoritySample(stationaryNasaWithFloodData)
	_, smotedData := p.smoteReplaceMethod(minoritySample, stationaryNasaWithFloodData, smoteK)
	undersampledSmoteData := p.undersample(smotedData, 10)

	var smoteStatisticData []map[string]interface{}
	p.PrepareDifferencedStatistics(smotedData, startDate, endDate, city, &smoteStatisticData)

	knnResultSmoteReplace, nearestDataSmoteReplace, nearestDistancesSmoteReplace := p.knnClassification(predictedValues, smotedData, kValue)
	floodSmoteReplace := "No Flood"
	if knnResultSmoteReplace == 1 {
		floodSmoteReplace = "Flood"
	}
	_, _, _, confMatrixSmoteResult := p.evaluateVARAndKNN(smotedData, 1.0, 6.0, kValue)

	_, _, _, confMatrixUndersampledSmote := p.evaluateVARAndKNN(undersampledSmoteData, 1.0, 6.0, kValue)

	knnDataSmoteReplace := nearestDataSmoteReplace[:len(nearestDataSmoteReplace)-1]
	knnDataSmoteReplace = append(knnDataSmoteReplace, nearestDistancesSmoteReplace)
	knnDataSmoteReplaceStr := twoDimFloatToTwoDimString(transpose(knnDataSmoteReplace))
	smoteDataStr := twoDimFloatToTwoDimString(transpose(smotedData))

	// findDifference(stationaryNasaWithFloodData, smoteReplacedData)

	p.logger.LogAndContinue("Done Processing Request")
	viewData := map[string]interface{}{
		"NasaHeaders":                             nasaDataStr[0],
		"NasaStat":                                nasaDataStr[1:6],
		"NasaValues":                              nasaDataStr[6:],
		"NasaFloodHeaders":                        append(nasaDataStr[0], "FLOOD"),
		"NasaFloodValues":                         nasaWithFloodDataStr,
		"BnpbHeaders":                             bnpbData[0],
		"BnpbValues":                              bnpbData[1:],
		"BnpbHeadersOri":                          bnpbDataOri[0],
		"BnpbValuesOri":                           bnpbDataOri[1:],
		"NewsHeadersOri":                          newsDataOri[0],
		"NewsValuesOri":                           newsDataOri[1:],
		"NRMSEEvaluationHeaders":                  nrmseResult[0],
		"NRMSEEvaluationValues":                   nrmseResult[1:],
		"ConfusionMatrixHeaders":                  confMatrixResult[0],
		"ConfusionMatrixValues":                   confMatrixResult[1:],
		"ConfusionMatrixSmoteHeaders":             confMatrixSmoteResult[0],
		"ConfusionMatrixSmoteValues":              confMatrixSmoteResult[1:],
		"ConfusionMatrixUndersampledSmoteHeaders": confMatrixUndersampledSmote[0],
		"ConfusionMatrixUndersampledSmoteValues":  confMatrixUndersampledSmote[1:],
		"ADFWithParam":                            pairAdfWithParam(criticalValues, adfScores),
		"StationaryDataHeaders":                   []string{"WS10M", "RH2M", "PRECTOTCORR", "T2M", "T2M_MAX", "T2M_MIN", "FLOOD"},
		"StationaryDataValues":                    stationaryNasaWithFloodDataStr,
		"SmoteDataHeaders":                        []string{"WS10M", "RH2M", "PRECTOTCORR", "T2M", "T2M_MAX", "T2M_MIN", "FLOOD"},
		"SmoteDataValues":                         smoteDataStr,
		"StatisticData":                           statisticData,
		"StationaryStatisticData":                 stationaryStatisticData,
		"SmoteStatisticData":                      smoteStatisticData,
		"StartDate":                               startDate.Format("2006/01/02"),
		"EndDate":                                 endDate.Format("2006/01/02"),
		"Latitude":                                latitude,
		"Longitude":                               longitude,
		"DifferencingStep":                        strconv.Itoa(maxDifferencingStep),
		"PredictedHeaders":                        []string{"WS10M", "RH2M", "PRECTOTCORR", "T2M", "T2M_MAX", "T2M_MIN"},
		"PredictedValues":                         predictedValuesStr,
		"KNNResult":                               flood,
		"KNNDataHeaders":                          []string{"WS10M", "RH2M", "PRECTOTCORR", "T2M", "T2M_MAX", "T2M_MIN", "FLOOD", "DISTANCE"},
		"KNNDataValues":                           knnDataStr,
		"KNNResultSmoteReplace":                   floodSmoteReplace,
		"KNNDataHeadersSmoteReplace":              []string{"WS10M", "RH2M", "PRECTOTCORR", "T2M", "T2M_MAX", "T2M_MIN", "FLOOD", "DISTANCE"},
		"KNNDataValuesSmoteReplace":               knnDataSmoteReplaceStr,
		"Timestamp":                               time.Now().Unix(),
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
			mergedDataOri = append(mergedDataOri, data...)
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

func (p *WebProcessorImpl) PreprocessFloodNewsCSV(newsData *[][]string, newsDataOri *[][]interface{}, floodData *[]float64, startDate, endDate time.Time, city string) error {
	wd, err := os.Getwd()
	if err != nil {
		return errors.New("Get working directory fails Preprocess Flood News Data")
	}

	csvFile, err := os.Open(filepath.Join(wd, "tmp/data_berita_banjir.csv"))
	if err != nil {
		return errors.New("Opening data csv file fails")
	}
	defer csvFile.Close()

	reader := csv.NewReader(csvFile)
	records, err := reader.ReadAll()
	if err != nil {
		return errors.New("Reading data csv file fails")
	}

	headersStrInterface := []interface{}{"CITY", "FLOOD DATE", "LINK"}
	headersStr := []string{"CITY", "FLOOD DATE"}
	records = records[1:]
	cityUppercase := strings.ToUpper(city)

	preparedNewsData := [][]string{}
	preparedNewsDataOri := [][]interface{}{}
	recordDateMap := make(map[string]bool)
	for _, record := range records {
		dateSlice := strings.Split(record[1], "/")
		year, _ := strconv.Atoi(dateSlice[0])
		month, _ := strconv.Atoi(dateSlice[1])
		day, _ := strconv.Atoi(dateSlice[2])
		date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)

		if date.After(startDate) && date.Before(endDate) && strings.ToLower(record[0]) == city {
			if _, exists := recordDateMap[record[1]]; !exists {
				var recordInterface []interface{}
				preparedNewsData = append(preparedNewsData, []string{cityUppercase, record[1]})
				recordInterface = append(recordInterface, record[0], record[1])
				recordInterface = append(recordInterface, template.HTML(fmt.Sprintf("<a class=\"text-blue-800\" href=\"%s\">Link</a>", record[2])))
				preparedNewsDataOri = append(preparedNewsDataOri, recordInterface)
				recordDateMap[record[1]] = true
			}
		}
	}

	dayCount := int(endDate.Sub(startDate).Hours()/24) + 1
	var flood []float64
	for i := 0; i < dayCount; i++ {
		curr := startDate.Add(time.Hour * 24 * time.Duration(i))
		currStr := curr.Format("2006/01/02")
		if _, exists := recordDateMap[currStr]; exists {
			flood = append(flood, 1)
		} else {
			flood = append(flood, 0)
		}
	}

	*floodData = flood
	*newsData = append([][]string{headersStr}, preparedNewsData...)
	*newsDataOri = append([][]interface{}{headersStrInterface}, preparedNewsDataOri...)

	return nil
}

func (p *WebProcessorImpl) MergeFloodData(newsFloodData, bnpbFloodData [][]string, newsFlood, bnpbFlood []float64) (mergedFloodData [][]string, mergedFlood []float64) {
	dateMap := make(map[string]bool)

	headers := bnpbFloodData[0]
	bnpbFloodData = bnpbFloodData[1:]
	newsFloodData = newsFloodData[1:]
	mergedData := [][]string{}

	for _, bd := range bnpbFloodData {
		if _, exists := dateMap[bd[1]]; !exists {
			mergedData = append(mergedData, bd)
			dateMap[bd[1]] = true
		}
	}

	for _, nd := range newsFloodData {
		if _, exists := dateMap[nd[1]]; !exists {
			mergedData = append(mergedData, nd)
			dateMap[nd[1]] = true
		}
	}

	for i := 0; i < len(newsFlood); i++ {
		if newsFlood[i] == 1 || bnpbFlood[i] == 1 {
			mergedFlood = append(mergedFlood, 1)
		} else {
			mergedFlood = append(mergedFlood, 0)
		}
	}

	mergedFloodData = append([][]string{headers}, mergedData...)

	return
}

func (p *WebProcessorImpl) MergeNASAWithFlood(nasaDataStr [][]string, floodData []float64) (result [][]string) {
	for i, nasaData := range nasaDataStr[6:] {
		floodStr := strconv.FormatFloat(floodData[i], 'f', 2, 64)
		nasaData = append(nasaData, floodStr)
		result = append(result, nasaData)
	}

	return
}

func (p *WebProcessorImpl) PrepareStatistics(bnpbData, nasaData *[][]string, startDate, endDate time.Time, city string, statisticData *[]map[string]interface{}) {
	stats := []map[string]interface{}{}
	stats = append(stats, map[string]interface{}{"StartDate": startDate.Format("2006/01/02")})
	stats = append(stats, map[string]interface{}{"EndDate": endDate.Format("2006/01/02")})
	stats = append(stats, map[string]interface{}{"City": strings.ToUpper(city)})
	stats = append(stats, map[string]interface{}{"DayCount": int(endDate.Sub(startDate).Hours()/24) + 1})
	stats = append(stats, map[string]interface{}{"DataCount": len(*nasaData) - 6})
	stats = append(stats, map[string]interface{}{"FloodCount": len(*bnpbData) - 1})
	stats = append(stats, map[string]interface{}{"FloodPercentage": fmt.Sprintf("%0.3f%s", float64((float64(len(*bnpbData))-1)/(float64(len(*nasaData))-1)*100), "%")})

	*statisticData = stats
}

func (p *WebProcessorImpl) PrepareDifferencedStatistics(stationaryData [][]float64, startDate, endDate time.Time, city string, statisticData *[]map[string]interface{}) {
	transposedStationaryData := transpose(stationaryData)
	floodCount := 0
	stats := []map[string]interface{}{}
	stats = append(stats, map[string]interface{}{"StartDate": startDate.Format("2006/01/02")})
	stats = append(stats, map[string]interface{}{"EndDate": endDate.Format("2006/01/02")})
	stats = append(stats, map[string]interface{}{"City": strings.ToUpper(city)})
	stats = append(stats, map[string]interface{}{"DayCount": int(endDate.Sub(startDate).Hours()/24) + 1})
	stats = append(stats, map[string]interface{}{"DataCount": len(transposedStationaryData)})

	for _, d := range transposedStationaryData {
		if d[6] == float64(1) {
			floodCount++
		}
	}

	stats = append(stats, map[string]interface{}{"FloodCount": floodCount})
	stats = append(stats, map[string]interface{}{"FloodPercentage": fmt.Sprintf("%0.3f%s", float64(floodCount)/float64(len(transposedStationaryData))*100, "%")})

	*statisticData = stats
}

func (p *WebProcessorImpl) undersample(data [][]float64, multiplier int) (undersampledData [][]float64) {
	var minorityCount int
	var minorityData [][]float64
	transposedData := transpose(data)
	for _, d := range transposedData {
		if d[6] == 1 {
			minorityCount++
			minorityData = append(minorityData, d)
		}
	}

	picked := make(map[int]bool)
	for i := 0; i < minorityCount*multiplier; i++ {
		var randomInt int
		for j := 0; j < len(transposedData); j++ {
			randomInt = rand.Intn(len(transposedData))
			if _, exists := picked[randomInt]; !exists {
				if transposedData[randomInt][6] == 0 {
					break
				}
			}
		}
		undersampledData = append(undersampledData, transposedData[randomInt])
	}
	undersampledData = append(undersampledData, minorityData...)
	undersampledData = transpose(undersampledData)
	return
}

func (p *WebProcessorImpl) getMax(data []float64) (max float64) {
	max = data[0]
	for _, d := range data {
		if d == -999 {
			continue
		}
		if max < d {
			max = d
		}
	}
	return
}

func (p *WebProcessorImpl) getMin(data []float64) (min float64) {
	min = data[0]
	for _, d := range data {
		if d == -999 {
			continue
		}
		if min > d {
			min = d
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

func (p *WebProcessorImpl) adfTest(originalData []float64) (bool, float64, float64, error) {
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
		return false, 0.0, 0.0, err
	}

	olsResult := mat.NewDense(3, 1, nil)
	olsResult.Mul(&inverseXtxMatrix, xtyMatrix)
	olsData := olsResult.RawMatrix().Data
	return criticalValue > olsData[1], criticalValue, olsData[1], nil
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

func (p *WebProcessorImpl) knnClassification(dataPoints []float64, nasaData [][]float64, kValue int) (int, [][]float64, []float64) {
	var distances []float64
	var nearest [][]float64
	var nearestDistances []float64
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

	transposedNasaData := transpose(appendIndex(nasaData))
	sortedDistances := make([]float64, len(distances))
	sortedFlood := make([]float64, len(nasaData[6]))
	sortedNasaData := make([][]float64, len(distances))

	for i, idx := range indices {
		sortedDistances[i] = distances[idx]
		sortedFlood[i] = nasaData[6][idx]
		sortedNasaData[i] = transposedNasaData[idx]
	}

	var kScore float64
	for i := 0; i < kValue; i++ {
		nearestDistances = append(nearestDistances, sortedDistances[i])
		nearest = append(nearest, sortedNasaData[i])
		kScore += sortedFlood[i]
	}

	result := 0
	if kScore/float64(kValue) >= 0.5 {
		result = 1
	}

	return result, transpose(nearest), nearestDistances
}

func (p *WebProcessorImpl) evaluateVARAndKNN(data [][]float64, start, stop float64, kValue int) (rmse [][]float64, rmseStr [][]string, confMatrix [][]float64, confMatrixStr [][]string) {
	var lengthSplit []int
	for i := start; i < stop+1; i++ {
		split := math.Floor(float64(len(data[0])) * ((100 - i*5) / 100))
		lengthSplit = append(lengthSplit, int(split))
	}

	var dataRange []float64
	for i := 0; i < len(data)-1; i++ {
		dataRange = append(dataRange, p.getMax(data[i])-p.getMin(data[i]))
	}

	// Rolling VAR & KNN
	for _, split := range lengthSplit {
		mse := make([]float64, 6)
		confusionMatrix := make([]float64, 4)
		for j := split; j < len(data[0]); j++ {
			var trainData [][]float64
			var testData []float64
			for k := 0; k < len(data); k++ {
				trainData = append(trainData, data[k][:j])
				testData = append(testData, data[k][j-1])
			}

			varResult, _ := p.vectorAutoregression(trainData)
			for k := 0; k < len(varResult); k++ {
				mse[k] += math.Pow(varResult[k]-data[k][j], 2)
			}

			knnResult, _, _ := p.knnClassification(varResult, trainData, kValue)
			actualResult := int(testData[6])

			if knnResult == 1 && actualResult == 1 { // True Positive
				confusionMatrix[0] += 1
			} else if knnResult == 1 && actualResult == 0 { // False Positive
				confusionMatrix[1] += 1
			} else if knnResult == 0 && actualResult == 0 { // True Negative
				confusionMatrix[2] += 1
			} else if knnResult == 0 && actualResult == 1 { // False Negative
				confusionMatrix[3] += 1
			}
		}

		for j := 0; j < len(mse); j++ {
			mse[j] = math.Sqrt(mse[j]/float64(split)) / dataRange[j]
		}
		confMatrix = append(confMatrix, confusionMatrix)
		rmse = append(rmse, mse)
	}

	rmseStr = append(rmseStr, []string{"TRAIN-TEST-SPLIT", "WS10M", "RH2M", "PRECTOTCORR", "T2M", "T2M_MAX", "T2M_MIN"})
	confMatrixStr = append(confMatrixStr, []string{"TRAIN-TEST-SPLIT", "TRUE POSITIVE", "FALSE POSITIVE", "TRUE NEGATIVE", "FALSE NEGATIVE", "ACCURACY", "PRECISION", "RECALL", "F1 SCORE"})

	for i := 0; i < len(lengthSplit); i++ {
		var accuracy, precision, recall, f1Score float64
		testLength := 5 * (i + 1)
		trainLength := 100 - testLength
		trainTest := fmt.Sprintf("%d-%d", trainLength, testLength)
		strSlice := oneDimFloatToOneDimString(rmse[i])
		confusionStr := oneDimFloatToOneDimString(confMatrix[i])
		rmseStr = append(rmseStr, []string{trainTest, strSlice[0], strSlice[1], strSlice[2], strSlice[3], strSlice[4], strSlice[5]})

		if (confMatrix[i][0] + confMatrix[i][1] + confMatrix[i][2] + confMatrix[i][3]) != 0 {
			accuracy = (confMatrix[i][0] + confMatrix[i][2]) / (confMatrix[i][0] + confMatrix[i][1] + confMatrix[i][2] + confMatrix[i][3])
		} else {
			accuracy = 0
		}

		if (confMatrix[i][0] + confMatrix[i][1]) != 0 {
			precision = confMatrix[i][0] / (confMatrix[i][0] + confMatrix[i][1])
		} else {
			precision = 0
		}

		if (confMatrix[i][0] + confMatrix[i][3]) != 0 {
			recall = confMatrix[i][0] / (confMatrix[i][0] + confMatrix[i][3])
		} else {
			recall = 0
		}

		if (precision + recall) != 0 {
			f1Score = 2 * (precision * recall) / (precision + recall)
		} else {
			f1Score = 0
		}

		confMatrixData := []string{trainTest, confusionStr[0], confusionStr[1], confusionStr[2], confusionStr[3], strconv.FormatFloat(accuracy, 'f', 3, 64), strconv.FormatFloat(precision, 'f', 3, 64), strconv.FormatFloat(recall, 'f', 3, 64), strconv.FormatFloat(f1Score, 'f', 3, 64)}
		confMatrixStr = append(confMatrixStr, confMatrixData)
	}
	return
}

func (p *WebProcessorImpl) getMinoritySample(data [][]float64) (minoritySample [][]float64) {
	data = transpose(data)
	for i := range data {
		if data[i][len(data[i])-1] == 1 {
			minoritySample = append(minoritySample, data[i])
		}
	}
	minoritySample = transpose(minoritySample)
	return
}

func (p *WebProcessorImpl) smoteMethod(minoritySample, data [][]float64, smoteK int) (smotedData [][]float64) {
	transposedMinoritySample := transpose(minoritySample)
	transposedData := transpose(data)
	newDataPoints := [][]float64{}

	if smoteK > len(transposedMinoritySample) {
		smoteK = len(transposedMinoritySample) - 1
		if smoteK <= 0 {
			return data
		}
	}

	type indexedFloat struct {
		index int
		value float64
	}

	for i := 0; i < len(transposedMinoritySample); i++ {
		distances := make([]float64, len(transposedMinoritySample))
		currentData := transposedMinoritySample[i]
		for j := 0; j < len(transposedMinoritySample); j++ {
			if i == j {
				continue
			}
			checkData := transposedMinoritySample[j]

			distance := 0.0
			for k := 0; k < len(checkData)-1; k++ {
				distance += math.Pow(currentData[k]-checkData[k], 2)
			}

			distances[j] = math.Sqrt(distance)
		}

		indexedDistances := make([]indexedFloat, len(distances))
		for j, v := range distances {
			indexedDistances[j] = indexedFloat{index: j, value: v}
		}

		sort.Slice(indexedDistances, func(j, k int) bool {
			return indexedDistances[j].value < indexedDistances[k].value
		})

		for j := 0; j < smoteK; j++ {
			index := indexedDistances[j].index
			neighborData := transposedMinoritySample[index]
			newDataPoint := make([]float64, 7)
			for k := 0; k < len(neighborData)-1; k++ {
				lambda := rand.Float64()
				newDataPoint[k] = currentData[k] + lambda*(neighborData[k]-currentData[k])
			}
			newDataPoint[6] = 1
			newDataPoints = append(newDataPoints, newDataPoint)
		}
	}

	transposedData = append(transposedData, newDataPoints...)
	smotedData = transpose(transposedData)

	return
}

func (p *WebProcessorImpl) smoteReplaceMethod(minoritySample, data [][]float64, smoteK int) (toBeReplacedMap map[float64][]float64, smotedData [][]float64) {
	// Select randomly from minority sample
	transposedMinoritySample := transpose(minoritySample)
	transposedData := transpose(data)
	toBeReplacedMap = make(map[float64][]float64)

	for i := 0; i < len(transposedMinoritySample); i++ {
		randomSample := transposedMinoritySample[rand.Intn(len(transposedMinoritySample))]

		_, nearestPoints, _ := p.knnClassification(randomSample[:len(randomSample)-1], data, smoteK+1)

		// Remove First because itself is chosen
		transposedNearestPoints := transpose(nearestPoints)[1:]
		for _, nearestDataPoint := range transposedNearestPoints {
			lambda := rand.Float64()
			syntheticDataPoint := make([]float64, 6)
			for k := 0; k < 6; k++ {
				syntheticDataPoint[k] = randomSample[k] + lambda*(nearestDataPoint[k]-randomSample[k])
			}

			if _, exists := toBeReplacedMap[nearestDataPoint[7]]; !exists {
				toBeReplacedMap[nearestDataPoint[7]] = syntheticDataPoint
			}
		}
	}

	transposedSmotedData := transposedData
	for index, replace := range toBeReplacedMap {
		if transposedSmotedData[int(index)][6] == 1 {
			continue
		}
		syntheticData := replace
		syntheticData = append(syntheticData, 1)
		transposedSmotedData[int(index)] = syntheticData
	}
	smotedData = transpose(transposedSmotedData)

	return
}

func matPrint(X mat.Matrix) {
	fa := mat.Formatted(X, mat.Prefix(""), mat.Squeeze())
	fmt.Printf("%v\n", fa)
}

func oneDimFloatToOneDimString(input []float64) (output []string) {
	for i := 0; i < len(input); i++ {
		output = append(output, fmt.Sprintf("%0.3f", input[i]))
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

func transpose(data [][]float64) [][]float64 {
	numFeatures := len(data)
	numSamples := len(data[0])
	transposed := make([][]float64, numSamples)
	for i := range transposed {
		transposed[i] = make([]float64, numFeatures)
		for j := range data {
			transposed[i][j] = data[j][i]
		}
	}
	return transposed
}

// Only use to the untransposed Data
func appendIndex(input [][]float64) (output [][]float64) {
	index := make([]float64, len(input[0]))
	for i := range input[0] {
		index[i] = float64(i)
	}
	output = input
	output = append(output, index)
	return
}

func pairAdfWithParam(criticalValues, adfScore []float64) (output []adfWithParam) {
	paramNames := []string{"WS10M", "RH2M", "PRECTOTCORR", "T2M", "T2M_MAX", "T2M_MIN"}
	for i, name := range paramNames {
		output = append(output, adfWithParam{
			Name:          name,
			CriticalValue: strconv.FormatFloat(criticalValues[i], 'f', 3, 64),
			ADFScore:      strconv.FormatFloat(adfScore[i], 'f', 3, 64),
		})
	}

	return
}

type adfWithParam struct {
	Name          string
	CriticalValue string
	ADFScore      string
}

func findDifference(data1, data2 [][]float64) {
	data1 = transpose(data1)
	data2 = transpose(data2)
	for i := 0; i < len(data1); i++ {
		for j := 0; j < len(data1[i]); j++ {
			if data1[i][j] != data2[i][j] {
				fmt.Printf("DATA DIFFERENCE INDEX: %d\nData 1: %v\nData 2: %v\n\n", i, data1[i], data2[i])
				break
			}
		}
	}
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

func (w *Weathers) KNearestNeighbor(kValue int, new Weather) (neighbors Weathers, result string) {
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

func (w *Weathers) KNearestNeighborEval(step, magnitude, kValue, lagOrder int) (confusionMatrix []ConfusionMatrix) {
	if magnitude*step > 100 {
		return
	}

	confusionMatrix = make([]ConfusionMatrix, step)
	for i := 1; i <= step; i++ {
		testPerc := fmt.Sprintf("%d", i*magnitude)
		trainPerc := fmt.Sprintf("%d", 100-i*magnitude)

		test := magnitude * i
		testSize := len(w.Items) * test / 100
		trainSize := len(w.Items) - testSize

		predictionCount := 0

		for j := trainSize; j < len(w.Items)-1; j++ {
			trainSlice := w.Items[:j]
			trainDataset := Weathers{
				Items: trainSlice,
			}

			predicted := trainDataset.VectorAutoregression(lagOrder)
			_, knnResult := trainDataset.KNearestNeighbor(kValue, predicted)
			actual := w.Items[j]

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

func (w *Weathers) SmoteOversampling(smoteKValue int) (oversampledData Weathers) {
	minoritySample := w.GetMinoritySample()

	var syntheticData []Weather
	for _, d := range minoritySample.Items {
		neighbors, _ := minoritySample.KNearestNeighbor(smoteKValue, d)
		for _, e := range neighbors.Items {
			syntheticData = append(syntheticData, d.InterpolateSyntheticData(e))
		}
	}

	oversampledData.Items = make([]Weather, len(w.Items))
	copy(oversampledData.Items, w.Items)
	oversampledData.Oversample.SynthData = syntheticData
	oversampledData.Oversample.SynthDataCount = len(syntheticData)
	oversampledData.Items = append(oversampledData.Items, syntheticData...)

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
}

func (w *Weather) InterpolateSyntheticData(neighbor Weather) (synth Weather) {
	synth.WindSpeed = w.WindSpeed + (rand.Float64() * (neighbor.WindSpeed - w.WindSpeed))
	synth.RelHumidity = w.RelHumidity + (rand.Float64() * (neighbor.RelHumidity - w.RelHumidity))
	synth.Precipitation = w.Precipitation + (rand.Float64() * (neighbor.Precipitation - w.Precipitation))
	synth.TempAverage = w.TempAverage + (rand.Float64() * (neighbor.TempAverage - w.TempAverage))
	synth.TempMax = w.TempMax + (rand.Float64() * (neighbor.TempMax - w.TempMax))
	synth.TempMin = w.TempMin + (rand.Float64() * (neighbor.TempMin - w.TempMin))
	synth.Flood = true
	return
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
	s.Smote.DataCount = strconv.Itoa(len(s.Ref.Smote.Items))

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
	for _, d := range s.Ref.Smote.Items {
		if d.Flood {
			smoteFloodCount++
		}
	}
	var weathersFloodPercentage, differencedWeathersFloodPercentage, smoteFloodPercentage float64
	weathersFloodPercentage = float64(weathersFloodCount) / float64(len(s.Ref.Weathers.Items))
	differencedWeathersFloodPercentage = float64(differencedWeathersFloodCount) / float64(len(s.Ref.DifferencedWeathers.Items))
	smoteFloodPercentage = float64(smoteFloodCount) / float64(len(s.Ref.DifferencedWeathers.Items))

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
