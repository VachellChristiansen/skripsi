package constant

const (
	LoggerPrefixDebug   = "[ DEBUG ]"
	LoggerPrefixInfo    = "[ INFO ]"
	LoggerPrefixWarning = "[ WARNING ]"
	LoggerPrefixError   = "[ ERROR ]"

	LoggerFileDebug   = "logger/debug.log"
	LoggerFileInfo    = "logger/info.log"
	LoggerFileWarning = "logger/warning.log"
	LoggerFileError   = "logger/error.log"

	NasaPowerAPIBaseURL = "https://power.larc.nasa.gov/api/temporal/daily/point"
	NasaPowerAPIParams  = "community=ag&parameters=TMIN%2CTMAX%2CPRECTOT%2CWS10M%2CT2M%2CRH2M&format=csv&user=V&header=true&time-standard=utc"
)
