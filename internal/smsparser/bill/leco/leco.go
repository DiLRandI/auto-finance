package leco

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type LecoParser struct{}

func (p *LecoParser) Parse(sms string) (*ElectricityBill, error) {
	bill := &ElectricityBill{}
	lines := strings.Split(sms, "\n")
	monthAbbr := map[string]string{
		"JAN": "Jan", "FEB": "Feb", "MAR": "Mar", "APR": "Apr",
		"MAY": "May", "JUN": "Jun", "JUL": "Jul", "AUG": "Aug",
		"SEP": "Sep", "OCT": "Oct", "NOV": "Nov", "DEC": "Dec",
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Remove leading '>' if present
		if strings.HasPrefix(line, ">") {
			line = strings.TrimSpace(line[1:])
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) < 2 {
			continue // Skip malformed lines
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "Read On":
			if t, err := parseDate(value, monthAbbr); err == nil {
				bill.ReadOn = t
			}
		case "Imp":
			parseReading(value, &bill.ImportReadingPrevious, &bill.ImportReadingCurrent, &bill.ImportUnits)
		case "Exp":
			parseReading(value, &bill.ExportReadingPrevious, &bill.ExportReadingCurrent, &bill.ExportUnits)
		case "Net Units":
			parseNetUnits(value, &bill.NetUnits, &bill.NetUnitsType)
		case "Monthly Bill":
			bill.MonthlyBill = parseAmount(value)
		case "Other Charges":
			bill.OtherCharges = parseAmount(value)
		case "SSCL":
			bill.SSCL = parseAmount(value)
		case "Opening Balance":
			parseBalanceWithDate(value, &bill.OpeningBalance, &bill.OpeningBalanceDate, monthAbbr)
		case "Total Payable":
			bill.TotalPayable = parseAmount(value)
		case "Last Payment":
			parsePayment(value, &bill.LastPaymentAmount, &bill.LastPaymentDate, monthAbbr)
		case "Last Amount Paid for Generation":
			bill.LastGenPayment = parseAmount(value)
		}
	}
	return bill, nil
}

// Helper functions
func parseDate(s string, monthMap map[string]string) (time.Time, error) {
	parts := strings.Split(s, "-")
	if len(parts) == 3 {
		if mon, ok := monthMap[parts[1]]; ok {
			s = fmt.Sprintf("%s-%s-%s", parts[0], mon, parts[2])
		}
	}
	return time.Parse("02-Jan-06", s)
}

func parseReading(s string, prev, curr, units *int) {
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == '-' || r == '=' })
	if len(parts) >= 3 {
		*prev, _ = strconv.Atoi(parts[0])
		*curr, _ = strconv.Atoi(parts[1])
		*units, _ = strconv.Atoi(parts[2])
	}
}

func parseNetUnits(s string, units *int, unitType *string) {
	if idx := strings.Index(s, "("); idx != -1 {
		*units, _ = strconv.Atoi(strings.TrimSpace(s[:idx]))
		*unitType = strings.Trim(s[idx+1:], ")")
	}
}

func parseAmount(s string) float64 {
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimPrefix(s, "Rs. ")
	s = strings.TrimSpace(s)
	val, _ := strconv.ParseFloat(s, 64)
	return val
}

func parseBalanceWithDate(s string, balance *float64, date *time.Time, monthMap map[string]string) {
	parts := strings.Split(s, " on ")
	if len(parts) >= 1 {
		*balance = parseAmount(parts[0])
	}
	if len(parts) >= 2 {
		t, _ := parseDate(parts[1], monthMap)
		*date = t
	}
}

func parsePayment(s string, amount *float64, date *time.Time, monthMap map[string]string) {
	parts := strings.Split(s, " on ")
	if len(parts) >= 1 {
		*amount = parseAmount(parts[0])
	}
	if len(parts) >= 2 {
		t, _ := parseDate(parts[1], monthMap)
		*date = t
	}
}
