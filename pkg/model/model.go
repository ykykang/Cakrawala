package model

import "time"

type Sentiment string
type Confidence string
type DocType string

const (
	SentimentPositive Sentiment = "POSITIVE"
	SentimentNegative Sentiment = "NEGATIVE"
	SentimentNeutral  Sentiment = "NEUTRAL"
)

const (
	ConfidenceHigh   Confidence = "high"
	ConfidenceMedium Confidence = "low"
	ConfidenceLow    Confidence = "medium"
)
const (
	DocTypeDisclosure DocType = "disclosure"
	DocTypeLapKeu     DocType = "laporan_keuangan"
	DocTypeOther      DocType = "other"
)

type Disclosure struct {
	Emiten   string
	Title    string
	Date     time.Time
	PDFURL   string
	Category string
	Text     string
	Hash     string
}

type Article struct {
	Title       string
	URL         string
	Source      string
	PublishedAt time.Time
	Snippet     string
	Tickets     []string
	Hash        string
}

type ClassifiedItem struct {
	Sentiment  Sentiment
	Confidence Confidence
	Reaseon    string
	Tickers    []string

	Disclosure *Disclosure
	Article    *Article
}

type DailyBrief struct {
	Date     time.Time
	Positive []ClassifiedItem
	Negative []ClassifiedItem
	Neutral  []ClassifiedItem
}
