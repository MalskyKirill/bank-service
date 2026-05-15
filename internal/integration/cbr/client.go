package cbr

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/beevik/etree"
)

const defaultEndpoint = "https://www.cbr.ru/DailyInfoWebServ/DailyInfo.asmx"

type Client struct {
	endpoint     string
	margin       float64
	lookbackDays int
	httpClient   *http.Client
}

type keyRatePoint struct {
	Date time.Time
	Rate float64
}

func NewClient(endpoint string, margin float64, lookbackDays int) *Client {
	if strings.TrimSpace(endpoint) == "" {
		endpoint = defaultEndpoint
	}

	if lookbackDays <= 0 {
		lookbackDays = 365
	}

	return &Client{
		endpoint:     endpoint,
		margin:       margin,
		lookbackDays: lookbackDays,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) GetCreditRate(ctx context.Context) (float64, error) {
	toDate := time.Now()
	fromDate := toDate.AddDate(0, 0, -c.lookbackDays)

	soapRequest := buildSOAPRequest(fromDate, toDate)

	rawBody, err := c.sendRequest(ctx, soapRequest)
	if err != nil {
		return 0, err
	}

	keyRate, err := parseKeyRate(rawBody)
	if err != nil {
		return 0, err
	}

	return keyRate + c.margin, nil
}

func buildSOAPRequest(fromDate time.Time, toDate time.Time) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope 
	xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" 
	xmlns:xsd="http://www.w3.org/2001/XMLSchema" 
	xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
	<soap:Body>
		<KeyRate xmlns="http://web.cbr.ru/">
			<fromDate>%s</fromDate>
			<ToDate>%s</ToDate>
		</KeyRate>
	</soap:Body>
</soap:Envelope>`,
		fromDate.Format("2006-01-02"),
		toDate.Format("2006-01-02"),
	)
}

func (c *Client) sendRequest(ctx context.Context, soapRequest string) ([]byte, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.endpoint,
		bytes.NewBufferString(soapRequest),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create cbr request: %w", err)
	}

	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("SOAPAction", "http://web.cbr.ru/KeyRate")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send cbr request: %w", err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read cbr response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("cbr returned status %d: %s", resp.StatusCode, string(rawBody))
	}

	return rawBody, nil
}

func parseKeyRate(rawBody []byte) (float64, error) {
	doc := etree.NewDocument()

	if err := doc.ReadFromBytes(rawBody); err != nil {
		return 0, fmt.Errorf("failed to parse cbr xml: %w", err)
	}

	krElements := findElementsByLocalName(doc.Root(), "KR")
	if len(krElements) == 0 {
		return 0, errors.New("key rate data not found")
	}

	var latest *keyRatePoint

	for _, kr := range krElements {
		rateElement := findFirstChildByLocalName(kr, "Rate")
		if rateElement == nil {
			continue
		}

		rate, err := parseRate(rateElement.Text())
		if err != nil {
			continue
		}

		var rateDate time.Time

		dateElement := findFirstChildByLocalName(kr, "DT")
		if dateElement != nil {
			parsedDate, err := parseDate(dateElement.Text())
			if err == nil {
				rateDate = parsedDate
			}
		}

		point := &keyRatePoint{
			Date: rateDate,
			Rate: rate,
		}

		if latest == nil || point.Date.After(latest.Date) {
			latest = point
		}
	}

	if latest == nil {
		return 0, errors.New("valid key rate value not found")
	}

	return latest.Rate, nil
}

func parseRate(value string) (float64, error) {
	normalized := strings.TrimSpace(value)
	normalized = strings.ReplaceAll(normalized, ",", ".")

	return strconv.ParseFloat(normalized, 64)
}

func parseDate(value string) (time.Time, error) {
	normalized := strings.TrimSpace(value)

	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02",
	}

	var lastErr error

	for _, layout := range layouts {
		parsed, err := time.Parse(layout, normalized)
		if err == nil {
			return parsed, nil
		}

		lastErr = err
	}

	return time.Time{}, lastErr
}

func findElementsByLocalName(root *etree.Element, name string) []*etree.Element {
	if root == nil {
		return nil
	}

	result := make([]*etree.Element, 0)

	var walk func(element *etree.Element)

	walk = func(element *etree.Element) {
		if localName(element.Tag) == name {
			result = append(result, element)
		}

		for _, child := range element.ChildElements() {
			walk(child)
		}
	}

	walk(root)

	return result
}

func findFirstChildByLocalName(root *etree.Element, name string) *etree.Element {
	if root == nil {
		return nil
	}

	for _, child := range root.ChildElements() {
		if localName(child.Tag) == name {
			return child
		}
	}

	return nil
}

func localName(tag string) string {
	if index := strings.LastIndex(tag, ":"); index >= 0 {
		return tag[index+1:]
	}

	return tag
}
