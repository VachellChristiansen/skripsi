package processor

import (
	"html/template"
	"time"
)

type IndexData struct {
	Data       map[string]interface{}
	JSData     string
	Message    string
	Err        string
	StatusCode int
	Timestamp  int64
}

type Weather struct {
	Date             time.Time
	WindSpeed        float64
	RelHumidity      float64
	Precipitation    float64
	TempAverage      float64
	TempMax          float64
	TempMin          float64
	Flood            bool
	DateStr          string
	WindSpeedStr     string
	RelHumidityStr   string
	PrecipitationStr string
	TempAverageStr   string
	TempMaxStr       string
	TempMinStr       string
	FloodStr         template.HTML
}

type Weathers struct {
	Items []Weather
	Diff  DifferencedStatistics
	Err   error
}

type DifferencedStatistics struct {
	Step                   int
	CriticalValues         Weather
	Gamma                  Weather
	CriticalValuesGammaMap []KeyValue
}

type Nasa struct {
	WindSpeed        float64
	RelHumidity      float64
	Precipitation    float64
	TempAverage      float64
	TempMax          float64
	TempMin          float64
	DateStr          string
	WindSpeedStr     string
	RelHumidityStr   string
	PrecipitationStr string
	TempAverageStr   string
	TempMaxStr       string
	TempMinStr       string
}

type NasaData struct {
	Items    []Nasa
	Max      Nasa
	Min      Nasa
	Mean     Nasa
	StdDev   Nasa
	Variance Nasa
}

type Bnpb struct {
	Code      string
	CityID    string
	Date      string
	Occurence string
	Location  string
	City      string
	Province  string
	Cause     string
}

type BnpbData struct {
	Items []Bnpb
}

type News struct {
	City string
	Date string
	Link template.HTML
}

type NewsData struct {
	Items []News
}

type KeyValue struct {
	Key   string
	Value string
}

type Statistics struct {
	City                   string
	StartDate              string
	EndDate                string
	Nasa                   NasaStatistic
	NasaMap                []KeyValue
	Bnpb                   BnpbStatistic
	BnpbMap                []KeyValue
	News                   NewsStatistic
	NewsMap                []KeyValue
	Weathers               WeatherStatistic
	WeathersMap            []KeyValue
	DifferencedWeathers    WeatherStatistic
	DifferencedWeathersMap []KeyValue
	Ref                    StatisticsReference
}

type StatisticsReference struct {
	Nasa                *NasaData
	Bnpb                *BnpbData
	News                *NewsData
	Weathers            *Weathers
	DifferencedWeathers *Weathers
}

type NasaStatistic struct {
	DayCount  string
	DataCount string
}

type BnpbStatistic struct {
	FloodCount string
}

type NewsStatistic struct {
	FloodCount string
}

type WeatherStatistic struct {
	DataCount       string
	FloodCount      string
	FloodPercentage string
}
