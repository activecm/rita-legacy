package useragent

//UserAgent holds the results of the user agent analysis
type UserAgent struct {
	UserAgent string `bson:"user_agent"`
	TimesUsed int64  `bson:"times_used"`
}
