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
	Distance         float64
	Flood            bool
	DateStr          string
	WindSpeedStr     string
	RelHumidityStr   string
	PrecipitationStr string
	TempAverageStr   string
	TempMaxStr       string
	TempMinStr       string
	DistanceStr      string
	FloodStr         template.HTML
}

type Weathers struct {
	Items      []Weather
	Diff       DifferencedStatistics
	Oversample OversampledStatistics
	Err        error
}

type DifferencedStatistics struct {
	Step                   int
	CriticalValues         Weather
	Gamma                  Weather
	CriticalValuesGammaMap []KeyValue
}

type OversampledStatistics struct {
	SynthDataCount int
	SynthData      []Weather
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
	Smote                  WeatherStatistic
	SmoteMap               []KeyValue
	Ref                    StatisticsReference
}

type StatisticsReference struct {
	Nasa                *NasaData
	Bnpb                *BnpbData
	News                *NewsData
	Weathers            *Weathers
	DifferencedWeathers *Weathers
	Smote               *Weathers
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

type ConfusionMatrix struct {
	TruePositive     int
	TrueNegative     int
	FalsePositive    int
	FalseNegative    int
	Accuracy         float64
	Precision        float64
	Recall           float64
	F1Score          float64
	TrainTestStr     string
	TruePositiveStr  string
	TrueNegativeStr  string
	FalsePositiveStr string
	FalseNegativeStr string
	AccuracyStr      string
	PrecisionStr     string
	RecallStr        string
	F1ScoreStr       string
}

type KeyValue struct {
	Key   string
	Value string
}
