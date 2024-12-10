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
	Date                time.Time     `json:"date"`
	WindSpeed           float64       `json:"wind_speed"`
	RelHumidity         float64       `json:"rel_humidity"`
	Precipitation       float64       `json:"precipitation"`
	TempAverage         float64       `json:"temp_average"`
	TempMax             float64       `json:"temp_max"`
	TempMin             float64       `json:"temp_min"`
	Distance            float64       `json:"distance"`
	Flood               bool          `json:"flood"`
	CosineSimilarity    []float64     `json:"cosine_similarity"`
	AvgCosineSimilarity float64       `json:"avg_cosine_similarity"`
	DateStr             string        `json:"date_str"`
	WindSpeedStr        string        `json:"wind_speed_str"`
	RelHumidityStr      string        `json:"rel_humidity_str"`
	PrecipitationStr    string        `json:"precipitation_str"`
	TempAverageStr      string        `json:"temp_average_str"`
	TempMaxStr          string        `json:"temp_max_str"`
	TempMinStr          string        `json:"temp_min_str"`
	DistanceStr         string        `json:"distance_str"`
	FloodStr            template.HTML `json:"flood_str"`
}

type Weathers struct {
	Items      []Weather             `json:"items"`
	SynthItems []Weather             `json:"synth_items"`
	Diff       DifferencedStatistics `json:"diff"`
	Oversample OversampledStatistics `json:"oversample"`
	Err        error                 `json:"err"`
}

type DifferencedStatistics struct {
	Step                   int        `json:"step"`
	CriticalValues         Weather    `json:"critical_values"`
	Gamma                  Weather    `json:"gamma"`
	CriticalValuesGammaMap []KeyValue `json:"critical_values_gamma_map"`
}

type OversampledStatistics struct {
	SynthDataCount int       `json:"synth_data_count"`
	SynthData      []Weather `json:"synth_data"`
}

type Nasa struct {
	WindSpeed        float64 `json:"wind_speed"`
	RelHumidity      float64 `json:"rel_humidity"`
	Precipitation    float64 `json:"precipitation"`
	TempAverage      float64 `json:"temp_average"`
	TempMax          float64 `json:"temp_max"`
	TempMin          float64 `json:"temp_min"`
	DateStr          string  `json:"date_str"`
	WindSpeedStr     string  `json:"wind_speed_str"`
	RelHumidityStr   string  `json:"rel_humidity_str"`
	PrecipitationStr string  `json:"precipitation_str"`
	TempAverageStr   string  `json:"temp_average_str"`
	TempMaxStr       string  `json:"temp_max_str"`
	TempMinStr       string  `json:"temp_min_str"`
}

type NasaData struct {
	Items    []Nasa `json:"items"`
	Max      Nasa   `json:"max"`
	Min      Nasa   `json:"min"`
	Mean     Nasa   `json:"mean"`
	StdDev   Nasa   `json:"std_dev"`
	Variance Nasa   `json:"variance"`
}

type Bnpb struct {
	Code      string `json:"code"`
	CityID    string `json:"city_id"`
	Date      string `json:"date"`
	Occurence string `json:"occurence"`
	Location  string `json:"location"`
	City      string `json:"city"`
	Province  string `json:"province"`
	Cause     string `json:"cause"`
}

type BnpbData struct {
	Items []Bnpb `json:"items"`
}

type News struct {
	City string        `json:"city"`
	Date string        `json:"date"`
	Link template.HTML `json:"link"`
}

type NewsData struct {
	Items []News `json:"items"`
}

type Statistics struct {
	City                   string              `json:"city"`
	StartDate              string              `json:"start_date"`
	EndDate                string              `json:"end_date"`
	Nasa                   NasaStatistic       `json:"nasa"`
	NasaMap                []KeyValue          `json:"nasa_map"`
	Bnpb                   BnpbStatistic       `json:"bnpb"`
	BnpbMap                []KeyValue          `json:"bnpb_map"`
	News                   NewsStatistic       `json:"news"`
	NewsMap                []KeyValue          `json:"news_map"`
	Weathers               WeatherStatistic    `json:"weathers"`
	WeathersMap            []KeyValue          `json:"weathers_map"`
	DifferencedWeathers    WeatherStatistic    `json:"differenced_weathers"`
	DifferencedWeathersMap []KeyValue          `json:"differenced_weathers_map"`
	Smote                  WeatherStatistic    `json:"smote"`
	SmoteMap               []KeyValue          `json:"smote_map"`
	Ref                    StatisticsReference `json:"ref"`
}

type StatisticsReference struct {
	Nasa                *NasaData `json:"nasa"`
	Bnpb                *BnpbData `json:"bnpb"`
	News                *NewsData `json:"news"`
	Weathers            *Weathers `json:"weathers"`
	DifferencedWeathers *Weathers `json:"differenced_weathers"`
	Smote               *Weathers `json:"smote"`
}

type NasaStatistic struct {
	DayCount  string `json:"day_count"`
	DataCount string `json:"data_count"`
}

type BnpbStatistic struct {
	FloodCount string `json:"flood_count"`
}

type NewsStatistic struct {
	FloodCount string `json:"flood_count"`
}

type WeatherStatistic struct {
	DataCount        string `json:"data_count"`
	FloodCount       string `json:"flood_count"`
	FloodPercentage  string `json:"flood_percentage"`
	OversampledCount string `json:"oversampled_count"`
}

type ConfusionMatrix struct {
	TruePositive     int     `json:"true_positive"`
	TrueNegative     int     `json:"true_negative"`
	FalsePositive    int     `json:"false_positive"`
	FalseNegative    int     `json:"false_negative"`
	Accuracy         float64 `json:"accuracy"`
	Precision        float64 `json:"precision"`
	Recall           float64 `json:"recall"`
	F1Score          float64 `json:"f1_score"`
	TrainTestStr     string  `json:"train_test_str"`
	TruePositiveStr  string  `json:"true_positive_str"`
	TrueNegativeStr  string  `json:"true_negative_str"`
	FalsePositiveStr string  `json:"false_positive_str"`
	FalseNegativeStr string  `json:"false_negative_str"`
	AccuracyStr      string  `json:"accuracy_str"`
	PrecisionStr     string  `json:"precision_str"`
	RecallStr        string  `json:"recall_str"`
	F1ScoreStr       string  `json:"f1_score_str"`
}

type KeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
