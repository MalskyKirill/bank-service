package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"
)

func GenerateCardNumber() (string, error) {
	prefix := "2200"

	var builder strings.Builder

	builder.WriteString(prefix)

	for builder.Len() < 15 {
		digit, err := randomDigit()
		if err != nil {
			return "", err
		}

		builder.WriteString(fmt.Sprintf("%d", digit))
	}

	numberWithoutCheckDigit := builder.String()

	checkDigit, err := calculateLuhnCheckDigit(numberWithoutCheckDigit)
	if err != nil {
		return "", err
	}

	cardNumber := fmt.Sprintf("%s%d", numberWithoutCheckDigit, checkDigit)

	if !IsValidLuhn(cardNumber) {
		return "", fmt.Errorf("generated card number is invalid")
	}

	return cardNumber, nil
}

func GenerateCardExpiry() string {
	return time.Now().AddDate(4, 0, 0).Format("01/06")
}

func GenerateCVV() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%03d", n.Int64()), nil
}

func ComputeHMAC(data string, secret []byte) string {
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(data))

	return hex.EncodeToString(h.Sum(nil))
}

func VerifyHMAC(data string, expectedHMAC string, secret []byte) bool {
	actualHMAC := ComputeHMAC(data, secret)

	return hmac.Equal([]byte(actualHMAC), []byte(expectedHMAC))
}

func MaskCardNumber(number string) string {
	if len(number) < 10 {
		return number
	}

	return number[:6] + "******" + number[len(number)-4:]
}

func IsValidLuhn(number string) bool {
	sum := 0
	shouldDouble := false

	for i := len(number) - 1; i >= 0; i-- {
		if number[i] < '0' || number[i] > '9' {
			return false
		}

		digit := int(number[i] - '0')

		if shouldDouble {
			digit *= 2

			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
		shouldDouble = !shouldDouble
	}

	return sum%10 == 0
}

func calculateLuhnCheckDigit(numberWithoutCheckDigit string) (int, error) {
	sum := 0
	shouldDouble := true

	for i := len(numberWithoutCheckDigit) - 1; i >= 0; i-- {
		if numberWithoutCheckDigit[i] < '0' || numberWithoutCheckDigit[i] > '9' {
			return 0, fmt.Errorf("number contains non-digit characters")
		}

		digit := int(numberWithoutCheckDigit[i] - '0')

		if shouldDouble {
			digit *= 2

			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
		shouldDouble = !shouldDouble
	}

	return (10 - sum%10) % 10, nil
}

func randomDigit() (int, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(10))
	if err != nil {
		return 0, err
	}

	return int(n.Int64()), nil
}
